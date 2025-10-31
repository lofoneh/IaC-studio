package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// ProjectGraph stores the graph version of project resources.
type ProjectGraph struct {
	ID        uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	ProjectID uuid.UUID      `gorm:"type:uuid;index;not null" json:"project_id" validate:"required"`
	Version   int            `gorm:"not null;index:idx_graph_project_version,unique" json:"version" validate:"gte=1"`
	Nodes     datatypes.JSON `gorm:"type:jsonb" json:"nodes" validate:"required"`
	Edges     datatypes.JSON `gorm:"type:jsonb" json:"edges" validate:"required"`
	IsCurrent bool           `gorm:"not null;default:false;index" json:"is_current"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}


