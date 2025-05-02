package httpz

import (
	"log/slog"
	"net/http"
)

type (
	config struct {
		transport http.RoundTripper
		paths     map[string]path
		logger    *slog.Logger
	}
	path struct {
		path string
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

func WithPaths(p map[string]path) option {
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
