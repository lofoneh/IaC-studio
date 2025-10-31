package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// Project represents an IaC project owned by a user.
type Project struct {
	ID            uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	UserID        uuid.UUID      `gorm:"type:uuid;index;not null" json:"user_id" validate:"required"`
	Name          string         `gorm:"not null;index:idx_projects_user_name,unique" json:"name" validate:"required"`
	Description   string         `gorm:"type:text" json:"description"`
	CloudProvider string         `gorm:"type:varchar(32);index" json:"cloud_provider" validate:"required,oneof=aws gcp azure do"`
	Settings      datatypes.JSON `gorm:"type:jsonb" json:"settings"`
	Archived      bool           `gorm:"not null;default:false;index" json:"archived"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
}



