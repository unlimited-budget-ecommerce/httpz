package httpz

import (
	"log/slog"

	"github.com/go-resty/resty/v2"
)

func logReqRes(l *slog.Logger) resty.ResponseMiddleware {
	return func(_ *resty.Client, res *resty.Response) error {
		logger := l.With(
			slog.String("url", res.Request.URL),
			slog.String("method", res.Request.Method),
			slog.Int64("total_time_ms", res.Time().Milliseconds()),
			slog.Int("status_code", res.StatusCode()),
			slog.Any("request_header", res.Request.Header),
			slog.Any("request_body", res.Request.Body),
			slog.Any("response_header", res.Header),
			slog.Any("response_body", res.Body()),
		)

		if res.IsError() {
			logger.Error("[HTTPZ] error")
		} else {
			logger.Info("[HTTPZ] success")
		}

		return nil
	}
}
