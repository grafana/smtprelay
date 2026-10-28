package main

import (
	"bytes"
	"context"
	"net"
	"os"
	"os/exec"
	"syscall"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const mainHelperEnv = "SMTPRELAY_TEST_MAIN"

// TestMainHelper runs main() when re-executed as a subprocess by the exit code
// tests. Arguments for main are passed after "--".
//
//nolint:paralleltest
func TestMainHelper(t *testing.T) {
	if os.Getenv(mainHelperEnv) != "1" {
		t.Skip("subprocess helper")
	}

	args := os.Args
	for i, a := range args {
		if a == "--" {
			args = args[i+1:]
			break
		}
	}

	os.Args = append([]string{os.Args[0]}, args...)
	main()
	os.Exit(0)
}

func listen(t *testing.T) net.Listener {
	t.Helper()

	lc := net.ListenConfig{}
	l, err := lc.Listen(t.Context(), "tcp", "127.0.0.1:0")
	require.NoError(t, err)

	return l
}

func freeAddr(t *testing.T) string {
	t.Helper()

	l := listen(t)
	addr := l.Addr().String()
	require.NoError(t, l.Close())

	return addr
}

func waitForListener(t *testing.T, addr string) {
	t.Helper()

	d := net.Dialer{}
	require.Eventually(t, func() bool {
		conn, err := d.DialContext(t.Context(), "tcp", addr)
		if err != nil {
			return false
		}
		_ = conn.Close()
		return true
	}, 10*time.Second, 50*time.Millisecond)
}

func mainCmd(t *testing.T, listen string) (*exec.Cmd, *bytes.Buffer) {
	t.Helper()

	//nolint:gosec // re-executes the test binary itself
	cmd := exec.CommandContext(t.Context(), os.Args[0], "-test.run=^TestMainHelper$", "--",
		"-listen", listen,
		"-metrics_listen", "127.0.0.1:0",
		"-remote_host", "127.0.0.1:1",
		"-log_format", "logfmt",
		"-log_level", "info",
	)
	// disable OTel so shutdown doesn't wait 5s trying to flush spans to a
	// collector that isn't running
	cmd.Env = append(os.Environ(), mainHelperEnv+"=1", "OTEL_SDK_DISABLED=true")

	stderr := &bytes.Buffer{}
	cmd.Stderr = stderr

	return cmd, stderr
}

func TestRunReturnsNilOnContextCancel(t *testing.T) {
	t.Setenv("OTEL_SDK_DISABLED", "true") // avoid 5s span flush on shutdown
	metricsRegistry = prometheus.NewRegistry()

	addr := freeAddr(t)
	cfg := &config{
		listen:        addr,
		metricsListen: "127.0.0.1:0",
		remoteHost:    "127.0.0.1:1",
	}

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	done := make(chan error, 1)
	go func() { done <- run(ctx, cfg) }()

	waitForListener(t, addr)
	cancel()

	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(10 * time.Second):
		t.Fatal("run did not return after context cancel")
	}
}

//nolint:paralleltest
func TestRunReturnsErrorOnListenFailure(t *testing.T) {
	metricsRegistry = prometheus.NewRegistry()

	l := listen(t)
	defer l.Close()

	cfg := &config{
		listen:        l.Addr().String(),
		metricsListen: "127.0.0.1:0",
		remoteHost:    "127.0.0.1:1",
	}

	assert.Error(t, run(t.Context(), cfg))
}

func TestMainExitCode(t *testing.T) {
	t.Parallel()

	for _, sig := range []syscall.Signal{syscall.SIGTERM, syscall.SIGINT} {
		t.Run(sig.String(), func(t *testing.T) {
			t.Parallel()

			addr := freeAddr(t)
			cmd, stderr := mainCmd(t, addr)
			require.NoError(t, cmd.Start())

			waitForListener(t, addr)
			require.NoError(t, cmd.Process.Signal(sig))

			require.NoError(t, cmd.Wait(), stderr.String())
			assert.Contains(t, stderr.String(), "level=INFO")
			assert.Contains(t, stderr.String(), "received shutdown signal")
			assert.NotContains(t, stderr.String(), "level=ERROR")
		})
	}
}

func TestMainExitCodeOnError(t *testing.T) {
	t.Parallel()

	l := listen(t)
	defer l.Close()

	cmd, stderr := mainCmd(t, l.Addr().String())

	var exitErr *exec.ExitError
	require.ErrorAs(t, cmd.Run(), &exitErr)
	assert.Equal(t, 1, exitErr.ExitCode())
	assert.Contains(t, stderr.String(), "level=ERROR")
}
