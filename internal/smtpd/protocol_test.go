package smtpd

import "testing"

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
