package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/iac-studio/engine/internal/api"
	"github.com/iac-studio/engine/internal/api/handlers"
	"github.com/iac-studio/engine/internal/repository"
	"github.com/iac-studio/engine/pkg/config"
	"github.com/iac-studio/engine/pkg/database"
	"github.com/iac-studio/engine/pkg/logger"
	"go.uber.org/zap"

	// Import generated docs (will be created after running swag init)
	_ "github.com/iac-studio/engine/docs"
)

// @title           IaC Studio API
// @version         1.0
// @description     Infrastructure as Code management platform with AI-powered recommendations
// @termsOfService  https://iacstudio.io/terms

// @contact.name   IaC Studio Support
// @contact.url    https://iacstudio.io/support
// @contact.email  support@iacstudio.io

// @license.name  MIT
// @license.url   https://opensource.org/licenses/MIT

// @host      localhost:8080
// @BasePath  /api/v1

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.

func main() {
	// Load configuration
	cfg := config.MustLoad()
	
	// Initialize logger
	log, err := logger.Init(cfg.LogLevel, cfg.LogFormat)
	if err != nil {
		panic(err)
	}
	defer logger.Sync()

	log.Info("Starting IaC Studio Engine", 
		zap.String("env", cfg.AppEnv),
		zap.String("addr", cfg.HTTPAddr),
	)

	// Connect to database
	ctx := context.Background()
	db, err := database.OpenPostgres(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal("Failed to connect to database", zap.Error(err))
	}
	log.Info("Database connected successfully")

	// Initialize repositories
	userRepo := repository.NewUserRepository(db)

	// JWT Secret from environment
	jwtSecret := []byte(os.Getenv("JWT_SECRET"))
	if len(jwtSecret) == 0 {
		log.Warn("JWT_SECRET not set, using default (INSECURE for production)")
		jwtSecret = []byte("change-me-in-production-please")
	}

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(userRepo, jwtSecret)

	// Create router with dependencies
	router := api.NewRouter(api.Dependencies{
		HMACSecret:  jwtSecret,
		AuthHandler: authHandler,
	})

	// Create HTTP server
	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       90 * time.Second,
	}

	// Start server in goroutine
	errCh := make(chan error, 1)
	go func() {
		log.Info("HTTP server starting", zap.String("addr", cfg.HTTPAddr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	// Graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-sigCh:
		log.Info("shutdown signal received", zap.String("signal", sig.String()))
	case err := <-errCh:
		log.Error("server error", zap.Error(err))
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("server shutdown error", zap.Error(err))
	} else {
		log.Info("server exited gracefully")
	}
}