package main

import (
	"bufio"
	"bytes"
	"context"
	"net"
	"net/smtp"
	"net/textproto"
	"strings"
	"testing"
	"time"

	"github.com/grafana/smtprelay/v2/internal/smtpd"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testSMTPServer struct {
	msgs *[]smtpd.Envelope
	addr string
}

func startTestSMTPServer(ctx context.Context, t *testing.T) *testSMTPServer {
	t.Helper()

	msgs := &[]smtpd.Envelope{}
	srv := &smtpd.Server{
		ConnectionChecker: func(_ context.Context, peer smtpd.Peer) error {
			t.Logf("Connection from %s", peer.HeloName)
			return nil
		},
		HeloChecker: func(_ context.Context, peer smtpd.Peer, name string) error {
			t.Logf("HELO (%s) %s", peer.Protocol, name)
			return nil
		},
		SenderChecker: func(_ context.Context, _ smtpd.Peer, addr string) error {
			t.Logf("MAIL FROM %s", addr)
			return nil
		},
		RecipientChecker: func(_ context.Context, _ smtpd.Peer, addr string) error {
			t.Logf("RCPT TO %s", addr)
			return nil
		},
		Handler: func(_ context.Context, _ smtpd.Peer, env smtpd.Envelope) error {
			t.Logf("DATA\n----\n%s\n----", env.Data)
			m := append(*msgs, env)
			*msgs = m
			return nil
		},
	}

	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	go func(l net.Listener) {
		_ = srv.Serve(ctx, l)
	}(l)

	t.Cleanup(func() {
		_ = srv.Shutdown(false)
	})

	return &testSMTPServer{addr: l.Addr().String(), msgs: msgs}
}

func sendMsg(t *testing.T, addr string, to []string, from, subject string, hdrs textproto.MIMEHeader, body string) error {
	t.Helper()

	data := bytes.Buffer{}
	data.WriteString("From: " + from + "\r\n")
	data.WriteString("To: " + strings.Join(to, ", ") + "\r\n")
	data.WriteString("Subject: " + subject + "\r\n")
	for k, v := range hdrs {
		data.WriteString(k + ": " + strings.Join(v, ", ") + "\r\n")
	}
	data.WriteString("\r\n")
	data.WriteString(body)

	return smtp.SendMail(addr, nil, from, to, data.Bytes())
}

// helper function to start the smtprelay
//
// TODO: refactor smtprelay to be more testable (allow passing in the logger and
// metrics registry, provide a good way to shut down the server, etc...)
func startRelay(ctx context.Context, t *testing.T, srvAddr string) string {
	t.Helper()
	return startRelayWithConfig(ctx, t, srvAddr, nil)
}

func startRelayWithConfig(ctx context.Context, t *testing.T, srvAddr string, cfgOverrides func(*config)) string {
	t.Helper()

	addr := ""
	// pick a random port
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	addr = l.Addr().String()
	_ = l.Close()

	go func() {
		metricsRegistry = prometheus.NewRegistry()

		cfg := &config{
			listen:        addr,
			metricsListen: "127.0.0.1:0",
			remoteHost:    srvAddr,
			logLevel:      "debug",
		}

		if cfgOverrides != nil {
			cfgOverrides(cfg)
		}

		_ = run(ctx, cfg)
	}()

	// wait for the server to start
	for range 10 {
		conn, err := net.Dial("tcp", addr)
		if err == nil {
			_ = conn.Close()
			break
		}

		time.Sleep(100 * time.Millisecond)
	}

	return addr
}

func TestSendMail(t *testing.T) {
	ctx := t.Context()

	srv := startTestSMTPServer(ctx, t)

	addr := startRelay(ctx, t, srv.addr)

	err := sendMsg(t, addr, []string{"alice@example.com"},
		"bob@example.com", "test message", textproto.MIMEHeader{}, "hello world")
	require.NoError(t, err)
	assert.Len(t, *srv.msgs, 1)

	r := bufio.NewReader(bytes.NewReader((*srv.msgs)[0].Data))
	msg := textproto.NewReader(r)
	hdr, err := msg.ReadMIMEHeader()
	require.NoError(t, err)

	assert.Equal(t, "bob@example.com", hdr.Get("From"))
	assert.Equal(t, "alice@example.com", hdr.Get("To"))
	assert.Equal(t, "test message", hdr.Get("Subject"))
	assert.NotEmpty(t, hdr.Get("Received"))

	line, err := msg.ReadLine()
	require.NoError(t, err)
	assert.Equal(t, "hello world", line)
}

func TestRateLimitBySender(t *testing.T) {
	ctx := t.Context()

	srv := startTestSMTPServer(ctx, t)

	addr := startRelayWithConfig(ctx, t, srv.addr, func(cfg *config) {
		cfg.rateLimitEnabled = true
		cfg.rateLimitMessagesPerMin = 1
		cfg.rateLimitBurst = 1
	})

	// first message should be accepted
	err := sendMsg(t, addr, []string{"alice@example.com"}, "bob@example.com", "message 1", textproto.MIMEHeader{}, "body 1")
	require.NoError(t, err)

	// second message from same sender should be rate limited
	err = sendMsg(t, addr, []string{"alice@example.com"}, "bob@example.com", "message 2", textproto.MIMEHeader{}, "body 2")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "421")

	// third message from different sender should be accepted
	err = sendMsg(t, addr, []string{"alice@example.com"}, "charlie@example.com", "message 3", textproto.MIMEHeader{}, "body 3")
	require.NoError(t, err)

	// verify two messages received
	assert.Len(t, *srv.msgs, 2)
}

func TestRateLimitByHeader(t *testing.T) {
	ctx := t.Context()

	srv := startTestSMTPServer(ctx, t)

	addr := startRelayWithConfig(ctx, t, srv.addr, func(cfg *config) {
		cfg.rateLimitEnabled = true
		cfg.rateLimitMessagesPerMin = 1
		cfg.rateLimitBurst = 1
		cfg.rateLimitHeader = "X-Sender-ID"
	})

	// first message should be accepted
	headers := textproto.MIMEHeader{"X-Sender-ID": []string{"user-123"}}
	err := sendMsg(t, addr, []string{"alice@example.com"}, "bob@example.com", "message 1", headers, "body 1")
	require.NoError(t, err)

	// second message with same header value should be rate limited
	err = sendMsg(t, addr, []string{"alice@example.com"}, "bob@example.com", "message 2", headers, "body 2")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "421")

	// third message with different header value should be accepted
	headers2 := textproto.MIMEHeader{"X-Sender-ID": []string{"user-456"}}
	err = sendMsg(t, addr, []string{"alice@example.com"}, "bob@example.com", "message 3", headers2, "body 3")
	require.NoError(t, err)

	// verify two messages received
	assert.Len(t, *srv.msgs, 2)
}
