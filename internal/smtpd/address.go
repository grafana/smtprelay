package smtpd

import (
	"fmt"
	"net/mail"
)

func parseAddress(src string) (string, error) {
	// While a RFC5321 mailbox specification is not the same as an RFC5322
	// email address specification, it is better to accept that format and
	// parse it down to the actual address, as there are a lot of badly
	// behaving MTAs and MUAs that do it wrongly. It therefore makes sense
	// to rely on Go's built-in address parser. This does have the benefit
	// of allowing "email@example.com" as input as thats commonly used,
	// though not RFC compliant.
	addr, err := mail.ParseAddress(src)
	if err != nil {
		return "", fmt.Errorf("malformed e-mail address: %s", src)
	}

	return addr.Address, nil
}
