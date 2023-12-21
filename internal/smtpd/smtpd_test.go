package smtpd_test

import (
	"bytes"
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

	"github.com/grafana/smtprelay/internal/smtpd"
	"github.com/stretchr/testify/require"
)

var localhostCert = []byte(`-----BEGIN CERTIFICATE-----
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

var localhostKey = []byte(`-----BEGIN PRIVATE KEY-----
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
hBgHM6A0WJC9MO3aAKRBcp48y6DXNA==
-----END PRIVATE KEY-----`)

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

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen failed: %v", err)
	}

	go func() {
		_ = server.Serve(ln)
	}()

	done := make(chan bool)

	go func() {
		<-done
		ln.Close()
	}()

	return ln.Addr().String(), func() {
		done <- true
	}

}

func runsslserver(t *testing.T, server *smtpd.Server) (addr string, closer func()) {
	t.Helper()

	cert, err := tls.X509KeyPair(localhostCert, localhostKey)
	if err != nil {
		t.Fatalf("Cert load failed: %v", err)
	}

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
	if err != nil {
		t.Fatalf("Dial failed: %v", err)
	}

	if err := c.Hello("localhost"); err != nil {
		t.Fatalf("HELO failed: %v", err)
	}

	if supported, _ := c.Extension("AUTH"); supported {
		t.Fatal("AUTH supported before TLS")
	}

	if supported, _ := c.Extension("8BITMIME"); !supported {
		t.Fatal("8BITMIME not supported")
	}

	if supported, _ := c.Extension("STARTTLS"); supported {
		t.Fatal("STARTTLS supported")
	}

	if err := c.Mail("sender@example.org"); err != nil {
		t.Fatalf("Mail failed: %v", err)
	}

	if err := c.Rcpt("recipient@example.net"); err != nil {
		t.Fatalf("Rcpt failed: %v", err)
	}

	if err := c.Rcpt("recipient2@example.net"); err != nil {
		t.Fatalf("Rcpt2 failed: %v", err)
	}

	wc, err := c.Data()
	if err != nil {
		t.Fatalf("Data failed: %v", err)
	}

	_, err = fmt.Fprintf(wc, "This is the email body")
	if err != nil {
		t.Fatalf("Data body failed: %v", err)
	}

	err = wc.Close()
	if err != nil {
		t.Fatalf("Data close failed: %v", err)
	}

	if err := c.Reset(); err != nil {
		t.Fatalf("Reset failed: %v", err)
	}

	if err := c.Verify("foobar@example.net"); err == nil {
		t.Fatal("Unexpected support for VRFY")
	}

	if err := cmd(c.Text, 250, "NOOP"); err != nil {
		t.Fatalf("NOOP failed: %v", err)
	}

	if err := c.Quit(); err != nil {
		t.Fatalf("Quit failed: %v", err)
	}
}

func TestListenAndServe(t *testing.T) {

	addr, closer := runserver(t, &smtpd.Server{})
	closer()

	server := &smtpd.Server{
		ProtocolLogger: log.New(os.Stdout, "log: ", log.Lshortfile),
	}

	go func() {
		_ = server.ListenAndServe(addr)
	}()

	time.Sleep(100 * time.Millisecond)

	c, err := smtp.Dial(addr)
	if err != nil {
		t.Fatalf("Dial failed: %v", err)
	}

	if err := c.Quit(); err != nil {
		t.Fatalf("Quit failed: %v", err)
	}

}

func TestSTARTTLS(t *testing.T) {

	addr, closer := runsslserver(t, &smtpd.Server{
		Authenticator:  func(peer smtpd.Peer, username, password string) error { return nil },
		ForceTLS:       true,
		ProtocolLogger: log.New(os.Stdout, "log: ", log.Lshortfile),
	})

	defer closer()

	c, err := smtp.Dial(addr)
	if err != nil {
		t.Fatalf("Dial failed: %v", err)
	}

	if supported, _ := c.Extension("AUTH"); supported {
		t.Fatal("AUTH supported before TLS")
	}

	if err := c.Mail("sender@example.org"); err == nil {
		t.Fatal("Mail workded before TLS with ForceTLS")
	}

	if err := cmd(c.Text, 220, "STARTTLS"); err != nil {
		t.Fatalf("STARTTLS failed: %v", err)
	}

	if err := cmd(c.Text, 250, "foobar"); err == nil {
		t.Fatal("STARTTLS didn't fail with invalid handshake")
	}

	//nolint:gosec
	if err := c.StartTLS(&tls.Config{InsecureSkipVerify: true}); err != nil {
		t.Fatalf("STARTTLS failed: %v", err)
	}

	//nolint:gosec
	if err := c.StartTLS(&tls.Config{InsecureSkipVerify: true}); err == nil {
		t.Fatal("STARTTLS worked twice")
	}

	if supported, _ := c.Extension("AUTH"); !supported {
		t.Fatal("AUTH not supported after TLS")
	}

	if _, mechs := c.Extension("AUTH"); !strings.Contains(mechs, "PLAIN") {
		t.Fatal("PLAIN AUTH not supported after TLS")
	}

	if _, mechs := c.Extension("AUTH"); !strings.Contains(mechs, "LOGIN") {
		t.Fatal("LOGIN AUTH not supported after TLS")
	}

	if err := c.Auth(smtp.PlainAuth("foo", "foo", "bar", "127.0.0.1")); err != nil {
		t.Fatalf("Auth failed: %v", err)
	}

	if err := c.Mail("sender@example.org"); err != nil {
		t.Fatalf("Mail failed: %v", err)
	}

	if err := c.Rcpt("recipient@example.net"); err != nil {
		t.Fatalf("Rcpt failed: %v", err)
	}

	if err := c.Rcpt("recipient2@example.net"); err != nil {
		t.Fatalf("Rcpt2 failed: %v", err)
	}

	wc, err := c.Data()
	if err != nil {
		t.Fatalf("Data failed: %v", err)
	}

	_, err = fmt.Fprintf(wc, "This is the email body")
	if err != nil {
		t.Fatalf("Data body failed: %v", err)
	}

	err = wc.Close()
	if err != nil {
		t.Fatalf("Data close failed: %v", err)
	}

	if err := c.Quit(); err != nil {
		t.Fatalf("Quit failed: %v", err)
	}
}

func TestAuthRejection(t *testing.T) {

	addr, closer := runsslserver(t, &smtpd.Server{
		Authenticator: func(peer smtpd.Peer, username, password string) error {
			return smtpd.Error{Code: 550, Message: "Denied"}
		},
		ForceTLS:       true,
		ProtocolLogger: log.New(os.Stdout, "log: ", log.Lshortfile),
	})

	defer closer()

	c, err := smtp.Dial(addr)
	if err != nil {
		t.Fatalf("Dial failed: %v", err)
	}

	//nolint:gosec
	if err := c.StartTLS(&tls.Config{InsecureSkipVerify: true}); err != nil {
		t.Fatalf("STARTTLS failed: %v", err)
	}

	if err := c.Auth(smtp.PlainAuth("foo", "foo", "bar", "127.0.0.1")); err == nil {
		t.Fatal("Auth worked despite rejection")
	}

}

func TestAuthNotSupported(t *testing.T) {

	addr, closer := runsslserver(t, &smtpd.Server{
		ForceTLS:       true,
		ProtocolLogger: log.New(os.Stdout, "log: ", log.Lshortfile),
	})

	defer closer()

	c, err := smtp.Dial(addr)
	if err != nil {
		t.Fatalf("Dial failed: %v", err)
	}

	//nolint:gosec
	if err := c.StartTLS(&tls.Config{InsecureSkipVerify: true}); err != nil {
		t.Fatalf("STARTTLS failed: %v", err)
	}

	if err := c.Auth(smtp.PlainAuth("foo", "foo", "bar", "127.0.0.1")); err == nil {
		t.Fatal("Auth worked despite no authenticator")
	}

}

func TestAuthBypass(t *testing.T) {

	addr, closer := runsslserver(t, &smtpd.Server{
		Authenticator: func(peer smtpd.Peer, username, password string) error {
			return smtpd.Error{Code: 550, Message: "Denied"}
		},
		ForceTLS:       true,
		ProtocolLogger: log.New(os.Stdout, "log: ", log.Lshortfile),
	})

	defer closer()

	c, err := smtp.Dial(addr)
	if err != nil {
		t.Fatalf("Dial failed: %v", err)
	}

	//nolint:gosec
	if err := c.StartTLS(&tls.Config{InsecureSkipVerify: true}); err != nil {
		t.Fatalf("STARTTLS failed: %v", err)
	}

	if err := c.Mail("sender@example.org"); err == nil {
		t.Fatal("Unexpected MAIL success")
	}

}

func TestConnectionCheck(t *testing.T) {

	addr, closer := runserver(t, &smtpd.Server{
		ConnectionChecker: func(peer smtpd.Peer) error {
			return smtpd.Error{Code: 552, Message: "Denied"}
		},
		ProtocolLogger: log.New(os.Stdout, "log: ", log.Lshortfile),
	})

	defer closer()

	if _, err := smtp.Dial(addr); err == nil {
		t.Fatal("Dial succeeded despite ConnectionCheck")
	}

}

func TestConnectionCheckSimpleError(t *testing.T) {

	addr, closer := runserver(t, &smtpd.Server{
		ConnectionChecker: func(peer smtpd.Peer) error {
			return errors.New("Denied")
		},
		ProtocolLogger: log.New(os.Stdout, "log: ", log.Lshortfile),
	})

	defer closer()

	if _, err := smtp.Dial(addr); err == nil {
		t.Fatal("Dial succeeded despite ConnectionCheck")
	}

}

func TestHELOCheck(t *testing.T) {

	addr, closer := runserver(t, &smtpd.Server{
		HeloChecker: func(peer smtpd.Peer, name string) error {
			if name != "foobar.local" {
				t.Fatal("Wrong HELO name")
			}
			return smtpd.Error{Code: 552, Message: "Denied"}
		},
		ProtocolLogger: log.New(os.Stdout, "log: ", log.Lshortfile),
	})

	defer closer()

	c, err := smtp.Dial(addr)
	if err != nil {
		t.Fatalf("Dial failed: %v", err)
	}

	if err := c.Hello("foobar.local"); err == nil {
		t.Fatal("Unexpected HELO success")
	}

}

func TestSenderCheck(t *testing.T) {

	addr, closer := runserver(t, &smtpd.Server{
		SenderChecker: func(peer smtpd.Peer, addr string) error {
			return smtpd.Error{Code: 552, Message: "Denied"}
		},
		ProtocolLogger: log.New(os.Stdout, "log: ", log.Lshortfile),
	})

	defer closer()

	c, err := smtp.Dial(addr)
	if err != nil {
		t.Fatalf("Dial failed: %v", err)
	}

	if err := c.Mail("sender@example.org"); err == nil {
		t.Fatal("Unexpected MAIL success")
	}

}

func TestRecipientCheck(t *testing.T) {

	addr, closer := runserver(t, &smtpd.Server{
		RecipientChecker: func(peer smtpd.Peer, addr string) error {
			return smtpd.Error{Code: 552, Message: "Denied"}
		},
		ProtocolLogger: log.New(os.Stdout, "log: ", log.Lshortfile),
	})

	defer closer()

	c, err := smtp.Dial(addr)
	if err != nil {
		t.Fatalf("Dial failed: %v", err)
	}

	if err := c.Mail("sender@example.org"); err != nil {
		t.Fatalf("Mail failed: %v", err)
	}

	if err := c.Rcpt("recipient@example.net"); err == nil {
		t.Fatal("Unexpected RCPT success")
	}

}

func TestMaxMessageSize(t *testing.T) {

	addr, closer := runserver(t, &smtpd.Server{
		MaxMessageSize: 5,
		ProtocolLogger: log.New(os.Stdout, "log: ", log.Lshortfile),
	})

	defer closer()

	c, err := smtp.Dial(addr)
	if err != nil {
		t.Fatalf("Dial failed: %v", err)
	}

	if err := c.Mail("sender@example.org"); err != nil {
		t.Fatalf("MAIL failed: %v", err)
	}

	if err := c.Rcpt("recipient@example.net"); err != nil {
		t.Fatalf("RCPT failed: %v", err)
	}

	wc, err := c.Data()
	if err != nil {
		t.Fatalf("Data failed: %v", err)
	}

	_, err = fmt.Fprintf(wc, "This is the email body")
	if err != nil {
		t.Fatalf("Data body failed: %v", err)
	}

	err = wc.Close()
	if err == nil {
		t.Fatal("Allowed message larger than 5 bytes to pass.")
	}

	if err := c.Quit(); err != nil {
		t.Fatalf("QUIT failed: %v", err)
	}

}

func TestHandler(t *testing.T) {

	addr, closer := runserver(t, &smtpd.Server{
		Handler: func(peer smtpd.Peer, env smtpd.Envelope) error {
			if env.Sender != "sender@example.org" {
				t.Fatalf("Unknown sender: %v", env.Sender)
			}
			if len(env.Recipients) != 1 {
				t.Fatalf("Too many recipients: %d", len(env.Recipients))
			}
			if env.Recipients[0] != "recipient@example.net" {
				t.Fatalf("Unknown recipient: %v", env.Recipients[0])
			}
			if string(env.Data) != "This is the email body\n" {
				t.Fatalf("Wrong message body: %v", string(env.Data))
			}
			return nil
		},
		ProtocolLogger: log.New(os.Stdout, "log: ", log.Lshortfile),
	})

	defer closer()

	c, err := smtp.Dial(addr)
	if err != nil {
		t.Fatalf("Dial failed: %v", err)
	}

	if err := c.Mail("sender@example.org"); err != nil {
		t.Fatalf("MAIL failed: %v", err)
	}

	if err := c.Rcpt("recipient@example.net"); err != nil {
		t.Fatalf("RCPT failed: %v", err)
	}

	wc, err := c.Data()
	if err != nil {
		t.Fatalf("Data failed: %v", err)
	}

	_, err = fmt.Fprintf(wc, "This is the email body")
	if err != nil {
		t.Fatalf("Data body failed: %v", err)
	}

	err = wc.Close()
	if err != nil {
		t.Fatalf("Data close failed: %v", err)
	}

	if err := c.Quit(); err != nil {
		t.Fatalf("QUIT failed: %v", err)
	}

}

func TestRejectHandler(t *testing.T) {

	addr, closer := runserver(t, &smtpd.Server{
		Handler: func(peer smtpd.Peer, env smtpd.Envelope) error {
			return smtpd.Error{Code: 550, Message: "Rejected"}
		},
		ProtocolLogger: log.New(os.Stdout, "log: ", log.Lshortfile),
	})

	defer closer()

	c, err := smtp.Dial(addr)
	if err != nil {
		t.Fatalf("Dial failed: %v", err)
	}

	if err := c.Mail("sender@example.org"); err != nil {
		t.Fatalf("MAIL failed: %v", err)
	}

	if err := c.Rcpt("recipient@example.net"); err != nil {
		t.Fatalf("RCPT failed: %v", err)
	}

	wc, err := c.Data()
	if err != nil {
		t.Fatalf("Data failed: %v", err)
	}

	_, err = fmt.Fprintf(wc, "This is the email body")
	if err != nil {
		t.Fatalf("Data body failed: %v", err)
	}

	err = wc.Close()
	if err == nil {
		t.Fatal("Unexpected accept of data")
	}

	if err := c.Quit(); err != nil {
		t.Fatalf("QUIT failed: %v", err)
	}

}

func TestMaxConnections(t *testing.T) {

	addr, closer := runserver(t, &smtpd.Server{
		MaxConnections: 1,
		ProtocolLogger: log.New(os.Stdout, "log: ", log.Lshortfile),
	})

	defer closer()

	c1, err := smtp.Dial(addr)
	if err != nil {
		t.Fatalf("Dial failed: %v", err)
	}

	_, err = smtp.Dial(addr)
	if err == nil {
		t.Fatal("Dial succeeded despite MaxConnections = 1")
	}

	c1.Close()
}

func TestNoMaxConnections(t *testing.T) {

	addr, closer := runserver(t, &smtpd.Server{
		MaxConnections: -1,
		ProtocolLogger: log.New(os.Stdout, "log: ", log.Lshortfile),
	})

	defer closer()

	c1, err := smtp.Dial(addr)
	if err != nil {
		t.Fatalf("Dial failed: %v", err)
	}

	c1.Close()
}

func TestMaxRecipients(t *testing.T) {

	addr, closer := runserver(t, &smtpd.Server{
		MaxRecipients:  1,
		ProtocolLogger: log.New(os.Stdout, "log: ", log.Lshortfile),
	})

	defer closer()

	c, err := smtp.Dial(addr)
	if err != nil {
		t.Fatalf("Dial failed: %v", err)
	}

	if err := c.Mail("sender@example.org"); err != nil {
		t.Fatalf("MAIL failed: %v", err)
	}

	if err := c.Rcpt("recipient@example.net"); err != nil {
		t.Fatalf("RCPT failed: %v", err)
	}

	if err := c.Rcpt("recipient@example.net"); err == nil {
		t.Fatal("RCPT succeeded despite MaxRecipients = 1")
	}

	if err := c.Quit(); err != nil {
		t.Fatalf("QUIT failed: %v", err)
	}

}

func TestInvalidHelo(t *testing.T) {

	addr, closer := runserver(t, &smtpd.Server{
		ProtocolLogger: log.New(os.Stdout, "log: ", log.Lshortfile),
	})

	defer closer()

	c, err := smtp.Dial(addr)
	if err != nil {
		t.Fatalf("Dial failed: %v", err)
	}

	if err := c.Hello(""); err == nil {
		t.Fatal("Unexpected HELO success")
	}

}

func TestInvalidSender(t *testing.T) {

	addr, closer := runserver(t, &smtpd.Server{
		ProtocolLogger: log.New(os.Stdout, "log: ", log.Lshortfile),
	})

	defer closer()

	c, err := smtp.Dial(addr)
	if err != nil {
		t.Fatalf("Dial failed: %v", err)
	}

	if err := c.Mail("invalid@@example.org"); err == nil {
		t.Fatal("Unexpected MAIL success")
	}

}

func TestInvalidRecipient(t *testing.T) {

	addr, closer := runserver(t, &smtpd.Server{
		ProtocolLogger: log.New(os.Stdout, "log: ", log.Lshortfile),
	})

	defer closer()

	c, err := smtp.Dial(addr)
	if err != nil {
		t.Fatalf("Dial failed: %v", err)
	}

	if err := c.Mail("sender@example.org"); err != nil {
		t.Fatalf("Mail failed: %v", err)
	}

	if err := c.Rcpt("invalid@@example.org"); err == nil {
		t.Fatal("Unexpected RCPT success")
	}

}

func TestRCPTbeforeMAIL(t *testing.T) {

	addr, closer := runserver(t, &smtpd.Server{
		ProtocolLogger: log.New(os.Stdout, "log: ", log.Lshortfile),
	})

	defer closer()

	c, err := smtp.Dial(addr)
	if err != nil {
		t.Fatalf("Dial failed: %v", err)
	}

	if err := c.Rcpt("recipient@example.net"); err == nil {
		t.Fatal("Unexpected RCPT success")
	}

}

func TestDATAbeforeRCPT(t *testing.T) {

	addr, closer := runserver(t, &smtpd.Server{
		ProtocolLogger: log.New(os.Stdout, "log: ", log.Lshortfile),
	})

	defer closer()

	c, err := smtp.Dial(addr)
	if err != nil {
		t.Fatalf("Dial failed: %v", err)
	}

	if err := c.Mail("sender@example.org"); err != nil {
		t.Fatalf("MAIL failed: %v", err)
	}

	if _, err := c.Data(); err == nil {
		t.Fatal("Data accepted despite no recipients")
	}

	if err := c.Quit(); err != nil {
		t.Fatalf("QUIT failed: %v", err)
	}

}

func TestInterruptedDATA(t *testing.T) {

	addr, closer := runserver(t, &smtpd.Server{
		Handler: func(peer smtpd.Peer, env smtpd.Envelope) error {
			t.Fatal("Accepted DATA despite disconnection")
			return nil
		},
		ProtocolLogger: log.New(os.Stdout, "log: ", log.Lshortfile),
	})

	defer closer()

	c, err := smtp.Dial(addr)
	if err != nil {
		t.Fatalf("Dial failed: %v", err)
	}

	if err := c.Mail("sender@example.org"); err != nil {
		t.Fatalf("MAIL failed: %v", err)
	}

	if err := c.Rcpt("recipient@example.net"); err != nil {
		t.Fatalf("RCPT failed: %v", err)
	}

	wc, err := c.Data()
	if err != nil {
		t.Fatalf("Data failed: %v", err)
	}

	_, err = fmt.Fprintf(wc, "This is the email body")
	if err != nil {
		t.Fatalf("Data body failed: %v", err)
	}

	c.Close()

}

func TestTimeoutClose(t *testing.T) {

	addr, closer := runserver(t, &smtpd.Server{
		MaxConnections: 1,
		ReadTimeout:    time.Second,
		WriteTimeout:   time.Second,
		ProtocolLogger: log.New(os.Stdout, "log: ", log.Lshortfile),
	})

	defer closer()

	c1, err := smtp.Dial(addr)
	if err != nil {
		t.Fatalf("Dial failed: %v", err)
	}

	time.Sleep(time.Second * 2)

	c2, err := smtp.Dial(addr)
	if err != nil {
		t.Fatalf("Dial failed: %v", err)
	}

	if err := c1.Mail("sender@example.org"); err == nil {
		t.Fatal("MAIL succeeded despite being timed out.")
	}

	if err := c2.Mail("sender@example.org"); err != nil {
		t.Fatalf("MAIL failed: %v", err)
	}

	if err := c2.Quit(); err != nil {
		t.Fatalf("Quit failed: %v", err)
	}

	c2.Close()
}

func TestTLSTimeout(t *testing.T) {

	addr, closer := runsslserver(t, &smtpd.Server{
		ReadTimeout:    time.Second * 2,
		WriteTimeout:   time.Second * 2,
		ProtocolLogger: log.New(os.Stdout, "log: ", log.Lshortfile),
	})

	defer closer()

	c, err := smtp.Dial(addr)
	if err != nil {
		t.Fatalf("Dial failed: %v", err)
	}

	//nolint:gosec
	if err := c.StartTLS(&tls.Config{InsecureSkipVerify: true}); err != nil {
		t.Fatalf("STARTTLS failed: %v", err)
	}

	time.Sleep(time.Second)

	if err := c.Mail("sender@example.org"); err != nil {
		t.Fatalf("MAIL failed: %v", err)
	}

	time.Sleep(time.Second)

	if err := c.Rcpt("recipient@example.net"); err != nil {
		t.Fatalf("RCPT failed: %v", err)
	}

	time.Sleep(time.Second)

	if err := c.Rcpt("recipient@example.net"); err != nil {
		t.Fatalf("RCPT failed: %v", err)
	}

	time.Sleep(time.Second)

	if err := c.Quit(); err != nil {
		t.Fatalf("Quit failed: %v", err)
	}

}

func TestLongLine(t *testing.T) {

	addr, closer := runserver(t, &smtpd.Server{
		ProtocolLogger: log.New(os.Stdout, "log: ", log.Lshortfile),
	})

	defer closer()

	c, err := smtp.Dial(addr)
	if err != nil {
		t.Fatalf("Dial failed: %v", err)
	}

	if err := c.Mail(fmt.Sprintf("%s@example.org", strings.Repeat("x", 65*1024))); err == nil {
		t.Fatalf("MAIL failed: %v", err)
	}

	if err := c.Quit(); err != nil {
		t.Fatalf("Quit failed: %v", err)
	}

}

func TestXCLIENT(t *testing.T) {

	addr, closer := runserver(t, &smtpd.Server{
		EnableXCLIENT: true,
		SenderChecker: func(peer smtpd.Peer, addr string) error {
			if peer.HeloName != "new.example.net" {
				t.Fatalf("Didn't override HELO name: %v", peer.HeloName)
			}
			if peer.Addr.String() != "42.42.42.42:4242" {
				t.Fatalf("Didn't override IP/Port: %v", peer.Addr)
			}
			if peer.Username != "newusername" {
				t.Fatalf("Didn't override username: %v", peer.Username)
			}
			if peer.Protocol != smtpd.SMTP {
				t.Fatalf("Didn't override protocol: %v", peer.Protocol)
			}
			return nil
		},
		ProtocolLogger: log.New(os.Stdout, "log: ", log.Lshortfile),
	})

	defer closer()

	c, err := smtp.Dial(addr)
	if err != nil {
		t.Fatalf("Dial failed: %v", err)
	}

	if supported, _ := c.Extension("XCLIENT"); !supported {
		t.Fatal("XCLIENT not supported")
	}

	err = cmd(c.Text, 220, "XCLIENT NAME=ignored ADDR=42.42.42.42 PORT=4242 PROTO=SMTP HELO=new.example.net LOGIN=newusername")
	if err != nil {
		t.Fatalf("XCLIENT failed: %v", err)
	}

	if err := c.Mail("sender@example.org"); err != nil {
		t.Fatalf("Mail failed: %v", err)
	}

	if err := c.Rcpt("recipient@example.net"); err != nil {
		t.Fatalf("Rcpt failed: %v", err)
	}

	if err := c.Rcpt("recipient2@example.net"); err != nil {
		t.Fatalf("Rcpt2 failed: %v", err)
	}

	wc, err := c.Data()
	if err != nil {
		t.Fatalf("Data failed: %v", err)
	}

	_, err = fmt.Fprintf(wc, "This is the email body")
	if err != nil {
		t.Fatalf("Data body failed: %v", err)
	}

	err = wc.Close()
	if err != nil {
		t.Fatalf("Data close failed: %v", err)
	}

	if err := c.Quit(); err != nil {
		t.Fatalf("Quit failed: %v", err)
	}

}

func TestEnvelopeReceived(t *testing.T) {

	addr, closer := runsslserver(t, &smtpd.Server{
		Hostname: "foobar.example.net",
		Handler: func(peer smtpd.Peer, env smtpd.Envelope) error {
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
	if err != nil {
		t.Fatalf("Dial failed: %v", err)
	}

	//nolint:gosec
	if err := c.StartTLS(&tls.Config{InsecureSkipVerify: true}); err != nil {
		t.Fatalf("STARTTLS failed: %v", err)
	}

	if err := c.Mail("sender@example.org"); err != nil {
		t.Fatalf("MAIL failed: %v", err)
	}

	if err := c.Rcpt("recipient@example.net"); err != nil {
		t.Fatalf("RCPT failed: %v", err)
	}

	wc, err := c.Data()
	if err != nil {
		t.Fatalf("Data failed: %v", err)
	}

	_, err = fmt.Fprintf(wc, "This is the email body")
	if err != nil {
		t.Fatalf("Data body failed: %v", err)
	}

	err = wc.Close()
	if err != nil {
		t.Fatalf("Data close failed: %v", err)
	}

	if err := c.Quit(); err != nil {
		t.Fatalf("QUIT failed: %v", err)
	}

}

func TestHELO(t *testing.T) {

	addr, closer := runserver(t, &smtpd.Server{
		ProtocolLogger: log.New(os.Stdout, "log: ", log.Lshortfile),
	})

	defer closer()

	c, err := smtp.Dial(addr)
	if err != nil {
		t.Fatalf("Dial failed: %v", err)
	}

	if err := cmd(c.Text, 502, "MAIL FROM:<test@example.org>"); err != nil {
		t.Fatalf("MAIL before HELO didn't fail: %v", err)
	}

	if err := cmd(c.Text, 250, "HELO localhost"); err != nil {
		t.Fatalf("HELO failed: %v", err)
	}

	if err := cmd(c.Text, 250, "MAIL FROM:<test@example.org>"); err != nil {
		t.Fatalf("MAIL after HELO failed: %v", err)
	}

	if err := cmd(c.Text, 250, "HELO localhost"); err != nil {
		t.Fatalf("double HELO failed: %v", err)
	}

	if err := c.Quit(); err != nil {
		t.Fatalf("Quit failed: %v", err)
	}

}

func TestLOGINAuth(t *testing.T) {

	addr, closer := runsslserver(t, &smtpd.Server{
		Authenticator:  func(peer smtpd.Peer, username, password string) error { return nil },
		ProtocolLogger: log.New(os.Stdout, "log: ", log.Lshortfile),
	})

	defer closer()

	c, err := smtp.Dial(addr)
	if err != nil {
		t.Fatalf("Dial failed: %v", err)
	}

	//nolint:gosec
	if err := c.StartTLS(&tls.Config{InsecureSkipVerify: true}); err != nil {
		t.Fatalf("STARTTLS failed: %v", err)
	}

	if err := cmd(c.Text, 334, "AUTH LOGIN"); err != nil {
		t.Fatalf("AUTH didn't work: %v", err)
	}

	if err := cmd(c.Text, 502, "foo"); err != nil {
		t.Fatalf("AUTH didn't fail: %v", err)
	}

	if err := cmd(c.Text, 334, "AUTH LOGIN"); err != nil {
		t.Fatalf("AUTH didn't work: %v", err)
	}

	if err := cmd(c.Text, 334, "Zm9v"); err != nil {
		t.Fatalf("AUTH didn't work: %v", err)
	}

	if err := cmd(c.Text, 502, "foo"); err != nil {
		t.Fatalf("AUTH didn't fail: %v", err)
	}

	if err := cmd(c.Text, 334, "AUTH LOGIN"); err != nil {
		t.Fatalf("AUTH didn't work: %v", err)
	}

	if err := cmd(c.Text, 334, "Zm9v"); err != nil {
		t.Fatalf("AUTH didn't work: %v", err)
	}

	if err := cmd(c.Text, 235, "Zm9v"); err != nil {
		t.Fatalf("AUTH didn't work: %v", err)
	}

	if err := c.Quit(); err != nil {
		t.Fatalf("Quit failed: %v", err)
	}

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

			err = cmd(c.Text, 250, "MAIL FROM:"+d.from)
			require.NoError(t, err)

			err = c.Quit()
			require.NoError(t, err)
		})
	}
}

func TestErrors(t *testing.T) {

	cert, err := tls.X509KeyPair(localhostCert, localhostKey)
	if err != nil {
		t.Fatalf("Cert load failed: %v", err)
	}

	server := &smtpd.Server{
		Authenticator:  func(peer smtpd.Peer, username, password string) error { return nil },
		ProtocolLogger: log.New(os.Stdout, "log: ", log.Lshortfile),
	}

	addr, closer := runserver(t, server)

	defer closer()

	c, err := smtp.Dial(addr)
	if err != nil {
		t.Fatalf("Dial failed: %v", err)
	}

	if err := cmd(c.Text, 502, "AUTH PLAIN foobar"); err != nil {
		t.Fatalf("AUTH didn't fail: %v", err)
	}

	if err := c.Hello("localhost"); err != nil {
		t.Fatalf("HELO failed: %v", err)
	}

	if err := cmd(c.Text, 502, "AUTH PLAIN foobar"); err != nil {
		t.Fatalf("AUTH didn't fail: %v", err)
	}

	if err := c.Mail("sender@example.org"); err == nil {
		t.Fatalf("MAIL didn't fail")
	}

	if err := cmd(c.Text, 502, "STARTTLS"); err != nil {
		t.Fatalf("STARTTLS didn't fail: %v", err)
	}

	server.TLSConfig = &tls.Config{
		MinVersion:   tls.VersionTLS12,
		Certificates: []tls.Certificate{cert},
	}

	//nolint:gosec
	if err := c.StartTLS(&tls.Config{InsecureSkipVerify: true}); err != nil {
		t.Fatalf("STARTTLS failed: %v", err)
	}

	if err := cmd(c.Text, 502, "AUTH UNKNOWN"); err != nil {
		t.Fatalf("AUTH didn't fail: %v", err)
	}

	if err := cmd(c.Text, 502, "AUTH PLAIN foobar"); err != nil {
		t.Fatalf("AUTH didn't fail: %v", err)
	}

	if err := cmd(c.Text, 502, "AUTH PLAIN Zm9vAGJhcg=="); err != nil {
		t.Fatalf("AUTH didn't fail: %v", err)
	}

	if err := cmd(c.Text, 334, "AUTH PLAIN"); err != nil {
		t.Fatalf("AUTH didn't work: %v", err)
	}

	if err := cmd(c.Text, 235, "Zm9vAGJhcgBxdXV4"); err != nil {
		t.Fatalf("AUTH didn't work: %v", err)
	}

	if err := c.Mail("sender@example.org"); err != nil {
		t.Fatalf("MAIL failed: %v", err)
	}

	if err := c.Mail("sender@example.org"); err == nil {
		t.Fatalf("Duplicate MAIL didn't fail")
	}

	if err := c.Quit(); err != nil {
		t.Fatalf("Quit failed: %v", err)
	}

}

func TestMailformedMAILFROM(t *testing.T) {

	addr, closer := runserver(t, &smtpd.Server{
		SenderChecker: func(peer smtpd.Peer, addr string) error {
			if addr != "test@example.org" {
				return smtpd.Error{Code: 502, Message: "Denied"}
			}
			return nil
		},
		ProtocolLogger: log.New(os.Stdout, "log: ", log.Lshortfile),
	})

	defer closer()

	c, err := smtp.Dial(addr)
	if err != nil {
		t.Fatalf("Dial failed: %v", err)
	}

	if err := c.Hello("localhost"); err != nil {
		t.Fatalf("HELO failed: %v", err)
	}

	if err := cmd(c.Text, 250, "MAIL FROM: <test@example.org>"); err != nil {
		t.Fatalf("MAIL FROM failed with extra whitespace: %v", err)
	}

	if err := c.Quit(); err != nil {
		t.Fatalf("Quit failed: %v", err)
	}
}

func TestTLSListener(t *testing.T) {

	cert, err := tls.X509KeyPair(localhostCert, localhostKey)
	if err != nil {
		t.Fatalf("Cert load failed: %v", err)
	}

	cfg := &tls.Config{
		MinVersion:   tls.VersionTLS12,
		Certificates: []tls.Certificate{cert},
	}

	ln, err := tls.Listen("tcp", "127.0.0.1:0", cfg)
	defer ln.Close()

	addr := ln.Addr().String()

	server := &smtpd.Server{
		Authenticator: func(peer smtpd.Peer, username, password string) error {
			if peer.TLS == nil {
				t.Error("didn't correctly set connection state on TLS connection")
			}
			return nil
		},
		ProtocolLogger: log.New(os.Stdout, "log: ", log.Lshortfile),
	}

	go func() {
		_ = server.Serve(ln)
	}()

	//nolint:gosec
	conn, err := tls.Dial("tcp", addr, &tls.Config{InsecureSkipVerify: true})
	if err != nil {
		t.Fatalf("couldn't connect to tls socket: %v", err)
	}

	c, err := smtp.NewClient(conn, "localhost")
	if err != nil {
		t.Fatalf("couldn't create client: %v", err)
	}

	if err := c.Hello("localhost"); err != nil {
		t.Fatalf("HELO failed: %v", err)
	}

	if err := cmd(c.Text, 334, "AUTH PLAIN"); err != nil {
		t.Fatalf("AUTH didn't work: %v", err)
	}

	if err := cmd(c.Text, 235, "Zm9vAGJhcgBxdXV4"); err != nil {
		t.Fatalf("AUTH didn't work: %v", err)
	}

	if err := c.Quit(); err != nil {
		t.Fatalf("Quit failed: %v", err)
	}

}

func TestShutdown(t *testing.T) {
	fmt.Println("Starting test")
	server := &smtpd.Server{
		ProtocolLogger: log.New(os.Stdout, "log: ", log.Lshortfile),
	}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen failed: %v", err)
	}

	srvres := make(chan error)
	go func() {
		t.Log("Starting server")
		srvres <- server.Serve(ln)
	}()

	// Connect a client
	c, err := smtp.Dial(ln.Addr().String())
	if err != nil {
		t.Fatalf("Dial failed: %v", err)
	}

	if err := c.Hello("localhost"); err != nil {
		t.Fatalf("HELO failed: %v", err)
	}

	// While the client connection is open, shut down the server (without
	// waiting for it to finish)
	err = server.Shutdown(false)
	if err != nil {
		t.Fatalf("Shutdown returned error: %v", err)
	}

	// Verify that Shutdown() worked by attempting to connect another client
	_, err = smtp.Dial(ln.Addr().String())
	if err == nil {
		t.Fatalf("Dial did not fail as expected")
	}

	var operr *net.OpError
	if !errors.As(err, &operr) {
		t.Fatalf("Dial did not return net.OpError as expected: %v (%T)", err, err)
	}

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
	if err := c.Quit(); err != nil {
		t.Fatalf("QUIT failed: %v", err)
	}
	c.Close()

	// Wait for Wait() to return
	t.Log("Waiting for Wait() to return")
	select {
	case shuterr := <-shutres:
		if shuterr != nil {
			t.Fatalf("Wait() returned error: %v", shuterr)
		}
	case <-time.After(15 * time.Second):
		t.Fatalf("Timed out waiting for Wait() to return")
	}

	// Wait for Serve() to return
	t.Log("Waiting for Serve() to return")
	select {
	case srverr := <-srvres:
		if !errors.Is(srverr, smtpd.ErrServerClosed) {
			t.Fatalf("Serve() returned error: %v", srverr)
		}
	case <-time.After(15 * time.Second):
		t.Fatalf("Timed out waiting for Serve() to return")
	}
}

func TestServeFailsIfShutdown(t *testing.T) {
	server := &smtpd.Server{}
	err := server.Shutdown(true)
	if err != nil {
		t.Fatalf("Shutdown() failed: %v", err)
	}
	err = server.Serve(nil)
	if !errors.Is(err, smtpd.ErrServerClosed) {
		t.Fatalf("Serve() did not return ErrServerClosed: %v", err)
	}
}

func TestWaitFailsIfNotShutdown(t *testing.T) {
	server := &smtpd.Server{}
	err := server.Wait()
	if err == nil {
		t.Fatalf("Wait() did not fail as expected")
	}
}
