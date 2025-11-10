package tasks

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"

	"github.com/iac-studio/engine/internal/models"
	"github.com/iac-studio/engine/internal/provisioner"
	"github.com/iac-studio/engine/internal/services"
	"github.com/iac-studio/engine/pkg/logger"
)

func TestMain(m *testing.M) {
	// Initialize logger for tests (required by tasks)
	_, err := logger.Init("info", "json")
	if err != nil {
		panic("failed to init logger: " + err.Error())
	}
	os.Exit(m.Run())
}

// Mock implementations
type mockProvisioner struct {
	mock.Mock
}

func (m *mockProvisioner) Plan(ctx context.Context, config *provisioner.InfraConfig) (*provisioner.Plan, error) {
	args := m.Called(ctx, config)
	if v := args.Get(0); v != nil {
		return v.(*provisioner.Plan), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockProvisioner) Apply(ctx context.Context, config *provisioner.InfraConfig) (*provisioner.Result, error) {
	args := m.Called(ctx, config)
	if v := args.Get(0); v != nil {
		return v.(*provisioner.Result), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockProvisioner) Destroy(ctx context.Context, deploymentID uuid.UUID, state []byte) (*provisioner.Result, error) {
	args := m.Called(ctx, deploymentID, state)
	if v := args.Get(0); v != nil {
		return v.(*provisioner.Result), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockProvisioner) GetState(ctx context.Context, deploymentID uuid.UUID) ([]byte, error) {
	args := m.Called(ctx, deploymentID)
	if v := args.Get(0); v != nil {
		return v.([]byte), args.Error(1)
	}
	return nil, args.Error(1)
}

type mockDeploymentService struct {
	mock.Mock
}

func (m *mockDeploymentService) CreateDeployment(ctx context.Context, projectID, userID uuid.UUID, input *services.CreateDeploymentInput) (*models.Deployment, error) {
	args := m.Called(ctx, projectID, userID, input)
	if v := args.Get(0); v != nil {
		return v.(*models.Deployment), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockDeploymentService) GetDeployment(ctx context.Context, deploymentID, userID uuid.UUID) (*models.Deployment, error) {
	args := m.Called(ctx, deploymentID, userID)
	if v := args.Get(0); v != nil {
		return v.(*models.Deployment), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockDeploymentService) ListDeployments(ctx context.Context, projectID, userID uuid.UUID, filters *services.DeploymentFilters) ([]models.Deployment, error) {
	args := m.Called(ctx, projectID, userID, filters)
	if v := args.Get(0); v != nil {
		return v.([]models.Deployment), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockDeploymentService) GetDeploymentLogs(ctx context.Context, deploymentID, userID uuid.UUID) ([]services.DeploymentLog, error) {
	args := m.Called(ctx, deploymentID, userID)
	if v := args.Get(0); v != nil {
		return v.([]services.DeploymentLog), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockDeploymentService) DestroyDeployment(ctx context.Context, deploymentID, userID uuid.UUID) error {
	args := m.Called(ctx, deploymentID, userID)
	return args.Error(0)
}

func (m *mockDeploymentService) CancelDeployment(ctx context.Context, deploymentID, userID uuid.UUID) error {
	args := m.Called(ctx, deploymentID, userID)
	return args.Error(0)
}

func (m *mockDeploymentService) UpdateDeploymentStatus(ctx context.Context, deploymentID uuid.UUID, status string) error {
	args := m.Called(ctx, deploymentID, status)
	return args.Error(0)
}

func (m *mockDeploymentService) SaveDeploymentOutputs(ctx context.Context, deploymentID uuid.UUID, outputs map[string]interface{}) error {
	args := m.Called(ctx, deploymentID, outputs)
	return args.Error(0)
}

func (m *mockDeploymentService) SaveTerraformState(ctx context.Context, deploymentID uuid.UUID, state []byte) error {
	args := m.Called(ctx, deploymentID, state)
	return args.Error(0)
}

func (m *mockDeploymentService) AppendLog(ctx context.Context, deploymentID uuid.UUID, log services.DeploymentLog) error {
	args := m.Called(ctx, deploymentID, log)
	return args.Error(0)
}

type mockProjectRepository struct {
	mock.Mock
}

func (m *mockProjectRepository) Create(ctx context.Context, obj *models.Project) error {
	args := m.Called(ctx, obj)
	return args.Error(0)
}

func (m *mockProjectRepository) GetByID(ctx context.Context, id any, dest *models.Project) error {
	args := m.Called(ctx, id, dest)
	if args.Error(0) == nil && args.Get(1) != nil {
		src := args.Get(1).(*models.Project)
		*dest = *src
	}
	return args.Error(0)
}

func (m *mockProjectRepository) Update(ctx context.Context, obj *models.Project) error {
	args := m.Called(ctx, obj)
	return args.Error(0)
}

func (m *mockProjectRepository) Delete(ctx context.Context, id any) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *mockProjectRepository) ListByUser(ctx context.Context, userID uuid.UUID) ([]models.Project, error) {
	args := m.Called(ctx, userID)
	if v := args.Get(0); v != nil {
		return v.([]models.Project), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockProjectRepository) Archive(ctx context.Context, projectID uuid.UUID) error {
	args := m.Called(ctx, projectID)
	return args.Error(0)
}

type mockDeploymentRepository struct {
	mock.Mock
}

func (m *mockDeploymentRepository) Create(ctx context.Context, obj *models.Deployment) error {
	args := m.Called(ctx, obj)
	return args.Error(0)
}

func (m *mockDeploymentRepository) GetByID(ctx context.Context, id any, dest *models.Deployment) error {
	args := m.Called(ctx, id, dest)
	if args.Error(0) == nil && args.Get(1) != nil {
		src := args.Get(1).(*models.Deployment)
		*dest = *src
	}
	return args.Error(0)
}

func (m *mockDeploymentRepository) Update(ctx context.Context, obj *models.Deployment) error {
	args := m.Called(ctx, obj)
	return args.Error(0)
}

func (m *mockDeploymentRepository) Delete(ctx context.Context, id any) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *mockDeploymentRepository) ListByProject(ctx context.Context, projectID uuid.UUID) ([]models.Deployment, error) {
	args := m.Called(ctx, projectID)
	if v := args.Get(0); v != nil {
		return v.([]models.Deployment), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockDeploymentRepository) GetLatestByProject(ctx context.Context, projectID uuid.UUID, dest *models.Deployment) error {
	args := m.Called(ctx, projectID, dest)
	if args.Error(0) == nil && args.Get(1) != nil {
		src := args.Get(1).(*models.Deployment)
		*dest = *src
	}
	return args.Error(0)
}

func (m *mockDeploymentRepository) UpdateStatus(ctx context.Context, deploymentID uuid.UUID, status string) error {
	args := m.Called(ctx, deploymentID, status)
	return args.Error(0)
}

type mockGraphRepository struct {
	mock.Mock
}

func (m *mockGraphRepository) Create(ctx context.Context, obj *models.ProjectGraph) error {
	args := m.Called(ctx, obj)
	return args.Error(0)
}

func (m *mockGraphRepository) GetByID(ctx context.Context, id any, dest *models.ProjectGraph) error {
	args := m.Called(ctx, id, dest)
	if args.Error(0) == nil && args.Get(1) != nil {
		src := args.Get(1).(*models.ProjectGraph)
		*dest = *src
	}
	return args.Error(0)
}

func (m *mockGraphRepository) Update(ctx context.Context, obj *models.ProjectGraph) error {
	args := m.Called(ctx, obj)
	return args.Error(0)
}

func (m *mockGraphRepository) Delete(ctx context.Context, id any) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *mockGraphRepository) GetCurrentByProject(ctx context.Context, projectID uuid.UUID, dest *models.ProjectGraph) error {
	args := m.Called(ctx, projectID, dest)
	if args.Error(0) == nil && args.Get(1) != nil {
		src := args.Get(1).(*models.ProjectGraph)
		*dest = *src
	}
	return args.Error(0)
}

func (m *mockGraphRepository) GetByVersion(ctx context.Context, projectID uuid.UUID, version int, dest *models.ProjectGraph) error {
	args := m.Called(ctx, projectID, version, dest)
	if args.Error(0) == nil && args.Get(1) != nil {
		src := args.Get(1).(*models.ProjectGraph)
		*dest = *src
	}
	return args.Error(0)
}

func (m *mockGraphRepository) ListByProject(ctx context.Context, projectID uuid.UUID) ([]models.ProjectGraph, error) {
	args := m.Called(ctx, projectID)
	if v := args.Get(0); v != nil {
		return v.([]models.ProjectGraph), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockGraphRepository) SetCurrent(ctx context.Context, projectID uuid.UUID, version int) error {
	args := m.Called(ctx, projectID, version)
	return args.Error(0)
}

func TestProvisionTaskHandler_HandleProvision(t *testing.T) {
	// Setup test data
	deploymentID := uuid.New()
	projectID := uuid.New()
	graphID := uuid.New()
	userID := uuid.New()

	// Create mock instances
	prov := &mockProvisioner{}
	deploySvc := &mockDeploymentService{}
	projectRepo := &mockProjectRepository{}
	graphRepo := &mockGraphRepository{}
	deployRepo := &mockDeploymentRepository{}

	// Create handler with mocks
	handler := NewProvisionTaskHandler(prov, deploySvc, projectRepo, graphRepo, deployRepo)

	// Test successful provision flow
	t.Run("successful provision", func(t *testing.T) {
		// Create task payload
		payload := ProvisionPayload{DeploymentID: deploymentID.String()}
		payloadBytes, _ := json.Marshal(payload)
		task := asynq.NewTask("deployment:provision", payloadBytes)

		// Mock deployment load
		deployment := &models.Deployment{
			ID:        deploymentID,
			ProjectID: projectID,
			GraphID:   graphID,
			Status:    "pending",
		}
		deployRepo.On("GetByID", mock.Anything, deploymentID, &models.Deployment{}).
			Run(func(args mock.Arguments) {
				dest := args.Get(2).(*models.Deployment)
				*dest = *deployment
			}).Return(nil, deployment).Once()

		// Mock project load
		project := &models.Project{
			ID:            projectID,
			UserID:        userID,
			Name:          "test-project",
			CloudProvider: "aws",
		}
		projectRepo.On("GetByID", mock.Anything, projectID, &models.Project{}).
			Run(func(args mock.Arguments) {
				dest := args.Get(2).(*models.Project)
				*dest = *project
			}).Return(nil, project).Once()

		// Mock graph load
		nodes := []provisioner.Node{{ID: "n1", Type: "aws_instance"}}
		nodesJSON, _ := json.Marshal(nodes)
		graph := &models.ProjectGraph{
			ID:        graphID,
			ProjectID: projectID,
			Version:   1,
			Nodes:     datatypes.JSON(nodesJSON),
			Edges:     datatypes.JSON("[]"),
			IsCurrent: true,
		}
		graphRepo.On("GetByID", mock.Anything, graphID, &models.ProjectGraph{}).
			Run(func(args mock.Arguments) {
				dest := args.Get(2).(*models.ProjectGraph)
				*dest = *graph
			}).Return(nil, graph).Once()

		// Mock status updates
		deploySvc.On("UpdateDeploymentStatus", mock.Anything, deploymentID, "planning").Return(nil).Once()
		deploySvc.On("UpdateDeploymentStatus", mock.Anything, deploymentID, "applying").Return(nil).Once()
		deploySvc.On("UpdateDeploymentStatus", mock.Anything, deploymentID, "applied").Return(nil).Once()

		// Mock provisioner apply
		result := &provisioner.Result{
			Success: true,
			Outputs: map[string]interface{}{"instance_ip": "1.2.3.4"},
			State:   []byte(`{"version":4}`),
		}
		prov.On("Apply", mock.Anything, mock.MatchedBy(func(cfg *provisioner.InfraConfig) bool {
			return cfg.DeploymentID == deploymentID && cfg.ProjectID == projectID
		})).Return(result, nil).Once()

		// Mock output/state persistence
		deploySvc.On("SaveDeploymentOutputs", mock.Anything, deploymentID, result.Outputs).Return(nil).Once()
		deploySvc.On("SaveTerraformState", mock.Anything, deploymentID, result.State).Return(nil).Once()
		deploySvc.On("AppendLog", mock.Anything, deploymentID, mock.MatchedBy(func(log services.DeploymentLog) bool {
			return log.Level == "info" && log.Message == "apply completed"
		})).Return(nil).Once()

		// Run the task handler
		err := handler.HandleProvision(context.Background(), task)
		require.NoError(t, err)

		// Verify all mocked calls were made
		mock.AssertExpectationsForObjects(t, prov, deploySvc, projectRepo, graphRepo, deployRepo)
	})

	// Test provision failure flow
	t.Run("provision failure", func(t *testing.T) {
		// Reset mocks
		prov = &mockProvisioner{}
		deploySvc = &mockDeploymentService{}
		projectRepo = &mockProjectRepository{}
		graphRepo = &mockGraphRepository{}
		deployRepo = &mockDeploymentRepository{}
		handler = NewProvisionTaskHandler(prov, deploySvc, projectRepo, graphRepo, deployRepo)

		// Create task payload
		payload := ProvisionPayload{DeploymentID: deploymentID.String()}
		payloadBytes, _ := json.Marshal(payload)
		task := asynq.NewTask("deployment:provision", payloadBytes)

		// Mock deployment load
		deployment := &models.Deployment{
			ID:        deploymentID,
			ProjectID: projectID,
			GraphID:   graphID,
			Status:    "pending",
		}
		deployRepo.On("GetByID", mock.Anything, deploymentID, &models.Deployment{}).
			Run(func(args mock.Arguments) {
				dest := args.Get(2).(*models.Deployment)
				*dest = *deployment
			}).Return(nil, deployment).Once()

		// Mock project load
		project := &models.Project{
			ID:            projectID,
			UserID:        userID,
			Name:          "test-project",
			CloudProvider: "aws",
		}
		projectRepo.On("GetByID", mock.Anything, projectID, &models.Project{}).
			Run(func(args mock.Arguments) {
				dest := args.Get(2).(*models.Project)
				*dest = *project
			}).Return(nil, project).Once()

		// Mock graph load
		nodes := []provisioner.Node{{ID: "n1", Type: "aws_instance"}}
		nodesJSON, _ := json.Marshal(nodes)
		graph := &models.ProjectGraph{
			ID:        graphID,
			ProjectID: projectID,
			Version:   1,
			Nodes:     datatypes.JSON(nodesJSON),
			Edges:     datatypes.JSON("[]"),
			IsCurrent: true,
		}
		graphRepo.On("GetByID", mock.Anything, graphID, &models.ProjectGraph{}).
			Run(func(args mock.Arguments) {
				dest := args.Get(2).(*models.ProjectGraph)
				*dest = *graph
			}).Return(nil, graph).Once()

		// Mock status updates
		deploySvc.On("UpdateDeploymentStatus", mock.Anything, deploymentID, "planning").Return(nil).Once()
		deploySvc.On("UpdateDeploymentStatus", mock.Anything, deploymentID, "applying").Return(nil).Once()
		deploySvc.On("UpdateDeploymentStatus", mock.Anything, deploymentID, "failed").Return(nil).Once()

		// Mock provisioner failure
		prov.On("Apply", mock.Anything, mock.MatchedBy(func(cfg *provisioner.InfraConfig) bool {
			return cfg.DeploymentID == deploymentID && cfg.ProjectID == projectID
		})).Return(nil, provisioner.ErrInvalidInput).Once()

		// Mock error logging
		deploySvc.On("AppendLog", mock.Anything, deploymentID, mock.MatchedBy(func(log services.DeploymentLog) bool {
			return log.Level == "error" && log.Message == "apply error: invalid input"
		})).Return(nil).Once()

		// Run the task handler
		err := handler.HandleProvision(context.Background(), task)
		require.Error(t, err)
		require.Equal(t, provisioner.ErrInvalidInput, err)

		// Verify all mocked calls were made
		mock.AssertExpectationsForObjects(t, prov, deploySvc, projectRepo, graphRepo, deployRepo)
	})
}

func TestProvisionTaskHandler_HandleDestroy(t *testing.T) {
	// Setup test data
	deploymentID := uuid.New()
	projectID := uuid.New()

	// Create mock instances
	prov := &mockProvisioner{}
	deploySvc := &mockDeploymentService{}
	projectRepo := &mockProjectRepository{}
	graphRepo := &mockGraphRepository{}
	deployRepo := &mockDeploymentRepository{}

	// Create handler with mocks
	handler := NewProvisionTaskHandler(prov, deploySvc, projectRepo, graphRepo, deployRepo)

	// Test successful destroy flow
	t.Run("successful destroy", func(t *testing.T) {
		// Create task payload
		payload := ProvisionPayload{DeploymentID: deploymentID.String()}
		payloadBytes, _ := json.Marshal(payload)
		task := asynq.NewTask("deployment:destroy", payloadBytes)

		// Mock deployment load with state
		state := []byte(`{"version":4,"resources":[]}`)
		deployment := &models.Deployment{
			ID:             deploymentID,
			ProjectID:      projectID,
			Status:         "destroying",
			TerraformState: datatypes.JSON(state),
		}
		deployRepo.On("GetByID", mock.Anything, deploymentID, &models.Deployment{}).
			Run(func(args mock.Arguments) {
				dest := args.Get(2).(*models.Deployment)
				*dest = *deployment
			}).Return(nil, deployment).Once()

		// Mock status updates
		deploySvc.On("UpdateDeploymentStatus", mock.Anything, deploymentID, "destroying").Return(nil).Once()
		deploySvc.On("UpdateDeploymentStatus", mock.Anything, deploymentID, "destroyed").Return(nil).Once()

		// Mock provisioner destroy
		result := &provisioner.Result{Success: true}
		prov.On("Destroy", mock.Anything, deploymentID, state).Return(result, nil).Once()

		// Mock logging
		deploySvc.On("AppendLog", mock.Anything, deploymentID, mock.MatchedBy(func(log services.DeploymentLog) bool {
			return log.Level == "info" && log.Message == "destroy completed"
		})).Return(nil).Once()

		// Run the task handler
		err := handler.HandleDestroy(context.Background(), task)
		require.NoError(t, err)

		// Verify all mocked calls were made
		mock.AssertExpectationsForObjects(t, prov, deploySvc, projectRepo, graphRepo, deployRepo)
	})

	// Test destroy failure flow
	t.Run("destroy failure", func(t *testing.T) {
		// Reset mocks
		prov = &mockProvisioner{}
		deploySvc = &mockDeploymentService{}
		projectRepo = &mockProjectRepository{}
		graphRepo = &mockGraphRepository{}
		deployRepo = &mockDeploymentRepository{}
		handler = NewProvisionTaskHandler(prov, deploySvc, projectRepo, graphRepo, deployRepo)

		// Create task payload
		payload := ProvisionPayload{DeploymentID: deploymentID.String()}
		payloadBytes, _ := json.Marshal(payload)
		task := asynq.NewTask("deployment:destroy", payloadBytes)

		// Mock deployment load with state
		state := []byte(`{"version":4,"resources":[]}`)
		deployment := &models.Deployment{
			ID:             deploymentID,
			ProjectID:      projectID,
			Status:         "destroying",
			TerraformState: datatypes.JSON(state),
		}
		deployRepo.On("GetByID", mock.Anything, deploymentID, &models.Deployment{}).
			Run(func(args mock.Arguments) {
				dest := args.Get(2).(*models.Deployment)
				*dest = *deployment
			}).Return(nil, deployment).Once()

		// Mock status updates
		deploySvc.On("UpdateDeploymentStatus", mock.Anything, deploymentID, "destroying").Return(nil).Once()
		deploySvc.On("UpdateDeploymentStatus", mock.Anything, deploymentID, "failed").Return(nil).Once()

		// Mock provisioner failure
		prov.On("Destroy", mock.Anything, deploymentID, state).Return(nil, provisioner.ErrInvalidState).Once()

		// Mock error logging
		deploySvc.On("AppendLog", mock.Anything, deploymentID, mock.MatchedBy(func(log services.DeploymentLog) bool {
			return log.Level == "error" && log.Message == "destroy error: invalid terraform state"
		})).Return(nil).Once()

		// Run the task handler
		err := handler.HandleDestroy(context.Background(), task)
		require.Error(t, err)
		require.Equal(t, provisioner.ErrInvalidState, err)

		// Verify all mocked calls were made
		mock.AssertExpectationsForObjects(t, prov, deploySvc, projectRepo, graphRepo, deployRepo)
	})
}
