package middleware

import (
    "context"
    "net/http"
    "github.com/google/uuid"
)

type ctxKey string

const RequestIDKey ctxKey = "request_id"

// RequestID ensures each request has an ID in context and response headers.
func RequestID(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        id := r.Header.Get("X-Request-ID")
        if id == "" {
            id = uuid.NewString()
        }
        w.Header().Set("X-Request-ID", id)
        ctx := context.WithValue(r.Context(), RequestIDKey, id)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}

// GetRequestID returns the request id from context.
func GetRequestID(ctx context.Context) string {
    if v := ctx.Value(RequestIDKey); v != nil {
        if s, ok := v.(string); ok { return s }
    }
    return ""
}


