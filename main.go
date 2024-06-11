package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/grafana/smtprelay/v2/internal/smtpd"
	"github.com/grafana/smtprelay/v2/internal/traceutil"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/version"
	"go.opentelemetry.io/otel"
)

var tracer = otel.Tracer("github.com/grafana/smtprelay/v2")

// metrics registry - overridable for tests
var metricsRegistry = prometheus.DefaultRegisterer

const applicationName = "smtprelay"

func main() {
	// load config as first thing
	cfg, err := loadConfig()
	if err != nil {
		slog.Error("error loading config", slog.Any("error", err))
		os.Exit(1)
	}

	if cfg.versionInfo {
		fmt.Printf("%s %s\n", applicationName, version.Info())
		return
	}

	// print version on start
	slog.Debug("config loaded", slog.String("version", version.Version))

	if err := run(context.Background(), cfg); err != nil {
		slog.Error("error running smtprelay", slog.Any("error", err))
		os.Exit(1)
	}
}

func run(ctx context.Context, cfg *config) error {
	ctx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM)
	defer stop()

	metricsSrv, err := handleMetrics(ctx, cfg.metricsListen, metricsRegistry)
	if err != nil {
		return fmt.Errorf("could not start metrics server: %w", err)
	}
	defer metricsSrv.Stop()

	closer, err := traceutil.InitTraceExporter(ctx, "smtprelay")
	if err != nil {
		return fmt.Errorf("init trace exporter: %w", err)
	}
	//nolint:errcheck
	defer closer(ctx)

	addresses := strings.Split(cfg.listen, " ")

	errch := make(chan error)

	for i := range addresses {
		address := addresses[i]

		var relay *relay
		relay, err = newRelay(cfg)
		if err != nil {
			return fmt.Errorf("error creating relay: %w", err)
		}

		var listener net.Listener
		listener, err = relay.listen(address)
		if err != nil {
			return fmt.Errorf("error listening on address %q: %w", address, err)
		}

		slog.InfoContext(ctx, "listening on address", slog.String("address", address))

		defer func(ctx context.Context) {
			slog.WarnContext(ctx, "closing listener", slog.String("address", address))

			_ = relay.shutdown(ctx)
		}(ctx)

		go func() {
			serveErr := relay.serve(ctx, listener)
			if serveErr != nil && !errors.Is(serveErr, smtpd.ErrServerClosed) {
				serveErr = fmt.Errorf("relay shutdown with an error: %w", serveErr)
			}

			errch <- serveErr
		}()
	}

	// Now wait for the server to stop, either by a signal or by an error
	select {
	case err = <-errch:
		err = fmt.Errorf("relay error: %w", err)
	case <-ctx.Done():
		// if we got to this point without err being set, it's probably due to
		// a signal being received
		err = fmt.Errorf("exiting: %w", ctx.Err())
	}

	return err
}
