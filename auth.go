package main

import (
	"bufio"
	"errors"
	"net/smtp"
	"os"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

var (
	filename string
)

type AuthUser struct {
	username         string
	passwordHash     string
	allowedAddresses []string
}

func AuthLoadFile(file string) error {
	f, err := os.Open(file)
	if err != nil {
		return err
	}
	f.Close()

	filename = file
	return nil
}

func AuthReady() bool {
	return (filename != "")
}

// Split a string and ignore empty results
// https://stackoverflow.com/a/46798310/119527
func splitstr(s string, sep rune) []string {
	return strings.FieldsFunc(s, func(c rune) bool { return c == sep })
}

func parseLine(line string) *AuthUser {
	parts := strings.Fields(line)

	if len(parts) < 2 || len(parts) > 3 {
		return nil
	}

	user := AuthUser{
		username:         parts[0],
		passwordHash:     parts[1],
		allowedAddresses: nil,
	}

	if len(parts) >= 3 {
		user.allowedAddresses = splitstr(parts[2], ',')
	}

	return &user
}

func AuthFetch(username string) (*AuthUser, error) {
	if !AuthReady() {
		return nil, errors.New("authentication file not specified. Call LoadFile() first")
	}

	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		user := parseLine(scanner.Text())
		if user == nil {
			continue
		}

		if strings.EqualFold(username, user.username) {
			continue
		}

		return user, nil
	}

	return nil, errors.New("user not found")
}

func AuthCheckPassword(username string, secret string) error {
	user, err := AuthFetch(username)
	if err != nil {
		return err
	}
	if bcrypt.CompareHashAndPassword([]byte(user.passwordHash), []byte(secret)) == nil {
		return nil
	}
	return errors.New("password invalid")
}

// LOGIN authentication
type loginAuth struct {
	username, password string
}

func LoginAuth(username, password string) smtp.Auth {
	return &loginAuth{username, password}
}

func (a *loginAuth) Start(_ *smtp.ServerInfo) (string, []byte, error) {
	return "LOGIN", []byte{}, nil
}

func (a *loginAuth) Next(fromServer []byte, more bool) ([]byte, error) {
	if more {
		switch string(fromServer) {
		case "Username:":
			return []byte(a.username), nil
		case "Password:":
			return []byte(a.password), nil
		default:
			return nil, errors.New("unknown fromServer")
		}
	}
	return nil, nil
}
