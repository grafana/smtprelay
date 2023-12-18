package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/vharitonsky/iniflags"
)

// config values
var (
	logFile           string
	hostName          string
	welcomeMsg        string
	listen            string
	metricsListen     string
	localCert         string
	localKey          string
	localForceTLS     bool
	allowedNetsStr    string
	allowedSender     string
	allowedRecipients string
	deniedRecipients  string
	allowedUsers      string
	remoteHost        string
	remoteUser        string
	maxMessageSize    int
	maxConnections    int
	maxRecipients     int
	readTimeout       time.Duration
	writeTimeout      time.Duration
	dataTimeout       time.Duration
	remotePass        string
	remoteAuth        string
	remoteSender      string
	versionInfo       bool
	logLevel          string
	logHeadersStr     string
)

var (
	allowedNets []*net.IPNet
	logHeaders  map[string]string
	log         *logrus.Entry
)

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

func ConfigLoad() error {
	registerFlags(flag.CommandLine)

	iniflags.Parse()

	logger, err := setupLogger(logFile, logLevel)
	if err != nil {
		return fmt.Errorf("setupLogger: %w", err)
	}
	log = logger

	// if remotePass is not set, try reading it from env var
	if remotePass == "" {
		log.Debug("remote_pass not set, trying REMOTE_PASS env var")
		remotePass = os.Getenv("REMOTE_PASS")
		if remotePass != "" {
			log.Debug("found data in REMOTE_PASS env var")
		} else {
			log.Debug("no data found in REMOTE_PASS env var")
		}
	}

	allowedNets, err = setupAllowedNetworks(allowedNetsStr)
	if err != nil {
		return fmt.Errorf("setupAllowedNetworks: %w", err)
	}

	logHeaders = parseLogHeaders(logHeadersStr)

	return nil
}

func registerFlags(f *flag.FlagSet) {
	f.StringVar(&logFile, "logfile", "/dev/stdout", "Path to logfile")
	f.StringVar(&hostName, "hostname", "localhost.localdomain", "Server hostname")
	f.StringVar(&welcomeMsg, "welcome_msg", "", "Welcome message for SMTP session")
	f.StringVar(&listen, "listen", "127.0.0.1:25 [::1]:25", "Address and port to listen for incoming SMTP")
	f.StringVar(&metricsListen, "metrics_listen", ":8080", "Address and port to listen for metrics exposition")
	f.StringVar(&localCert, "local_cert", "", "SSL certificate for STARTTLS/TLS")
	f.StringVar(&localKey, "local_key", "", "SSL private key for STARTTLS/TLS")
	f.BoolVar(&localForceTLS, "local_forcetls", false, "Force STARTTLS (needs local_cert and local_key)")
	f.StringVar(&allowedNetsStr, "allowed_nets", "127.0.0.0/8 ::/128", "Networks allowed to send mails (set to \"\" to disable")
	f.StringVar(&allowedSender, "allowed_sender", "", "Regular expression for valid FROM email addresses (leave empty to allow any sender)")
	f.StringVar(&allowedRecipients, "allowed_recipients", "", "Regular expression for valid 'to' email addresses (leave empty to allow any recipient)")
	f.StringVar(&deniedRecipients, "denied_recipients", "", "Regular expression for email addresses for which will never deliver any emails.")
	f.StringVar(&allowedUsers, "allowed_users", "", "Path to file with valid users/passwords (leave empty to allow any user)")
	f.StringVar(&remoteHost, "remote_host", "smtp.gmail.com:587", "Outgoing SMTP server")
	f.StringVar(&remoteUser, "remote_user", "", "Username for authentication on outgoing SMTP server")
	f.IntVar(&maxMessageSize, "max_message_size", 51200000, "Max message size allowed in bytes")
	f.IntVar(&maxConnections, "max_connections", 100, "Max number of concurrent connections, use -1 to disable")
	f.IntVar(&maxRecipients, "max_recipients", 100, "Max number of recipients on an email")
	f.DurationVar(&readTimeout, "read_timeout", 60*time.Second, "Socket timeout for read operations")
	f.DurationVar(&writeTimeout, "write_timeout", 60*time.Second, "Socket timeout for write operations")
	f.DurationVar(&dataTimeout, "data_timeout", 5*time.Minute, "Socket timeout for DATA command")
	f.StringVar(&remotePass, "remote_pass", "", "Password for authentication on outgoing SMTP server (set $REMOTE_PASS to use env var instead)")
	f.StringVar(&remoteAuth, "remote_auth", "plain", "Auth method on outgoing SMTP server (plain, login)")
	f.StringVar(&remoteSender, "remote_sender", "", "Sender email address on outgoing SMTP server")
	f.BoolVar(&versionInfo, "version", false, "Show version information")
	f.StringVar(&logLevel, "log_level", "debug", "Minimum log level to output")
	f.StringVar(&logHeadersStr, "log_header", "", "Log this mail header's value (log_field=Header-Name) set multiples with spaces")
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
