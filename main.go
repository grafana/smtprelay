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

	"github.com/grafana/smtprelay/internal/smtpd"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/version"
)

// metrics registry - overridable for tests
var metricsRegistry = prometheus.DefaultRegisterer

const applicationName = "smtprelay"

func main() {
	// load config as first thing
	cfg, err := loadConfig()
	if err != nil {
		slog.Default().Error("error loading config", slog.Any("error", err))
		os.Exit(1)
	}

	if cfg.versionInfo {
		fmt.Printf("%s %s\n", applicationName, version.Info())
		return
	}

	logger := slog.Default()

	// print version on start
	logger.Debug("config loaded", slog.String("version", version.Version))

	if err := run(cfg); err != nil {
		logger.Error("error running smtprelay", slog.Any("error", err))
		os.Exit(1)
	}
}

func run(cfg *config) error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM)
	defer stop()

	metricsSrv, err := handleMetrics(ctx, cfg.metricsListen, metricsRegistry)
	if err != nil {
		return fmt.Errorf("could not start metrics server: %w", err)
	}
	defer metricsSrv.Stop()

	logger := slog.Default()

	addresses := strings.Split(cfg.listen, " ")

	for i := range addresses {
		address := addresses[i]

		var relay *relay
		relay, err = newRelay(logger, cfg)
		if err != nil {
			return fmt.Errorf("error creating relay: %w", err)
		}

		var listener net.Listener
		listener, err = relay.listen(address)
		if err != nil {
			return fmt.Errorf("error listening on address %q: %w", address, err)
		}

		logger.Info("listening on address", slog.String("address", address))

		defer func(ctx context.Context) {
			logger.Warn("closing listener", slog.String("address", address))

			_ = relay.shutdown(ctx)
		}(ctx)

		go func() {
			err = relay.serve(listener)
			if err != nil && !errors.Is(err, smtpd.ErrServerClosed) {
				err = fmt.Errorf("relay shutdown with an error: %w", err)
			}

			// cancel the context so we can exit
			stop()
		}()
	}

	// Now wait for the context to be cancelled through a signal or other cause
	<-ctx.Done()

	if err == nil {
		// if we got to this point without err being set, it's probably due to
		// a signal being received
		err = fmt.Errorf("exiting: %w", ctx.Err())
	}

	return err
}
