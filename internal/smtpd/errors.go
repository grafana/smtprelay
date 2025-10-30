package smtpd

import "net/textproto"

var (
	ErrBusy              = &textproto.Error{Code: 421, Msg: "Too busy. Try again later."}
	ErrIPDenied          = &textproto.Error{Code: 421, Msg: "Denied - IP out of allowed network range"}
	ErrRateLimitExceeded = &textproto.Error{Code: 421, Msg: "Rate limit exceeded. Try again later."}
	ErrRecipientDenied   = &textproto.Error{Code: 451, Msg: "Denied recipient address"}
	ErrRecipientInvalid  = &textproto.Error{Code: 451, Msg: "Invalid recipient address"}
	ErrSenderDenied      = &textproto.Error{Code: 451, Msg: "sender address not allowed"}
	ErrTooManyRecipients = &textproto.Error{Code: 452, Msg: "Too many recipients"}

	ErrLineTooLong           = &textproto.Error{Code: 500, Msg: "Line too long"}
	ErrDuplicateMAIL         = &textproto.Error{Code: 502, Msg: "Duplicate MAIL"}
	ErrDuplicateSTARTTLS     = &textproto.Error{Code: 502, Msg: "Already running in TLS"}
	ErrInvalidSyntax         = &textproto.Error{Code: 502, Msg: "Invalid syntax."}
	ErrMalformedAuth         = &textproto.Error{Code: 502, Msg: "Couldn't decode your credentials"}
	ErrMalformedCommand      = &textproto.Error{Code: 502, Msg: "Couldn't decode the command"}
	ErrMalformedEmail        = &textproto.Error{Code: 502, Msg: "Malformed email address"} // TODO: should this be a 502 or 451?
	ErrMissingParam          = &textproto.Error{Code: 502, Msg: "Missing parameter"}
	ErrNoHELO                = &textproto.Error{Code: 502, Msg: "Please introduce yourself first."}
	ErrNoMAIL                = &textproto.Error{Code: 502, Msg: "Missing MAIL FROM command."}
	ErrNoRCPT                = &textproto.Error{Code: 502, Msg: "Missing RCPT TO command."}
	ErrNoSTARTTLS            = &textproto.Error{Code: 502, Msg: "Please turn on TLS by issuing a STARTTLS command."}
	ErrTLSNotSupported       = &textproto.Error{Code: 502, Msg: "TLS not supported"}
	ErrUnknownAuth           = &textproto.Error{Code: 502, Msg: "Unknown authentication mechanism"}
	ErrUnsupportedCommand    = &textproto.Error{Code: 502, Msg: "Unsupported command"}
	ErrUnsupportedConn       = &textproto.Error{Code: 502, Msg: "Unsupported network connection"}
	ErrUnsupportedAuthMethod = &textproto.Error{Code: 530, Msg: "Authentication method not supported"}
	ErrAuthRequired          = &textproto.Error{Code: 530, Msg: "Authentication required."}
	ErrAuthInvalid           = &textproto.Error{Code: 535, Msg: "Authentication credentials invalid"}
	ErrBadHandshake          = &textproto.Error{Code: 550, Msg: "Handshake error"}
	ErrTooBig                = &textproto.Error{Code: 552, Msg: "Message exceeded maximum size"}
	ErrForwardingFailed      = &textproto.Error{Code: 554, Msg: "Forwarding failed"}
)
