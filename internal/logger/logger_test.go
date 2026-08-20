package logger

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func TestInitLogger(t *testing.T) {
	logger := zap.NewNop()
	InitLogger(logger)

	sugar := Sugar()
	assert.NotNil(t, sugar, "Sugar() should not be nil after InitLogger")
}

func TestSugar_BeforeInit(t *testing.T) {
	original := sugarLogger
	sugarLogger = nil
	defer func() { sugarLogger = original }()

	sugar := Sugar()
	assert.Nil(t, sugar, "Sugar() should be nil before InitLogger")
}

func TestSugar_AfterInit(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	InitLogger(logger)

	sugar := Sugar()
	assert.NotNil(t, sugar)

	sugar.Infoln("test message")
}

func TestLoggingResponseWriter_Write(t *testing.T) {
	rec := httptest.NewRecorder()
	data := &responseData{status: 0, size: 0}
	lw := loggingResponseWriter{
		ResponseWriter: rec,
		responseData:   data,
	}

	n, err := lw.Write([]byte("hello"))
	assert.NoError(t, err)
	assert.Equal(t, 5, n)
	assert.Equal(t, 5, data.size)
}

func TestLoggingResponseWriter_WriteHeader(t *testing.T) {
	rec := httptest.NewRecorder()
	data := &responseData{status: 0, size: 0}
	lw := loggingResponseWriter{
		ResponseWriter: rec,
		responseData:   data,
	}

	lw.WriteHeader(http.StatusCreated)
	assert.Equal(t, http.StatusCreated, data.status)
	assert.Equal(t, http.StatusCreated, rec.Code)
}

func TestWithLogging(t *testing.T) {
	core, observed := observer.New(zap.InfoLevel)
	logger := zap.New(core)
	InitLogger(logger)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	middleware := WithLogging(handler)
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)
	res := w.Result()
	defer res.Body.Close()

	assert.Equal(t, http.StatusOK, res.StatusCode)

	events := observed.All()
	assert.NotEmpty(t, events, "no log events recorded")
}

func TestWithLogging_DifferentMethods(t *testing.T) {
	methods := []string{
		http.MethodGet,
		http.MethodPost,
		http.MethodPut,
		http.MethodDelete,
		http.MethodPatch,
	}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			core, observed := observer.New(zap.InfoLevel)
			logger := zap.New(core)
			InitLogger(logger)

			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusCreated)
			})

			middleware := WithLogging(handler)
			req := httptest.NewRequest(method, "/path", nil)
			w := httptest.NewRecorder()

			middleware.ServeHTTP(w, req)

			events := observed.All()
			assert.NotEmpty(t, events, "no log events for %s", method)
		})
	}
}

func TestWithLogging_Errors(t *testing.T) {
	core, observed := observer.New(zap.InfoLevel)
	logger := zap.New(core)
	InitLogger(logger)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	middleware := WithLogging(handler)
	req := httptest.NewRequest(http.MethodGet, "/error", nil)
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)
	res := w.Result()
	defer res.Body.Close()

	assert.Equal(t, http.StatusInternalServerError, res.StatusCode)

	events := observed.All()
	assert.NotEmpty(t, events)
}

func TestResponseData_Struct(t *testing.T) {
	data := &responseData{
		status: 200,
		size:   42,
	}
	assert.Equal(t, 200, data.status)
	assert.Equal(t, 42, data.size)
}

func TestWithLogging_EmptyPath(t *testing.T) {
	core, observed := observer.New(zap.InfoLevel)
	logger := zap.New(core)
	InitLogger(logger)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := WithLogging(handler)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	events := observed.All()
	assert.NotEmpty(t, events)
}

func TestWithLogging_WithBody(t *testing.T) {
	core, observed := observer.New(zap.InfoLevel)
	logger := zap.New(core)
	InitLogger(logger)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("response body content"))
	})

	middleware := WithLogging(handler)
	req := httptest.NewRequest(http.MethodPost, "/data", os.Stdin)
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	events := observed.All()
	assert.NotEmpty(t, events)
}
