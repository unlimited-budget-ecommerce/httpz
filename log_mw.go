package httpz

import (
	"log/slog"

	"github.com/unlimited-budget-ecommerce/logz"
	semconv "go.opentelemetry.io/otel/semconv/v1.30.0"
	"resty.dev/v3"
)

func logRequest(cfg *config) resty.RequestMiddleware {
	return func(_ *resty.Client, req *resty.Request) error {
		if !cfg.logMWEnabled {
			return nil
		}

		cfg.logger.InfoContext(req.Context(), "[HTTPZ][OUTGOING REQUEST] success",
			slog.String(string(semconv.URLFullKey), req.URL),
			slog.String(string(semconv.HTTPRequestMethodKey), req.Method),
			slog.Any("http.request.header", logz.MaskHttpHeader(req.Header)),
			slog.Any("http.request.body", req.Body),
		)

		return nil
	}
}

func logResponse(cfg *config) resty.ResponseMiddleware {
	return func(_ *resty.Client, res *resty.Response) error {
		if !cfg.logMWEnabled {
			return nil
		}

		logger := cfg.logger.With(
			slog.String(string(semconv.URLFullKey), res.Request.URL),
			slog.String(string(semconv.HTTPRequestMethodKey), res.Request.Method),
			slog.Duration(semconv.HTTPClientRequestDurationName, res.Duration()),
			slog.Int(string(semconv.HTTPResponseStatusCodeKey), res.StatusCode()),
			slog.Any("http.response.header", logz.MaskHttpHeader(res.Header())),
			slog.Any("http.response.body", res.Result()),
		)

		ctx := res.Request.Context()
		if res.IsError() {
			logger.ErrorContext(ctx, "[HTTPZ][INCOMING RESPONSE] error")
		} else {
			logger.InfoContext(ctx, "[HTTPZ][INCOMING RESPONSE] success")
		}

		return nil
	}
}
