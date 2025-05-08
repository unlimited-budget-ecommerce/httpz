package httpz

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/unlimited-budget-ecommerce/logz"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/semconv/v1.20.0/httpconv"
	semconv "go.opentelemetry.io/otel/semconv/v1.30.0"
	"go.opentelemetry.io/otel/trace"
)

func startTrace(cfg *config) resty.RequestMiddleware {
	return func(_ *resty.Client, req *resty.Request) error {
		if !cfg.otelMWEnabled {
			return nil
		}

		ctx := req.Context()
		parentSpanCtx := trace.SpanFromContext(ctx).SpanContext()
		if !parentSpanCtx.IsValid() {
			return nil
		}

		tracer := cfg.tracer.Tracer("httpz-tracer-middleware")
		ctx, span := tracer.Start(
			ctx,
			fmt.Sprintf("[HTTPZ][OUTGOING REQUEST] %s %s", req.Method, req.URL),
			trace.WithSpanKind(trace.SpanKindClient),
			trace.WithAttributes(
				semconv.URLFull(req.URL),
				semconv.HTTPRequestMethodKey.String(req.Method),
			),
			trace.WithTimestamp(time.Now()),
		)

		ctx = logz.SetContextAttrs(
			ctx,
			slog.String(logz.SpanKey, span.SpanContext().SpanID().String()),
			slog.String(logz.ParentSpanKey, parentSpanCtx.SpanID().String()),
		)

		cfg.propagator.Inject(ctx, propagation.HeaderCarrier(req.Header))
		req.SetContext(ctx)

		return nil
	}
}

func endTraceSuccess(cfg *config) resty.ResponseMiddleware {
	return func(_ *resty.Client, res *resty.Response) error {
		if !cfg.otelMWEnabled {
			return nil
		}

		span := trace.SpanFromContext(res.Request.Context())
		defer span.End()
		span.SetAttributes(
			attribute.KeyValue{
				Key:   semconv.HTTPClientRequestDurationName,
				Value: attribute.Int64Value(res.Time().Milliseconds()),
			},
			semconv.HTTPResponseStatusCode(res.StatusCode()),
		)

		code := codes.Ok
		if res.IsError() {
			code = codes.Error
		}
		span.SetStatus(code, res.Status())

		return nil
	}
}

func endTraceError(cfg *config) resty.ErrorHook {
	return func(req *resty.Request, err error) {
		if !cfg.otelMWEnabled {
			return
		}

		span := trace.SpanFromContext(req.Context())
		defer span.End()
		span.SetAttributes(httpconv.ClientRequest(req.RawRequest)...)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
}
