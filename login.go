package main

import (
	"errors"
	"net/smtp"
)

type loginAuth struct {
	username string
	password string
	step     int
}

func (a *loginAuth) Start(_ *smtp.ServerInfo) (string, []byte, error) {
	return "LOGIN", nil, nil
}

func (a *loginAuth) Next(_ []byte, more bool) ([]byte, error) {
	if !more {
		return nil, nil
	}

	// LOGIN auth works in two steps:
	// 1. Server challenges with "Username:"
	// 2. Server challenges with "Password:"

	a.step++
	switch a.step {
	case 1:
		// First challenge: send username
		return []byte(a.username), nil
	case 2:
		// Second challenge: send password
		return []byte(a.password), nil
	default:
		return nil, errors.New("unexpected challenge from server")
	}
}
