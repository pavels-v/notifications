package api

import (
	"net/http"
	"runtime/debug"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

func (s *Server) logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetReqID(r.Context())
		if reqID != "" {
			w.Header().Set(middleware.RequestIDHeader, reqID)
		}

		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		start := time.Now()

		defer func() {
			s.logger.Info("request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", ww.Status(),
				"bytes", ww.BytesWritten(),
				"duration_ms", time.Since(start).Milliseconds(),
				"request_id", reqID,
			)
		}()

		next.ServeHTTP(ww, r)
	})
}

func (s *Server) recoverPanics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			rvr := recover()
			if rvr == nil {
				return
			}

			s.logger.Error("failed to serve request after panic",
				"method", r.Method,
				"path", r.URL.Path,
				"panic", rvr,
				"stack", string(debug.Stack()),
				"request_id", middleware.GetReqID(r.Context()),
			)

			if ww, ok := w.(middleware.WrapResponseWriter); ok && ww.Status() != 0 {
				return
			}

			s.writeError(w, http.StatusInternalServerError, codeInternal, "the request could not be handled")
		}()

		next.ServeHTTP(w, r)
	})
}
