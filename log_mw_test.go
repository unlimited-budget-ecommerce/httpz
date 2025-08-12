package httpz

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"testing"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
			SetQueryParam("test-query", "query-val").
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
		assert.Contains(t, logs, `"url.full":"`+server.URL+`/test/log?test-query=query-val"`)
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

func TestConcurrentLogMiddleware(t *testing.T) {
	type testLogReq struct {
		Input1 string `json:"input1"`
		Input2 string `json:"input2"`
		Input3 string `json:"input3"`
	}
	type testLogRes struct {
		Output1 string `json:"output1"`
		Output2 string `json:"output2"`
		Output3 string `json:"output3"`
	}
	wantReqBody := testLogReq{Input1: "foo", Input2: "bar", Input3: "baz"}
	wantResBody := testLogRes{Output2: "out1", Output1: "out2", Output3: "out3"}
	server := startTestServer(t, testHandler{
		method: http.MethodPost,
		path:   "/test/log/concurrent",
		handlerFunc: func(w http.ResponseWriter, r *http.Request) {
			var reqBody testLogReq

			err := json.NewDecoder(r.Body).Decode(&reqBody)
			if err != nil {
				http.Error(w, "error decoding request body", http.StatusBadRequest)
				return
			}
			assert.ObjectsAreEqual(wantReqBody, reqBody)

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)

			err = json.NewEncoder(w).Encode(wantResBody)
			if err != nil {
				http.Error(w, "error encoding response body", http.StatusInternalServerError)
				return
			}
		},
	})
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	client := NewClient("", server.URL,
		WithPaths(map[string]string{"testConcurrentLog": "/test/log/concurrent"}),
		WithLogger(logger),
		WithLogMWEnabled(true),
	)

	// ensure no unknown panic occurred
	defer func() {
		require.Nil(t, recover())
	}()

	var wg sync.WaitGroup
	defer wg.Wait()
	for range 100 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			result := &testLogRes{}

			res, err := client.NewRequest(context.Background()).
				SetBody(wantReqBody).
				SetResult(result).
				Post(client.GetPath("testConcurrentLog"))

			require.NoError(t, err)
			assert.Equal(t, http.StatusOK, res.StatusCode())
			assert.Equal(t, "application/json", res.Header().Get("Content-Type"))
			assert.Equal(t, &wantResBody, res.Result())
		}()
	}
}
