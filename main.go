package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"

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
	listeners := make([]net.Listener, len(addresses))

	for i := range addresses {
		address := addresses[i]

		relay, err := newRelay(logger, cfg)
		if err != nil {
			return fmt.Errorf("error creating relay: %w", err)
		}

		listener, err := relay.listen(address)
		if err != nil {
			return fmt.Errorf("error listening on address %q: %w", address, err)
		}

		logger.Info("listening on address", slog.String("addrress", address))

		go func() {
			_ = relay.serve(listener)
		}()

		listeners[i] = listener
	}

	handleSignals(listeners)

	return nil
}

func handleSignals(servers []net.Listener) {
	sigs := make(chan os.Signal, 1)
	done := make(chan bool, 1)

	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigs

		logger := slog.Default().With(slog.String("component", "signal_handler"))

		for _, s := range servers {
			logger.Warn("closing listener in response to received signal",
				slog.String("signal", sig.String()), slog.String("addr", s.Addr().String()))

			err := s.Close()
			if err != nil {
				logger.Warn("could not close listener", slog.Any("error", err))
			}
		}

		done <- true
	}()

	<-done
}
