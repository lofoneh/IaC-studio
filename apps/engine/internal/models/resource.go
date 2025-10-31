package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// Resource represents a managed cloud resource in a deployment.
type Resource struct {
	ID           uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	DeploymentID uuid.UUID      `gorm:"type:uuid;index;not null" json:"deployment_id" validate:"required"`
	ResourceType string         `gorm:"type:varchar(64);index;not null" json:"resource_type" validate:"required"`
	ResourceName string         `gorm:"type:varchar(128);index;not null" json:"resource_name" validate:"required"`
	Properties   datatypes.JSON `gorm:"type:jsonb" json:"properties"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}



