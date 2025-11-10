package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/iac-studio/engine/internal/models"
	appErr "github.com/iac-studio/engine/pkg/errors"
	"gorm.io/gorm"
)

type GraphRepository interface {
	BaseRepository[models.ProjectGraph]
	GetCurrentByProject(ctx context.Context, projectID uuid.UUID, dest *models.ProjectGraph) error
	GetByVersion(ctx context.Context, projectID uuid.UUID, version int, dest *models.ProjectGraph) error
	ListByProject(ctx context.Context, projectID uuid.UUID) ([]models.ProjectGraph, error)
	SetCurrent(ctx context.Context, projectID uuid.UUID, version int) error
}

type graphRepository struct {
	BaseRepository[models.ProjectGraph]
	db *gorm.DB
}

func NewGraphRepository(db *gorm.DB) GraphRepository {
	return &graphRepository{BaseRepository: NewBaseRepository[models.ProjectGraph](db), db: db}
}

func (r *graphRepository) GetCurrentByProject(ctx context.Context, projectID uuid.UUID, dest *models.ProjectGraph) error {
	if err := r.db.WithContext(ctx).Where("project_id = ? AND is_current = true", projectID).First(dest).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return appErr.New(appErr.CodeNotFound, "no current graph found")
		}
		return appErr.Wrap(err, appErr.CodeInternal, "get current graph failed")
	}
	return nil
}

func (r *graphRepository) GetByVersion(ctx context.Context, projectID uuid.UUID, version int, dest *models.ProjectGraph) error {
	if err := r.db.WithContext(ctx).Where("project_id = ? AND version = ?", projectID, version).First(dest).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return appErr.New(appErr.CodeNotFound, "graph version not found")
		}
		return appErr.Wrap(err, appErr.CodeInternal, "get graph version failed")
	}
	return nil
}

func (r *graphRepository) ListByProject(ctx context.Context, projectID uuid.UUID) ([]models.ProjectGraph, error) {
	var out []models.ProjectGraph
	if err := r.db.WithContext(ctx).Where("project_id = ?", projectID).Order("version DESC").Find(&out).Error; err != nil {
		return nil, appErr.Wrap(err, appErr.CodeInternal, "list graphs failed")
	}
	return out, nil
}

// SetCurrent marks the specified version as current and clears previous current flag in a transaction
func (r *graphRepository) SetCurrent(ctx context.Context, projectID uuid.UUID, version int) error {
	tx := r.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return appErr.Wrap(tx.Error, appErr.CodeInternal, "begin transaction failed")
	}

	// clear existing
	if err := tx.Model(&models.ProjectGraph{}).Where("project_id = ? AND is_current = true", projectID).Update("is_current", false).Error; err != nil {
		tx.Rollback()
		return appErr.Wrap(err, appErr.CodeInternal, "clear current flag failed")
	}

	// set desired version
	res := tx.Model(&models.ProjectGraph{}).Where("project_id = ? AND version = ?", projectID, version).Update("is_current", true)
	if res.Error != nil {
		tx.Rollback()
		return appErr.Wrap(res.Error, appErr.CodeInternal, "set current flag failed")
	}
	if res.RowsAffected == 0 {
		tx.Rollback()
		return appErr.New(appErr.CodeNotFound, "graph version not found")
	}

	if err := tx.Commit().Error; err != nil {
		return appErr.Wrap(err, appErr.CodeInternal, "commit transaction failed")
	}
	return nil
}
