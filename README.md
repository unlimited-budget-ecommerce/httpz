# HTTPZ - HTTP client

`httpz` simplify the process of making HTTP requests. It a wrapper around `resty.dev/v3`, with built-in request/response logging and opentelemetry tracing middleware. It also support retry mechanism and circuit breaker pattern.

## Features

- Configurable
- Typed response result
- Structured logging middleware
- OpenTelemetry tracing middleware
- Retry mechanism
- Circuit breaker

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
	// read function doc for more details
	httpz.WithCircuitBreaker(0, 0, 0, nil), // passing zero values will result to default values: 10s, 3, 1, Status Code 500 and above
	httpz.WithCircuitBreakerEnabled(true),  // default: false
)
```

### Making a POST request

```go
// expected result struct when making a request
result := &CreateUserRes{}

res, err := client.NewRequest(context.Background()).
	SetBody(&CreateUserReq{}). // example request body
	SetResult(result).
	Post(client.GetPath("createUser"))
if err != nil {
	return nil, fmt.Errorf("failed to create user: %w", err)
}
if res.IsError() {
	return res, fmt.Errorf("error creating user, got status: %d" ,res.StatusCode())
}
```

### Making a GET request

```go
// expected result struct when making a request
result := &GetUserRes{}

res, err := client.NewRequest(context.Background()).
	SetPathParams(map[string]string{"id": "1"}).
	SetQueryParams(map[string]string{"foo": "bar"}).
	SetResult(result).
	Get(client.GetPath("getUser"))

// handle error
```

### Making a request with retries

You can configure retry attempts, wait times, and conditions for retrying a request. Default retry strategy is exponential backoff with a jitter

To enable and configure retries, you would typically interact with the `Client` or `Request` struct. Reference: https://resty.dev/docs/retry-mechanism/

```go
client :=  httpz.NewClient("", "")
client.
	SetAllowNonIdempotentRetry(true).         // default: false (enable retry for POST request)
	SetRetryCount(1).                         // default: 0 (total attempt = initial attempt + retry count)
	SetRetryWaitTime(100 * time.Millisecond). // default: 100ms
	SetRetryMaxWaitTime(2 * time.Second)      // default: 2s
```
