package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	chimid "github.com/go-chi/chi/v5/middleware"
	httpSwagger "github.com/swaggo/http-swagger"

	"github.com/iac-studio/engine/internal/api/handlers"
	mw "github.com/iac-studio/engine/internal/api/middleware"
)

type Dependencies struct {
	HMACSecret  []byte
	AuthHandler *handlers.AuthHandler
}

func NewRouter(dep Dependencies) http.Handler {
	r := chi.NewRouter()

	// Built-in middleware
	r.Use(mw.RequestID)
	r.Use(mw.Recovery)
	r.Use(mw.Logging)
	r.Use(mw.CORS)
	r.Use(mw.RateLimit(10, 20))
	r.Use(chimid.Compress(5))

	// Health endpoints
	hh := handlers.NewHealthHandler()
	r.Get("/healthz", hh.Liveness)
	r.Get("/readyz", hh.Readiness)

	// Swagger documentation
	r.Get("/docs/*", httpSwagger.Handler(
		httpSwagger.URL("http://localhost:8080/docs/doc.json"), // The url pointing to API definition
	))

	// API routes
	r.Route("/api/v1", func(api chi.Router) {
		// Auth routes (public)
		api.Route("/auth", func(ar chi.Router) {
			ar.Post("/register", dep.AuthHandler.Register)
			ar.Post("/login", dep.AuthHandler.Login)
			ar.Post("/logout", dep.AuthHandler.Logout)
			ar.Post("/refresh", dep.AuthHandler.Refresh)
		})

		// Protected routes
		api.Group(func(protected chi.Router) {
			protected.Use(mw.Auth(dep.HMACSecret))

			protected.Route("/projects", func(pr chi.Router) {
				pr.Get("/", func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(200)
					w.Write([]byte("[]"))
				})
				pr.Get("/{id}", func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(200)
					w.Write([]byte("{}"))
				})
			})

			protected.Route("/deployments", func(dr chi.Router) {
				dr.Get("/", func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(200)
					w.Write([]byte("[]"))
				})
			})

			protected.Route("/ai", func(ar chi.Router) {
				ar.Get("/status", func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(200)
					w.Write([]byte("{}"))
				})
			})
		})
	})

	return r
}