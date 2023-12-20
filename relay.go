package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/smtp"
	"net/textproto"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/grafana/smtprelay/internal/smtpd"
)

// relay is an SMTP relay server which can listen on a single address
type relay struct {
	server *smtpd.Server

	logger *slog.Logger

	cfg *config
}

func newRelay(logger *slog.Logger, cfg *config) (*relay, error) {
	r := &relay{
		cfg: cfg,
	}

	if logger == nil {
		r.logger = slog.Default()
	}

	r.server = &smtpd.Server{
		HeloChecker:       r.heloChecker,
		ConnectionChecker: r.connectionChecker(cfg.allowedNets),
		SenderChecker:     r.senderChecker(cfg.allowedSender, cfg.allowedUsers),
		RecipientChecker:  r.recipientChecker(cfg.allowedRecipients, cfg.deniedRecipients),
		Handler:           r.mailHandler(cfg),

		Hostname:       cfg.hostName,
		WelcomeMessage: cfg.welcomeMsg,
		MaxMessageSize: cfg.maxMessageSize,
		MaxConnections: cfg.maxConnections,
		MaxRecipients:  cfg.maxRecipients,
		ReadTimeout:    cfg.readTimeout,
		WriteTimeout:   cfg.writeTimeout,
		DataTimeout:    cfg.dataTimeout,
	}

	if cfg.allowedUsers != "" {
		err := AuthLoadFile(cfg.allowedUsers)
		if err != nil {
			return nil, fmt.Errorf("cannot load allowed users file %q: %w", cfg.allowedUsers, err)
		}

		r.server.Authenticator = r.authChecker
	}

	return r, nil
}

func (r *relay) serve(ln net.Listener) error {
	return r.server.Serve(ln)
}

func (r *relay) shutdown(ctx context.Context) error {
	// context propagation isn't yet implemented in smtpd, so let's build in
	// a timeout here
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()

	// shutdown without a wait - we'll wait asynchronously after
	err := r.server.Shutdown(false)
	if err != nil {
		return err
	}

	go func() {
		// wait for the server to shut down, then cancel the context
		_ = r.server.Wait()

		if ctx.Err() == nil {
			cancel()
		}
	}()

	<-ctx.Done()

	return nil
}

func (r *relay) listen(address string) (net.Listener, error) {
	var ln net.Listener

	switch {
	case !strings.Contains(address, "://"):
		listener, err := net.Listen("tcp", address)
		if err != nil {
			return nil, fmt.Errorf("could not listen on address %q: %w", address, err)
		}
		ln = listener
	case strings.HasPrefix(address, "starttls://"):
		tlsConfig, err := getServerTLSConfig(r.cfg.localCert, r.cfg.localKey)
		if err != nil {
			return nil, fmt.Errorf("error getting Server TLS config: %w", err)
		}

		r.server.TLSConfig = tlsConfig
		r.server.ForceTLS = r.cfg.localForceTLS

		address = strings.TrimPrefix(address, "starttls://")

		listener, err := net.Listen("tcp", address)
		if err != nil {
			return nil, fmt.Errorf("could not listen on address %q: %w", address, err)
		}
		ln = listener
	case strings.HasPrefix(address, "tls://"):
		// TODO: deprecate this in favor of starttls://
		tlsConfig, err := getServerTLSConfig(r.cfg.localCert, r.cfg.localKey)
		if err != nil {
			return nil, fmt.Errorf("error getting Server TLS config: %w", err)
		}

		r.server.TLSConfig = tlsConfig

		address = strings.TrimPrefix(address, "tls://")

		listener, err := tls.Listen("tcp", address, tlsConfig)
		if err != nil {
			return nil, fmt.Errorf("could not listen on address %q: %w", address, err)
		}
		ln = listener
	default:
		return nil, fmt.Errorf("unknown protocol in address %q", address)
	}

	return ln, nil
}

func (r *relay) authChecker(_ smtpd.Peer, username string, password string) error {
	err := AuthCheckPassword(username, password)
	if err != nil {
		slog.Default().Warn("auth error", slog.String("component", "auth_checker"),
			slog.String("username", username), slog.Any("error", err))

		return observeErr(smtpd.Error{Code: 535, Message: "Authentication credentials invalid"})
	}
	return nil
}

func (r *relay) heloChecker(_ smtpd.Peer, _ string) error {
	// every SMTP request starts with a HELO
	requestsCounter.Inc()

	return nil
}

func (r *relay) connectionChecker(allowedNets []*net.IPNet) func(peer smtpd.Peer) error {
	return func(peer smtpd.Peer) error {
		// This can't panic because we only have TCP listeners
		peerIP := peer.Addr.(*net.TCPAddr).IP

		if len(allowedNets) == 0 {
			// Special case: empty string means allow everything
			return nil
		}

		for _, allowedNet := range allowedNets {
			if allowedNet.Contains(peerIP) {
				return nil
			}
		}

		r.logger.Warn("IP out of allowed network range", slog.String("ip", peerIP.String()))

		return observeErr(smtpd.Error{Code: 421, Message: "Denied - IP out of allowed network range"})
	}
}

func (r *relay) senderChecker(allowedSender, allowedUsers string) func(peer smtpd.Peer, addr string) error {
	return func(peer smtpd.Peer, addr string) error {
		if allowedSender == "" {
			// disable sender check, allow anyone to send mail
			return nil
		}

		log := slog.Default().With(slog.String("sender_address", addr))

		// check sender address from auth file if user is authenticated
		if allowedUsers != "" && peer.Username != "" {
			user, err := AuthFetch(peer.Username)
			if err != nil {
				log.Warn("sender address not allowed", slog.Any("error", err))
				return observeErr(smtpd.Error{Code: 451, Message: "sender address not allowed"})
			}

			if !addrAllowed(addr, user.allowedAddresses) {
				log.Warn("sender address not allowed")
				return observeErr(smtpd.Error{Code: 451, Message: "sender address not allowed"})
			}
		}

		// TODO: precompile this regexp and reject it at config time
		re, err := regexp.Compile(allowedSender)
		if err != nil {
			log.Warn("allowed_sender invalid", slog.Any("error", err), slog.String("allowed_sender", allowedSender))
			return observeErr(smtpd.Error{Code: 451, Message: "sender address not allowed"})
		}

		if re.MatchString(addr) {
			return nil
		}

		log.Warn("sender address not allowed")

		return observeErr(smtpd.Error{Code: 451, Message: "sender address not allowed"})
	}
}

func (r *relay) recipientChecker(allowed, denied string) func(peer smtpd.Peer, addr string) error {
	log := slog.Default().With(slog.String("component", "recipient_checker"))

	return func(peer smtpd.Peer, addr string) error {
		// First, we check the deny list as that one takes precedence.
		if denied != "" {
			deniedRegexp, err := regexp.Compile(denied)
			if err != nil {
				log.Warn("denied_recipients invalid", slog.String("denied_recipients", denied), slog.Any("error", err))

				return observeErr(smtpd.Error{Code: 451, Message: "Invalid recipient address"})
			}

			if deniedRegexp.MatchString(addr) {
				log.Warn("receipt address is part of the deny list", slog.String("address", addr))
				return observeErr(smtpd.Error{Code: 451, Message: "Denied recipient address"})
			}
		}

		// Then, we check the allow list.
		if allowed != "" {
			allowedRegexp, err := regexp.Compile(allowed)
			if err != nil {
				log.Warn("allowed_recipients invalid", slog.String("allowed_recipients", allowed), slog.Any("error", err))
				return observeErr(smtpd.Error{Code: 451, Message: "Invalid recipient address"})
			}

			if allowedRegexp.MatchString(addr) {
				return nil
			}

			log.Warn("Invalid recipient address", slog.String("address", addr))
			return observeErr(smtpd.Error{Code: 451, Message: "Invalid recipient address"})
		}

		// No deny nor allow list, receipient check disabled.
		return nil
	}
}

func (r *relay) mailHandler(cfg *config) func(peer smtpd.Peer, env smtpd.Envelope) error {
	return func(peer smtpd.Peer, env smtpd.Envelope) error {
		uniqueID := generateUUID()

		peerIP := ""
		if addr, ok := peer.Addr.(*net.TCPAddr); ok {
			peerIP = addr.IP.String()
		}

		logger := slog.Default().With(
			slog.String("component", "mail_handler"),
			slog.String("uuid", uniqueID),
		)

		// parse headers from data if we need to log any of them
		var err error
		deliveryLog := logger.With(
			slog.String("from", env.Sender),
			slog.Any("to", env.Recipients),
			slog.String("peer", peerIP),
			slog.String("host", cfg.remoteHost),
		)
		if len(cfg.logHeaders) > 0 {
			deliveryLog, err = addLogHeaderFields(cfg.logHeaders, deliveryLog, env.Data)
			if err != nil {
				logger.Warn("could not parse headers", slog.Any("error", err))
			}
		}

		deliveryLog.Info("delivering mail from peer using smarthost")

		var auth smtp.Auth
		host, _, _ := net.SplitHostPort(cfg.remoteHost)

		if cfg.remoteUser != "" && cfg.remotePass != "" {
			switch cfg.remoteAuth {
			case "plain":
				auth = smtp.PlainAuth("", cfg.remoteUser, cfg.remotePass, host)
			case "login":
				auth = LoginAuth(cfg.remoteUser, cfg.remotePass)
			default:
				return observeErr(smtpd.Error{Code: 530, Message: "Authentication method not supported"})
			}
		}

		env.AddReceivedLine(peer)

		var sender string

		if cfg.remoteSender == "" {
			sender = env.Sender
		} else {
			sender = cfg.remoteSender
		}

		msgSizeHistogram.Observe(float64(len(env.Data)))

		start := time.Now()
		err = SendMail(
			cfg.remoteHost,
			auth,
			sender,
			env.Recipients,
			env.Data,
		)

		if err != nil {
			err = fmt.Errorf("sendMail: %w", err)

			var smtpError smtpd.Error
			var tperr *textproto.Error

			if errors.As(err, &tperr) {
				smtpError = smtpd.Error{Code: tperr.Code, Message: tperr.Msg}

				logger.Error("delivery failed",
					slog.Int("err_code", tperr.Code), slog.String("err_msg", tperr.Msg))
			} else {
				smtpError = smtpd.Error{Code: 554, Message: "Forwarding failed for message ID " + uniqueID}

				logger.Error("delivery failed", slog.Any("error", err))
			}

			durationHistogram.WithLabelValues(fmt.Sprintf("%v", smtpError.Code)).
				Observe(time.Since(start).Seconds())
			return observeErr(smtpError)
		}

		durationHistogram.WithLabelValues("none").
			Observe(time.Since(start).Seconds())

		logger.Debug("delivery successful", slog.String("host", cfg.remoteHost))

		return nil
	}
}

func observeErr(err smtpd.Error) smtpd.Error {
	errorsCounter.WithLabelValues(fmt.Sprintf("%v", err.Code)).Inc()

	return err
}

func addrAllowed(addr string, allowedAddrs []string) bool {
	if allowedAddrs == nil {
		// If absent, all addresses are allowed
		return true
	}

	addr = strings.ToLower(addr)

	// Extract optional domain part
	domain := ""
	if idx := strings.LastIndex(addr, "@"); idx != -1 {
		domain = strings.ToLower(addr[idx+1:])
	}

	// Test each address from allowedUsers file
	for _, allowedAddr := range allowedAddrs {
		if matchAddr(allowedAddr, addr, domain) {
			return true
		}
	}

	return false
}

func matchAddr(allowedAddr, addr, domain string) bool {
	allowedAddr = strings.ToLower(allowedAddr)

	// Three cases for allowedAddr format:
	idx := strings.Index(allowedAddr, "@")
	switch {
	case idx == -1:
		// 1. local address (no @) -- must match exactly
		if allowedAddr == addr {
			return true
		}
	case idx != 0:
		// 2. email address (user@domain.com) -- must match exactly
		if allowedAddr == addr {
			return true
		}
	default:
		// 3. domain (@domain.com) -- must match addr domain
		allowedDomain := allowedAddr[idx+1:]
		if allowedDomain == domain {
			return true
		}
	}

	return false
}

func addLogHeaderFields(logHeaders map[string]string, log *slog.Logger, data []byte) (*slog.Logger, error) {
	buf := bufio.NewReader(bytes.NewReader(data))
	headers, err := textproto.NewReader(buf).ReadMIMEHeader()
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("readMIMEHeader: %w", err)
	}

	for field, hdrname := range logHeaders {
		val := headers.Get(hdrname)
		if val != "" {
			// we assume a single value for the header, and get the first
			log = log.With(slog.String(field, val))
		}
	}

	return log, nil
}

func generateUUID() string {
	uniqueID, err := uuid.NewRandom()
	if err != nil {
		slog.Default().Error("could not generate UUIDv4", slog.Any("error", err))

		return ""
	}

	return uniqueID.String()
}

func getServerTLSConfig(certpath, keypath string) (*tls.Config, error) {
	if certpath == "" {
		return nil, fmt.Errorf("empty local_cert")
	}

	if keypath == "" {
		return nil, fmt.Errorf("empty local_key")
	}

	cert, err := tls.LoadX509KeyPair(certpath, keypath)
	if err != nil {
		return nil, fmt.Errorf("cannot load X509 keypair: %w", err)
	}

	return &tls.Config{
		MinVersion:   tls.VersionTLS12,
		Certificates: []tls.Certificate{cert},
	}, nil
}
