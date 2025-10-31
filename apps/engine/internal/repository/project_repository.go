package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/iac-studio/engine/internal/models"
	appErr "github.com/iac-studio/engine/pkg/errors"
	"gorm.io/gorm"
)

type ProjectRepository interface {
	BaseRepository[models.Project]
	ListByUser(ctx context.Context, userID uuid.UUID) ([]models.Project, error)
	Archive(ctx context.Context, projectID uuid.UUID) error
}

type projectRepository struct {
	BaseRepository[models.Project]
	db *gorm.DB
}

func NewProjectRepository(db *gorm.DB) ProjectRepository {
	return &projectRepository{BaseRepository: NewBaseRepository[models.Project](db), db: db}
}

func (r *projectRepository) ListByUser(ctx context.Context, userID uuid.UUID) ([]models.Project, error) {
	var out []models.Project
	if err := r.db.WithContext(ctx).Where("user_id = ? AND archived = false", userID).Order("created_at DESC").Find(&out).Error; err != nil {
		return nil, appErr.Wrap(err, appErr.CodeInternal, "list projects by user failed")
	}
	return out, nil
}

func (r *projectRepository) Archive(ctx context.Context, projectID uuid.UUID) error {
	res := r.db.WithContext(ctx).Model(&models.Project{}).Where("id = ?", projectID).Update("archived", true)
	if res.Error != nil {
		return appErr.Wrap(res.Error, appErr.CodeInternal, "archive project failed")
	}
	if res.RowsAffected == 0 {
		return appErr.New(appErr.CodeNotFound, "project not found")
	}
	return nil
}


