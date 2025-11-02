package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// CloudAccount stores provider credentials for a user.
type CloudAccount struct {
	ID          uuid.UUID       `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	UserID      uuid.UUID       `gorm:"type:uuid;index;not null" json:"user_id" validate:"required"`
	Provider    string          `gorm:"type:varchar(32);index;not null" json:"provider" validate:"required,oneof=aws azure gcp"`
	Credentials []byte          `gorm:"type:bytea;not null" json:"-"` 
	Metadata    datatypes.JSON  `gorm:"type:jsonb" json:"metadata"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
	DeletedAt   gorm.DeletedAt  `gorm:"index" json:"-"`
}

// TableName overrides the table name
func (CloudAccount) TableName() string {
	return "cloud_accounts"
}