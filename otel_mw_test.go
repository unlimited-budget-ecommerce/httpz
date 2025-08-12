package httpz

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	semconv "go.opentelemetry.io/otel/semconv/v1.30.0"
	"go.opentelemetry.io/otel/trace"
)

func TestOtelMiddleware(t *testing.T) {
	server := startTestServer(t,
		testHandler{
			method: http.MethodGet,
			path:   "/test/otel",
			handlerFunc: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"status":"ok"}`))
			},
		},
		testHandler{
			method: http.MethodPost,
			path:   "/test/otel/error",
			handlerFunc: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"status":"error"}`))
			},
		},
	)

	t.Run("otel middleware disabled", func(t *testing.T) {
		rec := tracetest.NewSpanRecorder()
		tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(rec))
		propagator := propagation.NewCompositeTextMapPropagator(propagation.TraceContext{})
		client := NewClient("test-otel-client", server.URL,
			WithPaths(map[string]string{"otel": "/test/otel"}),
			WithTracer(tp),
			WithPropagator(propagator),
			WithOtelMWEnabled(false),
		)

		res, err := client.NewRequest(context.Background()).Get(client.GetPath("otel"))

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, res.StatusCode())

		spans := rec.Ended()

		assert.Empty(t, spans)
	})

	t.Run("successful request", func(t *testing.T) {
		rec := tracetest.NewSpanRecorder()
		tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(rec))
		propagator := propagation.NewCompositeTextMapPropagator(propagation.TraceContext{})
		client := NewClient("test-otel-client", server.URL,
			WithPaths(map[string]string{"otel": "/test/otel"}),
			WithTracer(tp),
			WithPropagator(propagator),
			WithOtelMWEnabled(true),
		)

		res, err := client.NewRequest(context.Background()).Get(client.GetPath("otel"))

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, res.StatusCode())

		spans := rec.Ended()

		require.Len(t, spans, 1)

		span := spans[0]

		assert.Equal(t, "HTTP GET", span.Name())
		assert.Equal(t, trace.SpanKindClient, span.SpanKind())
		assert.Equal(t, codes.Ok, span.Status().Code)
		assert.Equal(t, "", span.Status().Description)
		assert.Equal(t, http.StatusOK, findIntAttribute(span.Attributes(), semconv.HTTPResponseStatusCodeKey))
		assert.Equal(t, client.GetPath("otel"), findStringAttribute(span.Attributes(), semconv.URLFullKey))
		assert.Equal(t, "GET", findStringAttribute(span.Attributes(), semconv.HTTPRequestMethodKey))
	})

	t.Run("request with http error", func(t *testing.T) {
		rec := tracetest.NewSpanRecorder()
		tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(rec))
		propagator := propagation.NewCompositeTextMapPropagator(propagation.TraceContext{})
		client := NewClient("test-otel-client", server.URL,
			WithPaths(map[string]string{"otelError": "/test/otel/error"}),
			WithTracer(tp),
			WithPropagator(propagator),
			WithOtelMWEnabled(true),
		)

		res, err := client.NewRequest(context.Background()).Post(client.GetPath("otelError"))

		require.NoError(t, err)
		assert.Equal(t, http.StatusInternalServerError, res.StatusCode())

		spans := rec.Ended()

		require.Len(t, spans, 1)

		span := spans[0]

		assert.Equal(t, "HTTP POST", span.Name())
		assert.Equal(t, trace.SpanKindClient, span.SpanKind())
		assert.Equal(t, codes.Error, span.Status().Code)
		assert.Equal(t, "500 Internal Server Error", span.Status().Description)
		assert.Equal(t, http.StatusInternalServerError, findIntAttribute(span.Attributes(), semconv.HTTPResponseStatusCodeKey))
	})

	t.Run("request with transport error", func(t *testing.T) {
		rec := tracetest.NewSpanRecorder()
		tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(rec))
		propagator := propagation.NewCompositeTextMapPropagator(propagation.TraceContext{})
		clientWithBadHost := NewClient("test-otel-client", "http://localhost:9999",
			WithTracer(tp),
			WithPropagator(propagator),
			WithOtelMWEnabled(true),
		)

		_, err := clientWithBadHost.NewRequest(context.Background()).Get("/test")

		require.Error(t, err)

		spans := rec.Ended()

		require.Len(t, spans, 1)

		span := spans[0]

		assert.Equal(t, "HTTP GET", span.Name())
		assert.Equal(t, trace.SpanKindClient, span.SpanKind())
		assert.Equal(t, codes.Error, span.Status().Code)
		assert.Equal(t, err.Error(), span.Status().Description)
		require.NotEmpty(t, span.Events())
		assert.Equal(t, "exception", span.Events()[0].Name)
	})

	t.Run("with parent span", func(t *testing.T) {
		rec := tracetest.NewSpanRecorder()
		tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(rec))
		propagator := propagation.NewCompositeTextMapPropagator(propagation.TraceContext{})
		client := NewClient("test-otel-client", server.URL,
			WithPaths(map[string]string{"otel": "/test/otel"}),
			WithTracer(tp),
			WithPropagator(propagator),
			WithOtelMWEnabled(true),
		)
		tracer := tp.Tracer("test-tracer")
		ctx, parentSpan := tracer.Start(context.Background(), "parent-span")

		res, err := client.NewRequest(ctx).Get(client.GetPath("otel"))

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, res.StatusCode())

		parentSpan.End()
		spans := rec.Ended()

		require.Len(t, spans, 2) // parent and child

		var childSpan, endedParentSpan sdktrace.ReadOnlySpan
		for _, s := range spans {
			if s.Parent().HasSpanID() {
				childSpan = s
			} else {
				endedParentSpan = s
			}
		}

		require.NotNil(t, childSpan)
		require.NotNil(t, endedParentSpan)
		assert.Equal(t, endedParentSpan.SpanContext().SpanID(), childSpan.Parent().SpanID())
		assert.Equal(t, "parent-span", endedParentSpan.Name())
	})
}

func findIntAttribute(attrs []attribute.KeyValue, key attribute.Key) int {
	for _, attr := range attrs {
		if attr.Key == key {
			return int(attr.Value.AsInt64())
		}
	}
	return 0
}

func findStringAttribute(attrs []attribute.KeyValue, key attribute.Key) string {
	for _, attr := range attrs {
		if attr.Key == key {
			return attr.Value.AsString()
		}
	}
	return ""
}
