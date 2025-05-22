package main

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/textproto"
	"strings"
	"testing"

	"github.com/grafana/smtprelay/v2/internal/smtpd"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_RecepientsCheck(t *testing.T) {
	registry := prometheus.NewRegistry()
	err := registerMetrics(registry)
	require.NoError(t, err)

	tc := []struct {
		name     string
		allowed  string
		denied   string
		expected error
		emails   []string
	}{
		{
			name:   "without any list, all emails are allowed",
			emails: []string{"delivery@example.com", "example@email.com", "<example@email.com>"},
		},
		{
			name:     "with emails not in the allow list",
			emails:   []string{"delivery@grafana.com"},
			allowed:  "(.+@example.(org|com)|.+@email.com)",
			expected: smtpd.ErrRecipientInvalid,
		},
		{
			name:     "with emails that are denied",
			emails:   []string{"delivery@example.com", "example@email.com", "<example@email.com>"},
			denied:   "(.+@example.(org|com)|.+@email.com)",
			expected: smtpd.ErrRecipientDenied,
		},
		{
			name:   "with valid email that are not denied",
			emails: []string{"someone@anemail.com", "someone@anemail.com"},
			denied: "(.+@example.(org|com)|.+@email.com)",
		},
		{
			name:    "with valid email that are allowed",
			emails:  []string{"someone@example.com", "someone@email.com"},
			allowed: "(.+@example.(org|com)|.+@email.com)",
		},
		{
			name:    "with valid email that complies with both the allowed and denied list",
			emails:  []string{"josue@grafana.com", "goutham@grafana.com"},
			denied:  "(.+@example.(org|com)|.+@email.com)",
			allowed: ".+@grafana.com",
		},
		{
			name:     "with an email that is not in any of the lists",
			emails:   []string{"random@deliver.org"},
			denied:   "(.+@example.(org|com)|.+@email.com)",
			allowed:  ".+@grafana.com",
			expected: smtpd.ErrRecipientInvalid,
		},
	}

	ctx := context.Background()
	r := &relay{}
	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			checker := r.recipientChecker(tt.allowed, tt.denied)

			for _, e := range tt.emails {
				if err := checker(ctx, smtpd.Peer{}, e); err != nil {
					if !errors.Is(err, tt.expected) {
						t.Errorf("got %d, want %d for the email %s", err, tt.expected, e)
					}
				}
			}
		})
	}
}

func TestAddLogHeaderFields(t *testing.T) {
	out := &bytes.Buffer{}
	logger := slog.New(slog.NewTextHandler(out, &slog.HandlerOptions{
		// remove time, level, and msg for simpler testing
		ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
			switch a.Key {
			case "time", "level", "msg":
				return slog.Attr{}
			}

			return a
		},
	}))

	data := []byte(`Subject: test
Message-ID: 9a7f8b9c-6d1d-4b9a-8c0a-9e4b9c6d1d4b
DKIM-Signature: v=1; a=rsa-sha256; d=example.com; s=dkim; c=relaxed/relaxed;
 q=dns/txt;
Date: Thu, 01 Jan 1970 00:00:00 +0000
From: Alice <alice@example.com>
To: Bob <bob@example.com>

This is a test message.
`)

	treader := textproto.NewReader(bufio.NewReader(bytes.NewBuffer(data)))
	hdr, err := treader.ReadMIMEHeader()
	require.NoError(t, err)

	t.Run("no logHeaders", func(t *testing.T) {
		log := addLogHeaderFields(nil, logger, textproto.MIMEHeader{})

		out.Reset()

		log.Info("")
		assert.Empty(t, strings.TrimSpace(out.String()))
	})

	t.Run("with logHeaders", func(t *testing.T) {
		hdrs := map[string]string{"header1": "field1", "header2": "field2"}
		log := addLogHeaderFields(hdrs, logger, textproto.MIMEHeader{})

		out.Reset()

		log.Info("")
		assert.Empty(t, strings.TrimSpace(out.String()))
	})

	t.Run("with simple data, logHeaders not present", func(t *testing.T) {
		hdrs := map[string]string{"header1": "field1", "header2": "field2"}
		log := addLogHeaderFields(hdrs, logger, hdr)

		out.Reset()

		log.Info("")
		assert.Empty(t, strings.TrimSpace(out.String()))
	})

	t.Run("with simple data, logHeaders found", func(t *testing.T) {
		hdrs := map[string]string{"subject": "Subject", "msgid": "Message-ID"}
		log := addLogHeaderFields(hdrs, logger, hdr)

		out.Reset()

		log.Info("")

		// we can't do a straight string compare because the order of the fields
		// will change
		pairs := strings.Split(
			strings.TrimSpace(out.String()),
			" ",
		)
		assert.Len(t, pairs, 2)
		assert.Contains(t, pairs, "subject=test")
		assert.Contains(t, pairs, "msgid=9a7f8b9c-6d1d-4b9a-8c0a-9e4b9c6d1d4b")
	})
}

func TestAddrAllowedNoDomain(t *testing.T) {
	allowedAddrs := []string{"joe@abc.com"}
	if addrAllowed("bob.com", allowedAddrs) {
		t.FailNow()
	}
}

func TestAddrAllowedSingle(t *testing.T) {
	allowedAddrs := []string{"joe@abc.com"}

	if !addrAllowed("joe@abc.com", allowedAddrs) {
		t.FailNow()
	}
	if addrAllowed("bob@abc.com", allowedAddrs) {
		t.FailNow()
	}
}

func TestAddrAllowedDifferentCase(t *testing.T) {
	allowedAddrs := []string{"joe@abc.com"}
	testAddrs := []string{
		"joe@ABC.com",
		"Joe@abc.com",
		"JOE@abc.com",
		"JOE@ABC.COM",
	}
	for _, addr := range testAddrs {
		if !addrAllowed(addr, allowedAddrs) {
			t.Errorf("Address %v not allowed, but should be", addr)
		}
	}
}

func TestAddrAllowedLocal(t *testing.T) {
	allowedAddrs := []string{"joe"}

	if !addrAllowed("joe", allowedAddrs) {
		t.FailNow()
	}
	if addrAllowed("bob", allowedAddrs) {
		t.FailNow()
	}
}

func TestAddrAllowedMulti(t *testing.T) {
	allowedAddrs := []string{"joe@abc.com", "bob@def.com"}
	if !addrAllowed("joe@abc.com", allowedAddrs) {
		t.FailNow()
	}
	if !addrAllowed("bob@def.com", allowedAddrs) {
		t.FailNow()
	}
	if addrAllowed("bob@abc.com", allowedAddrs) {
		t.FailNow()
	}
}

func TestAddrAllowedSingleDomain(t *testing.T) {
	allowedAddrs := []string{"@abc.com"}
	if !addrAllowed("joe@abc.com", allowedAddrs) {
		t.FailNow()
	}
	if addrAllowed("joe@def.com", allowedAddrs) {
		t.FailNow()
	}
}

func TestAddrAllowedMixed(t *testing.T) {
	allowedAddrs := []string{"app", "app@example.com", "@appsrv.example.com"}
	if !addrAllowed("app", allowedAddrs) {
		t.FailNow()
	}
	if !addrAllowed("app@example.com", allowedAddrs) {
		t.FailNow()
	}
	if addrAllowed("ceo@example.com", allowedAddrs) {
		t.FailNow()
	}
	if !addrAllowed("root@appsrv.example.com", allowedAddrs) {
		t.FailNow()
	}
	if !addrAllowed("dev@appsrv.example.com", allowedAddrs) {
		t.FailNow()
	}
	if addrAllowed("appsrv@example.com", allowedAddrs) {
		t.FailNow()
	}
}
