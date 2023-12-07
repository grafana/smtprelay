package main

import (
	"bufio"
	"bytes"
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

func heloChecker(peer smtpd.Peer, addr string) error {
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
		allowedAddr = strings.ToLower(allowedAddr)

		// Three cases for allowedAddr format:
		if idx := strings.Index(allowedAddr, "@"); idx == -1 {
			// 1. local address (no @) -- must match exactly
			if allowedAddr == addr {
				return true
			}
		} else {
			if idx != 0 {
				// 2. email address (user@domain.com) -- must match exactly
				if allowedAddr == addr {
					return true
				}
			} else {
				// 3. domain (@domain.com) -- must match addr domain
				allowedDomain := allowedAddr[idx+1:]
				if allowedDomain == domain {
					return true
				}
			}
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
				WithField("err", err).
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
			WithField("err", err).
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
					WithField("err", err).
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
					WithField("err", err).
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

func authChecker(peer smtpd.Peer, username string, password string) error {
	err := AuthCheckPassword(username, password)
	if err != nil {
		log.WithField("username", username).
			WithField("err", err).
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
			log.WithField("err", err).
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
		var smtpError smtpd.Error

		switch err.(type) {
		case *textproto.Error:
			err := err.(*textproto.Error)
			smtpError = smtpd.Error{Code: err.Code, Message: err.Msg}

			log.WithField("err_code", err.Code).
				WithField("err_msg", err.Msg).
				WithField("uuid", uniqueID).
				Error("delivery failed")
		default:
			smtpError = smtpd.Error{Code: 554, Message: "Forwarding failed"}

			log.WithField("err", err).
				WithField("uuid", uniqueID).
				Error("delivery failed")
		}

		durationHistogram.WithLabelValues(fmt.Sprintf("%v", smtpError.Code)).
			Observe(time.Now().Sub(start).Seconds())
		return observeErr(smtpError)
	}

	durationHistogram.WithLabelValues("none").
		Observe(time.Now().Sub(start).Seconds())

	log.WithField("host", remoteHost).
		WithField("uuid", uniqueID).
		Debug("delivery successful")

	return nil
}

func generateUUID() string {
	uniqueID, err := uuid.NewRandom()

	if err != nil {
		log.WithField("err", err).
			Error("could not generate UUIDv4")

		return ""
	}

	return uniqueID.String()
}

func main() {
	// load config as first thing
	ConfigLoad()

	// config is used here, call after config load
	go handleMetrics()

	// Cipher suites as defined in stock Go but without 3DES and RC4
	// https://golang.org/src/crypto/tls/cipher_suites.go
	var tlsCipherSuites = []uint16{
		tls.TLS_AES_128_GCM_SHA256,
		tls.TLS_AES_256_GCM_SHA384,
		tls.TLS_CHACHA20_POLY1305_SHA256,
		tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256,
		tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256,
		tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256,
		tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,
		tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA256,
		tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA,
		tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
		tls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA,
		tls.TLS_RSA_WITH_AES_128_GCM_SHA256, // does not provide PFS
		tls.TLS_RSA_WITH_AES_256_GCM_SHA384, // does not provide PFS
		tls.TLS_RSA_WITH_AES_128_CBC_SHA256,
		tls.TLS_RSA_WITH_AES_128_CBC_SHA,
		tls.TLS_RSA_WITH_AES_256_CBC_SHA,
	}

	if versionInfo {
		fmt.Printf("smtprelay/%s\n", VERSION)
		os.Exit(0)
	}

	// print version on start
	log.WithField("version", VERSION).
		Debug("starting smtprelay")

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
				log.WithField("err", err).
					WithField("file", allowedUsers).
					Fatal("cannot load allowed users file")
			}

			server.Authenticator = authChecker
		}

		if strings.Index(addresses[i], "://") == -1 {
			log.WithField("address", address).
				Info("listening on address")

			listener := listenOnAddr(server, address)
			listeners = append(listeners, listener)
		} else if strings.HasPrefix(addresses[i], "starttls://") {
			address = strings.TrimPrefix(address, "starttls://")

			if localCert == "" || localKey == "" {
				log.WithField("cert_file", localCert).
					WithField("key_file", localKey).
					Fatal("TLS certificate/key file not defined in config")
			}

			cert, err := tls.LoadX509KeyPair(localCert, localKey)
			if err != nil {
				log.WithField("error", err).
					Fatal("cannot load X509 keypair")
			}

			server.TLSConfig = &tls.Config{
				PreferServerCipherSuites: true,
				MinVersion:               tls.VersionTLS11,
				CipherSuites:             tlsCipherSuites,
				Certificates:             []tls.Certificate{cert},
			}
			server.ForceTLS = localForceTLS

			log.WithField("address", address).
				Info("listening on STARTTLS address")

			listener := listenOnAddr(server, address)
			listeners = append(listeners, listener)
		} else if strings.HasPrefix(addresses[i], "tls://") {

			address = strings.TrimPrefix(address, "tls://")

			if localCert == "" || localKey == "" {
				log.WithField("cert_file", localCert).
					WithField("key_file", localKey).
					Fatal("TLS certificate/key file not defined in config")
			}

			cert, err := tls.LoadX509KeyPair(localCert, localKey)
			if err != nil {
				log.WithField("error", err).
					Fatal("cannot load X509 keypair")
			}

			server.TLSConfig = &tls.Config{
				PreferServerCipherSuites: true,
				MinVersion:               tls.VersionTLS11,
				CipherSuites:             tlsCipherSuites,
				Certificates:             []tls.Certificate{cert},
			}

			log.WithField("address", address).
				Info("listening on TLS address")

			listener := listenOnTLSAddr(server, address, server.TLSConfig)
			listeners = append(listeners, listener)

		} else {
			log.WithField("address", address).
				Fatal("unknown protocol in address")
		}
	}

	handleSignals(listeners)
}

func listenOnAddr(server *smtpd.Server, addr string) net.Listener {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.WithField("err", err).
			WithField("address", addr).
			Fatal("could not listen on address")
	}

	go server.Serve(listener)
	return listener
}

func listenOnTLSAddr(server *smtpd.Server, addr string, config *tls.Config) net.Listener {
	listener, err := tls.Listen("tcp", addr, config)
	if err != nil {
		log.WithField("err", err).
			WithField("address", addr).
			Fatal("could not listen on address")
	}

	go server.Serve(listener)
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
				log.WithField("err", err).
					Warn("could not close listener")
			}
		}

		done <- true
	}()

	<-done
	os.Exit(0)
}
