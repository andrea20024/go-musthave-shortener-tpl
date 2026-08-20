// Package compress provides HTTP middleware for request/response gzip
// decompression and compression.
package compress

import (
	"compress/gzip"
	"net/http"
	"strings"
	"sync"
)

// gzipResponseWriter wraps an http.ResponseWriter and writes to a gzip.Writer.
type gzipResponseWriter struct {
	http.ResponseWriter
	gzipWriter *gzip.Writer
}

// Write writes the data to the gzip.Writer.
// Write writes the provided bytes to the gzip writer.
func (w *gzipResponseWriter) Write(b []byte) (int, error) {
	return w.gzipWriter.Write(b)
}

// Close finalizes the gzip writing and returns the writer to the pool.
func (w *gzipResponseWriter) Close() error {
	err := w.gzipWriter.Close()
	gzipPool.Put(w.gzipWriter)
	w.gzipWriter = nil
	return err
}

// gzipPool is a pool of gzip.Writer instances for reuse.
var gzipPool = sync.Pool{
	New: func() interface{} {
		return new(gzip.Writer)
	},
}

// GzipHandle is an HTTP middleware that:
//
//  1. Decompresses incoming request bodies with Content-Encoding: gzip.
//  2. Compresses response bodies with Accept-Encoding: gzip for
//     application/json and text/html content types.
//
// A pool of gzip.Writer instances is reused to minimize allocations.
func GzipHandle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.Header.Get("Content-Encoding"), "gzip") {
			reader, err := gzip.NewReader(r.Body)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			defer reader.Close()
			r.Body = reader
		}

		contentType := r.Header.Get("Content-Type")
		if contentType != "application/json" && contentType != "text/html" {
			next.ServeHTTP(w, r)
			return
		}

		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}

		gz := gzipPool.Get().(*gzip.Writer)
		gz.Reset(w)

		grw := &gzipResponseWriter{
			ResponseWriter: w,
			gzipWriter:     gz,
		}

		w.Header().Set("Content-Encoding", "gzip")
		next.ServeHTTP(grw, r)
		grw.Close()
	})
}
