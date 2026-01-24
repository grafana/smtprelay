package main

import (
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"os"
	"strings"
	"time"

	"github.com/vharitonsky/iniflags"
)

//nolint:govet
type config struct {
	logFormat                  string
	hostName                   string
	welcomeMsg                 string
	listen                     string
	metricsListen              string
	localCert                  string
	localKey                   string
	localForceTLS              bool
	allowedNetsStr             string
	allowedSender              string
	allowedRecipients          string
	deniedRecipients           string
	allowedUsers               string
	remoteHost                 string
	remoteUser                 string
	maxMessageSize             int
	maxConnections             int
	maxRecipients              int
	readTimeout                time.Duration
	writeTimeout               time.Duration
	dataTimeout                time.Duration
	remotePass                 string
	remoteAuth                 string
	remoteSender               string
	versionInfo                bool
	logLevel                   string
	logHeadersStr              string
	rateLimitEnabled           bool
	rateLimitMessagesPerSecond float64
	rateLimitBurst             int
	rateLimitHeader            string
	xoauth2ClientID            string
	xoauth2ClientSecret        string
	xoauth2TokenURL            string
	xoauth2RefreshToken        string
	allowedNets                []*net.IPNet
	logHeaders                 map[string]string
}

func setupAllowedNetworks(s string) ([]*net.IPNet, error) {
	nets := []*net.IPNet{}

	for _, netstr := range splitstr(s, ' ') {
		baseIP, allowedNet, err := net.ParseCIDR(netstr)
		if err != nil {
			return nil, fmt.Errorf("parseCIDR %q: %w", netstr, err)
		}

		// Reject any network specification where any host bits are set,
		// meaning the address refers to a host and not a network.
		if !allowedNet.IP.Equal(baseIP) {
			return nil, fmt.Errorf("invalid network (host bits set): %q", netstr)
		}

		nets = append(nets, allowedNet)
	}

	return nets, nil
}

func loadConfig() (*config, error) {
	cfg := config{}
	registerFlags(flag.CommandLine, &cfg)

	iniflags.Parse()

	setupLogger(cfg.logFormat, cfg.logLevel)

	logger := slog.With(slog.String("component", "config"))

	// if remotePass is not set, try reading it from env var
	if cfg.remotePass == "" {
		logger.Debug("remote_pass not set, trying REMOTE_PASS env var")
		cfg.remotePass = os.Getenv("REMOTE_PASS")
		if cfg.remotePass != "" {
			logger.Debug("found data in REMOTE_PASS env var")
		} else {
			logger.Debug("no data found in REMOTE_PASS env var")
		}
	}

	//nolint:nestif
	if cfg.remoteAuth == "xoauth2" {
		if cfg.remoteUser == "" {
			return nil, errors.New("remote_user is required for xoauth2 authentication")
		}
		if cfg.xoauth2ClientID == "" {
			return nil, errors.New("xoauth2_client_id is required for xoauth2 authentication")
		}
		if cfg.xoauth2ClientSecret == "" {
			return nil, errors.New("xoauth2_client_secret is required for xoauth2 authentication")
		}
		if cfg.xoauth2RefreshToken == "" {
			return nil, errors.New("xoauth2_refresh_token is required for xoauth2 authentication")
		}
		if cfg.xoauth2TokenURL == "" {
			return nil, errors.New("xoauth2_token_url is required for xoauth2 authentication")
		}
	}

	allowedNets, err := setupAllowedNetworks(cfg.allowedNetsStr)
	if err != nil {
		return nil, fmt.Errorf("setupAllowedNetworks: %w", err)
	}
	cfg.allowedNets = allowedNets

	cfg.logHeaders = parseLogHeaders(cfg.logHeadersStr)

	return &cfg, nil
}

func registerFlags(f *flag.FlagSet, cfg *config) {
	f.StringVar(&cfg.logFormat, "log_format", "json", "Log format - json or logfmt")
	f.StringVar(&cfg.hostName, "hostname", "localhost.localdomain", "Server hostname")
	f.StringVar(&cfg.welcomeMsg, "welcome_msg", "", "Welcome message for SMTP session")
	f.StringVar(&cfg.listen, "listen", "127.0.0.1:25 [::1]:25", "Address and port to listen for incoming SMTP")
	f.StringVar(&cfg.metricsListen, "metrics_listen", ":8080", "Address and port to listen for metrics exposition")
	f.StringVar(&cfg.localCert, "local_cert", "", "SSL certificate for STARTTLS/TLS")
	f.StringVar(&cfg.localKey, "local_key", "", "SSL private key for STARTTLS/TLS")
	f.BoolVar(&cfg.localForceTLS, "local_forcetls", false, "Force STARTTLS (needs local_cert and local_key)")
	f.StringVar(&cfg.allowedNetsStr, "allowed_nets", "127.0.0.0/8 ::/128", "Networks allowed to send mails (set to \"\" to disable")
	f.StringVar(&cfg.allowedSender, "allowed_sender", "", "Regular expression for valid FROM email addresses (leave empty to allow any sender)")
	f.StringVar(&cfg.allowedRecipients, "allowed_recipients", "", "Regular expression for valid 'to' email addresses (leave empty to allow any recipient)")
	f.StringVar(&cfg.deniedRecipients, "denied_recipients", "", "Regular expression for email addresses for which will never deliver any emails.")
	f.StringVar(&cfg.allowedUsers, "allowed_users", "", "Path to file with valid users/passwords (leave empty to allow any user)")
	f.StringVar(&cfg.remoteHost, "remote_host", "smtp.gmail.com:587", "Outgoing SMTP server")
	f.StringVar(&cfg.remoteUser, "remote_user", "", "Username for authentication on outgoing SMTP server")
	f.IntVar(&cfg.maxMessageSize, "max_message_size", 51200000, "Max message size allowed in bytes")
	f.IntVar(&cfg.maxConnections, "max_connections", 100, "Max number of concurrent connections, use -1 to disable")
	f.IntVar(&cfg.maxRecipients, "max_recipients", 100, "Max number of recipients on an email")
	f.DurationVar(&cfg.readTimeout, "read_timeout", 60*time.Second, "Socket timeout for read operations")
	f.DurationVar(&cfg.writeTimeout, "write_timeout", 60*time.Second, "Socket timeout for write operations")
	f.DurationVar(&cfg.dataTimeout, "data_timeout", 5*time.Minute, "Socket timeout for DATA command")
	f.StringVar(&cfg.remotePass, "remote_pass", "", "Password for authentication on outgoing SMTP server (set $REMOTE_PASS to use env var instead)")
	f.StringVar(&cfg.remoteAuth, "remote_auth", "plain", "Auth method on outgoing SMTP server (plain, login, xoauth2)")
	f.StringVar(&cfg.remoteSender, "remote_sender", "", "Sender email address on outgoing SMTP server")
	f.BoolVar(&cfg.versionInfo, "version", false, "Show version information")
	f.StringVar(&cfg.logLevel, "log_level", "debug", "Minimum log level to output")
	f.StringVar(&cfg.logHeadersStr, "log_header", "", "Log this mail header's value (log_field=Header-Name) set multiples with spaces")
	f.BoolVar(&cfg.rateLimitEnabled, "rate_limit_enabled", false, "Enable per-sender rate limiting")
	f.Float64Var(&cfg.rateLimitMessagesPerSecond, "rate_limit_messages_per_second", 10, "Maximum messages per second per sender")
	f.IntVar(&cfg.rateLimitBurst, "rate_limit_burst", 5, "Burst capacity for rate limiter")
	f.StringVar(&cfg.rateLimitHeader, "rate_limit_header", "", "Email header to extract sender identity for rate limiting (by default, the sender address is used)")
	f.StringVar(&cfg.xoauth2ClientID, "xoauth2_client_id", "", "Client ID for OAuth2 authentication")
	f.StringVar(&cfg.xoauth2ClientSecret, "xoauth2_client_secret", "", "Client secret for OAuth2 authentication")
	f.StringVar(&cfg.xoauth2RefreshToken, "xoauth2_refresh_token", "", "Refresh token for OAuth2 authentication")
	f.StringVar(&cfg.xoauth2TokenURL, "xoauth2_token_url", "", "OAuth2 token endpoint URL")
}

// parse the input into a map[string]string. It should be in the form of
// "field1=Header-Name1 field2=Header-Name2" (key=vaue pairs, separated by
// spaces)
func parseLogHeaders(s string) map[string]string {
	h := map[string]string{}
	if s == "" {
		return h
	}

	entries := strings.Split(s, " ")
	for _, entry := range entries {
		field, hdr, found := strings.Cut(entry, "=")
		if !found {
			continue
		}

		h[field] = hdr
	}

	return h
}
