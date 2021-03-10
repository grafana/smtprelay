package main

import (
	"io/ioutil"
	"testing"

	"github.com/chrj/smtpd"
	"github.com/sirupsen/logrus"
)

func init() {
	logger := logrus.New()
	logger.SetOutput(ioutil.Discard)
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
			name:   "with emails that are denied",
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
