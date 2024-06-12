package smtpd_test

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log"
	"net"
	"net/smtp"
	"net/textproto"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/grafana/smtprelay/v2/internal/smtpd"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	localhostCert = []byte(`-----BEGIN CERTIFICATE-----
MIIFkzCCA3ugAwIBAgIUQvhoyGmvPHq8q6BHrygu4dPp0CkwDQYJKoZIhvcNAQEL
BQAwWTELMAkGA1UEBhMCQVUxEzARBgNVBAgMClNvbWUtU3RhdGUxITAfBgNVBAoM
GEludGVybmV0IFdpZGdpdHMgUHR5IEx0ZDESMBAGA1UEAwwJbG9jYWxob3N0MB4X
DTIwMDUyMTE2MzI1NVoXDTMwMDUxOTE2MzI1NVowWTELMAkGA1UEBhMCQVUxEzAR
BgNVBAgMClNvbWUtU3RhdGUxITAfBgNVBAoMGEludGVybmV0IFdpZGdpdHMgUHR5
IEx0ZDESMBAGA1UEAwwJbG9jYWxob3N0MIICIjANBgkqhkiG9w0BAQEFAAOCAg8A
MIICCgKCAgEAk773plyfK4u2uIIZ6H7vEnTb5qJT6R/KCY9yniRvCFV+jCrISAs9
0pgU+/P8iePnZRGbRCGGt1B+1/JAVLIYFZuawILHNs4yWKAwh0uNpR1Pec8v7vpq
NpdUzXKQKIqFynSkcLA8c2DOZwuhwVc8rZw50yY3r4i4Vxf0AARGXapnBfy6WerR
/6xT7y/OcK8+8aOirDQ9P6WlvZ0ynZKi5q2o1eEVypT2us9r+HsCYosKEEAnjzjJ
wP5rvredxUqb7OupIkgA4Nq80+4tqGGQfWetmoi3zXRhKpijKjgxBOYEqSUWm9ws
/aC91Iy5RawyTB0W064z75OgfuI5GwFUbyLD0YVN4DLSAI79GUfvc8NeLEXpQvYq
+f8P+O1Hbv2AQ28IdbyQrNefB+/WgjeTvXLploNlUihVhpmLpptqnauw/DY5Ix51
w60lHIZ6esNOmMQB+/z/IY5gpmuo66yH8aSCPSYBFxQebB7NMqYGOS9nXx62/Bn1
OUVXtdtrhfbbdQW6zMZjka0t8m83fnGw3ISyBK2NNnSzOgycu0ChsW6sk7lKyeWa
85eJGsQWIhkOeF9v9GAIH/qsrgVpToVC9Krbk+/gqYIYF330tHQrzp6M6LiG5OY1
P7grUBovN2ZFt10B97HxWKa2f/8t9sfHZuKbfLSFbDsyI2JyNDh+Vk0CAwEAAaNT
MFEwHQYDVR0OBBYEFOLdIQUr3gDQF5YBor75mlnCdKngMB8GA1UdIwQYMBaAFOLd
IQUr3gDQF5YBor75mlnCdKngMA8GA1UdEwEB/wQFMAMBAf8wDQYJKoZIhvcNAQEL
BQADggIBAGddhQMVMZ14TY7bU8CMuc9IrXUwxp59QfqpcXCA2pHc2VOWkylv2dH7
ta6KooPMKwJ61d+coYPK1zMUvNHHJCYVpVK0r+IGzs8mzg91JJpX2gV5moJqNXvd
Fy6heQJuAvzbb0Tfsv8KN7U8zg/ovpS7MbY+8mRJTQINn2pCzt2y2C7EftLK36x0
KeBWqyXofBJoMy03VfCRqQlWK7VPqxluAbkH+bzji1g/BTkoCKzOitAbjS5lT3sk
oCrF9N6AcjpFOH2ZZmTO4cZ6TSWfrb/9OWFXl0TNR9+x5c/bUEKoGeSMV1YT1SlK
TNFMUlq0sPRgaITotRdcptc045M6KF777QVbrYm/VH1T3pwPGYu2kUdYHcteyX9P
8aRG4xsPGQ6DD7YjBFsif2fxlR3nQ+J/l/+eXHO4C+eRbxi15Z2NjwVjYpxZlUOq
HD96v516JkMJ63awbY+HkYdEUBKqR55tzcvNWnnfiboVmIecjAjoV4zStwDIti9u
14IgdqqAbnx0ALbUWnvfFloLdCzPPQhgLHpTeRSEDPljJWX8rmy8iQtRb0FWYQ3z
A2wsUyutzK19nt4hjVrTX0At9ku3gMmViXFlbvyA1Y4TuhdUYqJauMBrWKl2ybDW
yhdKg/V3yTwgBUtb3QO4m1khNQjQLuPFVxULGEA38Y5dXSONsYnt
-----END CERTIFICATE-----`)

	localhostKey = []byte(`-----BEGIN PRIVATE KEY-----
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
hBgHM6A0WJC9MO3aAKRBcp48y6DXNA==
-----END PRIVATE KEY-----`)
)

//nolint:gosec
var testTLSConfig = &tls.Config{InsecureSkipVerify: true}

func cmd(c *textproto.Conn, expectedCode int, format string, args ...interface{}) error {
	id, err := c.Cmd(format, args...)
	if err != nil {
		return err
	}

	c.StartResponse(id)
	_, _, err = c.ReadResponse(expectedCode)
	c.EndResponse(id)

	return err
}

func runserver(t *testing.T, server *smtpd.Server) (addr string, closer func()) {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	go func() {
		_ = server.Serve(ctx, ln)
	}()

	go func() {
		<-ctx.Done()

		ln.Close()
	}()

	return ln.Addr().String(), func() {
		cancel()
	}
}

func runsslserver(t *testing.T, server *smtpd.Server) (addr string, closer func()) {
	t.Helper()

	cert, err := tls.X509KeyPair(localhostCert, localhostKey)
	require.NoError(t, err)

	server.TLSConfig = &tls.Config{
		MinVersion:   tls.VersionTLS12,
		Certificates: []tls.Certificate{cert},
	}

	return runserver(t, server)
}

func TestSMTP(t *testing.T) {
	addr, closer := runserver(t, &smtpd.Server{
		ProtocolLogger: log.New(os.Stdout, "log: ", log.Lshortfile),
	})
	defer closer()

	c, err := smtp.Dial(addr)
	require.NoError(t, err)

	err = c.Hello("localhost")
	require.NoError(t, err)

	supported, _ := c.Extension("AUTH")
	require.False(t, supported, "AUTH supported before TLS")

	supported, _ = c.Extension("8BITMIME")
	require.True(t, supported, "8BITMIME not supported")

	supported, _ = c.Extension("STARTTLS")
	require.False(t, supported, "STARTTLS supported")

	err = c.Mail("sender@example.org")
	require.NoError(t, err)

	err = c.Rcpt("recipient@example.net")
	require.NoError(t, err)

	err = c.Rcpt("recipient2@example.net")
	require.NoError(t, err)

	wc, err := c.Data()
	require.NoError(t, err)

	_, err = fmt.Fprintf(wc, "This is the email body")
	require.NoError(t, err)

	err = wc.Close()
	require.NoError(t, err)

	err = c.Reset()
	require.NoError(t, err)

	err = c.Verify("foobar@example.net")
	require.Error(t, err)

	err = cmd(c.Text, 250, "NOOP")
	require.NoError(t, err)

	err = c.Quit()
	require.NoError(t, err)
}

func TestListenAndServe(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// get a random port
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := ln.Addr().String()
	ln.Close()

	server := &smtpd.Server{
		ProtocolLogger: log.New(os.Stdout, "log: ", log.Lshortfile),
	}

	go func() {
		_ = server.ListenAndServe(ctx, addr)
	}()

	// wait for the server to start
	for {
		select {
		case <-ctx.Done():
			t.Fatal("server failed to start")
		case <-time.After(10 * time.Millisecond):
		}

		cl, derr := smtp.Dial(addr)
		if derr != nil {
			continue
		}

		_ = cl.Close()
		break
	}

	c, err := smtp.Dial(addr)
	require.NoError(t, err)

	err = c.Quit()
	require.NoError(t, err)
}

func TestSTARTTLS(t *testing.T) {
	addr, closer := runsslserver(t, &smtpd.Server{
		Authenticator:  func(_ context.Context, _ smtpd.Peer, _, _ string) error { return nil },
		ForceTLS:       true,
		ProtocolLogger: log.New(os.Stdout, "log: ", log.Lshortfile),
	})

	defer closer()

	c, err := smtp.Dial(addr)
	require.NoError(t, err)

	supported, _ := c.Extension("AUTH")
	require.False(t, supported, "AUTH supported before TLS")

	err = c.Mail("sender@example.org")
	require.Error(t, err, "Mail worked before TLS with ForceTLS")

	err = cmd(c.Text, 220, "STARTTLS")
	require.NoError(t, err)

	err = cmd(c.Text, 250, "foobar")
	require.Error(t, err, "STARTTLS didn't fail with invalid handshake")

	testConfig := &tls.Config{
		//nolint:gosec
		InsecureSkipVerify: true,
	}

	err = c.StartTLS(testConfig)
	require.NoError(t, err)

	err = c.StartTLS(testConfig)
	require.Error(t, err, "STARTTLS worked twice")

	supported, _ = c.Extension("AUTH")
	require.True(t, supported, "AUTH not supported after TLS")

	_, mechs := c.Extension("AUTH")
	assert.Contains(t, mechs, "PLAIN", "PLAIN AUTH not supported after TLS")

	_, mechs = c.Extension("AUTH")
	assert.Contains(t, mechs, "LOGIN", "LOGIN AUTH not supported after TLS")

	err = c.Auth(smtp.PlainAuth("foo", "foo", "bar", "127.0.0.1"))
	require.NoError(t, err)

	err = c.Mail("sender@example.org")
	require.NoError(t, err)

	err = c.Rcpt("recipient@example.net")
	require.NoError(t, err)

	err = c.Rcpt("recipient2@example.net")
	require.NoError(t, err)

	wc, err := c.Data()
	require.NoError(t, err)

	_, err = fmt.Fprintf(wc, "This is the email body")
	require.NoError(t, err)

	err = wc.Close()
	require.NoError(t, err)

	err = c.Quit()
	require.NoError(t, err)
}

func TestAuthRejection(t *testing.T) {
	addr, closer := runsslserver(t, &smtpd.Server{
		Authenticator: func(_ context.Context, _ smtpd.Peer, _, _ string) error {
			return smtpd.ErrAuthInvalid
		},
		ForceTLS:       true,
		ProtocolLogger: log.New(os.Stdout, "log: ", log.Lshortfile),
	})
	defer closer()

	c, err := smtp.Dial(addr)
	require.NoError(t, err)

	err = c.StartTLS(testTLSConfig)
	require.NoError(t, err)

	err = c.Auth(smtp.PlainAuth("foo", "foo", "bar", "127.0.0.1"))
	require.Error(t, err, "Auth worked despite rejection")
}

func TestAuthNotSupported(t *testing.T) {
	addr, closer := runsslserver(t, &smtpd.Server{
		ForceTLS:       true,
		ProtocolLogger: log.New(os.Stdout, "log: ", log.Lshortfile),
	})
	defer closer()

	c, err := smtp.Dial(addr)
	require.NoError(t, err)

	err = c.StartTLS(testTLSConfig)
	require.NoError(t, err)

	err = c.Auth(smtp.PlainAuth("foo", "foo", "bar", "127.0.0.1"))
	require.Error(t, err, "Auth worked despite no authenticator")
}

func TestAuthBypass(t *testing.T) {
	addr, closer := runsslserver(t, &smtpd.Server{
		Authenticator: func(_ context.Context, _ smtpd.Peer, _, _ string) error {
			return smtpd.ErrAuthInvalid
		},
		ForceTLS:       true,
		ProtocolLogger: log.New(os.Stdout, "log: ", log.Lshortfile),
	})
	defer closer()

	c, err := smtp.Dial(addr)
	require.NoError(t, err)

	err = c.StartTLS(testTLSConfig)
	require.NoError(t, err)

	err = c.Mail("sender@example.org")
	require.Error(t, err, "MAIL succeeded despite AuthBypass")
}

func TestConnectionCheck(t *testing.T) {
	addr, closer := runserver(t, &smtpd.Server{
		ConnectionChecker: func(_ context.Context, _ smtpd.Peer) error {
			return smtpd.ErrIPDenied
		},
		ProtocolLogger: log.New(os.Stdout, "log: ", log.Lshortfile),
	})
	defer closer()

	_, err := smtp.Dial(addr)
	require.Error(t, err, "Dial succeeded despite ConnectionCheck")
}

func TestConnectionCheckSimpleError(t *testing.T) {
	addr, closer := runserver(t, &smtpd.Server{
		ConnectionChecker: func(_ context.Context, _ smtpd.Peer) error {
			return errors.New("Denied")
		},
		ProtocolLogger: log.New(os.Stdout, "log: ", log.Lshortfile),
	})
	defer closer()

	_, err := smtp.Dial(addr)
	require.Error(t, err, "Dial succeeded despite ConnectionCheck")
}

func TestHELOCheck(t *testing.T) {
	addr, closer := runserver(t, &smtpd.Server{
		HeloChecker: func(_ context.Context, _ smtpd.Peer, name string) error {
			require.Equal(t, "foobar.local", name)
			return smtpd.ErrUnsupportedCommand
		},
		ProtocolLogger: log.New(os.Stdout, "log: ", log.Lshortfile),
	})

	defer closer()

	c, err := smtp.Dial(addr)
	require.NoError(t, err)

	err = c.Hello("foobar.local")
	require.Error(t, err, "HELO succeeded despite HeloCheck")
}

func TestSenderCheck(t *testing.T) {
	addr, closer := runserver(t, &smtpd.Server{
		SenderChecker: func(_ context.Context, _ smtpd.Peer, _ string) error {
			return smtpd.ErrSenderDenied
		},
		ProtocolLogger: log.New(os.Stdout, "log: ", log.Lshortfile),
	})
	defer closer()

	c, err := smtp.Dial(addr)
	require.NoError(t, err)

	err = c.Mail("sender@example.org")
	require.Error(t, err, "MAIL succeeded despite SenderCheck")
}

func TestRecipientCheck(t *testing.T) {
	addr, closer := runserver(t, &smtpd.Server{
		RecipientChecker: func(_ context.Context, _ smtpd.Peer, _ string) error {
			return smtpd.ErrRecipientDenied
		},
		ProtocolLogger: log.New(os.Stdout, "log: ", log.Lshortfile),
	})
	defer closer()

	c, err := smtp.Dial(addr)
	require.NoError(t, err)

	err = c.Mail("sender@example.org")
	require.NoError(t, err)

	err = c.Rcpt("recipient@example.net")
	require.Error(t, err, "RCPT succeeded despite RecipientCheck")
}

func TestMaxMessageSize(t *testing.T) {
	addr, closer := runserver(t, &smtpd.Server{
		MaxMessageSize: 5,
		ProtocolLogger: log.New(os.Stdout, "log: ", log.Lshortfile),
	})
	defer closer()

	c, err := smtp.Dial(addr)
	require.NoError(t, err)

	err = c.Mail("sender@example.org")
	require.NoError(t, err)

	err = c.Rcpt("recipient@example.net")
	require.NoError(t, err)

	wc, err := c.Data()
	require.NoError(t, err)

	_, err = fmt.Fprintf(wc, "This is the email body")
	require.NoError(t, err)

	err = wc.Close()
	require.Error(t, err, "Allowed message larger than 5 bytes to pass.")

	err = c.Quit()
	require.NoError(t, err)
}

func TestHandler(t *testing.T) {
	expectedHeader := textproto.MIMEHeader{}
	body := "This is the email body"

	addr, closer := runserver(t, &smtpd.Server{
		Handler: func(_ context.Context, _ smtpd.Peer, env smtpd.Envelope) error {
			assert.Equal(t, "sender@example.org", env.Sender)
			assert.Len(t, env.Recipients, 1)
			assert.Equal(t, "recipient@example.net", env.Recipients[0])
			assert.Equal(t, body+"\n", string(env.Data))
			assert.Equal(t, env.Header, expectedHeader)

			return nil
		},
		ProtocolLogger: log.New(os.Stdout, "log: ", log.Lshortfile),
	})

	defer closer()

	// no header
	err := smtp.SendMail(addr, nil, "sender@example.org", []string{
		"recipient@example.net",
	}, []byte(body))
	require.NoError(t, err)

	// with header
	expectedHeader = textproto.MIMEHeader{
		"Foo": []string{"bar"},
	}
	body = "Foo: bar\n\nThis is the email body"

	err = smtp.SendMail(addr, nil, "sender@example.org", []string{
		"recipient@example.net",
	}, []byte(body))
	require.NoError(t, err)
}

func TestRejectHandler(t *testing.T) {
	addr, closer := runserver(t, &smtpd.Server{
		Handler: func(_ context.Context, _ smtpd.Peer, _ smtpd.Envelope) error {
			return smtpd.ErrTooBig
		},
		ProtocolLogger: log.New(os.Stdout, "log: ", log.Lshortfile),
	})

	defer closer()

	c, err := smtp.Dial(addr)
	require.NoError(t, err)

	err = c.Mail("sender@example.org")
	require.NoError(t, err)

	err = c.Rcpt("recipient@example.net")
	require.NoError(t, err)

	wc, err := c.Data()
	require.NoError(t, err)

	_, err = fmt.Fprintf(wc, "This is the email body")
	require.NoError(t, err)

	err = wc.Close()
	require.Error(t, err, "Unexpected accept of data")

	err = c.Quit()
	require.NoError(t, err)
}

func TestMaxConnections(t *testing.T) {
	addr, closer := runserver(t, &smtpd.Server{
		MaxConnections: 1,
		ProtocolLogger: log.New(os.Stdout, "log: ", log.Lshortfile),
	})
	defer closer()

	c1, err := smtp.Dial(addr)
	require.NoError(t, err)

	_, err = smtp.Dial(addr)
	require.Error(t, err, "Dial succeeded despite MaxConnections = 1")

	c1.Close()
}

func TestNoMaxConnections(t *testing.T) {
	addr, closer := runserver(t, &smtpd.Server{
		MaxConnections: -1,
		ProtocolLogger: log.New(os.Stdout, "log: ", log.Lshortfile),
	})
	defer closer()

	c1, err := smtp.Dial(addr)
	require.NoError(t, err)

	c1.Close()
}

func TestMaxRecipients(t *testing.T) {
	addr, closer := runserver(t, &smtpd.Server{
		MaxRecipients:  1,
		ProtocolLogger: log.New(os.Stdout, "log: ", log.Lshortfile),
	})
	defer closer()

	c, err := smtp.Dial(addr)
	require.NoError(t, err)

	err = c.Mail("sender@example.org")
	require.NoError(t, err)

	err = c.Rcpt("recipient@example.net")
	require.NoError(t, err)

	err = c.Rcpt("recipient@example.net")
	require.Error(t, err, "RCPT succeeded despite MaxRecipients = 1")

	err = c.Quit()
	require.NoError(t, err)
}

func TestInvalidHelo(t *testing.T) {
	addr, closer := runserver(t, &smtpd.Server{
		ProtocolLogger: log.New(os.Stdout, "log: ", log.Lshortfile),
	})
	defer closer()

	c, err := smtp.Dial(addr)
	require.NoError(t, err)

	err = c.Hello("")
	require.Error(t, err, "HELO succeeded despite empty name")
}

func TestInvalidSender(t *testing.T) {
	addr, closer := runserver(t, &smtpd.Server{
		ProtocolLogger: log.New(os.Stdout, "log: ", log.Lshortfile),
	})
	defer closer()

	c, err := smtp.Dial(addr)
	require.NoError(t, err)

	err = c.Mail("invalid@@example.org")
	require.Error(t, err, "MAIL succeeded despite invalid address")
}

func TestInvalidRecipient(t *testing.T) {
	addr, closer := runserver(t, &smtpd.Server{
		ProtocolLogger: log.New(os.Stdout, "log: ", log.Lshortfile),
	})
	defer closer()

	c, err := smtp.Dial(addr)
	require.NoError(t, err)

	err = c.Mail("sender@example.org")
	require.NoError(t, err)

	err = c.Rcpt("invalid@@example.org")
	require.Error(t, err, "RCPT succeeded despite invalid address")
}

func TestRCPTbeforeMAIL(t *testing.T) {
	addr, closer := runserver(t, &smtpd.Server{
		ProtocolLogger: log.New(os.Stdout, "log: ", log.Lshortfile),
	})
	defer closer()

	c, err := smtp.Dial(addr)
	require.NoError(t, err)

	err = c.Rcpt("recipient@example.net")
	require.Error(t, err, "RCPT succeeded despite no MAIL")
}

func TestDATAbeforeRCPT(t *testing.T) {
	addr, closer := runserver(t, &smtpd.Server{
		ProtocolLogger: log.New(os.Stdout, "log: ", log.Lshortfile),
	})

	defer closer()

	c, err := smtp.Dial(addr)
	require.NoError(t, err)

	err = c.Mail("sender@example.org")
	require.NoError(t, err)

	_, err = c.Data()
	require.Error(t, err, "Data accepted despite no recipients")

	err = c.Quit()
	require.NoError(t, err)
}

func TestInterruptedDATA(t *testing.T) {
	addr, closer := runserver(t, &smtpd.Server{
		Handler: func(_ context.Context, _ smtpd.Peer, _ smtpd.Envelope) error {
			t.Fatal("Accepted DATA despite disconnection")
			return nil
		},
		ProtocolLogger: log.New(os.Stdout, "log: ", log.Lshortfile),
	})

	defer closer()

	c, err := smtp.Dial(addr)
	require.NoError(t, err)

	err = c.Mail("sender@example.org")
	require.NoError(t, err)

	err = c.Rcpt("recipient@example.net")
	require.NoError(t, err)

	wc, err := c.Data()
	require.NoError(t, err)

	_, err = fmt.Fprintf(wc, "This is the email body")
	require.NoError(t, err)

	_ = c.Close()
}

func TestTimeoutClose(t *testing.T) {
	addr, closer := runserver(t, &smtpd.Server{
		MaxConnections: 1,
		ReadTimeout:    1000 * time.Millisecond,
		WriteTimeout:   1000 * time.Millisecond,
		ProtocolLogger: log.New(os.Stdout, "log: ", log.Lshortfile),
	})

	defer closer()

	c1, err := smtp.Dial(addr)
	require.NoError(t, err)

	// TODO: reduce this after fixing Serve to do an exponential backoff instead
	// of sleeping for a full second
	time.Sleep(2000 * time.Millisecond)

	c2, err := smtp.Dial(addr)
	require.NoError(t, err)

	err = c1.Mail("sender@example.org")
	require.Error(t, err, "MAIL succeeded despite being timed out.")

	err = c2.Mail("sender@example.org")
	require.NoError(t, err)

	err = c2.Quit()
	require.NoError(t, err)

	c2.Close()
}

func TestTLSTimeout(t *testing.T) {
	addr, closer := runsslserver(t, &smtpd.Server{
		ReadTimeout:    200 * time.Millisecond,
		WriteTimeout:   200 * time.Millisecond,
		ProtocolLogger: log.New(os.Stdout, "log: ", log.Lshortfile),
	})

	defer closer()

	c, err := smtp.Dial(addr)
	require.NoError(t, err)

	err = c.StartTLS(testTLSConfig)
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	err = c.Mail("sender@example.org")
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	err = c.Rcpt("recipient@example.net")
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	err = c.Rcpt("recipient@example.net")
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	err = c.Quit()
	require.NoError(t, err)
}

func TestLongLine(t *testing.T) {
	addr, closer := runserver(t, &smtpd.Server{
		ProtocolLogger: log.New(os.Stdout, "log: ", log.Lshortfile),
	})
	defer closer()

	c, err := smtp.Dial(addr)
	require.NoError(t, err)

	err = c.Mail(fmt.Sprintf("%s@example.org", strings.Repeat("x", 65*1024)))
	require.Error(t, err)

	err = c.Quit()
	require.NoError(t, err)
}

func TestXCLIENT(t *testing.T) {
	addr, closer := runserver(t, &smtpd.Server{
		EnableXCLIENT: true,
		SenderChecker: func(_ context.Context, peer smtpd.Peer, addr string) error {
			require.Equal(t, "new.example.net", peer.HeloName)
			require.Equal(t, "42.42.42.42:4242", peer.Addr.String())
			require.Equal(t, "newusername", peer.Username)
			require.Equal(t, smtpd.SMTP, peer.Protocol)
			require.Equal(t, "sender@example.org", addr)

			return nil
		},
		ProtocolLogger: log.New(os.Stdout, "log: ", log.Lshortfile),
	})
	defer closer()

	c, err := smtp.Dial(addr)
	require.NoError(t, err)

	supported, _ := c.Extension("XCLIENT")
	require.True(t, supported, "XCLIENT not supported")

	err = cmd(c.Text, 220, "XCLIENT NAME=ignored ADDR=42.42.42.42 PORT=4242 PROTO=SMTP HELO=new.example.net LOGIN=newusername")
	require.NoError(t, err)

	err = c.Mail("sender@example.org")
	require.NoError(t, err)

	err = c.Rcpt("recipient@example.net")
	require.NoError(t, err)

	err = c.Rcpt("recipient2@example.net")
	require.NoError(t, err)

	wc, err := c.Data()
	require.NoError(t, err)

	_, err = fmt.Fprintf(wc, "This is the email body")
	require.NoError(t, err)

	err = wc.Close()
	require.NoError(t, err)

	err = c.Quit()
	require.NoError(t, err)
}

func TestEnvelopeReceived(t *testing.T) {
	addr, closer := runsslserver(t, &smtpd.Server{
		Hostname: "foobar.example.net",
		Handler: func(_ context.Context, peer smtpd.Peer, env smtpd.Envelope) error {
			env.AddReceivedLine(peer)
			if !bytes.HasPrefix(env.Data, []byte("Received: from localhost ([127.0.0.1]) by foobar.example.net with ESMTP;")) {
				t.Fatal("Wrong received line.")
			}
			return nil
		},
		ForceTLS:       true,
		ProtocolLogger: log.New(os.Stdout, "log: ", log.Lshortfile),
	})
	defer closer()

	c, err := smtp.Dial(addr)
	require.NoError(t, err)

	err = c.StartTLS(testTLSConfig)
	require.NoError(t, err)

	err = c.Mail("sender@example.org")
	require.NoError(t, err)

	err = c.Rcpt("recipient@example.net")
	require.NoError(t, err)

	wc, err := c.Data()
	require.NoError(t, err)

	_, err = fmt.Fprintf(wc, "This is the email body")
	require.NoError(t, err)

	err = wc.Close()
	require.NoError(t, err)

	err = c.Quit()
	require.NoError(t, err)
}

func TestHELO(t *testing.T) {
	addr, closer := runserver(t, &smtpd.Server{
		ProtocolLogger: log.New(os.Stdout, "log: ", log.Lshortfile),
	})

	defer closer()

	c, err := smtp.Dial(addr)
	require.NoError(t, err)

	err = cmd(c.Text, 502, "MAIL FROM:<test@example.org>")
	require.NoError(t, err)

	err = cmd(c.Text, 250, "HELO localhost")
	require.NoError(t, err)

	err = cmd(c.Text, 250, "MAIL FROM:<test@example.org>")
	require.NoError(t, err)

	err = cmd(c.Text, 250, "HELO localhost")
	require.NoError(t, err)

	err = c.Quit()
	require.NoError(t, err)
}

func TestLOGINAuth(t *testing.T) {
	addr, closer := runsslserver(t, &smtpd.Server{
		Authenticator:  func(_ context.Context, _ smtpd.Peer, _, _ string) error { return nil },
		ProtocolLogger: log.New(os.Stdout, "log: ", log.Lshortfile),
	})

	defer closer()

	c, err := smtp.Dial(addr)
	require.NoError(t, err)

	err = c.StartTLS(testTLSConfig)
	require.NoError(t, err)

	err = cmd(c.Text, 334, "AUTH LOGIN")
	require.NoError(t, err)

	err = cmd(c.Text, 502, "foo")
	require.NoError(t, err)

	err = cmd(c.Text, 334, "AUTH LOGIN")
	require.NoError(t, err)

	err = cmd(c.Text, 334, "Zm9v")
	require.NoError(t, err)

	err = cmd(c.Text, 502, "foo")
	require.NoError(t, err)

	err = cmd(c.Text, 334, "AUTH LOGIN")
	require.NoError(t, err)

	err = cmd(c.Text, 334, "Zm9v")
	require.NoError(t, err)

	err = cmd(c.Text, 235, "Zm9v")
	require.NoError(t, err)

	err = c.Quit()
	require.NoError(t, err)
}

func TestMailFrom(t *testing.T) {
	addr, closer := runserver(t, &smtpd.Server{
		ProtocolLogger: log.New(os.Stdout, "log: ", log.Lshortfile),
	})
	defer closer()

	testdata := []struct {
		name, from string
	}{
		{"null", "<>"},
		{"no brackets", "test@example.org"},
	}

	for _, d := range testdata {
		t.Run(d.name, func(t *testing.T) {
			c, err := smtp.Dial(addr)
			require.NoError(t, err)

			err = cmd(c.Text, 250, "HELO localhost")
			require.NoError(t, err)

			err = cmd(c.Text, 250, "MAIL FROM:%s", d.from)
			require.NoError(t, err)

			err = c.Quit()
			require.NoError(t, err)
		})
	}
}

func TestErrors(t *testing.T) {
	cert, err := tls.X509KeyPair(localhostCert, localhostKey)
	require.NoError(t, err)

	server := &smtpd.Server{
		Authenticator:  func(_ context.Context, _ smtpd.Peer, _, _ string) error { return nil },
		ProtocolLogger: log.New(os.Stdout, "log: ", log.Lshortfile),
	}

	addr, closer := runserver(t, server)
	defer closer()

	c, err := smtp.Dial(addr)
	require.NoError(t, err)

	err = cmd(c.Text, 502, "AUTH PLAIN foobar")
	require.NoError(t, err)

	err = c.Hello("localhost")
	require.NoError(t, err)

	err = cmd(c.Text, 502, "AUTH PLAIN foobar")
	require.NoError(t, err)

	err = c.Mail("sender@example.org")
	require.Error(t, err, "MAIL didn't fail")

	err = cmd(c.Text, 502, "STARTTLS")
	require.NoError(t, err)

	server.TLSConfig = &tls.Config{
		MinVersion:   tls.VersionTLS12,
		Certificates: []tls.Certificate{cert},
	}

	err = c.StartTLS(testTLSConfig)
	require.NoError(t, err)

	err = cmd(c.Text, 502, "AUTH UNKNOWN")
	require.NoError(t, err)

	err = cmd(c.Text, 502, "AUTH PLAIN foobar")
	require.NoError(t, err)

	err = cmd(c.Text, 502, "AUTH PLAIN Zm9vAGJhcg==")
	require.NoError(t, err)

	err = cmd(c.Text, 334, "AUTH PLAIN")
	require.NoError(t, err)

	err = cmd(c.Text, 235, "Zm9vAGJhcgBxdXV4")
	require.NoError(t, err)

	err = c.Mail("sender@example.org")
	require.NoError(t, err)

	err = c.Mail("sender@example.org")
	require.Error(t, err, "Duplicate MAIL didn't fail")

	err = c.Quit()
	require.NoError(t, err)
}

func TestMalformedMAILFROM(t *testing.T) {
	addr, closer := runserver(t, &smtpd.Server{
		SenderChecker: func(_ context.Context, _ smtpd.Peer, addr string) error {
			if addr != "test@example.org" {
				return smtpd.ErrRecipientDenied
			}
			return nil
		},
		ProtocolLogger: log.New(os.Stdout, "log: ", log.Lshortfile),
	})
	defer closer()

	c, err := smtp.Dial(addr)
	require.NoError(t, err)

	err = c.Hello("localhost")
	require.NoError(t, err)

	err = cmd(c.Text, 250, "MAIL FROM: <test@example.org>")
	require.NoError(t, err)

	err = c.Quit()
	require.NoError(t, err)
}

func TestTLSListener(t *testing.T) {
	cert, err := tls.X509KeyPair(localhostCert, localhostKey)
	require.NoError(t, err)

	cfg := &tls.Config{
		MinVersion:   tls.VersionTLS12,
		Certificates: []tls.Certificate{cert},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ln, err := tls.Listen("tcp", "127.0.0.1:0", cfg)
	require.NoError(t, err)

	defer ln.Close()

	addr := ln.Addr().String()

	server := &smtpd.Server{
		Authenticator: func(_ context.Context, peer smtpd.Peer, _, _ string) error {
			require.NotNil(t, peer.TLS, "didn't correctly set connection state on TLS connection")
			return nil
		},
		ProtocolLogger: log.New(os.Stdout, "log: ", log.Lshortfile),
	}

	go func() {
		_ = server.Serve(ctx, ln)
	}()

	conn, err := tls.Dial("tcp", addr, testTLSConfig)
	require.NoError(t, err)

	c, err := smtp.NewClient(conn, "localhost")
	require.NoError(t, err)

	err = c.Hello("localhost")
	require.NoError(t, err)

	err = cmd(c.Text, 334, "AUTH PLAIN")
	require.NoError(t, err)

	err = cmd(c.Text, 235, "Zm9vAGJhcgBxdXV4")
	require.NoError(t, err)

	err = c.Quit()
	require.NoError(t, err)
}

func TestShutdown(t *testing.T) {
	server := &smtpd.Server{
		ProtocolLogger: log.New(os.Stdout, "log: ", log.Lshortfile),
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	srvres := make(chan error)
	go func() {
		t.Log("Starting server")
		srvres <- server.Serve(ctx, ln)
	}()

	// Connect a client
	c, err := smtp.Dial(ln.Addr().String())
	require.NoError(t, err)

	err = c.Hello("localhost")
	require.NoError(t, err)

	// While the client connection is open, shut down the server (without
	// waiting for it to finish)
	err = server.Shutdown(false)
	require.NoError(t, err)

	// Verify that Shutdown() worked by attempting to connect another client
	_, err = smtp.Dial(ln.Addr().String())
	require.Error(t, err, "Dial did not fail as expected")

	var operr *net.OpError
	require.ErrorAs(t, err, &operr, "Dial did not return net.OpError as expected")

	// Wait for shutdown to complete
	shutres := make(chan error)
	go func() {
		t.Log("Waiting for server shutdown to finish")
		shutres <- server.Wait()
	}()

	// Slight delay to ensure Shutdown() blocks
	time.Sleep(250 * time.Millisecond)

	// Wait() should not have returned yet due to open client conn
	select {
	case shuterr := <-shutres:
		t.Fatalf("Wait() returned early w/ error: %v", shuterr)
	default:
	}

	// Now close the client
	t.Log("Closing client connection")
	err = c.Quit()
	require.NoError(t, err)

	_ = c.Close()

	// Wait for Wait() to return
	t.Log("Waiting for Wait() to return")
	select {
	case shuterr := <-shutres:
		require.NoError(t, shuterr)
	case <-time.After(15 * time.Second):
		t.Fatalf("Timed out waiting for Wait() to return")
	}

	// Wait for Serve() to return
	t.Log("Waiting for Serve() to return")
	select {
	case srverr := <-srvres:
		require.ErrorIs(t, srverr, smtpd.ErrServerClosed)
	case <-time.After(15 * time.Second):
		t.Fatalf("Timed out waiting for Serve() to return")
	}
}

func TestServeFailsIfShutdown(t *testing.T) {
	server := &smtpd.Server{}
	err := server.Shutdown(true)
	require.NoError(t, err)

	err = server.Serve(context.Background(), nil)
	require.ErrorIs(t, err, smtpd.ErrServerClosed)
}

func TestWaitFailsIfNotShutdown(t *testing.T) {
	server := &smtpd.Server{}
	err := server.Wait()
	require.Error(t, err)
}

func TestServe_Context(t *testing.T) {
	lc := net.ListenConfig{}

	t.Run("cancelled context", func(t *testing.T) {
		server := &smtpd.Server{}
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		ln, err := lc.Listen(ctx, "tcp", "127.0.0.1:0")
		require.NoError(t, err)

		t.Cleanup(func() {
			_ = ln.Close()
		})

		errch := make(chan error)
		go func() {
			errch <- server.Serve(ctx, ln)
		}()

		select {
		case <-time.After(1 * time.Second):
			t.Fatal("Timed out waiting for Serve() to return")
		case err := <-errch:
			require.Error(t, err)
		}
	})

	t.Run("cancelled context after serve", func(t *testing.T) {
		server := &smtpd.Server{}
		t.Cleanup(func() {
			_ = server.Shutdown(false)
		})

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		ln, err := lc.Listen(ctx, "tcp", "127.0.0.1:0")
		require.NoError(t, err)

		t.Cleanup(func() {
			_ = ln.Close()
		})

		errch := make(chan error)
		go func() {
			errch <- server.Serve(ctx, ln)
		}()

		cancel()

		select {
		case <-time.After(1 * time.Second):
			t.Fatal("Timed out waiting for Serve() to return")
		case err := <-errch:
			require.Error(t, err)
		}
	})

	t.Run("cancelled context after serve and accept", func(t *testing.T) {
		server := &smtpd.Server{}
		t.Cleanup(func() {
			_ = server.Shutdown(false)
		})

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		ln, err := lc.Listen(ctx, "tcp", "127.0.0.1:0")
		require.NoError(t, err)

		t.Cleanup(func() {
			_ = ln.Close()
		})

		errch := make(chan error)
		go func() {
			errch <- server.Serve(ctx, ln)
		}()

		client, err := smtp.Dial(ln.Addr().String())
		require.NoError(t, err)

		t.Cleanup(func() {
			_ = client.Close()
		})

		// wait enough time for Serve() to get into the accept loop
		time.Sleep(50 * time.Millisecond)
		cancel()

		select {
		case <-time.After(1 * time.Second):
			t.Fatal("Timed out waiting for Serve() to return")
		case err := <-errch:
			require.Error(t, err)
		}
	})

	t.Run("connection context cancelled doesn't close server", func(t *testing.T) {
		server := &smtpd.Server{
			ConnContext: func(ctx context.Context, _ net.Conn) context.Context {
				ctx, cancel := context.WithCancel(ctx)
				cancel()
				return ctx
			},
		}
		t.Cleanup(func() {
			_ = server.Shutdown(false)
		})

		ln, err := lc.Listen(context.Background(), "tcp", "127.0.0.1:0")
		require.NoError(t, err)

		t.Cleanup(func() {
			_ = ln.Close()
		})

		errch := make(chan error)
		go func() {
			errch <- server.Serve(context.Background(), ln)
		}()

		_, err = smtp.Dial(ln.Addr().String())
		require.Error(t, err)
	})
}
