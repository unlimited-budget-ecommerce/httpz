package httpz

import (
	"fmt"
	"log/slog"

	"resty.dev/v3"
)

type logger struct{ *slog.Logger }

var _ resty.Logger = (*logger)(nil)

func (l logger) Debugf(format string, v ...any) {
	l.Debug("[HTTPZ] " + fmt.Sprintf(format, v...))
}

func (l logger) Warnf(format string, v ...any) {
	l.Warn("[HTTPZ] " + fmt.Sprintf(format, v...))
}

func (l logger) Errorf(format string, v ...any) {
	l.Error("[HTTPZ] " + fmt.Sprintf(format, v...))
}
