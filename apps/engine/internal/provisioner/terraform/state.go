package terraform

import (
    "context"

    "github.com/google/uuid"
    "github.com/iac-studio/engine/internal/models"
    "github.com/iac-studio/engine/internal/repository"
    appErr "github.com/iac-studio/engine/pkg/errors"
    "gorm.io/datatypes"
)

// StateStore handles Terraform state persistence
type StateStore interface {
    SaveState(ctx context.Context, deploymentID uuid.UUID, state []byte) error
    GetState(ctx context.Context, deploymentID uuid.UUID) ([]byte, error)
    LockState(ctx context.Context, deploymentID uuid.UUID) error
    UnlockState(ctx context.Context, deploymentID uuid.UUID) error
}

type DatabaseStateStore struct {
    deploymentRepo repository.DeploymentRepository
}

func NewDatabaseStateStore(deploymentRepo repository.DeploymentRepository) *DatabaseStateStore {
    return &DatabaseStateStore{
        deploymentRepo: deploymentRepo,
    }
}

func (s *DatabaseStateStore) SaveState(ctx context.Context, deploymentID uuid.UUID, state []byte) error {
    var d models.Deployment
    if err := s.deploymentRepo.GetByID(ctx, deploymentID, &d); err != nil {
        return err
    }
    if state == nil {
        d.TerraformState = datatypes.JSON([]byte("null"))
    } else {
        d.TerraformState = datatypes.JSON(state)
    }
    if err := s.deploymentRepo.Update(ctx, &d); err != nil {
        return appErr.Wrap(err, appErr.CodeInternal, "save state failed")
    }
    return nil
}

func (s *DatabaseStateStore) GetState(ctx context.Context, deploymentID uuid.UUID) ([]byte, error) {
    var d models.Deployment
    if err := s.deploymentRepo.GetByID(ctx, deploymentID, &d); err != nil {
        return nil, err
    }
    if len(d.TerraformState) == 0 {
        return nil, nil
    }
    return []byte(d.TerraformState), nil
}

func (s *DatabaseStateStore) LockState(ctx context.Context, deploymentID uuid.UUID) error {
    // For MVP, skipping explicit locking. DB row-level locks or advisory locks can be added later.
    return nil
}

func (s *DatabaseStateStore) UnlockState(ctx context.Context, deploymentID uuid.UUID) error {
    return nil
}


