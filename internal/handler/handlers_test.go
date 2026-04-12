package handlers

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	config "github.com/andrea20024/go-musthave-shortener-tpl/internal/config"
	storage "github.com/andrea20024/go-musthave-shortener-tpl/internal/repository"
)

func Test_GetHandler(t *testing.T) {
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
			shortURL, err := GenerateShortURL()
			if err != nil {
				return
			}

			storage.Add(shortURL, tt.ref)
			req := httptest.NewRequest(http.MethodGet, "/"+shortURL, nil)
			w := httptest.NewRecorder()

			GetHandler(w, req)
			res := w.Result()
			defer res.Body.Close()

			assert.Equal(t, tt.params.code, res.StatusCode)
			assert.Equal(t, tt.ref, res.Header.Get("location"))
		})
	}
}

func TestPostHandler(t *testing.T) {
	type params struct {
		code        int
		contentType string
	}
	tests := []struct {
		name   string
		body   string
		config config.Config
		params params
	}{
		{
			name:   "test post",
			body:   "https://practicum.yandex.ru",
			config: *config.InitConfig(),
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

			PostHandler(w, req, &tt.config)
			res := w.Result()
			defer res.Body.Close()

			assert.Equal(t, tt.params.code, res.StatusCode)
			assert.Equal(t, tt.params.contentType, res.Header.Get("Content-Type"))
		})
	}
}
