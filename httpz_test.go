package httpz

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

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
	client := New("test-client", server.URL, WithPaths(map[string]Path{
		"testGet": {Path: "/test/get/{id}"},
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
	assert.Equal(t, http.StatusOK, res.StatusCode())
	assert.Equal(t, "application/json", res.Header().Get("Content-Type"))
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
	client := New("test-client", server.URL, WithPaths(map[string]Path{
		"testPost": {Path: "/test/post"},
	}))
	req := Request{PathName: "testPost"}
	req.Method = http.MethodPost
	req.Body = wantReq

	res, err := Do[testPostRes](context.Background(), client, &req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, res.StatusCode())
	assert.Equal(t, "application/json", res.Header().Get("Content-Type"))
	assert.Equal(t, wantRes, res.Result)
}

func TestDoPathNotFound(t *testing.T) {
	client := New("", "")
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
	client := New("test-client", server.URL, WithPaths(map[string]Path{
		"testBasicAuth": {Path: "/test/auth/basic"},
	}))
	req := Request{PathName: "testBasicAuth"}
	req.Method = http.MethodPost
	req.SetBasicAuth(wantUser, wantPass)

	res, err := Do[testAuthRes](context.Background(), client, &req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, res.StatusCode())
	assert.Equal(t, "application/json", res.Header().Get("Content-Type"))
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
	client := New("test-client", server.URL, WithPaths(map[string]Path{
		"testBearerAuth": {Path: "/test/auth/bearer"},
	}))
	req := Request{PathName: "testBearerAuth"}
	req.Method = http.MethodPost
	req.SetAuthScheme("Bearer").SetAuthToken(wantToken)

	res, err := Do[testAuthRes](context.Background(), client, &req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, res.StatusCode())
	assert.Equal(t, "application/json", res.Header().Get("Content-Type"))
	assert.Equal(t, wantRes, res.Result)
}

func TestLogMiddleware(t *testing.T) {
	type testLogReq struct {
		Input string `json:"input"`
	}
	type testLogRes struct {
		Output string `json:"output"`
	}
	wantReqBody := testLogReq{Input: "ping"}
	wantResBody := testLogRes{Output: "pong"}
	server := startTestServer(t, testHandler{
		method: http.MethodPost,
		path:   "/test/log",
		handlerFunc: func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "test-client/", r.UserAgent())
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

			var reqBody testLogReq

			err := json.NewDecoder(r.Body).Decode(&reqBody)

			assert.NoError(t, err)
			assert.Equal(t, wantReqBody, reqBody)

			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("X-Test-Resp", "resp-header-val")
			w.WriteHeader(http.StatusOK)

			err = json.NewEncoder(w).Encode(wantResBody)

			assert.NoError(t, err)
		},
	})
	b := &bytes.Buffer{}
	logger := slog.New(slog.NewJSONHandler(b, nil))
	clientWithLog := New("test-client", server.URL,
		WithPaths(map[string]Path{"testLog": {Path: "/test/log"}}),
		WithLogger(logger),
		WithLogMWEnabled(true),
	)
	clientWithoutLog := New("test-client", server.URL,
		WithPaths(map[string]Path{"testLog": {Path: "/test/log"}}),
		WithLogger(logger),
		WithLogMWEnabled(false),
	)

	t.Run("logging enabled success msg", func(t *testing.T) {
		req := Request{PathName: "testLog"}
		req.Method = http.MethodPost
		req.Header = http.Header{"X-Test-Req": []string{"req-header-val"}}
		req.Body = wantReqBody

		res, err := Do[testLogRes](context.Background(), clientWithLog, &req)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, res.StatusCode())
		assert.Equal(t, "application/json", res.Header().Get("Content-Type"))
		assert.Equal(t, wantResBody, res.Result)

		logs := b.String()
		t.Log("captured logs:\n", logs)

		// request log
		assert.Contains(t, logs, "[HTTPZ][OUTGOING REQUEST]")
		assert.Contains(t, logs, `"url.full":"`+server.URL+`/test/log"`)
		assert.Contains(t, logs, `"http.request.method":"POST"`)
		assert.Contains(t, logs, `"X-Test-Req":["req-header-val"]`)
		assert.Contains(t, logs, `"http.request.body":{"input":"ping"}`)

		// response log
		assert.Contains(t, logs, "[HTTPZ][INCOMING RESPONSE] success")
		assert.Contains(t, logs, `"http.client.request.duration":`)
		assert.Contains(t, logs, `"http.response.status_code":200`)
		assert.Contains(t, logs, `"X-Test-Resp":["resp-header-val"]`)
		assert.Contains(t, logs, `"http.response.body":{"output":"pong"}`)
	})

	t.Run("logging disabled", func(t *testing.T) {
		b.Reset()

		req := Request{PathName: "testLog"}
		req.Method = http.MethodPost
		req.Body = wantReqBody

		res, err := Do[testLogRes](context.Background(), clientWithoutLog, &req)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, res.StatusCode())
		assert.Equal(t, "application/json", res.Header().Get("Content-Type"))
		assert.Equal(t, wantResBody, res.Result)

		logs := b.String()

		assert.Empty(t, logs)
	})

	// TODO: Add test cases for logging error request, response
}
