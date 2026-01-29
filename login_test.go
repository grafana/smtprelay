package main

import (
	"net/smtp"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_loginAuth_Start(t *testing.T) {
	t.Parallel()

	a := &loginAuth{
		username: "testuser",
		password: "testpass",
	}

	gotAuth, gotPayload, gotErr := a.Start(&smtp.ServerInfo{})

	assert.Equal(t, "LOGIN", gotAuth, "The auth method should be LOGIN")
	assert.Nil(t, gotPayload, "The initial payload should be nil")
	assert.NoError(t, gotErr)
}

func Test_loginAuth_Next(t *testing.T) {
	t.Parallel()

	username := "testuser"
	password := "testpass"

	t.Run("successful auth flow", func(t *testing.T) {
		a := &loginAuth{
			username: username,
			password: password,
		}

		// First challenge from server (Username:)
		got1, err1 := a.Next([]byte("VXNlcm5hbWU6"), true)
		assert.NoError(t, err1)
		assert.Equal(t, []byte(username), got1, "First challenge should return username")

		// Second challenge from server (Password:)
		got2, err2 := a.Next([]byte("UGFzc3dvcmQ6"), true)
		assert.NoError(t, err2)
		assert.Equal(t, []byte(password), got2, "Second challenge should return password")

		// Authentication complete
		got3, err3 := a.Next(nil, false)
		assert.NoError(t, err3)
		assert.Nil(t, got3, "When more=false, should return nil")
	})

	t.Run("unexpected third challenge", func(t *testing.T) {
		a := &loginAuth{
			username: username,
			password: password,
		}

		// Go through normal flow
		_, _ = a.Next([]byte("VXNlcm5hbWU6"), true) // username
		_, _ = a.Next([]byte("UGFzc3dvcmQ6"), true) // password

		// Unexpected third challenge
		got, err := a.Next([]byte("unexpected"), true)
		assert.Error(t, err)
		assert.EqualError(t, err, "unexpected challenge from server")
		assert.Nil(t, got)
	})
}
