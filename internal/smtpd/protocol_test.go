package smtpd

import "testing"

func FuzzParseLine(f *testing.F) {
	f.Add("MAIL FROM:<test@example.org>")
	f.Add("MAIL FROM: <test@example.org>")
	f.Add("RCPT TO:<recipient@example.com>")
	f.Add("EHLO localhost")
	f.Add("AUTH PLAIN dGVzdAB0ZXN0AHBhc3M=")
	f.Add("")
	f.Add("XCLIENT NAME=foo ADDR=1.2.3.4 PORT=1234")
	f.Add("PROXY TCP4 1.2.3.4 5.6.7.8 1234 25")

	f.Fuzz(func(_ *testing.T, line string) {
		parseLine(line)
	})
}

func TestParseLine(t *testing.T) {
	t.Parallel()

	cmd := parseLine("HELO hostname")
	if cmd.action != "HELO" {
		t.Fatalf("unexpected action: %s", cmd.action)
	}

	if len(cmd.fields) != 2 {
		t.Fatalf("unexpected fields length: %d", len(cmd.fields))
	}

	if len(cmd.params) != 1 {
		t.Fatalf("unexpected params length: %d", len(cmd.params))
	}

	if cmd.params[0] != "hostname" {
		t.Fatalf("unexpected value for param 0: %v", cmd.params[0])
	}

	cmd = parseLine("DATA")
	if cmd.action != "DATA" {
		t.Fatalf("unexpected action: %s", cmd.action)
	}

	if len(cmd.fields) != 1 {
		t.Fatalf("unexpected fields length: %d", len(cmd.fields))
	}

	if cmd.params != nil {
		t.Fatalf("unexpected params: %v", cmd.params)
	}

	cmd = parseLine("MAIL FROM:<test@example.org>")
	if cmd.action != "MAIL" {
		t.Fatalf("unexpected action: %s", cmd.action)
	}

	if len(cmd.fields) != 2 {
		t.Fatalf("unexpected fields length: %d", len(cmd.fields))
	}

	if len(cmd.params) != 2 {
		t.Fatalf("unexpected params length: %d", len(cmd.params))
	}

	if cmd.params[0] != "FROM" {
		t.Fatalf("unexpected value for param 0: %v", cmd.params[0])
	}

	if cmd.params[1] != "<test@example.org>" {
		t.Fatalf("unexpected value for param 1: %v", cmd.params[1])
	}

}

func TestParseLineMailformedMAILFROM(t *testing.T) {
	t.Parallel()

	cmd := parseLine("MAIL FROM: <test@example.org>")
	if cmd.action != "MAIL" {
		t.Fatalf("unexpected action: %s", cmd.action)
	}

	if len(cmd.fields) != 2 {
		t.Fatalf("unexpected fields length: %d", len(cmd.fields))
	}

	if len(cmd.params) != 2 {
		t.Fatalf("unexpected params length: %d", len(cmd.params))
	}

	if cmd.params[0] != "FROM" {
		t.Fatalf("unexpected value for param 0: %v", cmd.params[0])
	}

	if cmd.params[1] != "<test@example.org>" {
		t.Fatalf("unexpected value for param 1: %v", cmd.params[1])
	}
}
