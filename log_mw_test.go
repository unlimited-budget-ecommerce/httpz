package httpz

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"testing"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
)

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
	clientWithLog := NewClient("test-client", server.URL,
		WithPaths(map[string]string{"testLog": "/test/log"}),
		WithLogger(logger),
		WithLogMWEnabled(true),
	)
	clientWithoutLog := NewClient("test-client", server.URL,
		WithPaths(map[string]string{"testLog": "/test/log"}),
		WithLogger(logger),
		WithLogMWEnabled(false),
	)

	t.Run("logging enabled success msg", func(t *testing.T) {
		result := &testLogRes{}

		res, err := clientWithLog.NewRequest(context.Background()).
			SetHeader("X-Test-Req", "req-header-val").
			SetBody(wantReqBody).
			SetResult(result).
			Post(clientWithLog.GetPath("testLog"))

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, res.StatusCode())
		assert.Equal(t, "application/json", res.Header().Get("Content-Type"))
		assert.Equal(t, &wantResBody, res.Result())

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
		result := &testLogRes{}

		res, err := clientWithoutLog.NewRequest(context.Background()).
			SetBody(wantReqBody).
			SetResult(result).
			Post(clientWithoutLog.GetPath("testLog"))

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, res.StatusCode())
		assert.Equal(t, "application/json", res.Header().Get("Content-Type"))
		assert.Equal(t, &wantResBody, res.Result())

		logs := b.String()

		assert.Empty(t, logs)
	})

	// TODO: Add test cases for logging error request, response
}
