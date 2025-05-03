package httpz

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/go-resty/resty/v2"
)

type httpClient struct {
	resty.Client
	name  string
	paths map[string]path
}

func New(clientName, baseURL string, opts ...option) *httpClient {
	cfg := config{}
	for _, opt := range opts {
		opt(&cfg)
	}
	if cfg.transport == nil {
		cfg.transport = http.DefaultTransport
	}
	if cfg.paths == nil {
		cfg.paths = make(map[string]path)
	}
	if cfg.logger == nil {
		cfg.logger = slog.Default()
	}

	client := resty.NewWithClient(&http.Client{
		Transport: cfg.transport,
	})
	client.BaseURL = baseURL
	client.
		SetLogger(logger{cfg.logger}).
		OnBeforeRequest(nil). // TODO: otel req
		OnBeforeRequest(logRequest(&cfg)).
		OnAfterResponse(logResponse(&cfg)).
		OnAfterResponse(nil). // TODO: otel res
		OnError(nil)          // TODO: otel err

	return &httpClient{
		Client: *client,
		name:   clientName,
		paths:  cfg.paths,
	}
}

type (
	Request struct {
		PathName    string
		QueryParams map[string]string
		resty.Request
	}
	Response[T any] struct {
		Result *T
		*resty.Response
	}
)

func Do[T any](ctx context.Context, client *httpClient, req *Request) (Response[T], error) {
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
	request.Header.Set("User-Agent", client.name)

	res, err := request.Execute(req.Method, client.paths[req.PathName].path)
	if err != nil {
		return Response[T]{}, fmt.Errorf("error executing request: %w", err)
	}

	return Response[T]{Result: result, Response: res}, nil
}
