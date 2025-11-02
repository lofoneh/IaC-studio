package main

import (
	"gorm.io/gorm"
	
	"github.com/iac-studio/engine/internal/models"
)

// registerModels returns all models that need migration
func registerModels() []interface{} {
	return []interface{}{
		// User Management
		&models.User{},
		&models.CloudAccount{},
		
		// Projects & Infrastructure
		&models.Project{},
		&models.Resource{},
		&models.Deployment{},
		
		// AI & Recommendations
		// &models.Recommendation{}, // enable once Recommendation model is defined
		
		// Add other models as you create them
		// &models.Workspace{},
		// &models.Variable{},
		// &models.Job{},
	}
}

// runMigrations executes all database migrations
func runMigrations(db *gorm.DB) error {
	models := registerModels()
	
	// Run AutoMigrate for all models
	if err := db.AutoMigrate(models...); err != nil {
		return err
	}

	// Run custom migrations
	return runCustomMigrations(db)
}

// runCustomMigrations handles schema changes AutoMigrate can't handle
func runCustomMigrations(db *gorm.DB) error {
	migrations := []func(*gorm.DB) error{
		enableUUIDExtension,
		addCloudAccountIndexes,
		// Add more custom migrations as needed
	}

	for _, migration := range migrations {
		if err := migration(db); err != nil {
			return err
		}
	}

	return nil
}

// enableUUIDExtension ensures UUID generation is available
func enableUUIDExtension(db *gorm.DB) error {
	return db.Exec(`CREATE EXTENSION IF NOT EXISTS "pgcrypto"`).Error
}

// addCloudAccountIndexes adds custom indexes for performance
func addCloudAccountIndexes(db *gorm.DB) error {
	// Composite index for user + provider lookups
	if err := db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_cloud_accounts_user_provider 
		ON cloud_accounts(user_id, provider) 
		WHERE deleted_at IS NULL
	`).Error; err != nil {
		return err
	}

	return nil
}