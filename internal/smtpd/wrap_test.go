package smtpd

import (
	"strings"
	"testing"
)

func FuzzWrap(f *testing.F) {
	f.Add([]byte("short line"))
	f.Add([]byte(strings.Repeat("a ", 100)))
	f.Add([]byte("no-spaces-at-all" + strings.Repeat("x", 200)))
	f.Add([]byte("line1\r\nline2\r\nline3"))
	f.Add([]byte{})
	f.Add([]byte(strings.Repeat("word ", 50) + "\r\n" + strings.Repeat("more ", 50)))

	f.Fuzz(func(_ *testing.T, input []byte) {
		wrap(input)
	})
}

func TestWrap(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"foobar":         "foobar",
		"foobar quux":    "foobar quux",
		"foobar\r\n":     "foobar\r\n",
		"foobar\r\nquux": "foobar\r\nquux",
		"foobar quux foobar quux foobar quux foobar quux foobar quux foobar quux foobar quux foobar quux":      "foobar quux foobar quux foobar quux foobar quux foobar quux foobar quux foobar\r\n\tquux foobar quux",
		"foobar quux foobar quux foobar quux foobar quux foobar quux foobar\r\n\tquux foobar quux foobar quux": "foobar quux foobar quux foobar quux foobar quux foobar quux foobar\r\n\tquux foobar quux foobar quux",
	}

	for k, v := range cases {
		if string(wrap([]byte(k))) != v {
			t.Fatal("Didn't wrap correctly.")
		}
	}

}
