package middleware

import (
    "net"
    "net/http"
    "sync"
    "time"
    "golang.org/x/time/rate"
)

type limiterEntry struct {
    limiter *rate.Limiter
    last    time.Time
}

var (
    mu       sync.Mutex
    visitors = map[string]*limiterEntry{}
)

func getIP(r *http.Request) string {
    if ip := r.Header.Get("X-Forwarded-For"); ip != "" { return ip }
    host, _, err := net.SplitHostPort(r.RemoteAddr)
    if err != nil { return r.RemoteAddr }
    return host
}

// RateLimit applies a simple IP-based token bucket limiter.
func RateLimit(rps float64, burst int) func(http.Handler) http.Handler {
    gcTicker := time.NewTicker(5 * time.Minute)
    go func(){
        for range gcTicker.C {
            mu.Lock()
            for k, v := range visitors {
                if time.Since(v.last) > 10*time.Minute { delete(visitors, k) }
            }
            mu.Unlock()
        }
    }()
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            ip := getIP(r)
            mu.Lock()
            le, ok := visitors[ip]
            if !ok {
                le = &limiterEntry{limiter: rate.NewLimiter(rate.Limit(rps), burst)}
                visitors[ip] = le
            }
            le.last = time.Now()
            allow := le.limiter.Allow()
            mu.Unlock()
            if !allow {
                http.Error(w, http.StatusText(http.StatusTooManyRequests), http.StatusTooManyRequests)
                return
            }
            next.ServeHTTP(w, r)
        })
    }
}


