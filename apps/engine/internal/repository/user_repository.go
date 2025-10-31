package repository

import (
	"context"

	"github.com/iac-studio/engine/internal/models"
	appErr "github.com/iac-studio/engine/pkg/errors"
	"gorm.io/gorm"
)

type UserRepository interface {
	BaseRepository[models.User]
	GetByEmail(ctx context.Context, email string, dest *models.User) error
}

type userRepository struct {
	BaseRepository[models.User]
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{BaseRepository: NewBaseRepository[models.User](db), db: db}
}

func (r *userRepository) GetByEmail(ctx context.Context, email string, dest *models.User) error {
	if err := r.db.WithContext(ctx).Where("email = ?", email).First(dest).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return appErr.New(appErr.CodeNotFound, "user not found")
		}
		return appErr.Wrap(err, appErr.CodeInternal, "get user by email failed")
	}
	return nil
}


