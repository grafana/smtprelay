package main

import (
	"fmt"
	"net/smtp"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_xoauth2Auth_Start(t *testing.T) {
	t.Parallel()

	a := &xoauth2Auth{
		user:  "test@example.com",
		token: "abcdef",
	}

	gotAuth, gotPayload, gotErr := a.Start(&smtp.ServerInfo{})

	// XOAUTH2 expects: user={email}\001auth=Bearer {token}\001\001
	wantPayload := []byte("user=test@example.com\001auth=Bearer abcdef\001\001")

	assert.Equalf(t, "XOAUTH2", gotAuth, "The auth method should be XOAUTH2")
	assert.Equalf(t, wantPayload, gotPayload, "The payload does not match")
	assert.NoError(t, gotErr)
}

func Test_xoauth2Auth_Next(t *testing.T) {
	t.Parallel()

	user := "test@example.com"
	token := "abcdef"

	//nolint:govet
	tests := []struct {
		name    string
		more    bool
		want    []byte
		wantErr assert.ErrorAssertionFunc
	}{
		{"nothing expected", false, nil, assert.NoError},
		{"response expected", true, nil, assert.Error},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			a := &xoauth2Auth{
				user:  user,
				token: token,
			}
			got, err := a.Next(nil, tt.more)
			if !tt.wantErr(t, err, fmt.Sprintf("next(%v, %v)", nil, tt.more)) {
				return
			}
			assert.Equalf(t, tt.want, got, "next(%v, %v)", []byte{}, tt.more)
		})
	}
}
