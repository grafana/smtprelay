package main

import (
	"errors"
	"fmt"
	"net/smtp"
)

type xoauth2Auth struct {
	user  string
	token string
}

func (a *xoauth2Auth) Start(_ *smtp.ServerInfo) (string, []byte, error) {
	// XOAUTH2 expects: user={email}\001auth=Bearer {token}\001\001
	resp := []byte(fmt.Sprintf("user=%s\x01auth=Bearer %s\x01\x01", a.user, a.token))
	return "XOAUTH2", resp, nil
}

func (a *xoauth2Auth) Next(_ []byte, more bool) ([]byte, error) {
	if more {
		return nil, errors.New("unexpected challenge from server")
	}
	return nil, nil
}
