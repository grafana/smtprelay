package main

import (
	"bufio"
	"bytes"
	"net"
	"net/smtp"
	"net/textproto"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/grafana/smtprelay/internal/smtpd"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testSMTPServer struct {
	msgs *[]smtpd.Envelope
	addr string
}

func startTestSMTPServer(t *testing.T) *testSMTPServer {
	t.Helper()

	msgs := &[]smtpd.Envelope{}
	srv := &smtpd.Server{
		ConnectionChecker: func(peer smtpd.Peer) error {
			t.Logf("Connection from %s", peer.HeloName)
			return nil
		},
		HeloChecker: func(peer smtpd.Peer, name string) error {
			t.Logf("HELO (%s) %s", peer.Protocol, name)
			return nil
		},
		SenderChecker: func(peer smtpd.Peer, addr string) error {
			t.Logf("MAIL FROM %s", addr)
			return nil
		},
		RecipientChecker: func(peer smtpd.Peer, addr string) error {
			t.Logf("RCPT TO %s", addr)
			return nil
		},
		Handler: func(peer smtpd.Peer, env smtpd.Envelope) error {
			t.Logf("DATA\n----\n%s\n----", env.Data)
			m := append(*msgs, env)
			*msgs = m
			return nil
		},
	}

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}

	go func(l net.Listener) {
		_ = srv.Serve(l)
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
func startRelay(t *testing.T, srvAddr string) string {
	t.Helper()

	addr := ""
	// pick a random port
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	addr = l.Addr().String()
	_ = l.Close()

	go func() {
		os.Args = []string{"smtprelay",
			"-log_level", "debug",
			"-listen", addr,
			"-metrics_listen", "127.0.0.1:0",
			"-remote_host", srvAddr,
		}

		metricsRegistry = prometheus.NewRegistry()
		main()
	}()

	// wait for the server to start
	for n := 0; n < 10; n++ {
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
	srv := startTestSMTPServer(t)

	addr := startRelay(t, srv.addr)

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
