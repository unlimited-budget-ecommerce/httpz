package httpz

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/go-resty/resty/v2"
	"github.com/goccy/go-json"
	"github.com/unlimited-budget-ecommerce/logz"
	semconv "go.opentelemetry.io/otel/semconv/v1.30.0"
)

func logRequest(cfg *config) resty.RequestMiddleware {
	return func(_ *resty.Client, req *resty.Request) error {
		if !cfg.logMWEnabled {
			return nil
		}

		logger := cfg.logger.With(
			slog.String(string(semconv.URLFullKey), req.URL),
			slog.String(string(semconv.HTTPRequestMethodKey), req.Method),
			slog.Any("http.request.header", logz.MaskHttpHeader(req.Header)),
		)

		ctx := req.Context()
		body, err := json.Marshal(req.Body)
		if err != nil {
			logger.ErrorContext(
				ctx,
				"[HTTPZ][OUTGOING REQUEST] error marshalling request body: "+err.Error(),
			)
			return err
		}

		logger.InfoContext(ctx, "[HTTPZ][OUTGOING REQUEST] success",
			slog.Any("http.request.body", maskBytes(ctx, body, "request body")),
		)

		return nil
	}
}

func logResponse(cfg *config) resty.ResponseMiddleware {
	return func(_ *resty.Client, res *resty.Response) error {
		if !cfg.logMWEnabled {
			return nil
		}

		ctx := res.Request.Context()
		logger := cfg.logger.With(
			slog.String(string(semconv.URLFullKey), res.Request.URL),
			slog.String(string(semconv.HTTPRequestMethodKey), res.Request.Method),
			slog.Int64(semconv.HTTPClientRequestDurationName, res.Time().Milliseconds()),
			slog.Int(string(semconv.HTTPResponseStatusCodeKey), res.StatusCode()),
			slog.Any("http.response.header", logz.MaskHttpHeader(res.Header())),
			slog.Any("http.response.body", maskBytes(ctx, res.Body(), "response body")),
		)

		if res.IsError() {
			logger.ErrorContext(ctx, "[HTTPZ][INCOMING RESPONSE] error")
		} else {
			logger.InfoContext(ctx, "[HTTPZ][INCOMING RESPONSE] success")
		}

		return nil
	}
}

func maskBytes(ctx context.Context, b []byte, bodyType string) map[string]any {
	m := make(map[string]any)
	err := json.Unmarshal(b, &m)
	if err != nil {
		slog.ErrorContext(ctx, fmt.Sprintf("[HTTPZ] error unmarshalling %s: %s", bodyType, err.Error()))
		return m
	}
	return logz.MaskMap(m)
}
