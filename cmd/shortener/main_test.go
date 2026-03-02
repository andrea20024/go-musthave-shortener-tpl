package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_getHandler(t *testing.T) {
	type params struct {
		code        int
		contentType string
	}
	tests := []struct {
		name   string
		ref    string
		params params
	}{
		{
			name: "test get",
			ref:  "https://practicum.yandex.ru",
			params: params{
				code:        http.StatusTemporaryRedirect,
				contentType: "text/plain",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shortUrl, err := generateShortURL()
			if err != nil {
				return
			}

			dict[shortUrl] = tt.ref
			req := httptest.NewRequest(http.MethodGet, "/"+shortUrl, nil)
			w := httptest.NewRecorder()

			getHandler(w, req)
			res := w.Result()
			defer res.Body.Close()

			assert.Equal(t, tt.params.code, res.StatusCode)
			assert.Equal(t, tt.ref, res.Header.Get("location"))
		})
	}
}

func Test_postHandler(t *testing.T) {
	type params struct {
		code        int
		contentType string
	}
	tests := []struct {
		name   string
		body   string
		params params
	}{
		{
			name: "test post",
			body: "https://practicum.yandex.ru",
			params: params{
				code:        http.StatusCreated,
				contentType: "text/plain",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "text/plain")
			w := httptest.NewRecorder()

			postHandler(w, req)
			res := w.Result()
			defer res.Body.Close()

			assert.Equal(t, tt.params.code, res.StatusCode)
			assert.Equal(t, tt.params.contentType, res.Header.Get("Content-Type"))
		})
	}
}
