package api

import (
	"net/http"
	"github.com/go-chi/chi/v5"
	chimid "github.com/go-chi/chi/v5/middleware"
	"github.com/iac-studio/engine/internal/api/handlers"
	mw "github.com/iac-studio/engine/internal/api/middleware"
)

type Dependencies struct {
	HMACSecret         []byte
	AuthHandler        *handlers.AuthHandler
	ProjectsHandler    *handlers.ProjectsHandler
	DeploymentsHandler *handlers.DeploymentsHandler
	GraphsHandler      *handlers.GraphsHandler
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

	r.Route("/api/v1", func(api chi.Router) {
		// Auth routes (public - no auth required)
		api.Route("/auth", func(ar chi.Router) {
			ar.Post("/register", dep.AuthHandler.Register)
			ar.Post("/login", dep.AuthHandler.Login)
			ar.Post("/logout", dep.AuthHandler.Logout)
			ar.Post("/refresh", dep.AuthHandler.Refresh)
		})

		// Protected routes (require authentication)
		api.Group(func(protected chi.Router) {
			protected.Use(mw.Auth(dep.HMACSecret))

			// Projects
			protected.Route("/projects", func(pr chi.Router) {
				pr.Get("/", dep.ProjectsHandler.List)
				pr.Post("/", dep.ProjectsHandler.Create)
				pr.Get("/{id}", func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(200)
					w.Write([]byte(`{"success":true,"data":{}}`))
				})
			})

			// Deployments
			protected.Route("/deployments", func(dr chi.Router) {
				dr.Get("/", dep.DeploymentsHandler.List)
				dr.Post("/", dep.DeploymentsHandler.Create)
			})

			// Graphs
			protected.Route("/graphs", func(gr chi.Router) {
				gr.Post("/save", dep.GraphsHandler.Save)
				gr.Get("/load", dep.GraphsHandler.Load)
			})

			// AI (placeholder)
			protected.Route("/ai", func(ar chi.Router) {
				ar.Get("/status", func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(200)
					w.Write([]byte(`{"success":true,"data":{"status":"ready"}}`))
				})
			})
		})
	})

	return r
}