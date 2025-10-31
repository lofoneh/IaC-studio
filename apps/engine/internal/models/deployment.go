package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// Deployment represents an execution of a ProjectGraph.
type Deployment struct {
	ID             uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	ProjectID      uuid.UUID      `gorm:"type:uuid;index;not null" json:"project_id" validate:"required"`
	GraphID        uuid.UUID      `gorm:"type:uuid;index;not null" json:"graph_id" validate:"required"`
	Status         string         `gorm:"type:varchar(32);index;not null" json:"status" validate:"required,oneof=pending planning applying completed failed"`
	TerraformState datatypes.JSON `gorm:"type:jsonb" json:"terraform_state"`
	Outputs        datatypes.JSON `gorm:"type:jsonb" json:"outputs"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
}



