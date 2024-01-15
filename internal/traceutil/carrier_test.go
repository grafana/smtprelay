package traceutil

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

func TestMIMEHeaderCarrier(t *testing.T) {
	hc := MIMEHeaderCarrier{}
	hc.Set("Foo", "bar")
	hc.Set("Traceparent", "00-e775b110dfe5dd5e0f385d5afe2df71e-8cd5b7ec6ac3bcab-01")

	prop := propagation.TraceContext{}
	ctx := prop.Extract(context.Background(), hc)
	span := trace.SpanFromContext(ctx)
	assert.Equal(t, "e775b110dfe5dd5e0f385d5afe2df71e", span.SpanContext().TraceID().String())

	keys := hc.Keys()
	assert.EqualValues(t, []string{"Foo", "Traceparent"}, keys)
}
