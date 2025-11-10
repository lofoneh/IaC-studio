package services

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/iac-studio/engine/internal/models"
	"github.com/iac-studio/engine/internal/repository"
	appErr "github.com/iac-studio/engine/pkg/errors"
	"github.com/iac-studio/engine/pkg/logger"
	"go.uber.org/zap"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// Service interface and related DTOs
type ProjectService interface {
	// Project CRUD
	CreateProject(ctx context.Context, userID uuid.UUID, input *CreateProjectInput) (*models.Project, error)
	GetProject(ctx context.Context, projectID, userID uuid.UUID) (*models.Project, error)
	ListProjects(ctx context.Context, userID uuid.UUID, filters *ProjectFilters) ([]models.Project, error)
	UpdateProject(ctx context.Context, projectID, userID uuid.UUID, updates *UpdateProjectInput) (*models.Project, error)
	DeleteProject(ctx context.Context, projectID, userID uuid.UUID) error

	// Graph management
	SaveGraph(ctx context.Context, projectID, userID uuid.UUID, graphData *GraphData) (*models.ProjectGraph, error)
	GetCurrentGraph(ctx context.Context, projectID, userID uuid.UUID) (*models.ProjectGraph, error)
	GetGraphVersion(ctx context.Context, projectID, userID uuid.UUID, version int) (*models.ProjectGraph, error)
	ListGraphVersions(ctx context.Context, projectID, userID uuid.UUID) ([]models.ProjectGraph, error)
}

type CreateProjectInput struct {
	Name          string
	Description   string
	CloudProvider string
	Settings      map[string]interface{}
}

type UpdateProjectInput struct {
	Description   *string
	CloudProvider *string
	Settings      map[string]interface{}
}

type ProjectFilters struct {
	Archived bool
	Page     int
	PageSize int
}

type GraphData struct {
	Nodes []GraphNode `json:"nodes"`
	Edges []GraphEdge `json:"edges"`
}

type GraphNode struct {
	ID         string                 `json:"id"`
	Type       string                 `json:"type"`
	Properties map[string]interface{} `json:"properties"`
	Position   Position               `json:"position"`
}

type GraphEdge struct {
	ID   string `json:"id"`
	From string `json:"from"`
	To   string `json:"to"`
	Type string `json:"type"`
}

type Position struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type projectService struct {
	db          *gorm.DB
	projectRepo repository.ProjectRepository
}

func NewProjectService(db *gorm.DB, projectRepo repository.ProjectRepository) ProjectService {
	return &projectService{db: db, projectRepo: projectRepo}
}

// Ensure interfaces are satisfied at compile time
var _ ProjectService = (*projectService)(nil)

// CreateProject creates a new project for the given user.
func (s *projectService) CreateProject(ctx context.Context, userID uuid.UUID, input *CreateProjectInput) (*models.Project, error) {
	logger.L().Info("create project called", zap.String("user_id", userID.String()), zap.String("name", input.Name))

	var settings datatypes.JSON
	if input.Settings != nil {
		b, err := json.Marshal(input.Settings)
		if err != nil {
			return nil, appErr.Wrap(err, appErr.CodeInvalid, "invalid settings json")
		}
		settings = datatypes.JSON(b)
	}

	p := &models.Project{
		UserID:        userID,
		Name:          input.Name,
		Description:   input.Description,
		CloudProvider: input.CloudProvider,
		Settings:      settings,
	}

	if err := s.projectRepo.Create(ctx, p); err != nil {
		return nil, err
	}

	logger.L().Info("project created", zap.String("project_id", p.ID.String()), zap.String("user_id", userID.String()))
	return p, nil
}

func (s *projectService) GetProject(ctx context.Context, projectID, userID uuid.UUID) (*models.Project, error) {
	logger.L().Info("get project", zap.String("project_id", projectID.String()), zap.String("user_id", userID.String()))
	var p models.Project
	if err := s.projectRepo.GetByID(ctx, projectID, &p); err != nil {
		return nil, err
	}
	if p.UserID != userID {
		return nil, appErr.New(appErr.CodeUnauthorized, "user does not own project")
	}
	return &p, nil
}

func (s *projectService) ListProjects(ctx context.Context, userID uuid.UUID, filters *ProjectFilters) ([]models.Project, error) {
	logger.L().Info("list projects", zap.String("user_id", userID.String()))
	// repository handles user filtering
	return s.projectRepo.ListByUser(ctx, userID)
}

func (s *projectService) UpdateProject(ctx context.Context, projectID, userID uuid.UUID, updates *UpdateProjectInput) (*models.Project, error) {
	logger.L().Info("update project", zap.String("project_id", projectID.String()), zap.String("user_id", userID.String()))
	var p models.Project
	if err := s.projectRepo.GetByID(ctx, projectID, &p); err != nil {
		return nil, err
	}
	if p.UserID != userID {
		return nil, appErr.New(appErr.CodeUnauthorized, "user does not own project")
	}

	if updates.Description != nil {
		p.Description = *updates.Description
	}
	if updates.CloudProvider != nil {
		p.CloudProvider = *updates.CloudProvider
	}
	if updates.Settings != nil {
		b, err := json.Marshal(updates.Settings)
		if err != nil {
			return nil, appErr.Wrap(err, appErr.CodeInvalid, "invalid settings json")
		}
		p.Settings = datatypes.JSON(b)
	}

	if err := s.projectRepo.Update(ctx, &p); err != nil {
		return nil, err
	}

	logger.L().Info("project updated", zap.String("project_id", projectID.String()), zap.String("user_id", userID.String()))
	return &p, nil
}

func (s *projectService) DeleteProject(ctx context.Context, projectID, userID uuid.UUID) error {
	logger.L().Info("delete project", zap.String("project_id", projectID.String()), zap.String("user_id", userID.String()))
	var p models.Project
	if err := s.projectRepo.GetByID(ctx, projectID, &p); err != nil {
		return err
	}
	if p.UserID != userID {
		return appErr.New(appErr.CodeUnauthorized, "user does not own project")
	}
	if err := s.projectRepo.Delete(ctx, projectID); err != nil {
		return err
	}
	logger.L().Info("project deleted", zap.String("project_id", projectID.String()), zap.String("user_id", userID.String()))
	return nil
}

// Graph management
func (s *projectService) SaveGraph(ctx context.Context, projectID, userID uuid.UUID, graphData *GraphData) (*models.ProjectGraph, error) {
	logger.L().Info("save graph start", zap.String("project_id", projectID.String()), zap.String("user_id", userID.String()))

	// verify ownership
	var p models.Project
	if err := s.projectRepo.GetByID(ctx, projectID, &p); err != nil {
		return nil, err
	}
	if p.UserID != userID {
		return nil, appErr.New(appErr.CodeUnauthorized, "user does not own project")
	}

	// marshal nodes/edges
	nodesB, err := json.Marshal(graphData.Nodes)
	if err != nil {
		return nil, appErr.Wrap(err, appErr.CodeInvalid, "invalid nodes json")
	}
	edgesB, err := json.Marshal(graphData.Edges)
	if err != nil {
		return nil, appErr.Wrap(err, appErr.CodeInvalid, "invalid edges json")
	}

	tx := s.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return nil, appErr.Wrap(tx.Error, appErr.CodeInternal, "begin transaction failed")
	}

	// compute next version
	var maxVersion int
	if err := tx.Model(&models.ProjectGraph{}).Where("project_id = ?", projectID).Select("COALESCE(MAX(version),0)").Scan(&maxVersion).Error; err != nil {
		tx.Rollback()
		return nil, appErr.Wrap(err, appErr.CodeInternal, "compute graph version failed")
	}
	nextVersion := maxVersion + 1

	// mark previous graphs not current
	if err := tx.Model(&models.ProjectGraph{}).Where("project_id = ? AND is_current = true", projectID).Update("is_current", false).Error; err != nil {
		tx.Rollback()
		return nil, appErr.Wrap(err, appErr.CodeInternal, "mark previous graphs failed")
	}

	g := &models.ProjectGraph{
		ProjectID: projectID,
		Version:   nextVersion,
		Nodes:     datatypes.JSON(nodesB),
		Edges:     datatypes.JSON(edgesB),
		IsCurrent: true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := tx.Create(g).Error; err != nil {
		tx.Rollback()
		return nil, appErr.Wrap(err, appErr.CodeInternal, "create graph failed")
	}

	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		return nil, appErr.Wrap(err, appErr.CodeInternal, "commit transaction failed")
	}

	logger.L().Info("graph saved", zap.String("project_id", projectID.String()), zap.Int("version", nextVersion), zap.String("user_id", userID.String()))
	return g, nil
}

func (s *projectService) GetCurrentGraph(ctx context.Context, projectID, userID uuid.UUID) (*models.ProjectGraph, error) {
	logger.L().Info("get current graph", zap.String("project_id", projectID.String()), zap.String("user_id", userID.String()))
	var p models.Project
	if err := s.projectRepo.GetByID(ctx, projectID, &p); err != nil {
		return nil, err
	}
	if p.UserID != userID {
		return nil, appErr.New(appErr.CodeUnauthorized, "user does not own project")
	}

	var g models.ProjectGraph
	if err := s.db.WithContext(ctx).Where("project_id = ? AND is_current = true", projectID).First(&g).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, appErr.New(appErr.CodeNotFound, "graph not found")
		}
		return nil, appErr.Wrap(err, appErr.CodeInternal, "get current graph failed")
	}
	return &g, nil
}

func (s *projectService) GetGraphVersion(ctx context.Context, projectID, userID uuid.UUID, version int) (*models.ProjectGraph, error) {
	logger.L().Info("get graph version", zap.String("project_id", projectID.String()), zap.Int("version", version), zap.String("user_id", userID.String()))
	var p models.Project
	if err := s.projectRepo.GetByID(ctx, projectID, &p); err != nil {
		return nil, err
	}
	if p.UserID != userID {
		return nil, appErr.New(appErr.CodeUnauthorized, "user does not own project")
	}

	var g models.ProjectGraph
	if err := s.db.WithContext(ctx).Where("project_id = ? AND version = ?", projectID, version).First(&g).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, appErr.New(appErr.CodeNotFound, "graph version not found")
		}
		return nil, appErr.Wrap(err, appErr.CodeInternal, "get graph version failed")
	}
	return &g, nil
}

func (s *projectService) ListGraphVersions(ctx context.Context, projectID, userID uuid.UUID) ([]models.ProjectGraph, error) {
	logger.L().Info("list graph versions", zap.String("project_id", projectID.String()), zap.String("user_id", userID.String()))
	var p models.Project
	if err := s.projectRepo.GetByID(ctx, projectID, &p); err != nil {
		return nil, err
	}
	if p.UserID != userID {
		return nil, appErr.New(appErr.CodeUnauthorized, "user does not own project")
	}

	var out []models.ProjectGraph
	if err := s.db.WithContext(ctx).Where("project_id = ?", projectID).Order("version DESC").Find(&out).Error; err != nil {
		return nil, appErr.Wrap(err, appErr.CodeInternal, "list graph versions failed")
	}
	return out, nil
}
