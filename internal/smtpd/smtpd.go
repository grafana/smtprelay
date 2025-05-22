// Package smtpd implements an SMTP server with support for STARTTLS, authentication (PLAIN/LOGIN), XCLIENT and optional restrictions on the different stages of the SMTP session.
package smtpd

import (
	"bufio"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log"
	"net"
	"net/textproto"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"go.opentelemetry.io/otel"
)

var tracer = otel.Tracer("github.com/grafana/smtprelay/v2/internal/smtpd")

// Server defines the parameters for running the SMTP server
//
//nolint:govet
type Server struct {
	Hostname       string // Server hostname. (default: "localhost.localdomain")
	WelcomeMessage string // Initial server banner. (default: "<hostname> ESMTP ready.")

	ReadTimeout  time.Duration // Socket timeout for read operations. (default: 60s)
	WriteTimeout time.Duration // Socket timeout for write operations. (default: 60s)
	DataTimeout  time.Duration // Socket timeout for DATA command (default: 5m)

	MaxConnections int // Max concurrent connections, use -1 to disable. (default: 100)
	MaxMessageSize int // Max message size in bytes. (default: 10240000)
	MaxRecipients  int // Max RCPT TO calls for each envelope. (default: 100)

	// New e-mails are handed off to this function.
	// Can be left empty for a NOOP server.
	// If an error is returned, it will be reported in the SMTP session.
	Handler func(ctx context.Context, peer Peer, env Envelope) error

	// Enable various checks during the SMTP session.
	// Can be left empty for no restrictions.
	// If an error is returned, it will be reported in the SMTP session.
	// Use the Error struct for access to error codes.
	ConnectionChecker func(ctx context.Context, peer Peer) error              // Called upon new connection.
	HeloChecker       func(ctx context.Context, peer Peer, name string) error // Called after HELO/EHLO.
	SenderChecker     func(ctx context.Context, peer Peer, addr string) error // Called after MAIL FROM.
	RecipientChecker  func(ctx context.Context, peer Peer, addr string) error // Called after each RCPT TO.

	// Enable PLAIN/LOGIN authentication, only available after STARTTLS.
	// Can be left empty for no authentication support.
	Authenticator func(ctx context.Context, peer Peer, username, password string) error

	EnableXCLIENT       bool // Enable XCLIENT support (default: false)
	EnableProxyProtocol bool // Enable proxy protocol support (default: false)

	TLSConfig *tls.Config // Enable STARTTLS support.
	ForceTLS  bool        // Force STARTTLS usage.

	ProtocolLogger *log.Logger

	// ConnContext optionally specifies a function that modifies
	// the context used for a new connection c. The provided ctx
	// is derived from the base context.
	ConnContext func(ctx context.Context, c net.Conn) context.Context

	// mu guards doneChan and makes closing it and listener atomic from
	// perspective of Serve()
	mu         sync.Mutex
	doneChan   chan struct{}
	listener   *net.Listener
	waitgrp    sync.WaitGroup
	inShutdown atomic.Bool // true when server is in shutdown
}

// Protocol represents the protocol used in the SMTP session
type Protocol string

const (
	// SMTP
	SMTP Protocol = "SMTP"

	// Extended SMTP
	ESMTP = "ESMTP"
)

// Peer represents the client connecting to the server
type Peer struct {
	Addr       net.Addr             // Network address
	TLS        *tls.ConnectionState // TLS Connection details, if on TLS
	HeloName   string               // Server name used in HELO/EHLO command
	Username   string               // Username from authentication, if authenticated
	Password   string               // Password from authentication, if authenticated
	Protocol   Protocol             // Protocol used, SMTP or ESMTP
	ServerName string               // A copy of Server.Hostname
}

// ErrServerClosed is returned by the Server's Serve and ListenAndServe,
// methods after a call to Shutdown.
var ErrServerClosed = errors.New("smtp: Server closed")

var (
	// similar to net/http's LocalAddrContextKey
	localAddrContextKey = &struct{}{}
)

// LocalAddrFromContext can be used in handlers to access the local address the
// connection arrived on. If no local address is available, nil is returned.
func LocalAddrFromContext(ctx context.Context) net.Addr {
	if addr, ok := ctx.Value(localAddrContextKey).(net.Addr); ok {
		return addr
	}

	return nil
}

type session struct {
	server *Server

	envelope *Envelope

	conn net.Conn

	reader  *bufio.Reader
	writer  *bufio.Writer
	scanner *bufio.Scanner

	peer Peer

	tls bool
}

func (srv *Server) newSession(c net.Conn) *session {
	s := &session{
		server: srv,
		conn:   c,
		reader: bufio.NewReader(c),
		writer: bufio.NewWriter(c),
		peer: Peer{
			Addr:       c.RemoteAddr(),
			ServerName: srv.Hostname,
		},
	}

	// Check if the underlying connection is already TLS.
	// This will happen if the Listerner provided Serve()
	// is from tls.Listen()

	var tlsConn *tls.Conn

	tlsConn, s.tls = c.(*tls.Conn)

	if s.tls {
		// run handshake otherwise it's done when we first
		// read/write and connection state will be invalid
		_ = tlsConn.Handshake()
		state := tlsConn.ConnectionState()
		s.peer.TLS = &state
	}

	s.scanner = bufio.NewScanner(s.reader)

	return s
}

// ListenAndServe starts the SMTP server and listens on addr, using ctx as the
// base context for incoming requests.
func (srv *Server) ListenAndServe(ctx context.Context, addr string) error {
	if srv.shuttingDown() {
		return ErrServerClosed
	}

	lc := net.ListenConfig{}
	l, err := lc.Listen(ctx, "tcp", addr)
	if err != nil {
		return err
	}

	return srv.Serve(ctx, l)
}

// Serve starts the SMTP server and listens on l, using ctx as the base context
// for incoming requests.
func (srv *Server) Serve(ctx context.Context, l net.Listener) error {
	if srv.shuttingDown() {
		return ErrServerClosed
	}

	srv.configureDefaults()

	l = &onceCloseListener{Listener: l}
	defer l.Close()
	srv.listener = &l

	var limiter chan struct{}

	if srv.MaxConnections > 0 {
		limiter = make(chan struct{}, srv.MaxConnections)
	}

	// shut down the server if the context is cancelled, but wait for all
	// connections to finish
	go func() {
		<-ctx.Done()
		_ = srv.Shutdown(true)
	}()

	for {
		conn, e := l.Accept()
		if e != nil {
			select {
			case <-srv.getDoneChan():
				return ErrServerClosed
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			var ne net.Error
			//nolint:staticcheck
			if ok := errors.As(e, &ne); ok && ne.Temporary() {
				// TODO: Exponential backoff
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(time.Second):
				}

				continue
			}
			return e
		}

		connCtx := ctx
		if cc := srv.ConnContext; cc != nil {
			connCtx = cc(connCtx, conn)
			if connCtx == nil {
				panic("ConnContext returned nil")
			}
		}

		session := srv.newSession(conn)

		srv.waitgrp.Add(1)
		go func() {
			defer srv.waitgrp.Done()

			if limiter != nil {
				select {
				case limiter <- struct{}{}:
					session.serve(connCtx)
					<-limiter
				default:
					session.reject()
				}
			} else {
				session.serve(connCtx)
			}
		}()
	}
}

// Shutdown instructs the server to shutdown, starting by closing the
// associated listener. If wait is true, it will wait for the shutdown
// to complete. If wait is false, Wait must be called afterwards.
func (srv *Server) Shutdown(wait bool) error {
	var lnerr error
	srv.inShutdown.Store(true)

	// First close the listener
	srv.mu.Lock()
	if srv.listener != nil {
		lnerr = (*srv.listener).Close()
	}
	srv.closeDoneChanLocked()
	srv.mu.Unlock()

	// Now wait for all client connections to close
	if wait {
		_ = srv.Wait()
	}

	return lnerr
}

// Wait waits for all client connections to close and the server to finish
// shutting down.
func (srv *Server) Wait() error {
	if !srv.shuttingDown() {
		return errors.New("Server has not been Shutdown")
	}

	srv.waitgrp.Wait()
	return nil
}

// Address returns the listening address of the server
func (srv *Server) Address() net.Addr {
	return (*srv.listener).Addr()
}

func (srv *Server) configureDefaults() {
	if srv.MaxMessageSize == 0 {
		srv.MaxMessageSize = 10240000
	}

	if srv.MaxConnections == 0 {
		srv.MaxConnections = 100
	}

	if srv.MaxRecipients == 0 {
		srv.MaxRecipients = 100
	}

	if srv.ReadTimeout == 0 {
		srv.ReadTimeout = time.Second * 60
	}

	if srv.WriteTimeout == 0 {
		srv.WriteTimeout = time.Second * 60
	}

	if srv.DataTimeout == 0 {
		srv.DataTimeout = time.Minute * 5
	}

	if srv.ForceTLS && srv.TLSConfig == nil {
		log.Fatal("Cannot use ForceTLS with no TLSConfig")
	}

	if srv.Hostname == "" {
		srv.Hostname = "localhost.localdomain"
	}

	if srv.WelcomeMessage == "" {
		srv.WelcomeMessage = srv.Hostname + " ESMTP ready."
	}
}

func (session *session) serve(ctx context.Context) {
	ctx, span := tracer.Start(ctx, "smtpd.serve")
	defer span.End()

	defer session.close()

	ctx = context.WithValue(ctx, localAddrContextKey, session.conn.LocalAddr())

	if ctx.Err() != nil {
		session.reject()
		return
	}

	if !session.server.EnableProxyProtocol {
		session.welcome(ctx)
	}

	for {
		for session.scanner.Scan() {
			line := session.scanner.Text()
			session.logf("received: %s", strings.TrimSpace(line))
			session.handle(ctx, line)
		}

		err := session.scanner.Err()

		if errors.Is(err, bufio.ErrTooLong) {
			session.error(ErrLineTooLong)

			// Advance reader to the next newline

			_, _ = session.reader.ReadString('\n')
			session.scanner = bufio.NewScanner(session.reader)

			// Reset and have the client start over.

			session.reset()

			continue
		}

		break
	}

}

func (session *session) reject() {
	session.error(ErrBusy)
	session.close()
}

func (session *session) reset() {
	session.envelope = nil
}

func (session *session) welcome(ctx context.Context) {
	if session.server.ConnectionChecker != nil {
		err := session.server.ConnectionChecker(ctx, session.peer)
		if err != nil {
			session.error(err)
			session.close()
			return
		}
	}

	session.reply(220, session.server.WelcomeMessage)
}

func (session *session) reply(code int, message string) {
	session.logf("sending: %d %s", code, message)
	_, _ = fmt.Fprintf(session.writer, "%d %s\r\n", code, message)
	session.flush()
}

func (session *session) flush() {
	_ = session.conn.SetWriteDeadline(time.Now().Add(session.server.WriteTimeout))
	session.writer.Flush()
	_ = session.conn.SetReadDeadline(time.Now().Add(session.server.ReadTimeout))
}

func (session *session) error(err error) {
	var smtpdError *textproto.Error
	if errors.As(err, &smtpdError) {
		// session.reply(smtpdError.Code, err.Error())
		// the error code will be prefixed in the error message
		session.logf("sending: %s", err)
		_, _ = fmt.Fprintf(session.writer, "%s\r\n", err)
		session.flush()
	} else {
		session.reply(502, err.Error())
	}
}

func (session *session) logf(format string, v ...interface{}) {
	if session.server.ProtocolLogger == nil {
		return
	}

	_ = session.server.ProtocolLogger.Output(2, fmt.Sprintf(
		"%s [peer:%s]",
		fmt.Sprintf(format, v...),
		session.peer.Addr,
	))
}

func (session *session) logError(err error, desc string) {
	session.logf("%s: %v ", desc, err)
}

func (session *session) extensions() []string {
	extensions := []string{
		fmt.Sprintf("SIZE %d", session.server.MaxMessageSize),
		"8BITMIME",
		"PIPELINING",
	}

	if session.server.EnableXCLIENT {
		extensions = append(extensions, "XCLIENT")
	}

	if session.server.TLSConfig != nil && !session.tls {
		extensions = append(extensions, "STARTTLS")
	}

	if session.server.Authenticator != nil && session.tls {
		extensions = append(extensions, "AUTH PLAIN LOGIN")
	}

	return extensions
}

func (session *session) deliver(ctx context.Context) error {
	if session.server.Handler != nil {
		return session.server.Handler(ctx, session.peer, *session.envelope)
	}

	return nil
}

func (session *session) close() {
	session.writer.Flush()
	time.Sleep(200 * time.Millisecond)
	session.conn.Close()
}

// From net/http/server.go

func (srv *Server) shuttingDown() bool {
	return srv.inShutdown.Load()
}

func (srv *Server) getDoneChan() <-chan struct{} {
	srv.mu.Lock()
	defer srv.mu.Unlock()
	return srv.getDoneChanLocked()
}

func (srv *Server) getDoneChanLocked() chan struct{} {
	if srv.doneChan == nil {
		srv.doneChan = make(chan struct{})
	}
	return srv.doneChan
}

func (srv *Server) closeDoneChanLocked() {
	ch := srv.getDoneChanLocked()
	select {
	case <-ch:
		// Already closed. Don't close again.
	default:
		// Safe to close here. We're the only closer, guarded
		// by srv.mu.
		close(ch)
	}
}

// onceCloseListener wraps a net.Listener, protecting it from
// multiple Close calls.
type onceCloseListener struct {
	net.Listener
	closeErr error
	once     sync.Once
}

func (oc *onceCloseListener) Close() error {
	oc.once.Do(oc.close)
	return oc.closeErr
}

func (oc *onceCloseListener) close() { oc.closeErr = oc.Listener.Close() }
