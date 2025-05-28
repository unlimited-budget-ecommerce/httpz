package httpz

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

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

	restyClient := resty.NewWithClient(&http.Client{
		Transport: cfg.transport,
	})
	restyClient.
		SetBaseURL(baseURL).
		SetHeaders(cfg.baseHeaders).
		SetResponseBodyUnlimitedReads(true). // TODO: handle large body
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

type request struct {
	// pathName is the name registered with [httpz.WithPaths]
	pathName    string
	queryParams map[string]string
	basicAuth   *basicAuth
	r           resty.Request
}

type basicAuth struct {
	user, pass string
}

// NewRequest returns [*request] given pathName, and method.
func NewRequest(pathName, method string) *request {
	return &request{
		pathName: pathName,
		r:        resty.Request{Method: method},
	}
}

func (r *request) WithHeader(header http.Header) *request {
	if header != nil {
		r.r.Header = header
	}
	return r
}

func (r *request) WithBody(body any) *request {
	if body != nil {
		r.r.Body = body
	}
	return r
}

func (r *request) WithAuthScheme(scheme string) *request {
	r.r.AuthScheme = scheme
	return r
}

func (r *request) WithAuthToken(token string) *request {
	r.r.AuthToken = token
	return r
}

func (r *request) WithBasicAuth(user, pass string) *request {
	r.basicAuth = &basicAuth{user: user, pass: pass}
	return r
}

func (r *request) WithQueryParams(params map[string]string) *request {
	if params != nil {
		r.queryParams = params
	}
	return r
}

func (r *request) WithPathParams(params map[string]string) *request {
	if params != nil {
		r.r.PathParams = params
	}
	return r
}

type Response[T any] struct {
	Result T
	http.Response
}

// Do executes an HTTP request and returns a typed response *T and [*resty.Response].
//
// It looks up the request path by name from the client's registered paths.
// It also sets default headers including "Content-Type: application/json" and "User-Agent"
// based on the client name.
func Do[T any](ctx context.Context, client *Client, req *request) (*Response[T], error) {
	path, ok := client.paths[req.pathName]
	if !ok {
		return nil, fmt.Errorf("path %q not found", req.pathName)
	}

	if req.r.Header == nil {
		req.r.Header = make(http.Header)
	}
	if req.r.Header.Get("Content-Type") == "" {
		req.r.Header.Set("Content-Type", "application/json")
	}
	req.r.Header.Set("User-Agent", fmt.Sprintf("%s/%s", client.name, client.version))

	result := new(T)
	request := client.
		R().
		SetContext(ctx).
		SetAuthScheme(req.r.AuthScheme).
		SetAuthToken(req.r.AuthToken).
		SetHeaderMultiValues(req.r.Header).
		SetBody(req.r.Body).
		SetQueryParams(req.queryParams).
		SetPathParams(req.r.PathParams).
		SetResult(result)

	if req.basicAuth != nil {
		request.SetBasicAuth(req.basicAuth.user, req.basicAuth.pass)
	}

	res, err := request.Execute(req.r.Method, path)
	if err != nil {
		return nil, fmt.Errorf("error executing request: %w", err)
	}

	return &Response[T]{Result: *result, Response: *res.RawResponse}, nil
}
