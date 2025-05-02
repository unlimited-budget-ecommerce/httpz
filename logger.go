package httpz

import (
	"fmt"
	"log/slog"

	"github.com/go-resty/resty/v2"
)

type logger struct{ slog.Logger }

var _ resty.Logger = (*logger)(nil)

func (l logger) Debugf(format string, v ...any) {
	l.Debug(fmt.Sprintf(format, v...))
}

func (l logger) Warnf(format string, v ...any) {
	l.Warn(fmt.Sprintf(format, v...))
}

func (l logger) Errorf(format string, v ...any) {
	l.Error(fmt.Sprintf(format, v...))
}
