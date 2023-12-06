package main

import (
	"bytes"
	"io"
	"testing"

	"github.com/chrj/smtpd"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	logger := logrus.New()
	logger.SetOutput(io.Discard)
	log = logrus.NewEntry(logger)
	registerMetrics()
}

func Test_RecepientsCheck(t *testing.T) {
	tc := []struct {
		name     string
		emails   []string
		allowed  string
		denied   string
		expected error
	}{
		{
			name:   "without any list, all emails are allowed",
			emails: []string{"delivery@example.com", "example@email.com", "<example@email.com>"},
		},
		{
			name:     "with emails not in the allow list",
			emails:   []string{"delivery@grafana.com"},
			allowed:  "(.+@example.(org|com)|.+@email.com)",
			expected: observeErr(smtpd.Error{Code: 451, Message: "Invalid recipient address"}),
		},
		{
			name:     "with emails that are denied",
			emails:   []string{"delivery@example.com", "example@email.com", "<example@email.com>"},
			denied:   "(.+@example.(org|com)|.+@email.com)",
			expected: observeErr(smtpd.Error{Code: 451, Message: "Denied recipient address"}),
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
			expected: observeErr(smtpd.Error{Code: 451, Message: "Invalid recipient address"}),
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			checker := recipientChecker(tt.allowed, tt.denied)

			for _, e := range tt.emails {
				if err := checker(smtpd.Peer{}, e); err != nil {
					if err != tt.expected {
						t.Errorf("got %d, want %d for the email %s", err, tt.expected, e)
					}
				}
			}
		})
	}
}

func TestAddLogHeaderFields(t *testing.T) {
	out := &bytes.Buffer{}
	logger := logrus.New()
	logger.SetOutput(out)
	logger.SetFormatter(&logrus.TextFormatter{
		DisableTimestamp: true,
		DisableSorting:   false,
	})
	logger.SetLevel(logrus.InfoLevel)

	data := []byte(`Subject: test
Message-ID: 9a7f8b9c-6d1d-4b9a-8c0a-9e4b9c6d1d4b
DKIM-Signature: v=1; a=rsa-sha256; d=example.com; s=dkim; c=relaxed/relaxed;
 q=dns/txt;
Date: Thu, 01 Jan 1970 00:00:00 +0000
From: Alice <alice@example.com>
To: Bob <bob@example.com>

This is a test message.
`)

	t.Run("no logHeaders", func(t *testing.T) {
		log, err := addLogHeaderFields(nil, logrus.NewEntry(logger), nil)
		require.NoError(t, err)

		s, err := log.String()
		require.NoError(t, err)
		assert.Equal(t, "level=panic\n", s)
	})

	t.Run("with logHeaders", func(t *testing.T) {
		logHeaders := map[string]string{"header1": "field1", "header2": "field2"}
		log, err := addLogHeaderFields(logHeaders, logrus.NewEntry(logger), nil)
		require.NoError(t, err)

		s, err := log.String()
		require.NoError(t, err)
		assert.Equal(t, "level=panic\n", s)
	})

	t.Run("with simple data, logHeaders not found", func(t *testing.T) {
		log, err := addLogHeaderFields(logHeaders, logrus.NewEntry(logger), data)
		require.NoError(t, err)

		s, err := log.String()
		require.NoError(t, err)
		assert.Equal(t, "level=panic\n", s)
	})

	t.Run("with simple data, logHeaders found", func(t *testing.T) {
		logHeaders = map[string]string{"subject": "Subject", "msgid": "Message-ID"}
		log, err := addLogHeaderFields(logHeaders, logrus.NewEntry(logger), data)
		require.NoError(t, err)

		s, err := log.String()
		require.NoError(t, err)
		assert.Equal(t, "level=panic msgid=9a7f8b9c-6d1d-4b9a-8c0a-9e4b9c6d1d4b subject=test\n", s)
	})
}
