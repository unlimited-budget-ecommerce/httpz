package httpz

import (
	"encoding/json"
	"log/slog"

	"github.com/go-resty/resty/v2"
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
			slog.Any("http.request.header", req.Header), // TODO: wraps with mask map
		)

		body, err := json.Marshal(req.Body)
		if err != nil {
			logger.Error("[HTTPZ][OUTGOING REQUEST] error marshalling request body: " + err.Error())
			return err
		}

		logger.Info("[HTTPZ][OUTGOING REQUEST] success",
			slog.Any("http.request.body", body), // TODO: wraps with mask []byte
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
			slog.Int64(semconv.HTTPClientRequestDurationName, res.Time().Milliseconds()),
			slog.Int(string(semconv.HTTPResponseStatusCodeKey), res.StatusCode()),
			slog.Any("http.response.header", res.Header()), // TODO: wraps with mask map
			slog.Any("http.response.body", res.Body()),     // TODO: wraps with mask []byte
		)

		if res.IsError() {
			logger.Error("[HTTPZ][INCOMING RESPONSE] error")
		} else {
			logger.Info("[HTTPZ][INCOMING RESPONSE] success")
		}

		return nil
	}
}
