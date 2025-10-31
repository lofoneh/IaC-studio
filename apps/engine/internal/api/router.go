package api

import (
    "net/http"
    "strconv"
    "github.com/go-chi/chi/v5"
    chimid "github.com/go-chi/chi/v5/middleware"
    "github.com/iac-studio/engine/internal/api/handlers"
    mw "github.com/iac-studio/engine/internal/api/middleware"
)

type Dependencies struct {
    HMACSecret []byte
}

// NewRouter constructs the application router with all middleware and routes.
func NewRouter(dep Dependencies) http.Handler {
    r := chi.NewRouter()

    // built-in middleware
    r.Use(mw.RequestID)
    r.Use(mw.Recovery)
    r.Use(mw.Logging)
    r.Use(mw.CORS)
    r.Use(mw.RateLimit(10, 20))
    r.Use(chimid.Compress(5))

    // health endpoints
    hh := handlers.NewHealthHandler()
    r.Get("/healthz", hh.Liveness)
    r.Get("/readyz", hh.Readiness)

    r.Route("/api/v1", func(api chi.Router) {
        // Auth routes
        api.Route("/auth", func(ar chi.Router) {
            // handlers to be wired when implemented
            ar.Get("/ping", func(w http.ResponseWriter, r *http.Request){ w.WriteHeader(200); w.Write([]byte("ok")) })
        })

        // Protected routes
        api.Group(func(protected chi.Router) {
            protected.Use(mw.Auth(dep.HMACSecret))

            protected.Route("/projects", func(pr chi.Router) {
                pr.Get("/", func(w http.ResponseWriter, r *http.Request){ w.WriteHeader(200); w.Write([]byte("[]")) })
                pr.Get("/{id}", func(w http.ResponseWriter, r *http.Request){ w.WriteHeader(200); w.Write([]byte("{}")) })
            })

            protected.Route("/deployments", func(dr chi.Router) {
                dr.Get("/", func(w http.ResponseWriter, r *http.Request){ w.WriteHeader(200); w.Write([]byte("[]")) })
            })

            protected.Route("/ai", func(ar chi.Router) {
                ar.Get("/status", func(w http.ResponseWriter, r *http.Request){ w.WriteHeader(200); w.Write([]byte("{}")) })
            })
        })
    })

    // pagination helper to ensure import usage
    _, _ = strconv.Atoi("0")
    return r
}


