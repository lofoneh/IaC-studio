package middleware

import (
    "context"
    "net/http"
    "strings"
    "github.com/golang-jwt/jwt/v5"
)

type userKeyType string

const UserIDKey userKeyType = "user_id"

// Auth validates a Bearer JWT using the provided HMAC secret and adds user id to context.
func Auth(hmacSecret []byte) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            ah := r.Header.Get("Authorization")
            if !strings.HasPrefix(strings.ToLower(ah), "bearer ") {
                http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
                return
            }
            tokenStr := strings.TrimSpace(ah[len("Bearer "):])
            token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
                if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok { return nil, jwt.ErrSignatureInvalid }
                return hmacSecret, nil
            })
            if err != nil || !token.Valid {
                http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
                return
            }
            claims, ok := token.Claims.(jwt.MapClaims)
            if !ok {
                http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
                return
            }
            uid, _ := claims["sub"].(string)
            ctx := context.WithValue(r.Context(), UserIDKey, uid)
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}

func GetUserID(ctx context.Context) string {
    if v := ctx.Value(UserIDKey); v != nil {
        if s, ok := v.(string); ok { return s }
    }
    return ""
}


