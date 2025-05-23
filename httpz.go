package httpz

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/go-resty/resty/v2"
	"go.opentelemetry.io/otel"
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

	restyClient := resty.NewWithClient(&http.Client{
		Transport: cfg.transport,
	})
	restyClient.BaseURL = baseURL
	restyClient.
		SetHeaders(cfg.baseHeaders).
		SetLogger(logger{cfg.logger}).
		OnBeforeRequest(startTrace(&cfg)).
		OnBeforeRequest(logRequest(&cfg)).
		OnAfterResponse(logResponse(&cfg)).
		OnAfterResponse(endTraceSuccess(&cfg)).
		OnError(endTraceError(&cfg)).
		OnPanic(endTraceError(&cfg))

	return &Client{
		Client:  *restyClient,
		name:    clientName,
		version: cfg.serviceVersion,
		paths:   cfg.paths,
	}
}

type (
	Request struct {
		// PathName is the name registered with [httpz.WithPaths]
		PathName string
		// use this field instead of [resty.Request.QueryParam]
		QueryParams map[string]string
		resty.Request
	}
	Response[T any] struct {
		Result T
		http.Response
	}
)

// Do executes an HTTP request and returns a typed response *T and [*resty.Response].
//
// It looks up the request path by name from the client's registered paths.
// It also sets default headers including "Content-Type: application/json" and "User-Agent"
// based on the client name.
func Do[T any](ctx context.Context, client *Client, req *Request) (*Response[T], error) {
	path, ok := client.paths[req.PathName]
	if !ok {
		return nil, fmt.Errorf("path %q not found", req.PathName)
	}

	if req.Header == nil {
		req.Header = make(http.Header)
	}
	if req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("User-Agent", fmt.Sprintf("%s/%s", client.name, client.version))

	result := new(T)
	request := client.
		R().
		SetContext(ctx).
		SetAuthScheme(req.AuthScheme).
		SetAuthToken(req.Token).
		SetHeaderMultiValues(req.Header).
		SetBody(req.Body).
		SetQueryParams(req.QueryParams).
		SetPathParams(req.PathParams).
		SetResult(result)
	if req.UserInfo != nil {
		request.SetBasicAuth(req.UserInfo.Username, req.UserInfo.Password)
	}

	res, err := request.Execute(req.Method, path)
	if err != nil {
		return nil, fmt.Errorf("error executing request: %w", err)
	}

	return &Response[T]{Result: *result, Response: *res.RawResponse}, nil
}
