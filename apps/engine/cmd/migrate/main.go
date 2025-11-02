package main

import (
	"fmt"
	"os"
	"time"

	"github.com/iac-studio/engine/pkg/config"
	"github.com/iac-studio/engine/pkg/logger"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

func main() {
	cfg := config.MustLoad()
	log, err := logger.Init(cfg.LogLevel, cfg.LogFormat)
	if err != nil {
		panic(err)
	}
	defer logger.Sync()

	// Configure GORM logger
	var gormLogLevel gormlogger.LogLevel
	switch cfg.LogLevel {
	case "debug":
		gormLogLevel = gormlogger.Info
	default:
		gormLogLevel = gormlogger.Warn
	}

	log.Info("connecting to database...")

	db, err := gorm.Open(postgres.Open(cfg.DatabaseURL), &gorm.Config{
		Logger:                 gormlogger.Default.LogMode(gormLogLevel),
		NowFunc:                func() time.Time { return time.Now().UTC() },
		SkipDefaultTransaction: true,
	})
	if err != nil {
		log.Fatal("failed to connect to database", zap.Error(err))
	}

	// Set connection pool settings
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatal("failed to get database instance", zap.Error(err))
	}
	sqlDB.SetMaxOpenConns(10)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(time.Hour)

	log.Info("connection established, running migrations...")

	// Run migrations
	if err := runMigrations(db); err != nil {
		log.Fatal("migration failed", zap.Error(err))
	}

	log.Info("migrations completed successfully")
	fmt.Fprintln(os.Stdout, "✓ All migrations completed")
}