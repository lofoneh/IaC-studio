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

	// Swagger documentation
	r.Get("/docs/*", httpSwagger.Handler(
		httpSwagger.URL("http://localhost:8080/docs/doc.json"),
	))

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

			// Projects
			protected.Route("/projects", func(pr chi.Router) {
				pr.Get("/", dep.ProjectsHandler.List)
				pr.Post("/", dep.ProjectsHandler.Create)
				pr.Get("/{id}", dep.ProjectsHandler.Get)
				pr.Put("/{id}", dep.ProjectsHandler.Update)
				pr.Delete("/{id}", dep.ProjectsHandler.Delete)
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

			// AI
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