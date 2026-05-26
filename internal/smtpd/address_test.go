package smtpd

import (
	"strings"
	"testing"
)

func FuzzParseAddress(f *testing.F) {
	f.Add("<test@example.org>")
	f.Add("test@example.org")
	f.Add("<>")
	f.Add("\"quoted name\" <addr@example.com>")
	f.Add("")
	f.Add("<a@b@c>")
	f.Add("<" + strings.Repeat("a", 1000) + "@x.com>")

	f.Fuzz(func(t *testing.T, src string) {
		addr, err := parseAddress(src)
		if err != nil {
			return
		}
		if !strings.Contains(addr, "@") {
			t.Errorf("parseAddress(%q) = %q, expected '@' in result", src, addr)
		}
	})
}
