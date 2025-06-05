package httpz

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
	"resty.dev/v3"
)

type testHandler struct {
	method      string
	path        string
	handlerFunc http.HandlerFunc
}

func startTestServer(t *testing.T, handlers ...testHandler) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	for _, h := range handlers {
		mux.HandleFunc(h.path, func(w http.ResponseWriter, r *http.Request) {
			h.handlerFunc(w, r)
		})
	}
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return server
}

func TestGetRequest(t *testing.T) {
	type testGetRes struct {
		Code int    `json:"code"`
		Desc string `json:"desc"`
	}
	wantRes := testGetRes{Code: 123, Desc: "Hello"}
	server := startTestServer(t, testHandler{
		method: http.MethodGet,
		path:   "/test/get/{id}",
		handlerFunc: func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "test-client/", r.UserAgent())
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
			assert.Equal(t, "1", r.PathValue("id"))
			assert.Equal(t, "bar", r.URL.Query().Get("foo"))

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)

			err := json.NewEncoder(w).Encode(wantRes)

			assert.NoError(t, err)
		},
	})
	client := NewClient("test-client", server.URL, WithPaths(map[string]string{
		"testGet": "/test/get/{id}",
	}))
	result := &testGetRes{}

	res, err := client.NewRequest(context.Background()).
		SetPathParams(map[string]string{"id": "1"}).
		SetQueryParams(map[string]string{"foo": "bar"}).
		SetResult(result).
		Get(client.GetPath("testGet"))

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, res.StatusCode())
	assert.Equal(t, "application/json", res.Header().Get("Content-Type"))
	assert.Equal(t, &wantRes, res.Result())
}

func TestPostRequest(t *testing.T) {
	type testPostReq struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}
	type testPostRes struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	}
	wantReq := testPostReq{Name: "Alice", Age: 30}
	wantRes := testPostRes{ID: "abc-123", Status: "created"}
	server := startTestServer(t, testHandler{
		method: http.MethodPost,
		path:   "/test/post",
		handlerFunc: func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "test-client/", r.UserAgent())
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
			assert.Equal(t, "test-header-val", r.Header.Get("x-test-header"))

			body, err := io.ReadAll(r.Body)
			defer r.Body.Close()

			assert.NoError(t, err)

			var req testPostReq

			err = json.Unmarshal(body, &req)

			assert.NoError(t, err)
			assert.Equal(t, wantReq, req)

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)

			err = json.NewEncoder(w).Encode(wantRes)

			assert.NoError(t, err)
		},
	})
	client := NewClient("test-client", server.URL,
		WithBaseHeaders(map[string]string{
			"x-test-header": "test-header-val",
		}),
		WithPaths(map[string]string{
			"testPost": "/test/post",
		}),
	)
	result := &testPostRes{}

	res, err := client.NewRequest(context.Background()).
		SetBody(wantReq).
		SetResult(result).
		Post(client.GetPath("testPost"))

	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, res.StatusCode())
	assert.Equal(t, "application/json", res.Header().Get("Content-Type"))
	assert.Equal(t, &wantRes, res.Result())
}

func TestGetNonExistPath(t *testing.T) {
	server := startTestServer(t, testHandler{
		method: http.MethodGet,
		path:   "/foo",
		handlerFunc: func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		},
	})
	client := NewClient("test-client", server.URL)

	res, err := client.NewRequest(context.Background()).
		Get(client.GetPath("nonExistPath"))

	assert.NoError(t, err)
	assert.NotNil(t, res)
	assert.Equal(t, http.StatusNotFound, res.StatusCode())
}

func TestSetClientAndRequestHeaders(t *testing.T) {
	type testGetRes struct {
		Code int `json:"code"`
	}
	wantRes := testGetRes{Code: 123}
	server := startTestServer(t, testHandler{
		method: http.MethodGet,
		path:   "/test/get",
		handlerFunc: func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "test-client/", r.UserAgent())
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
			assert.Equal(t, "test-header-val1", r.Header.Get("x-test-header1"))
			assert.Equal(t, "test-header-val2", r.Header.Get("x-test-header2"))
			assert.Equal(t, "new-test-header-val", r.Header.Get("x-test-header3"))
			assert.Equal(t, "test-header-val4", r.Header.Get("x-test-header4"))
			assert.Equal(t, "test-header-val5", r.Header.Get("x-test-header5"))

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)

			err := json.NewEncoder(w).Encode(wantRes)

			assert.NoError(t, err)
		},
	})
	client := NewClient("test-client", server.URL,
		WithPaths(map[string]string{"testGet": "/test/get"}),
		WithBaseHeaders(map[string]string{
			"x-test-header1": "test-header-val1",
			"x-test-header2": "test-header-val2",
			"x-test-header3": "test-header-val3",
		}),
	)
	result := &testGetRes{}

	res, err := client.NewRequest(context.Background()).
		SetHeaders(map[string]string{
			"x-test-header3": "new-test-header-val",
			"x-test-header4": "test-header-val4",
			"x-test-header5": "test-header-val5",
		}).
		SetResult(result).
		Get(client.GetPath("testGet"))

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, res.StatusCode())
	assert.Equal(t, "application/json", res.Header().Get("Content-Type"))
	assert.Equal(t, &wantRes, res.Result())
}

func TestBasicAuthRequest(t *testing.T) {
	type testAuthRes struct {
		Authenticated bool `json:"authenticated"`
	}
	wantRes := testAuthRes{Authenticated: true}
	wantUser := "testuser"
	wantPass := "testpass"
	server := startTestServer(t, testHandler{
		method: http.MethodPost,
		path:   "/test/auth/basic",
		handlerFunc: func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "test-client/", r.UserAgent())
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
			assert.True(t, strings.HasPrefix(r.Header.Get("Authorization"), "Basic "))

			user, pass, ok := r.BasicAuth()

			assert.True(t, ok)
			assert.Equal(t, wantUser, user)
			assert.Equal(t, wantPass, pass)

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)

			err := json.NewEncoder(w).Encode(wantRes)

			assert.NoError(t, err)
		},
	})
	client := NewClient("test-client", server.URL, WithPaths(map[string]string{
		"testBasicAuth": "/test/auth/basic",
	}))
	result := &testAuthRes{}

	res, err := client.NewRequest(context.Background()).
		SetBasicAuth(wantUser, wantPass).
		SetResult(result).
		Post(client.GetPath("testBasicAuth"))

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, res.StatusCode())
	assert.Equal(t, "application/json", res.Header().Get("Content-Type"))
	assert.Equal(t, &wantRes, res.Result())
}

func TestBearerTokenRequest(t *testing.T) {
	type testAuthRes struct {
		Authenticated bool `json:"authenticated"`
	}
	wantRes := testAuthRes{Authenticated: true}
	wantToken := "test-token"
	server := startTestServer(t, testHandler{
		method: http.MethodPost,
		path:   "/test/auth/bearer",
		handlerFunc: func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "test-client/", r.UserAgent())
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
			assert.Equal(t, "Bearer "+wantToken, r.Header.Get("Authorization"))

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)

			err := json.NewEncoder(w).Encode(wantRes)

			assert.NoError(t, err)
		},
	})
	client := NewClient("test-client", server.URL, WithPaths(map[string]string{
		"testBearerAuth": "/test/auth/bearer",
	}))
	result := &testAuthRes{}

	res, err := client.NewRequest(context.Background()).
		SetAuthScheme("Bearer").
		SetAuthToken(wantToken).
		SetResult(result).
		Post(client.GetPath("testBearerAuth"))

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, res.StatusCode())
	assert.Equal(t, "application/json", res.Header().Get("Content-Type"))
	assert.Equal(t, &wantRes, res.Result())
}

func TestRequestWithRetry(t *testing.T) {
	type testRetryRes struct {
		Message string `json:"message"`
	}
	wantResBody := testRetryRes{Message: "success"}
	attempts := 0
	maxAttempts := 3 // Succeed on the 3rd attempt
	server := startTestServer(t, testHandler{
		method: http.MethodPost,
		path:   "/test/retry",
		handlerFunc: func(w http.ResponseWriter, r *http.Request) {
			attempts++
			if attempts < maxAttempts {
				w.WriteHeader(http.StatusServiceUnavailable)
				_, _ = w.Write([]byte("service unavailable"))
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			err := json.NewEncoder(w).Encode(wantResBody)
			assert.NoError(t, err)
		},
	})
	client := NewClient("test-retry-client", server.URL,
		WithPaths(map[string]string{"testRetry": "/test/retry"}),
	)
	client.SetAllowNonIdempotentRetry(true)
	// 1 initial attempt + 2 retries = 3 total attempts
	client.SetRetryCount(maxAttempts - 1)
	client.SetRetryWaitTime(1 * time.Millisecond)
	client.SetRetryMaxWaitTime(1 * time.Millisecond)
	result := &testRetryRes{}

	res, err := client.NewRequest(context.Background()).
		SetResult(result).
		Post(client.GetPath("testRetry"))

	assert.NoError(t, err)
	if err == nil {
		assert.Equal(t, http.StatusOK, res.StatusCode())
	}
	assert.Equal(t, maxAttempts, attempts)
}

func TestClientCircuitBreaker(t *testing.T) {
	server := startTestServer(t,
		testHandler{
			method: http.MethodGet,
			path:   "/200",
			handlerFunc: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"status":"ok"}`))
			},
		},
		testHandler{
			method: http.MethodGet,
			path:   "/500",
			handlerFunc: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"status":"error"}`))
			},
		},
	)
	failureThreshold := uint32(3)
	successThreshold := uint32(1)
	cbTimeout := 100 * time.Millisecond
	client := NewClient("test-circuit-breaker", server.URL,
		WithPaths(map[string]string{
			"success": "/200",
			"fail":    "/500",
		}),
		WithCircuitBreaker(cbTimeout, failureThreshold, successThreshold, nil),
		WithCircuitBreakerEnabled(true),
	)
	req := client.NewRequest(context.Background())

	for range failureThreshold {
		res, err := req.Get(client.GetPath("fail"))

		assert.NoError(t, err)
		assert.Equal(t, http.StatusInternalServerError, res.StatusCode())
		assert.NotNil(t, res)
	}
	t.Log("Circuit Breaker Open")

	res, err := req.Get(client.GetPath("success"))

	assert.ErrorIs(t, err, resty.ErrCircuitBreakerOpen)
	assert.Nil(t, res)

	time.Sleep(cbTimeout + 50*time.Millisecond)
	t.Log("Circuit Breaker Half-Open")

	res, err = req.Get(client.GetPath("fail"))

	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, res.StatusCode())
	assert.NotNil(t, res)
	t.Log("Circuit Breaker Open")

	res, err = req.Get(client.GetPath("success"))

	assert.ErrorIs(t, err, resty.ErrCircuitBreakerOpen)
	assert.Nil(t, res)

	time.Sleep(cbTimeout + 50*time.Millisecond)
	t.Log("Circuit Breaker Half-Open Again")

	res, err = req.Get(client.GetPath("success"))

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, res.StatusCode())
	assert.NotNil(t, res)
	t.Log("Circuit Breaker Closed")

	res, err = req.Get(client.GetPath("success"))

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, res.StatusCode())
	assert.NotNil(t, res)
}
