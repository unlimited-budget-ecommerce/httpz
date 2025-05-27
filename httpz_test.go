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

func TestDoGetRequest(t *testing.T) {
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
	req := Request{
		PathName: "testGet",
		QueryParams: map[string]string{
			"foo": "bar",
		},
	}
	req.Method = http.MethodGet
	req.PathParams = map[string]string{
		"id": "1",
	}

	res, err := Do[testGetRes](context.Background(), client, &req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Equal(t, "application/json", res.Header.Get("Content-Type"))
	assert.Equal(t, wantRes, res.Result)
}

func TestDoPostRequest(t *testing.T) {
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
	req := Request{PathName: "testPost"}
	req.Method = http.MethodPost
	req.Body = wantReq

	res, err := Do[testPostRes](context.Background(), client, &req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, res.StatusCode)
	assert.Equal(t, "application/json", res.Header.Get("Content-Type"))
	assert.Equal(t, wantRes, res.Result)
}

func TestDoPathNotFound(t *testing.T) {
	client := NewClient("", "")
	req := Request{PathName: "notExistPath"}

	_, err := Do[any](req.Context(), client, &req)

	assert.Error(t, err)
	assert.Equal(t, err.Error(), `path "notExistPath" not found`)
}

func TestDoBasicAuthRequest(t *testing.T) {
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
	req := Request{PathName: "testBasicAuth"}
	req.Method = http.MethodPost
	req.SetBasicAuth(wantUser, wantPass)

	res, err := Do[testAuthRes](context.Background(), client, &req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Equal(t, "application/json", res.Header.Get("Content-Type"))
	assert.Equal(t, wantRes, res.Result)
}

func TestDoBearerTokenRequest(t *testing.T) {
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
	req := Request{PathName: "testBearerAuth"}
	req.Method = http.MethodPost
	req.SetAuthScheme("Bearer").SetAuthToken(wantToken)

	res, err := Do[testAuthRes](context.Background(), client, &req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Equal(t, "application/json", res.Header.Get("Content-Type"))
	assert.Equal(t, wantRes, res.Result)
}

func TestDoRequestWithRetry(t *testing.T) {
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
	req := Request{PathName: "testRetry"}
	req.Method = http.MethodPost

	res, err := Do[testRetryRes](context.Background(), client, &req)

	assert.NoError(t, err)
	if err == nil {
		assert.Equal(t, http.StatusOK, res.StatusCode)
	}
	assert.Equal(t, maxAttempts, attempts)
}
