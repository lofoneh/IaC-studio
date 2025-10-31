package middleware

import (
    "net/http"
    "runtime/debug"
    "github.com/iac-studio/engine/pkg/logger"
    "go.uber.org/zap"
)

// Recovery logs panics and returns 500 with a generic message.
func Recovery(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        defer func() {
            if rec := recover(); rec != nil {
                logger.L().Error("panic recovered", zap.Any("panic", rec), zap.ByteString("stack", debug.Stack()))
                http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
            }
        }()
        next.ServeHTTP(w, r)
    })
}


