package httpz

import (
	"log/slog"
	"net/http"

	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

type (
	config struct {
		transport      http.RoundTripper
		baseHeaders    map[string]string
		paths          map[string]string
		logger         *slog.Logger
		logMWEnabled   bool
		tracer         trace.TracerProvider
		propagator     propagation.TextMapPropagator
		otelMWEnabled  bool
		serviceVersion string
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

func WithBaseHeaders(h map[string]string) option {
	return option(func(cfg *config) {
		if h != nil {
			cfg.baseHeaders = h
		}
	})
}

func WithPaths(p map[string]string) option {
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

func WithTracer(t trace.TracerProvider) option {
	return option(func(cfg *config) {
		if t != nil {
			cfg.tracer = t
		}
	})
}

func WithPropagator(p propagation.TextMapPropagator) option {
	return option(func(cfg *config) {
		if p != nil {
			cfg.propagator = p
		}
	})
}

func WithOtelMWEnabled(enabled bool) option {
	return option(func(cfg *config) {
		cfg.otelMWEnabled = enabled
	})
}

func WithServiceVersion(version string) option {
	return option(func(cfg *config) {
		cfg.serviceVersion = version
	})
}
