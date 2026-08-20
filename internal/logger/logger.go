// Package logger provides a centralized structured logging facility based on
// uber-go/zap and an HTTP middleware for request-level access logging.
package logger

import (
	"net/http"
	"time"

	"go.uber.org/zap"
)

// sugarLogger is the global SugaredLogger instance.
var sugarLogger *zap.SugaredLogger

// InitLogger initializes the global SugaredLogger instance used by the
// application. It should be called once during application startup with a
// configured *zap.Logger.
func InitLogger(logger *zap.Logger) {
	sugarLogger = logger.Sugar()
}

// Sugar returns the global SugaredLogger. Returns nil if InitLogger has not
// been called yet.
func Sugar() *zap.SugaredLogger {
	return sugarLogger
}

// responseData holds the HTTP response status code and body size.
type responseData struct {
	status int
	size   int
}

// loggingResponseWriter wraps an http.ResponseWriter to capture status
// code and body size for logging purposes.
type loggingResponseWriter struct {
	http.ResponseWriter
	responseData *responseData
}

// Write captures the number of bytes written and updates the size in
// the responseData struct.
// Write captures the number of bytes written and updates the size in
// the responseData struct.
func (r *loggingResponseWriter) Write(b []byte) (int, error) {
	size, err := r.ResponseWriter.Write(b)
	r.responseData.size += size
	return size, err
}

// WriteHeader captures the HTTP status code.
func (r *loggingResponseWriter) WriteHeader(statusCode int) {
	r.ResponseWriter.WriteHeader(statusCode)
	r.responseData.status = statusCode
}

// WithLogging is an HTTP middleware that logs each request's method, URI,
// response status code, duration, and response body size using the global
// SugaredLogger.
func WithLogging(h http.Handler) http.Handler {
	logFn := func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		responseData := &responseData{
			status: 0,
			size:   0,
		}
		lw := loggingResponseWriter{
			ResponseWriter: w,
			responseData:   responseData,
		}
		h.ServeHTTP(&lw, r)

		duration := time.Since(start)

		sugarLogger.Infoln(
			"uri", r.RequestURI,
			"method", r.Method,
			"status", responseData.status,
			"duration", duration,
			"size", responseData.size,
		)
	}
	return http.HandlerFunc(logFn)
}
