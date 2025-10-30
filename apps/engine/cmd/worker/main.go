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
	"github.com/iac-studio/engine/pkg/logger"
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
	// TODO: register task handlers, e.g., mux.HandleFunc("provision:apply", handleProvisionApply)

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
