package main

import (
	"net/http"

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
	requestsCounter = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "smtprelay",
		Name:      "requests_count",
		Help:      "count of message relay requests",
	})

	errorsCounter = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "smtprelay",
		Name:      "errors_count",
		Help:      "count of unsuccessfully relayed messages",
	}, []string{"error_code"})

	durationHistogram = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "smtprelay",
		Name:      "request_duration",
		Buckets:   prometheus.DefBuckets,
	}, []string{"error_code"})

	msgSizeHistogram = promauto.NewHistogram(prometheus.HistogramOpts{
		Namespace: "smtprelay",
		Name:      "message_bytes",
		Buckets:   []float64{1 * mb, 10 * mb, 20 * mb, 30 * mb, 40 * mb, 50 * mb},
	})
}

func handleMetrics() {
	registerMetrics()

	http.Handle("/metrics", promhttp.Handler())
	if err := http.ListenAndServe(*metricsListen, nil); err != nil {
		log.WithField("err", err.Error()).
			Fatal("cannot publish metrics")
	}
}
