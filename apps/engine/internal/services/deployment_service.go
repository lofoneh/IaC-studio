package services

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/iac-studio/engine/internal/models"
	"github.com/iac-studio/engine/internal/repository"
	appErr "github.com/iac-studio/engine/pkg/errors"
	"github.com/iac-studio/engine/pkg/logger"
	"go.uber.org/zap"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// Deployment service interface and DTOs
type DeploymentService interface {
	// Deployment lifecycle
	CreateDeployment(ctx context.Context, projectID, userID uuid.UUID, input *CreateDeploymentInput) (*models.Deployment, error)
	GetDeployment(ctx context.Context, deploymentID, userID uuid.UUID) (*models.Deployment, error)
	ListDeployments(ctx context.Context, projectID, userID uuid.UUID, filters *DeploymentFilters) ([]models.Deployment, error)
	GetDeploymentLogs(ctx context.Context, deploymentID, userID uuid.UUID) ([]DeploymentLog, error)

	// Actions
	DestroyDeployment(ctx context.Context, deploymentID, userID uuid.UUID) error
	CancelDeployment(ctx context.Context, deploymentID, userID uuid.UUID) error

	// Status updates (called by worker)
	UpdateDeploymentStatus(ctx context.Context, deploymentID uuid.UUID, status string) error
	SaveDeploymentOutputs(ctx context.Context, deploymentID uuid.UUID, outputs map[string]interface{}) error
	SaveTerraformState(ctx context.Context, deploymentID uuid.UUID, state []byte) error
	AppendLog(ctx context.Context, deploymentID uuid.UUID, log DeploymentLog) error
}

type CreateDeploymentInput struct {
	GraphID uuid.UUID
}

type DeploymentFilters struct {
	Status   string
	Page     int
	PageSize int
}

type DeploymentLog struct {
	Timestamp time.Time              `json:"timestamp"`
	Level     string                 `json:"level"`
	Message   string                 `json:"message"`
	Data      map[string]interface{} `json:"data,omitempty"`
}

type deploymentService struct {
	db          *gorm.DB
	projectRepo repository.ProjectRepository
	deployRepo  repository.DeploymentRepository
	asynqClient *asynq.Client
}

func NewDeploymentService(db *gorm.DB, projectRepo repository.ProjectRepository, deployRepo repository.DeploymentRepository, client *asynq.Client) DeploymentService {
	return &deploymentService{db: db, projectRepo: projectRepo, deployRepo: deployRepo, asynqClient: client}
}

var _ DeploymentService = (*deploymentService)(nil)

func (s *deploymentService) CreateDeployment(ctx context.Context, projectID, userID uuid.UUID, input *CreateDeploymentInput) (*models.Deployment, error) {
	logger.L().Info("create deployment", zap.String("project_id", projectID.String()), zap.String("user_id", userID.String()))

	var p models.Project
	if err := s.projectRepo.GetByID(ctx, projectID, &p); err != nil {
		return nil, err
	}
	if p.UserID != userID {
		return nil, appErr.New(appErr.CodeUnauthorized, "user does not own project")
	}

	// resolve graph if not provided
	var graph models.ProjectGraph
	if input.GraphID == uuid.Nil {
		if err := s.db.WithContext(ctx).Where("project_id = ?", projectID).Order("version DESC").First(&graph).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return nil, appErr.New(appErr.CodeNotFound, "no graph available for project")
			}
			return nil, appErr.Wrap(err, appErr.CodeInternal, "get latest graph failed")
		}
	} else {
		if err := s.db.WithContext(ctx).First(&graph, "id = ?", input.GraphID).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return nil, appErr.New(appErr.CodeNotFound, "graph not found")
			}
			return nil, appErr.Wrap(err, appErr.CodeInternal, "get graph failed")
		}
		if graph.ProjectID != projectID {
			return nil, appErr.New(appErr.CodeUnauthorized, "graph does not belong to project")
		}
	}

	// optional: prevent multiple active deployments for the same project
	var latest models.Deployment
	if err := s.deployRepo.GetLatestByProject(ctx, projectID, &latest); err == nil {
		if latest.Status == "pending" || latest.Status == "planning" || latest.Status == "applying" {
			return nil, appErr.New(appErr.CodeConflict, "another active deployment exists for project")
		}
	}

	d := &models.Deployment{
		ProjectID: projectID,
		GraphID:   graph.ID,
		Status:    "pending",
	}

	if err := s.deployRepo.Create(ctx, d); err != nil {
		return nil, err
	}

	// enqueue provision job
	payload := map[string]string{"deployment_id": d.ID.String()}
	pb, _ := json.Marshal(payload)
	task := asynq.NewTask("deployment:provision", pb)
	if s.asynqClient == nil {
		logger.L().Warn("asynq client not configured, skipping enqueue", zap.String("deployment_id", d.ID.String()))
	} else {
		if _, err := s.asynqClient.EnqueueContext(ctx, task); err != nil {
			// best-effort: mark deployment failed if enqueue fails
			logger.L().Error("enqueue provision task failed", zap.Error(err), zap.String("deployment_id", d.ID.String()))
			// try to update status
			_ = s.deployRepo.UpdateStatus(ctx, d.ID, "failed")
			return nil, appErr.Wrap(err, appErr.CodeInternal, "enqueue provision task failed")
		}
	}

	logger.L().Info("deployment created and enqueued", zap.String("deployment_id", d.ID.String()), zap.String("project_id", projectID.String()))
	return d, nil
}

func (s *deploymentService) GetDeployment(ctx context.Context, deploymentID, userID uuid.UUID) (*models.Deployment, error) {
	logger.L().Info("get deployment", zap.String("deployment_id", deploymentID.String()), zap.String("user_id", userID.String()))
	var d models.Deployment
	if err := s.deployRepo.GetByID(ctx, deploymentID, &d); err != nil {
		return nil, err
	}
	var p models.Project
	if err := s.projectRepo.GetByID(ctx, d.ProjectID, &p); err != nil {
		return nil, err
	}
	if p.UserID != userID {
		return nil, appErr.New(appErr.CodeUnauthorized, "user does not own project")
	}
	return &d, nil
}

func (s *deploymentService) ListDeployments(ctx context.Context, projectID, userID uuid.UUID, filters *DeploymentFilters) ([]models.Deployment, error) {
	logger.L().Info("list deployments", zap.String("project_id", projectID.String()), zap.String("user_id", userID.String()))
	var p models.Project
	if err := s.projectRepo.GetByID(ctx, projectID, &p); err != nil {
		return nil, err
	}
	if p.UserID != userID {
		return nil, appErr.New(appErr.CodeUnauthorized, "user does not own project")
	}
	return s.deployRepo.ListByProject(ctx, projectID)
}

func (s *deploymentService) GetDeploymentLogs(ctx context.Context, deploymentID, userID uuid.UUID) ([]DeploymentLog, error) {
	logger.L().Info("get deployment logs", zap.String("deployment_id", deploymentID.String()), zap.String("user_id", userID.String()))
	// validate ownership via deployment -> project -> user
	var d models.Deployment
	if err := s.deployRepo.GetByID(ctx, deploymentID, &d); err != nil {
		return nil, err
	}
	var p models.Project
	if err := s.projectRepo.GetByID(ctx, d.ProjectID, &p); err != nil {
		return nil, err
	}
	if p.UserID != userID {
		return nil, appErr.New(appErr.CodeUnauthorized, "user does not own project")
	}

	var out []DeploymentLog
	if len(d.Outputs) == 0 {
		return out, nil
	}
	var raw map[string]any
	if err := json.Unmarshal(d.Outputs, &raw); err != nil {
		return nil, appErr.Wrap(err, appErr.CodeInternal, "unmarshal outputs failed")
	}
	if logs, ok := raw["logs"]; ok {
		b, _ := json.Marshal(logs)
		_ = json.Unmarshal(b, &out) // ignore error, return what we can
	}
	return out, nil
}

func (s *deploymentService) DestroyDeployment(ctx context.Context, deploymentID, userID uuid.UUID) error {
	logger.L().Info("destroy deployment requested", zap.String("deployment_id", deploymentID.String()), zap.String("user_id", userID.String()))
	// ensure user owns the project
	var d models.Deployment
	if err := s.deployRepo.GetByID(ctx, deploymentID, &d); err != nil {
		return err
	}
	var p models.Project
	if err := s.projectRepo.GetByID(ctx, d.ProjectID, &p); err != nil {
		return err
	}
	if p.UserID != userID {
		return appErr.New(appErr.CodeUnauthorized, "user does not own project")
	}

	// enqueue destroy job
	payload := map[string]string{"deployment_id": d.ID.String()}
	pb, _ := json.Marshal(payload)
	task := asynq.NewTask("deployment:destroy", pb)
	if s.asynqClient == nil {
		logger.L().Warn("asynq client not configured, skipping destroy enqueue", zap.String("deployment_id", d.ID.String()))
	} else {
		if _, err := s.asynqClient.EnqueueContext(ctx, task); err != nil {
			logger.L().Error("enqueue destroy task failed", zap.Error(err), zap.String("deployment_id", d.ID.String()))
			return appErr.Wrap(err, appErr.CodeInternal, "enqueue destroy task failed")
		}
	}
	// mark as pending -> worker will set appropriate status
	_ = s.deployRepo.UpdateStatus(ctx, d.ID, "pending")
	return nil
}

func (s *deploymentService) CancelDeployment(ctx context.Context, deploymentID, userID uuid.UUID) error {
	logger.L().Info("cancel deployment", zap.String("deployment_id", deploymentID.String()), zap.String("user_id", userID.String()))
	var d models.Deployment
	if err := s.deployRepo.GetByID(ctx, deploymentID, &d); err != nil {
		return err
	}
	var p models.Project
	if err := s.projectRepo.GetByID(ctx, d.ProjectID, &p); err != nil {
		return err
	}
	if p.UserID != userID {
		return appErr.New(appErr.CodeUnauthorized, "user does not own project")
	}

	// We don't have a dedicated cancel flow in worker yet; mark as failed and log.
	if err := s.deployRepo.UpdateStatus(ctx, d.ID, "failed"); err != nil {
		return err
	}
	logger.L().Info("deployment cancelled (marked failed)", zap.String("deployment_id", deploymentID.String()))
	return nil
}

func (s *deploymentService) UpdateDeploymentStatus(ctx context.Context, deploymentID uuid.UUID, status string) error {
	logger.L().Info("update deployment status", zap.String("deployment_id", deploymentID.String()), zap.String("status", status))
	return s.deployRepo.UpdateStatus(ctx, deploymentID, status)
}

func (s *deploymentService) SaveDeploymentOutputs(ctx context.Context, deploymentID uuid.UUID, outputs map[string]interface{}) error {
	logger.L().Info("save deployment outputs", zap.String("deployment_id", deploymentID.String()))
	b, err := json.Marshal(outputs)
	if err != nil {
		return appErr.Wrap(err, appErr.CodeInvalid, "marshal outputs failed")
	}
	res := s.db.WithContext(ctx).Model(&models.Deployment{}).Where("id = ?", deploymentID).Update("outputs", datatypes.JSON(b))
	if res.Error != nil {
		return appErr.Wrap(res.Error, appErr.CodeInternal, "update outputs failed")
	}
	if res.RowsAffected == 0 {
		return appErr.New(appErr.CodeNotFound, "deployment not found")
	}
	return nil
}

func (s *deploymentService) SaveTerraformState(ctx context.Context, deploymentID uuid.UUID, state []byte) error {
	logger.L().Info("save terraform state", zap.String("deployment_id", deploymentID.String()))
	// store raw bytes as JSON (worker may compress/encode before calling)
	js := datatypes.JSON(state)
	res := s.db.WithContext(ctx).Model(&models.Deployment{}).Where("id = ?", deploymentID).Update("terraform_state", js)
	if res.Error != nil {
		return appErr.Wrap(res.Error, appErr.CodeInternal, "update terraform state failed")
	}
	if res.RowsAffected == 0 {
		return appErr.New(appErr.CodeNotFound, "deployment not found")
	}
	return nil
}

func (s *deploymentService) AppendLog(ctx context.Context, deploymentID uuid.UUID, logEntry DeploymentLog) error {
	logger.L().Info("append deployment log", zap.String("deployment_id", deploymentID.String()))
	var d models.Deployment
	if err := s.deployRepo.GetByID(ctx, deploymentID, &d); err != nil {
		return err
	}

	// read existing outputs -> logs[]
	var outputs map[string]any
	if len(d.Outputs) > 0 {
		if err := json.Unmarshal(d.Outputs, &outputs); err != nil {
			outputs = map[string]any{}
		}
	} else {
		outputs = map[string]any{}
	}

	var logs []DeploymentLog
	if v, ok := outputs["logs"]; ok {
		b, _ := json.Marshal(v)
		_ = json.Unmarshal(b, &logs)
	}
	logs = append(logs, logEntry)
	outputs["logs"] = logs

	b, err := json.Marshal(outputs)
	if err != nil {
		return appErr.Wrap(err, appErr.CodeInternal, "marshal outputs failed")
	}
	res := s.db.WithContext(ctx).Model(&models.Deployment{}).Where("id = ?", deploymentID).Update("outputs", datatypes.JSON(b))
	if res.Error != nil {
		return appErr.Wrap(res.Error, appErr.CodeInternal, "append log failed")
	}
	if res.RowsAffected == 0 {
		return appErr.New(appErr.CodeNotFound, "deployment not found")
	}
	return nil
}
