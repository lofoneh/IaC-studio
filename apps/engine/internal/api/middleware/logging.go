package middleware

import (
    "net/http"
    "time"
    "github.com/iac-studio/engine/pkg/logger"
    "go.uber.org/zap"
)

// Logging logs basic request information with request ID.
func Logging(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()
        rw := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
        next.ServeHTTP(rw, r)
        logger.L().Info("request",
            zap.String("id", GetRequestID(r.Context())),
            zap.String("method", r.Method),
            zap.String("path", r.URL.Path),
            zap.Int("status", rw.status),
            zap.Duration("duration", time.Since(start)),
            zap.String("remote", r.RemoteAddr),
        )
    })
}

type statusRecorder struct {
    http.ResponseWriter
    status int
}
func (s *statusRecorder) WriteHeader(code int) { s.status = code; s.ResponseWriter.WriteHeader(code) }


