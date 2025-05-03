package httpz

import (
	"log/slog"

	"github.com/go-resty/resty/v2"
)

func logRequest(l *slog.Logger) resty.RequestMiddleware {
	return func(_ *resty.Client, req *resty.Request) error {
		l.Info("[HTTPZ][OUTGOING REQUEST]",
			slog.String("url", req.URL),
			slog.String("method", req.Method),
			slog.Any("request_header", req.Header),
			slog.Any("request_body", req.Body),
		)

		return nil
	}
}

func logResponse(l *slog.Logger) resty.ResponseMiddleware {
	return func(_ *resty.Client, res *resty.Response) error {
		logger := l.With(
			slog.String("url", res.Request.URL),
			slog.String("method", res.Request.Method),
			slog.Int64("total_time_ms", res.Time().Milliseconds()),
			slog.Int("status_code", res.StatusCode()),
			slog.Any("response_header", res.Header),
			slog.Any("response_body", res.Body()),
		)

		if res.IsError() {
			logger.Error("[HTTPZ][INCOMING RESPONSE] error")
		} else {
			logger.Info("[HTTPZ][INCOMING RESPONSE] success")
		}

		return nil
	}
}
