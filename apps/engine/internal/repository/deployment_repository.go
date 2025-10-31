package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/iac-studio/engine/internal/models"
	appErr "github.com/iac-studio/engine/pkg/errors"
	"gorm.io/gorm"
)

type DeploymentRepository interface {
	BaseRepository[models.Deployment]
	ListByProject(ctx context.Context, projectID uuid.UUID) ([]models.Deployment, error)
	GetLatestByProject(ctx context.Context, projectID uuid.UUID, dest *models.Deployment) error
	UpdateStatus(ctx context.Context, deploymentID uuid.UUID, status string) error
}

type deploymentRepository struct {
	BaseRepository[models.Deployment]
	db *gorm.DB
}

func NewDeploymentRepository(db *gorm.DB) DeploymentRepository {
	return &deploymentRepository{BaseRepository: NewBaseRepository[models.Deployment](db), db: db}
}

func (r *deploymentRepository) ListByProject(ctx context.Context, projectID uuid.UUID) ([]models.Deployment, error) {
	var out []models.Deployment
	if err := r.db.WithContext(ctx).Where("project_id = ?", projectID).Order("created_at DESC").Find(&out).Error; err != nil {
		return nil, appErr.Wrap(err, appErr.CodeInternal, "list deployments failed")
	}
	return out, nil
}

func (r *deploymentRepository) GetLatestByProject(ctx context.Context, projectID uuid.UUID, dest *models.Deployment) error {
	if err := r.db.WithContext(ctx).Where("project_id = ?", projectID).Order("created_at DESC").First(dest).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return appErr.New(appErr.CodeNotFound, "no deployments found")
		}
		return appErr.Wrap(err, appErr.CodeInternal, "get latest deployment failed")
	}
	return nil
}

func (r *deploymentRepository) UpdateStatus(ctx context.Context, deploymentID uuid.UUID, status string) error {
	res := r.db.WithContext(ctx).Model(&models.Deployment{}).Where("id = ?", deploymentID).Update("status", status)
	if res.Error != nil {
		return appErr.Wrap(res.Error, appErr.CodeInternal, "update deployment status failed")
	}
	if res.RowsAffected == 0 {
		return appErr.New(appErr.CodeNotFound, "deployment not found")
	}
	return nil
}


