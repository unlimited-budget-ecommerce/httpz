# HTTPZ - HTTP client

`httpz` simplify the process of making HTTP requests. It a wrapper around `resty.dev/v3`, with built-in request/response logging and opentelemetry tracing middleware. It also support retry mechanism.

## Features

- Configurable
- Typed responses using generics
- Structured logging middleware
- OpenTelemetry tracing middleware

## Installation

```sh
go get github.com/unlimited-budget-ecommerce/httpz
```

## Usage

### Initializing httpz

```go
// paths should comes from config.yaml file
// this is just an example.
paths := map[string]string{
	"createUser": "/users",
	"getUser":    "/users/{id}",
}

client := httpz.NewClient(
	"service-name",                         // set to "User-Agent"
	"https://api.example.com",              // base url
	httpz.WithTransport(&http.Transport{}), // default: [http.DefaultTransport]
	httpz.WithBaseHeaders(nil),             // default: nil (type map[string]string)
	httpz.WithPaths(paths),                 // default: map[string]string{}
	httpz.WithLogger(slog.Default()),       // default: [slog.Default]
	httpz.WithLogMWEnabled(true),           // request/response logging, default: false
	httpz.WithTracer(nil),                  // default: [otel.GetTracerProvider]
	httpz.WithPropagator(nil),              // default: [otel.GetTextMapPropagator]
	httpz.WithOtelMWEnabled(true),          // opentelemetry tracing, default: false
	httpz.WithServiceVersion(""),           // set to "User-Agent", default: ""
)
```

### Making a POST request

```go
// prepare request
req := &httpz.Request{
	PathName: "createUser", // matches the key in WithPaths()
}
req.Method = http.MethodPost
req.Body = CreateUserReq{}

// example function
func (a *adapter) CreateUser(ctx context.Context, req *http.Request) (*httpz.Response[CreateUserRes], error) {
	resp, err := httpz.Do[CreateUserRes](ctx, a.client, &req)
	if err != nil {
		return resp, fmt.Errorf("failed to create user: %w", err) // resp is nil
	}

	if resp.IsError() {
		return resp, fmt.Errorf("error creating user, got status: %d" ,resp.StatusCode())
	}

	return resp, nil
}
```

### Making a GET request

```go
req := &httpz.Request{
	PathName: "getUser",            // matches the key in WithPaths()
	QueryParams: map[string]string{ // use this field instead of [resty.Request.QueryParam]
		"foo": "bar",
	},
}
req.Method = http.MethodGet
req.PathParams = map[string]string{
	"id": "1",
}

// the rest should be similar to the above example
```

### Making a request with retries

You can configure retry attempts, wait times, and conditions for retrying a request. Default retry strategy is exponential backoff with a jitter

To enable and configure retries, you would typically interact with the `Client` or `Request` struct. Reference: https://resty.dev/docs/retry-mechanism/

Option 1: client level retries

```go
client :=  httpz.NewClient("", "")
client.
	SetAllowNonIdempotentRetry(true).         // default: false (enable retry for POST request)
	SetRetryCount(1).                         // default: 0 (total attempt = initial attempt + retry count)
	SetRetryWaitTime(100 * time.Millisecond). // default: 100ms
	SetRetryMaxWaitTime(2 * time.Second)      // default: 2s
```

Option 2: request level retries

```go
req := &httpz.Request{
	PathName: "createUser", // matches the key in WithPaths()
}
req.
	SetAllowNonIdempotentRetry(true).
	SetRetryCount(1).
	SetRetryWaitTime(100 * time.Millisecond).
	SetRetryMaxWaitTime(2 * time.Second)
```
