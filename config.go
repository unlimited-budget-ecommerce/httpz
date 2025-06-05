package httpz

import (
	"log/slog"
	"net/http"
	"time"

	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"resty.dev/v3"
)

type (
	config struct {
		transport             http.RoundTripper
		baseHeaders           map[string]string
		paths                 map[string]string
		logger                *slog.Logger
		tracer                trace.TracerProvider
		propagator            propagation.TextMapPropagator
		serviceVersion        string
		circuitBreaker        *resty.CircuitBreaker
		logMWEnabled          bool
		otelMWEnabled         bool
		circuitBreakerEnabled bool
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

// WithCircuitBreaker accepts:
//   - timeout - duration window for circuit breaker to determine the state
//   - failureThreshold - number of failures that must occur within the timeout duration to transition to Open state
//   - successThreshold - number of successes that must occur to transition from Half-Open state to Closed state
//   - policies - determine whether a request is failed or successful by evaluating the response instance
//
// passing zero values will result to default values: 10s, 3, 1, Status Code 500 and above
func WithCircuitBreaker(
	timeout time.Duration,
	failureThreshold, successThreshold uint32,
	policies ...func(*http.Response) bool,
) option {
	return option(func(cfg *config) {
		cfg.circuitBreaker = resty.NewCircuitBreaker()
		if timeout > 0 {
			cfg.circuitBreaker.SetTimeout(timeout)
		}
		if failureThreshold > 0 {
			cfg.circuitBreaker.SetFailureThreshold(failureThreshold)
		}
		if successThreshold > 0 {
			cfg.circuitBreaker.SetSuccessThreshold(successThreshold)
		}
		if len(policies) > 0 {
			pp := make([]resty.CircuitBreakerPolicy, 0, len(policies))
			for _, p := range policies {
				if p != nil {
					pp = append(pp, resty.CircuitBreakerPolicy(p))
				}
			}
			if len(pp) > 0 {
				cfg.circuitBreaker.SetPolicies(pp...)
			}
		}
	})
}

func WithCircuitBreakerEnabled(enabled bool) option {
	return option(func(cfg *config) {
		cfg.circuitBreakerEnabled = enabled
	})
}
