package httpz

import (
	"log/slog"
	"net/http"
)

type (
	config struct {
		transport     http.RoundTripper
		paths         map[string]Path
		logger        *slog.Logger
		logMWEnabled  bool
		otelMWEnabled bool
	}
	Path struct {
		Path string
	}
)

type option func(*config)

func WithTransport(t *http.Transport) option {
	return option(func(cfg *config) {
		if t != nil {
			cfg.transport = t
		}
	})
}

func WithPaths(p map[string]Path) option {
	return option(func(cfg *config) {
		if p != nil {
			cfg.paths = p
		}
	})
}

func WithLogger(l *slog.Logger) option {
	return option(func(cfg *config) {
		if l != nil {
			cfg.logger = l
		}
	})
}

func WithLogMWEnabled(enabled bool) option {
	return option(func(cfg *config) {
		cfg.logMWEnabled = enabled
	})
}

func WithOtelMWEnabled(enabled bool) option {
	return option(func(cfg *config) {
		cfg.otelMWEnabled = enabled
	})
}
