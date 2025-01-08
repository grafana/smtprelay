package smtpd

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net"
	"net/textproto"
	"strconv"
	"strings"
	"time"
)

type command struct {
	line   string
	action string
	fields []string
	params []string
}

func parseLine(line string) command {
	cmd := command{
		line:   line,
		fields: strings.Fields(line),
	}

	if len(cmd.fields) > 0 {

		cmd.action = strings.ToUpper(cmd.fields[0])

		if len(cmd.fields) > 1 {
			// Account for some clients breaking the standard and having
			// an extra whitespace after the ':' character. Example:
			//
			// MAIL FROM: <test@example.org>
			//
			// Should be:
			//
			// MAIL FROM:<test@example.org>
			//
			// Thus, we add a check if the second field ends with ':'
			// and appends the rest of the third field.
			if cmd.fields[1][len(cmd.fields[1])-1] == ':' && len(cmd.fields) > 2 {
				cmd.fields[1] += cmd.fields[2]
				cmd.fields = cmd.fields[0:2]
			}

			cmd.params = strings.Split(cmd.fields[1], ":")

		}

	}

	return cmd
}

func (session *session) handle(ctx context.Context, line string) {
	cmd := parseLine(line)

	ctx, span := tracer.Start(ctx, "session.handle"+cmd.action)
	defer span.End()

	// Commands are dispatched to the appropriate handler functions.
	// If a network error occurs during handling, the handler should
	// just return and let the error be handled on the next read.
	switch cmd.action {
	case "PROXY":
		session.handlePROXY(ctx, cmd)
	case "HELO":
		session.handleHELO(ctx, cmd)
	case "EHLO":
		session.handleEHLO(ctx, cmd)
	case "MAIL":
		session.handleMAIL(ctx, cmd)
	case "RCPT":
		session.handleRCPT(ctx, cmd)
	case "STARTTLS":
		session.handleSTARTTLS(ctx, cmd)
	case "DATA":
		session.handleDATA(ctx, cmd)
	case "RSET":
		session.handleRSET(ctx, cmd)
	case "NOOP":
		session.handleNOOP(ctx, cmd)
	case "QUIT":
		session.handleQUIT(ctx, cmd)
	case "AUTH":
		session.handleAUTH(ctx, cmd)
	case "XCLIENT":
		session.handleXCLIENT(ctx, cmd)
	default:
		session.error(ErrUnsupportedCommand)
	}
}

func (session *session) handleHELO(ctx context.Context, cmd command) {
	if len(cmd.fields) < 2 {
		session.error(ErrMissingParam)
		return
	}

	if session.peer.HeloName != "" {
		// Reset envelope in case of duplicate HELO
		session.reset()
	}

	if session.server.HeloChecker != nil {
		err := session.server.HeloChecker(ctx, session.peer, cmd.fields[1])
		if err != nil {
			session.error(err)
			return
		}
	}

	session.peer.HeloName = cmd.fields[1]
	session.peer.Protocol = SMTP
	session.reply(250, "Go ahead")
}

func (session *session) handleEHLO(ctx context.Context, cmd command) {
	if len(cmd.fields) < 2 {
		session.error(ErrMissingParam)
		return
	}

	if session.peer.HeloName != "" {
		// Reset envelope in case of duplicate EHLO
		session.reset()
	}

	if session.server.HeloChecker != nil {
		err := session.server.HeloChecker(ctx, session.peer, cmd.fields[1])
		if err != nil {
			session.error(err)
			return
		}
	}

	session.peer.HeloName = cmd.fields[1]
	session.peer.Protocol = ESMTP

	fmt.Fprintf(session.writer, "250-%s\r\n", session.server.Hostname)

	extensions := session.extensions()

	if len(extensions) > 1 {
		for _, ext := range extensions[:len(extensions)-1] {
			fmt.Fprintf(session.writer, "250-%s\r\n", ext)
		}
	}

	session.reply(250, extensions[len(extensions)-1])
}

func (session *session) handleMAIL(ctx context.Context, cmd command) {
	if len(cmd.params) != 2 || strings.ToUpper(cmd.params[0]) != "FROM" {
		session.error(ErrInvalidSyntax)
		return
	}

	if session.peer.HeloName == "" {
		session.error(ErrNoHELO)
		return
	}

	if session.server.Authenticator != nil && session.peer.Username == "" {
		session.error(ErrAuthRequired)
		return
	}

	if !session.tls && session.server.ForceTLS {
		session.error(ErrNoSTARTTLS)
		return
	}

	if session.envelope != nil {
		session.error(ErrDuplicateMAIL)
		return
	}

	var err error
	addr := "" // null sender

	// We must accept a null sender as per rfc5321 section-6.1.
	if cmd.params[1] != "<>" {
		addr, err = parseAddress(cmd.params[1])
		if err != nil {
			session.error(ErrMalformedEmail)
			return
		}
	}

	if session.server.SenderChecker != nil {
		err = session.server.SenderChecker(ctx, session.peer, addr)
		if err != nil {
			session.error(err)
			return
		}
	}

	session.envelope = &Envelope{
		Sender: addr,
	}

	session.reply(250, "Go ahead")
}

func (session *session) handleRCPT(ctx context.Context, cmd command) {
	if len(cmd.params) != 2 || strings.ToUpper(cmd.params[0]) != "TO" {
		session.error(ErrInvalidSyntax)
		return
	}

	if session.envelope == nil {
		session.error(ErrNoMAIL)
		return
	}

	if len(session.envelope.Recipients) >= session.server.MaxRecipients {
		session.error(ErrTooManyRecipients)
		return
	}

	addr, err := parseAddress(cmd.params[1])
	if err != nil {
		session.error(ErrMalformedEmail)
		return
	}

	if session.server.RecipientChecker != nil {
		err = session.server.RecipientChecker(ctx, session.peer, addr)
		if err != nil {
			session.error(err)
			return
		}
	}

	session.envelope.Recipients = append(session.envelope.Recipients, addr)

	session.reply(250, "Go ahead")
}

func (session *session) handleSTARTTLS(_ context.Context, _ command) {
	if session.tls {
		session.error(ErrDuplicateSTARTTLS)
		return
	}

	if session.server.TLSConfig == nil {
		session.error(ErrTLSNotSupported)
		return
	}

	tlsConn := tls.Server(session.conn, session.server.TLSConfig)
	session.reply(220, "Go ahead")

	if err := tlsConn.Handshake(); err != nil {
		session.logError(err, "couldn't perform handshake")
		session.error(ErrBadHandshake)
		return
	}

	// Reset envelope as a new EHLO/HELO is required after STARTTLS
	session.reset()

	// Reset deadlines on the underlying connection before I replace it
	// with a TLS connection
	_ = session.conn.SetDeadline(time.Time{})

	// Replace connection with a TLS connection
	session.conn = tlsConn
	session.reader = bufio.NewReader(tlsConn)
	session.writer = bufio.NewWriter(tlsConn)
	session.scanner = bufio.NewScanner(session.reader)
	session.tls = true

	// Save connection state on peer
	state := tlsConn.ConnectionState()
	session.peer.TLS = &state

	// Flush the connection to set new timeout deadlines
	session.flush()
}

func (session *session) handleDATA(ctx context.Context, _ command) {
	if session.envelope == nil || len(session.envelope.Recipients) == 0 {
		session.error(ErrNoRCPT)
		return
	}

	session.reply(354, "Go ahead. End your data with <CR><LF>.<CR><LF>")
	_ = session.conn.SetDeadline(time.Now().Add(session.server.DataTimeout))

	data := &bytes.Buffer{}
	reader := textproto.NewReader(session.reader).DotReader()

	_, err := io.CopyN(data, reader, int64(session.server.MaxMessageSize))
	if errors.Is(err, io.EOF) {
		// EOF was reached before MaxMessageSize
		// Accept and deliver message
		session.envelope.Data = data.Bytes()

		// re-read to get the MIME header (if any)
		header, _ := textproto.NewReader(bufio.NewReader(data)).ReadMIMEHeader()
		session.envelope.Header = header

		err = session.deliver(ctx)
		if err != nil {
			session.error(err)
		} else {
			session.reply(250, "Thank you.")
		}

		session.reset()
		return
	} else if err != nil {
		// Other network error, ignore
		return
	}

	// Discard the rest and report an error.
	_, err = io.Copy(io.Discard, reader)
	if err != nil {
		// Network error, ignore
		return
	}

	session.error(fmt.Errorf("%w (max %d bytes)", ErrTooBig, session.server.MaxMessageSize))

	session.reset()
}

func (session *session) handleRSET(_ context.Context, _ command) {
	session.reset()
	session.reply(250, "Go ahead")
}

func (session *session) handleNOOP(_ context.Context, _ command) {
	session.reply(250, "Go ahead")
}

func (session *session) handleQUIT(_ context.Context, _ command) {
	session.reply(221, "OK, bye")
	session.close()
}

func (session *session) handleAUTH(ctx context.Context, cmd command) {
	if len(cmd.fields) < 2 {
		session.error(ErrInvalidSyntax)
		return
	}

	if session.server.Authenticator == nil {
		session.error(ErrUnsupportedCommand)
		return
	}

	if session.peer.HeloName == "" {
		session.error(ErrNoHELO)
		return
	}

	if !session.tls {
		session.error(ErrNoSTARTTLS)
		return
	}

	mechanism := strings.ToUpper(cmd.fields[1])

	username := ""
	password := ""

	switch mechanism {
	case "PLAIN":
		auth := ""

		if len(cmd.fields) < 3 {
			session.reply(334, "Give me your credentials")
			if !session.scanner.Scan() {
				return
			}
			auth = session.scanner.Text()
		} else {
			auth = cmd.fields[2]
		}

		data, err := base64.StdEncoding.DecodeString(auth)

		if err != nil {
			session.error(ErrMalformedAuth)
			return
		}

		parts := bytes.Split(data, []byte{0})

		if len(parts) != 3 {
			session.error(ErrMalformedAuth)
			return
		}

		username = string(parts[1])
		password = string(parts[2])
	case "LOGIN":
		encodedUsername := ""

		if len(cmd.fields) < 3 {
			session.reply(334, "VXNlcm5hbWU6")
			if !session.scanner.Scan() {
				return
			}
			encodedUsername = session.scanner.Text()
		} else {
			encodedUsername = cmd.fields[2]
		}

		byteUsername, err := base64.StdEncoding.DecodeString(encodedUsername)

		if err != nil {
			session.error(ErrMalformedAuth)
			return
		}

		session.reply(334, "UGFzc3dvcmQ6")

		if !session.scanner.Scan() {
			return
		}

		bytePassword, err := base64.StdEncoding.DecodeString(session.scanner.Text())

		if err != nil {
			session.error(ErrMalformedAuth)
			return
		}

		username = string(byteUsername)
		password = string(bytePassword)
	default:
		session.logf("unknown authentication mechanism: %s", mechanism)
		session.error(ErrUnknownAuth)
		return
	}

	err := session.server.Authenticator(ctx, session.peer, username, password)
	if err != nil {
		session.error(err)
		return
	}

	session.peer.Username = username
	session.peer.Password = password

	session.reply(235, "OK, you are now authenticated")
}

func (session *session) handleXCLIENT(ctx context.Context, cmd command) {
	if len(cmd.fields) < 2 {
		session.error(ErrInvalidSyntax)
		return
	}

	if !session.server.EnableXCLIENT {
		session.error(ErrUnsupportedCommand)
		return
	}

	var (
		newHeloName, newUsername string
		newProto                 Protocol
		newAddr                  net.IP
		newTCPPort               uint16
	)

	for _, item := range cmd.fields[1:] {
		parts := strings.Split(item, "=")

		if len(parts) != 2 {
			session.error(ErrMalformedCommand)

			return
		}

		name := parts[0]
		value := parts[1]

		switch name {

		case "NAME":
			// Unused in smtpd package
			continue
		case "HELO":
			newHeloName = value
			continue
		case "ADDR":
			newAddr = net.ParseIP(value)
			continue
		case "PORT":
			n, err := strconv.ParseUint(value, 10, 16)
			if err != nil {
				session.error(ErrMalformedCommand)

				return
			}
			newTCPPort = uint16(n)
			continue
		case "LOGIN":
			newUsername = value
			continue
		case "PROTO":
			if value == "SMTP" {
				newProto = SMTP
			} else if value == "ESMTP" {
				newProto = ESMTP
			}
			continue
		default:
			session.error(ErrMalformedCommand)

			return
		}
	}

	tcpAddr, ok := session.peer.Addr.(*net.TCPAddr)
	if !ok {
		session.error(ErrUnsupportedConn)
		return
	}

	if newHeloName != "" {
		session.peer.HeloName = newHeloName
	}

	if newAddr != nil {
		tcpAddr.IP = newAddr
	}

	if newTCPPort != 0 {
		tcpAddr.Port = int(newTCPPort)
	}

	if newUsername != "" {
		session.peer.Username = newUsername
	}

	if newProto != "" {
		session.peer.Protocol = newProto
	}

	session.welcome(ctx)
}

func (session *session) handlePROXY(ctx context.Context, cmd command) {
	if !session.server.EnableProxyProtocol {
		session.error(ErrUnsupportedCommand)
		return
	}

	if len(cmd.fields) < 6 {
		session.error(ErrMalformedCommand)
		return
	}

	newAddr := net.ParseIP(cmd.fields[2])

	n, err := strconv.ParseUint(cmd.fields[4], 10, 16)
	if err != nil {
		session.error(ErrMalformedCommand)
		return
	}
	newTCPPort := uint16(n)

	tcpAddr, ok := session.peer.Addr.(*net.TCPAddr)
	if !ok {
		session.error(ErrUnsupportedConn)
		return
	}

	if newAddr != nil {
		tcpAddr.IP = newAddr
	}

	if newTCPPort != 0 {
		tcpAddr.Port = int(newTCPPort)
	}

	session.welcome(ctx)
}
