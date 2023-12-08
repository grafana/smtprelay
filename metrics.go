package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	requestsCounter   prometheus.Counter
	errorsCounter     *prometheus.CounterVec
	durationHistogram *prometheus.HistogramVec
	msgSizeHistogram  prometheus.Histogram
)

const mb = 1024 * 1024

func registerMetrics() {
	// TODO: rename this to add a _total suffix
	//nolint:promlinter
	requestsCounter = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "smtprelay",
		Name:      "requests_count",
		Help:      "count of message relay requests",
	})

	// TODO: rename this to add a _total suffix
	//nolint:promlinter
	errorsCounter = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "smtprelay",
		Name:      "errors_count",
		Help:      "count of unsuccessfully relayed messages",
	}, []string{"error_code"})

	durationHistogram = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "smtprelay",
		Name:      "request_duration",
		Help:      "duration of message relay requests",
		Buckets:   prometheus.DefBuckets,
	}, []string{"error_code"})

	msgSizeHistogram = promauto.NewHistogram(prometheus.HistogramOpts{
		Namespace: "smtprelay",
		Name:      "message_bytes",
		Help:      "size of messages",
		Buckets:   []float64{0.05 * mb, 0.1 * mb, 0.25 * mb, 0.5 * mb, 1 * mb, 2 * mb, 5 * mb, 10 * mb, 20 * mb},
	})
}

func handleMetrics(ctx context.Context, addr string) (*instrumentationServer, error) {
	// Setup listeners first, so we can fail early if the address is in use.
	httpListener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to listen at %s: %w", addr, err)
	}

	registerMetrics()

	router := http.NewServeMux()
	router.Handle("/metrics", promhttp.InstrumentMetricHandler(
		prometheus.DefaultRegisterer,
		promhttp.HandlerFor(prometheus.DefaultGatherer, promhttp.HandlerOpts{
			EnableOpenMetrics: true,
		}),
	))

	srv := &http.Server{
		// 5s timeout for header reads to avoid Slowloris attacks (https://thetooth.io/blog/slowloris-attack/)
		ReadHeaderTimeout: 5 * time.Second,
		Handler:           router,
		BaseContext:       func(_ net.Listener) context.Context { return ctx },
	}

	go func() {
		err := srv.Serve(httpListener)
		if err != nil && err != http.ErrServerClosed {
			log.WithError(err).Error("instrumentation server terminated with error")
		}
	}()

	log.WithField("addr", addr).Info("instrumentation server listening")

	return &instrumentationServer{srv: srv}, nil
}

type instrumentationServer struct {
	srv *http.Server
}

func (m *instrumentationServer) Stop() {
	m.srv.Close()
}
