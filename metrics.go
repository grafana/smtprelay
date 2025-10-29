package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/http/pprof"
	"strconv"
	"time"

	deltapprof "github.com/grafana/pyroscope-go/godeltaprof/http/pprof"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors/version"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel/trace"
)

var (
	requestsCounter    prometheus.Counter
	errorsCounter      *prometheus.CounterVec
	durationHistogram  *prometheus.HistogramVec
	durationNative     *prometheus.HistogramVec
	msgSizeHistogram   prometheus.Histogram
	rateLimitedCounter *prometheus.CounterVec
)

const mb = 1024 * 1024

func init() {
	ns := applicationName

	// TODO: rename this to add a _total suffix
	requestsCounter = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: ns,
		Name:      "requests_count",
		Help:      "count of message relay requests",
	})

	// TODO: rename this to add a _total suffix
	errorsCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: ns,
		Name:      "errors_count",
		Help:      "count of unsuccessfully relayed messages",
	}, []string{"error_code"})

	// TODO: remove this
	durationHistogram = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: ns,
		Name:      "request_duration",
		Help:      "duration of message relay requests",
		Buckets:   prometheus.DefBuckets,
	}, []string{"error_code"})

	durationNative = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace:                       ns,
		Subsystem:                       "relay",
		Name:                            "duration_seconds",
		Help:                            "duration of message relay requests",
		Buckets:                         prometheus.DefBuckets,
		NativeHistogramBucketFactor:     1.1,
		NativeHistogramMaxBucketNumber:  160,
		NativeHistogramMinResetDuration: 1 * time.Hour,
	}, []string{"status_code"})

	msgSizeHistogram = prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: ns,
		Name:      "message_bytes",
		Help:      "size of messages",
		Buckets:   []float64{0.05 * mb, 0.1 * mb, 0.25 * mb, 0.5 * mb, 1 * mb, 2 * mb, 5 * mb, 10 * mb, 20 * mb},
	})

	rateLimitedCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: ns,
		Name:      "rate_limited_total",
		Help:      "count of rate limited messages by sender",
	}, []string{"sender"})
}

func registerMetrics(registry prometheus.Registerer) error {
	err := registry.Register(requestsCounter)
	if err != nil {
		return err
	}
	err = registry.Register(errorsCounter)
	if err != nil {
		return err
	}
	err = registry.Register(durationHistogram)
	if err != nil {
		return err
	}
	err = registry.Register(durationNative)
	if err != nil {
		return err
	}
	err = registry.Register(msgSizeHistogram)
	if err != nil {
		return err
	}

	err = registry.Register(rateLimitedCounter)
	if err != nil {
		return err
	}

	err = registry.Register(version.NewCollector(applicationName))
	if err != nil {
		return err
	}

	return nil
}

func handleMetrics(ctx context.Context, addr string, registry prometheus.Registerer) (*instrumentationServer, error) {
	log := slog.Default().With(slog.String("component", "metrics"))

	// Setup listeners first, so we can fail early if the address is in use.
	httpListener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to listen at %s: %w", addr, err)
	}

	if err = registerMetrics(registry); err != nil {
		return nil, fmt.Errorf("registerMetrics: %w", err)
	}

	router := http.NewServeMux()
	router.Handle("/metrics", promhttp.InstrumentMetricHandler(
		prometheus.DefaultRegisterer,
		promhttp.HandlerFor(prometheus.DefaultGatherer, promhttp.HandlerOpts{
			EnableOpenMetrics: true,
		}),
	))

	router.HandleFunc("/debug/pprof/", pprof.Index)
	router.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	router.HandleFunc("/debug/pprof/profile", pprof.Profile)
	router.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	router.HandleFunc("/debug/pprof/trace", pprof.Trace)
	router.HandleFunc("/debug/pprof/delta_heap", deltapprof.Heap)
	router.HandleFunc("/debug/pprof/delta_block", deltapprof.Block)
	router.HandleFunc("/debug/pprof/delta_mutex", deltapprof.Mutex)

	srv := &http.Server{
		// 5s timeout for header reads to avoid Slowloris attacks (https://thetooth.io/blog/slowloris-attack/)
		ReadHeaderTimeout: 5 * time.Second,
		Handler:           router,
		BaseContext:       func(_ net.Listener) context.Context { return ctx },
	}

	go func() {
		err := srv.Serve(httpListener)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("instrumentation server terminated with error", slog.Any("error", err))
		}
	}()

	log.Info("instrumentation server listening", slog.String("addr", addr))

	return &instrumentationServer{srv: srv}, nil
}

type instrumentationServer struct {
	srv *http.Server
}

func (m *instrumentationServer) Stop() {
	m.srv.Close()
}

// observeDuration records the duration of a message relay request.
func observeDuration(ctx context.Context, statusCode int, duration time.Duration) {
	traceID := trace.SpanFromContext(ctx).SpanContext().TraceID()
	exemplarLabels := prometheus.Labels{}
	if traceID.IsValid() {
		exemplarLabels["traceID"] = traceID.String()
	}

	durHist := durationNative.WithLabelValues(strconv.Itoa(statusCode)).(prometheus.ExemplarObserver)
	durHist.ObserveWithExemplar(duration.Seconds(), exemplarLabels)

	// legacy metric doesn't get exemplar - it's going away
	durationHistogram.WithLabelValues(strconv.Itoa(statusCode)).Observe(duration.Seconds())
}
