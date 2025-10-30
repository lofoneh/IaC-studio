package main

import (
	"fmt"
	"os"

	"github.com/iac-studio/engine/pkg/config"
	"github.com/iac-studio/engine/pkg/logger"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	cfg := config.MustLoad()
	log, err := logger.Init(cfg.LogLevel, cfg.LogFormat)
	if err != nil {
		panic(err)
	}
	defer logger.Sync()

	db, err := gorm.Open(postgres.Open(cfg.DatabaseURL), &gorm.Config{})
	if err != nil {
		log.Fatal("failed to connect to database", zap.Error(err))
	}

	// TODO: add model migrations here, e.g., db.AutoMigrate(&models.User{})
	if err := db.Exec("SELECT 1").Error; err != nil {
		log.Fatal("migration failed", zap.Error(err))
	}

	fmt.Fprintln(os.Stdout, "migrations completed")
}
