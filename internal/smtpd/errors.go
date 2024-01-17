package smtpd

import "net/textproto"

var (
	ErrIPDenied              = &textproto.Error{Code: 421, Msg: "Denied - IP out of allowed network range"}
	ErrRecipientDenied       = &textproto.Error{Code: 451, Msg: "Denied recipient address"}
	ErrRecipientInvalid      = &textproto.Error{Code: 451, Msg: "Invalid recipient address"}
	ErrSenderDenied          = &textproto.Error{Code: 451, Msg: "sender address not allowed"}
	ErrUnsupportedAuthMethod = &textproto.Error{Code: 530, Msg: "Authentication method not supported"}
	ErrAuthInvalid           = &textproto.Error{Code: 535, Msg: "Authentication credentials invalid"}
	ErrForwardingFailed      = &textproto.Error{Code: 554, Msg: "Forwarding failed"}

	//TODO: make sure these each make sense - there's overlap
	Err502Denied   = &textproto.Error{Code: 502, Msg: "Denied"}
	Err550Denied   = &textproto.Error{Code: 550, Msg: "Denied"}
	Err550Rejected = &textproto.Error{Code: 550, Msg: "Rejected"}
	Err552Denied   = &textproto.Error{Code: 552, Msg: "Denied"}
)
