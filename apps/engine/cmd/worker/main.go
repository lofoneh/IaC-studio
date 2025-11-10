package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/iac-studio/engine/pkg/config"
	"github.com/iac-studio/engine/pkg/database"
	"github.com/iac-studio/engine/pkg/logger"

	"github.com/iac-studio/engine/internal/provisioner"
	terraformstate "github.com/iac-studio/engine/internal/provisioner/terraform"
	"github.com/iac-studio/engine/internal/queue/tasks"
	"github.com/iac-studio/engine/internal/repository"
	"github.com/iac-studio/engine/internal/services"
)

func main() {
	cfg := config.MustLoad()
	log, err := logger.Init(cfg.LogLevel, cfg.LogFormat)
	if err != nil {
		panic(err)
	}
	defer logger.Sync()

	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       0,
	})

	if err := rdb.Ping(context.Background()).Err(); err != nil {
		log.Fatal("redis connection failed", zap.Error(err))
	}

	srv := asynq.NewServer(
		asynq.RedisClientOpt{
			Addr:     cfg.RedisAddr,
			Password: cfg.RedisPassword,
			DB:       0,
		},
		asynq.Config{
			Concurrency: cfg.AsynqConcurrency,
		},
	)

	mux := asynq.NewServeMux()
	// Initialize DB and repositories for task handlers
	ctx := context.Background()
	db, err := database.OpenPostgres(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.L().Fatal("failed to open database", zap.Error(err))
	}

	projectRepo := repository.NewProjectRepository(db)
	deploymentRepo := repository.NewDeploymentRepository(db)
	graphRepo := repository.NewGraphRepository(db)

	// provisioner + state store
	stateStore := terraformstate.NewDatabaseStateStore(deploymentRepo)

	// Determine working directory for the provisioner. If WORKING_DIR is set
	// in config, use it (create if necessary). Otherwise fall back to
	// os.TempDir(). This allows per-deployment/workspace control in tests or
	// deployments.
	workingDir := cfg.WorkingDir
	if workingDir == "" {
		workingDir = os.TempDir()
	} else {
		if err := os.MkdirAll(workingDir, 0o755); err != nil {
			logger.L().Fatal("failed to create working dir", zap.Error(err))
		}
	}

	prov := provisioner.NewTerraformProvisioner(workingDir, stateStore)

	// deployment service (worker doesn't need asynq client)
	deploySvc := services.NewDeploymentService(db, projectRepo, deploymentRepo, nil)

	handler := tasks.NewProvisionTaskHandler(prov, deploySvc, projectRepo, graphRepo, deploymentRepo)
	mux.HandleFunc("deployment:provision", handler.HandleProvision)
	mux.HandleFunc("deployment:destroy", handler.HandleDestroy)

	errCh := make(chan error, 1)
	go func() {
		logger.L().Info("asynq worker starting", zap.Int("concurrency", cfg.AsynqConcurrency))
		if err := srv.Run(mux); err != nil {
			errCh <- err
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-sigCh:
		logger.L().Info("shutdown signal received", zap.String("signal", sig.String()))
	case err := <-errCh:
		logger.L().Error("worker stopped with error", zap.Error(err))
	}

	// Allow in-flight tasks to finish gracefully
	// NOTE: asynq.Server's Shutdown does not take any arguments and returns no value.
	srv.Shutdown()
}
