package httpz

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/goccy/go-json"
	"go.opentelemetry.io/otel"
	"resty.dev/v3"
)

type Client struct {
	resty.Client
	name    string
	version string
	paths   map[string]string
}

func NewClient(clientName, baseURL string, opts ...option) *Client {
	cfg := config{}
	for _, opt := range opts {
		opt(&cfg)
	}
	if cfg.transport == nil {
		cfg.transport = http.DefaultTransport
	}
	if cfg.paths == nil {
		cfg.paths = make(map[string]string)
	}
	if cfg.logger == nil {
		cfg.logger = slog.Default()
	}
	if cfg.tracer == nil {
		cfg.tracer = otel.GetTracerProvider()
	}
	if cfg.propagator == nil {
		cfg.propagator = otel.GetTextMapPropagator()
	}
	if !cfg.circuitBreakerEnabled {
		cfg.circuitBreaker = nil
	}

	restyClient := resty.NewWithClient(&http.Client{
		Transport: cfg.transport,
	})
	restyClient.
		SetBaseURL(baseURL).
		SetCircuitBreaker(cfg.circuitBreaker).
		AddContentTypeDecoder("application/json", func(r io.Reader, v any) error {
			return json.NewDecoder(r).Decode(v)
		}).
		SetHeaders(cfg.baseHeaders).
		SetLogger(logger{cfg.logger}).
		AddRequestMiddleware(startTrace(&cfg)).
		AddRequestMiddleware(logRequest(&cfg)).
		AddResponseMiddleware(logResponse(&cfg)).
		AddResponseMiddleware(endTraceSuccess(&cfg)).
		OnError(endTraceError(&cfg)).
		OnPanic(endTraceError(&cfg))

	return &Client{
		Client:  *restyClient,
		name:    clientName,
		version: cfg.serviceVersion,
		paths:   cfg.paths,
	}
}

func (c *Client) GetPath(pathName string) string {
	return c.paths[pathName]
}

// NewRequest returns *[resty.Request] from given context.
//
// It sets default headers "Content-Type" to "application/json" and "User-Agent"
// based on the client name and version.
func (c *Client) NewRequest(ctx context.Context) *resty.Request {
	return c.R().
		SetContext(ctx).
		SetHeaders(map[string]string{
			"Content-Type": "application/json",
			"User-Agent":   fmt.Sprintf("%s/%s", c.name, c.version),
		})
}
