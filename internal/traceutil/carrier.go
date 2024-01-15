package traceutil

import (
	"net/textproto"
)

// MIMEHeaderCarrier adapts textproto.MIMEHeader to satisfy the TextMapCarrier interface.
type MIMEHeaderCarrier textproto.MIMEHeader

// Get returns the value associated with the passed key.
func (hc MIMEHeaderCarrier) Get(key string) string {
	return textproto.MIMEHeader(hc).Get(key)
}

// Set stores the key-value pair.
func (hc MIMEHeaderCarrier) Set(key string, value string) {
	textproto.MIMEHeader(hc).Set(key, value)
}

// Keys lists the keys stored in this carrier.
func (hc MIMEHeaderCarrier) Keys() []string {
	keys := make([]string, 0, len(hc))
	for k := range hc {
		keys = append(keys, k)
	}
	return keys
}
