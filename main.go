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
	"os"
	"os/signal"
	"regexp"
	"strings"
	"syscall"
	"time"

	"github.com/chrj/smtpd"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/version"
)

func observeErr(err smtpd.Error) smtpd.Error {
	errorsCounter.WithLabelValues(fmt.Sprintf("%v", err.Code)).Inc()

	return err
}

func connectionChecker(allowedNets []*net.IPNet) func(peer smtpd.Peer) error {
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

		slog.Default().Warn("IP out of allowed network range", slog.String("ip", peerIP.String()))

		return observeErr(smtpd.Error{Code: 421, Message: "Denied - IP out of allowed network range"})
	}
}

func heloChecker(_ smtpd.Peer, _ string) error {
	// every SMTP request starts with a HELO
	requestsCounter.Inc()

	return nil
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

func senderChecker(allowedSender, allowedUsers string) func(peer smtpd.Peer, addr string) error {
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

func recipientChecker(allowed, denied string) func(peer smtpd.Peer, addr string) error {
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

func authChecker(_ smtpd.Peer, username string, password string) error {
	err := AuthCheckPassword(username, password)
	if err != nil {
		slog.Default().Warn("auth error", slog.String("component", "auth_checker"),
			slog.String("username", username), slog.Any("error", err))

		return observeErr(smtpd.Error{Code: 535, Message: "Authentication credentials invalid"})
	}
	return nil
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

func mailHandler(cfg *config) func(peer smtpd.Peer, env smtpd.Envelope) error {
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

func generateUUID() string {
	uniqueID, err := uuid.NewRandom()
	if err != nil {
		slog.Default().Error("could not generate UUIDv4", slog.Any("error", err))

		return ""
	}

	return uniqueID.String()
}

// metrics registry - overridable for tests
var metricsRegistry = prometheus.DefaultRegisterer

const applicationName = "smtprelay"

func main() {
	// load config as first thing
	cfg, err := loadConfig()
	if err != nil {
		slog.Default().Error("error loading config", slog.Any("error", err))
		os.Exit(1)
	}

	if cfg.versionInfo {
		fmt.Printf("%s %s\n", applicationName, version.Info())
		return
	}

	logger := slog.Default()

	// print version on start
	logger.Debug("config loaded", slog.String("version", version.Version))

	if err := run(cfg); err != nil {
		logger.Error("error running smtprelay", slog.Any("error", err))
		os.Exit(1)
	}
}

func run(cfg *config) error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM)
	defer stop()

	// config is used here, call after config load
	metricsSrv, err := handleMetrics(ctx, cfg.metricsListen, metricsRegistry)
	if err != nil {
		return fmt.Errorf("could not start metrics server: %w", err)
	}
	defer metricsSrv.Stop()

	logger := slog.Default()

	var listeners []net.Listener
	addresses := strings.Split(cfg.listen, " ")

	for i := range addresses {
		address := addresses[i]

		server := &smtpd.Server{
			Hostname:          cfg.hostName,
			WelcomeMessage:    cfg.welcomeMsg,
			HeloChecker:       heloChecker,
			ConnectionChecker: connectionChecker(cfg.allowedNets),
			SenderChecker:     senderChecker(cfg.allowedSender, cfg.allowedUsers),
			RecipientChecker:  recipientChecker(cfg.allowedRecipients, cfg.deniedRecipients),
			Handler:           mailHandler(cfg),
			MaxMessageSize:    cfg.maxMessageSize,
			MaxConnections:    cfg.maxConnections,
			MaxRecipients:     cfg.maxRecipients,
			ReadTimeout:       cfg.readTimeout,
			WriteTimeout:      cfg.writeTimeout,
			DataTimeout:       cfg.dataTimeout,
		}

		if cfg.allowedUsers != "" {
			err := AuthLoadFile(cfg.allowedUsers)
			if err != nil {
				return fmt.Errorf("cannot load allowed users file %q: %w", cfg.allowedUsers, err)
			}

			server.Authenticator = authChecker
		}

		switch {
		case !strings.Contains(addresses[i], "://"):
			logger.Info("listening on address", slog.String("address", address))

			listener, err := listenOnAddr(server, address)
			if err != nil {
				return fmt.Errorf("error listening on address %q: %w", address, err)
			}

			listeners = append(listeners, listener)
		case strings.HasPrefix(addresses[i], "starttls://"):
			tlsConfig, err := getServerTLSConfig(cfg.localCert, cfg.localKey)
			if err != nil {
				return fmt.Errorf("error getting Server TLS config: %w", err)
			}

			server.TLSConfig = tlsConfig
			server.ForceTLS = cfg.localForceTLS

			address = strings.TrimPrefix(address, "starttls://")
			logger.Info("listening on STARTTLS address", slog.String("address", address))

			listener, err := listenOnAddr(server, address)
			if err != nil {
				return fmt.Errorf("error listening on address %q: %w", address, err)
			}

			listeners = append(listeners, listener)
		case strings.HasPrefix(addresses[i], "tls://"):
			tlsConfig, err := getServerTLSConfig(cfg.localCert, cfg.localKey)
			if err != nil {
				return fmt.Errorf("error getting Server TLS config: %w", err)
			}

			server.TLSConfig = tlsConfig

			address = strings.TrimPrefix(address, "tls://")

			logger.Info("listening on TLS address", slog.String("address", address))

			listener, err := listenOnTLSAddr(server, address, server.TLSConfig)
			if err != nil {
				return fmt.Errorf("error listening on address %q: %w", address, err)
			}

			listeners = append(listeners, listener)
		default:
			return fmt.Errorf("unknown protocol in address %q", address)
		}
	}

	handleSignals(listeners)

	return nil
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

func getClientTLSConfig(serverName string) *tls.Config {
	return &tls.Config{
		MinVersion: tls.VersionTLS12,
		ServerName: serverName,
	}
}

func listenOnAddr(server *smtpd.Server, addr string) (net.Listener, error) {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("could not listen on address %q: %w", addr, err)
	}

	go func() {
		_ = server.Serve(listener)
	}()
	return listener, nil
}

func listenOnTLSAddr(server *smtpd.Server, addr string, config *tls.Config) (net.Listener, error) {
	listener, err := tls.Listen("tcp", addr, config)
	if err != nil {
		return nil, fmt.Errorf("could not listen on address %q: %w", addr, err)
	}

	go func() {
		_ = server.Serve(listener)
	}()
	return listener, nil
}

func handleSignals(servers []net.Listener) {
	sigs := make(chan os.Signal, 1)
	done := make(chan bool, 1)

	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigs

		logger := slog.Default().With(slog.String("component", "signal_handler"))

		for _, s := range servers {
			logger.Warn("closing listener in response to received signal",
				slog.String("signal", sig.String()), slog.String("addr", s.Addr().String()))

			err := s.Close()
			if err != nil {
				logger.Warn("could not close listener", slog.Any("error", err))
			}
		}

		done <- true
	}()

	<-done
}
