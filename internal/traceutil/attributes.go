package traceutil

import (
	"go.opentelemetry.io/otel/attribute"
)

const (
	senderKey     = attribute.Key("smtp.sender")
	recipientsKey = attribute.Key("smtp.recipients")
	datasizeKey   = attribute.Key("smtp.data.size")
	statusCodeKey = attribute.Key("smtp.response.status_code")
)

// The sender address (from the 'MAIL FROM' SMTP command).
//
// Type: string
// Required: Yes
// Examples: "bob@example.com", "<alice@example.com>"
func Sender(name string) attribute.KeyValue {
	return senderKey.String(name)
}

// The recipient addresses (from the 'RCPT TO' SMTP command).
//
// Type: []string
// Required: Yes
// Examples: ["alice@example", "<bob@example>"]
func Recipients(names []string) attribute.KeyValue {
	return recipientsKey.StringSlice(names)
}

// The size of the message data (from the 'DATA' SMTP command).
//
// Type: int64
// Required: Yes
// Examples: 1024
func DataSize(size int64) attribute.KeyValue {
	return datasizeKey.Int64(size)
}

// The SMTP response status code.
//
// Type: int
// Required: Yes
// Examples: 250
func StatusCode(code int) attribute.KeyValue {
	return statusCodeKey.Int(code)
}
