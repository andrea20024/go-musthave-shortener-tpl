package compress

import (
	"compress/gzip"
	"net/http"
	"strings"
	"sync"
)

type gzipResponseWriter struct {
	http.ResponseWriter
	gzipWriter *gzip.Writer
}

func (w *gzipResponseWriter) Write(b []byte) (int, error) {
	return w.gzipWriter.Write(b)
}

func (w *gzipResponseWriter) Close() error {
	err := w.gzipWriter.Close()
	gzipPool.Put(w.gzipWriter)
	w.gzipWriter = nil
	return err
}

var gzipPool = sync.Pool{
	New: func() interface{} {
		return new(gzip.Writer)
	},
}

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
