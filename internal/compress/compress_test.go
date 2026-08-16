package compress

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGzipResponseWriter_Write(t *testing.T) {
	rec := httptest.NewRecorder()
	gz := gzip.NewWriter(io.Discard)
	grw := &gzipResponseWriter{
		ResponseWriter: rec,
		gzipWriter:     gz,
	}

	n, err := grw.Write([]byte("hello"))
	assert.NoError(t, err)
	assert.Equal(t, 5, n)

	err = grw.Close()
	assert.NoError(t, err)
}

func TestGzipResponseWriter_Close(t *testing.T) {
	rec := httptest.NewRecorder()
	gz := gzip.NewWriter(io.Discard)
	grw := &gzipResponseWriter{
		ResponseWriter: rec,
		gzipWriter:     gz,
	}

	err := grw.Close()
	assert.NoError(t, err)
	assert.Nil(t, grw.gzipWriter, "gzipWriter should be nil after Close")
}

func TestGzipPool(t *testing.T) {
	assert.NotNil(t, gzipPool)

	val := gzipPool.Get()
	assert.NotNil(t, val)

	gz := val.(*gzip.Writer)
	assert.NotNil(t, gz)

	gzipPool.Put(gz)
}

func TestGzipHandle_WithJSONContentType(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"result":"ok"}`))
	})

	middleware := GzipHandle(handler)
	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept-Encoding", "gzip")
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)
	res := w.Result()
	defer res.Body.Close()

	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Equal(t, "gzip", res.Header.Get("Content-Encoding"))
}

func TestGzipHandle_WithHTMLContentType(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("<html><body>test</body></html>"))
	})

	middleware := GzipHandle(handler)
	req := httptest.NewRequest(http.MethodGet, "/page", nil)
	req.Header.Set("Content-Type", "text/html")
	req.Header.Set("Accept-Encoding", "gzip")
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)
	res := w.Result()
	defer res.Body.Close()

	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Equal(t, "gzip", res.Header.Get("Content-Encoding"))
}

func TestGzipHandle_NoGzipAccept(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"result":"ok"}`))
	})

	middleware := GzipHandle(handler)
	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept-Encoding", "")
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)
	res := w.Result()
	defer res.Body.Close()

	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Equal(t, "", res.Header.Get("Content-Encoding"))
	assert.Equal(t, `{"result":"ok"}`, w.Body.String())
}

func TestGzipHandle_NonJSONHTMLContentType(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("plain text"))
	})

	middleware := GzipHandle(handler)
	req := httptest.NewRequest(http.MethodGet, "/text", nil)
	req.Header.Set("Content-Type", "text/plain")
	req.Header.Set("Accept-Encoding", "gzip")
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)
	res := w.Result()
	defer res.Body.Close()

	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Equal(t, "", res.Header.Get("Content-Encoding"))
	assert.Equal(t, "plain text", w.Body.String())
}

func TestGzipHandle_WithContentEncodingGzip(t *testing.T) {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	gz.Write([]byte(`{"url":"https://example.com"}`))
	gz.Close()

	bodyReceived := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if string(body) == `{"url":"https://example.com"}` {
			bodyReceived = true
		}
		w.WriteHeader(http.StatusOK)
	})

	middleware := GzipHandle(handler)
	req := httptest.NewRequest(http.MethodPost, "/api", bytes.NewReader(buf.Bytes()))
	req.Header.Set("Content-Encoding", "gzip")
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)
	res := w.Result()
	defer res.Body.Close()

	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.True(t, bodyReceived, "body should be decompressed before handler")
}

func TestGzipHandle_InvalidGzipBody(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := GzipHandle(handler)
	req := httptest.NewRequest(http.MethodPost, "/api", bytes.NewReader([]byte("not-gzip-data")))
	req.Header.Set("Content-Encoding", "gzip")
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)
	res := w.Result()
	defer res.Body.Close()

	assert.Equal(t, http.StatusInternalServerError, res.StatusCode)
}

func TestGzipHandle_MultipleRequests(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"ok":true}`))
	})

	middleware := GzipHandle(handler)

	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api", nil)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept-Encoding", "gzip")
		w := httptest.NewRecorder()

		middleware.ServeHTTP(w, req)
		res := w.Result()
		defer res.Body.Close()

		assert.Equal(t, http.StatusOK, res.StatusCode)
		assert.Equal(t, "gzip", res.Header.Get("Content-Encoding"))
	}
}

func TestGzipHandle_JSONContentTypeNoAccept(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"data":"value"}`))
	})

	middleware := GzipHandle(handler)
	req := httptest.NewRequest(http.MethodGet, "/json", nil)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)
	res := w.Result()
	defer res.Body.Close()

	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Equal(t, "", res.Header.Get("Content-Encoding"))
}
