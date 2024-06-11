package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/grafana/smtprelay/v2/internal/smtpd"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

func setupLogger(format, level string) {
	lvl := slog.LevelDebug
	switch level {
	case "debug":
		lvl = slog.LevelDebug
	case "info":
		lvl = slog.LevelInfo
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	}

	opts := &slog.HandlerOptions{
		Level:     lvl,
		AddSource: true,
	}

	var handler slog.Handler
	switch format {
	case "logfmt":
		handler = slog.NewTextHandler(os.Stderr, opts)
	default:
		handler = slog.NewJSONHandler(os.Stderr, opts)
	}

	handler = &traceLogHandler{handler}

	slog.SetDefault(slog.New(handler))
}

type traceLogHandler struct {
	slog.Handler
}

var _ slog.Handler = (*traceLogHandler)(nil)

func (h *traceLogHandler) Handle(ctx context.Context, r slog.Record) error {
	span := trace.SpanFromContext(ctx)
	if span.IsRecording() {
		var attrs []attribute.KeyValue
		r.Attrs(func(a slog.Attr) bool {
			attrs = append(attrs, attribute.String(a.Key, a.Value.String()))
			return true
		})

		r.Add(slog.String("traceID", span.SpanContext().TraceID().String()))

		span.AddEvent(r.Message, trace.WithAttributes(attrs...))
	}

	if addr := smtpd.LocalAddrFromContext(ctx); addr != nil {
		r.Add(slog.Any("peer", addr.String()))
	}

	return h.Handler.Handle(ctx, r)
}

func (h *traceLogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &traceLogHandler{h.Handler.WithAttrs(attrs)}
}

func (h *traceLogHandler) Enabled(ctx context.Context, lvl slog.Level) bool {
	return h.Handler.Enabled(ctx, lvl)
}

func (h *traceLogHandler) WithGroup(name string) slog.Handler {
	return &traceLogHandler{h.Handler.WithGroup(name)}
}
