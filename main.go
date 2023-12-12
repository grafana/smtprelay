package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
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
	"github.com/sirupsen/logrus"
)

func observeErr(err smtpd.Error) smtpd.Error {
	errorsCounter.WithLabelValues(fmt.Sprintf("%v", err.Code)).Inc()

	return err
}

func connectionChecker(peer smtpd.Peer) error {
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

	log.WithField("ip", peerIP).
		Warn("IP out of allowed network range")
	return observeErr(smtpd.Error{Code: 421, Message: "Denied - IP out of allowed network range"})
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

func senderChecker(peer smtpd.Peer, addr string) error {
	if allowedSender == "" {
		// disable sender check, allow anyone to send mail
		return nil
	}

	// check sender address from auth file if user is authenticated
	if allowedUsers != "" && peer.Username != "" {
		user, err := AuthFetch(peer.Username)
		if err != nil {
			log.WithField("sender_address", addr).
				WithError(err).
				Warn("sender address not allowed")
			return observeErr(smtpd.Error{Code: 451, Message: "sender address not allowed"})
		}

		if !addrAllowed(addr, user.allowedAddresses) {
			log.WithField("sender_address", addr).
				Warn("sender address not allowed")
			return observeErr(smtpd.Error{Code: 451, Message: "sender address not allowed"})
		}
	}

	re, err := regexp.Compile(allowedSender)
	if err != nil {
		log.WithField("allowed_sender", allowedSender).
			WithError(err).
			Warn("allowed_sender invalid")
		return observeErr(smtpd.Error{Code: 451, Message: "sender address not allowed"})
	}

	if re.MatchString(addr) {
		return nil
	}

	log.WithField("sender_address", addr).
		Warn("sender address not allowed")
	return observeErr(smtpd.Error{Code: 451, Message: "sender address not allowed"})
}

func recipientChecker(allowed, denied string) func(peer smtpd.Peer, addr string) error {
	return func(peer smtpd.Peer, addr string) error {
		// First, we check the deny list as that one takes precedence.
		if denied != "" {
			deniedRegexp, err := regexp.Compile(denied)
			if err != nil {
				log.WithField("denied_recipients", denied).
					WithError(err).
					Warn("denied_recipients invalid")
				return observeErr(smtpd.Error{Code: 451, Message: "Invalid recipient address"})
			}

			if deniedRegexp.MatchString(addr) {
				log.WithField("address", addr).Warn("receipt address is part of the deny list")
				return observeErr(smtpd.Error{Code: 451, Message: "Denied recipient address"})
			}
		}

		// Then, we check the allow list.
		if allowed != "" {
			allowedRegexp, err := regexp.Compile(allowed)
			if err != nil {
				log.WithField("allow_recipients", allowed).
					WithError(err).
					Warn("allowed_recipients invalid")
				return observeErr(smtpd.Error{Code: 451, Message: "Invalid recipient address"})
			}

			if allowedRegexp.MatchString(addr) {
				return nil
			}

			log.WithField("address", addr).Warn("Invalid recipient address")
			return observeErr(smtpd.Error{Code: 451, Message: "Invalid recipient address"})
		}

		// No deny nor allow list, receipient check disabled.
		return nil
	}
}

func authChecker(_ smtpd.Peer, username string, password string) error {
	err := AuthCheckPassword(username, password)
	if err != nil {
		log.WithField("username", username).
			WithError(err).
			Warn("auth error")

		return observeErr(smtpd.Error{Code: 535, Message: "Authentication credentials invalid"})
	}
	return nil
}

func addLogHeaderFields(logHeaders map[string]string, log *logrus.Entry, data []byte) (*logrus.Entry, error) {
	buf := bufio.NewReader(bytes.NewReader(data))
	headers, err := textproto.NewReader(buf).ReadMIMEHeader()
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("readMIMEHeader: %w", err)
	}

	for field, hdrname := range logHeaders {
		val := headers.Get(hdrname)
		if val != "" {
			// we assume a single value for the header, and get the first
			log = log.WithField(field, val)
		}
	}

	return log, nil
}

func mailHandler(peer smtpd.Peer, env smtpd.Envelope) error {
	uniqueID := generateUUID()

	peerIP := ""
	if addr, ok := peer.Addr.(*net.TCPAddr); ok {
		peerIP = addr.IP.String()
	}

	// parse headers from data if we need to log any of them
	var err error
	deliveryLog := log.WithField("from", env.Sender).
		WithField("to", env.Recipients).
		WithField("peer", peerIP).
		WithField("host", remoteHost).
		WithField("uuid", uniqueID)
	if len(logHeaders) > 0 {
		deliveryLog, err = addLogHeaderFields(logHeaders, deliveryLog, env.Data)
		if err != nil {
			log.WithError(err).
				WithField("uuid", uniqueID).
				Warn("could not parse headers")
		}
	}

	deliveryLog.Info("delivering mail from peer using smarthost")

	var auth smtp.Auth
	host, _, _ := net.SplitHostPort(remoteHost)

	if remoteUser != "" && remotePass != "" {
		switch remoteAuth {
		case "plain":
			auth = smtp.PlainAuth("", remoteUser, remotePass, host)
		case "login":
			auth = LoginAuth(remoteUser, remotePass)
		default:
			return observeErr(smtpd.Error{Code: 530, Message: "Authentication method not supported"})
		}
	}

	env.AddReceivedLine(peer)

	var sender string

	if remoteSender == "" {
		sender = env.Sender
	} else {
		sender = remoteSender
	}

	msgSizeHistogram.Observe(float64(len(env.Data)))

	start := time.Now()
	err = SendMail(
		remoteHost,
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

			log.WithField("err_code", tperr.Code).
				WithField("err_msg", tperr.Msg).
				WithField("uuid", uniqueID).
				Error("delivery failed")
		} else {
			smtpError = smtpd.Error{Code: 554, Message: "Forwarding failed for message ID " + uniqueID}

			log.WithError(err).
				WithField("uuid", uniqueID).
				Error("delivery failed")
		}

		durationHistogram.WithLabelValues(fmt.Sprintf("%v", smtpError.Code)).
			Observe(time.Since(start).Seconds())
		return observeErr(smtpError)
	}

	durationHistogram.WithLabelValues("none").
		Observe(time.Since(start).Seconds())

	log.WithField("host", remoteHost).
		WithField("uuid", uniqueID).
		Debug("delivery successful")

	return nil
}

func generateUUID() string {
	uniqueID, err := uuid.NewRandom()

	if err != nil {
		log.WithError(err).
			Error("could not generate UUIDv4")

		return ""
	}

	return uniqueID.String()
}

// metrics registry - overridable for tests
var metricsRegistry = prometheus.DefaultRegisterer

func main() {
	// load config as first thing
	err := ConfigLoad()
	if err != nil {
		log.WithError(err).
			Fatal("error loading config")
	}

	if versionInfo {
		fmt.Printf("smtprelay/%s\n", VERSION)
		return
	}

	// print version on start
	log.WithField("version", VERSION).Debug("starting smtprelay")

	if err := run(); err != nil {
		log.WithError(err).
			Fatal("error running smtprelay")
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM)
	defer stop()

	// config is used here, call after config load
	metricsSrv, err := handleMetrics(ctx, metricsListen, metricsRegistry)
	if err != nil {
		return fmt.Errorf("could not start metrics server: %w", err)
	}
	defer metricsSrv.Stop()

	var listeners []net.Listener
	addresses := strings.Split(listen, " ")

	for i := range addresses {
		address := addresses[i]

		server := &smtpd.Server{
			Hostname:          hostName,
			WelcomeMessage:    welcomeMsg,
			HeloChecker:       heloChecker,
			ConnectionChecker: connectionChecker,
			SenderChecker:     senderChecker,
			RecipientChecker:  recipientChecker(allowedRecipients, deniedRecipients),
			Handler:           mailHandler,
			MaxMessageSize:    maxMessageSize,
			MaxConnections:    maxConnections,
			MaxRecipients:     maxRecipients,
			ReadTimeout:       readTimeout,
			WriteTimeout:      writeTimeout,
			DataTimeout:       dataTimeout,
		}

		if allowedUsers != "" {
			err := AuthLoadFile(allowedUsers)
			if err != nil {
				return fmt.Errorf("cannot load allowed users file %q: %w", allowedUsers, err)
			}

			server.Authenticator = authChecker
		}

		switch {
		case !strings.Contains(addresses[i], "://"):
			log.WithField("address", address).Info("listening on address")

			listener := listenOnAddr(server, address)
			listeners = append(listeners, listener)
		case strings.HasPrefix(addresses[i], "starttls://"):
			tlsConfig, err := getServerTLSConfig(localCert, localKey)
			if err != nil {
				log.WithField("error", err).
					Fatal("error getting Server TLS config")
			}

			server.TLSConfig = tlsConfig
			server.ForceTLS = localForceTLS

			address = strings.TrimPrefix(address, "starttls://")
			log.WithField("address", address).Info("listening on STARTTLS address")

			listener := listenOnAddr(server, address)
			listeners = append(listeners, listener)
		case strings.HasPrefix(addresses[i], "tls://"):
			tlsConfig, err := getServerTLSConfig(localCert, localKey)
			if err != nil {
				log.WithField("error", err).
					Fatal("error getting Server TLS config")
			}

			server.TLSConfig = tlsConfig

			address = strings.TrimPrefix(address, "tls://")
			log.WithField("address", address).Info("listening on TLS address")

			listener := listenOnTLSAddr(server, address, server.TLSConfig)
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

func listenOnAddr(server *smtpd.Server, addr string) net.Listener {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.WithError(err).
			WithField("address", addr).
			Fatal("could not listen on address")
	}

	go func() {
		_ = server.Serve(listener)
	}()
	return listener
}

func listenOnTLSAddr(server *smtpd.Server, addr string, config *tls.Config) net.Listener {
	listener, err := tls.Listen("tcp", addr, config)
	if err != nil {
		log.WithError(err).
			WithField("address", addr).
			Fatal("could not listen on address")
	}

	go func() {
		_ = server.Serve(listener)
	}()
	return listener
}

func handleSignals(servers []net.Listener) {
	sigs := make(chan os.Signal, 1)
	done := make(chan bool, 1)

	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigs

		for _, s := range servers {
			log.WithField("signal", sig).
				WithField("addr", s.Addr().String()).
				Warn("closing listener in response to received signal")

			err := s.Close()
			if err != nil {
				log.WithError(err).
					Warn("could not close listener")
			}
		}

		done <- true
	}()

	<-done
	os.Exit(0)
}
