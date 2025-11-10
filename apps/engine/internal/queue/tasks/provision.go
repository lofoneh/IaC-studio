package tasks

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/iac-studio/engine/internal/models"
	"github.com/iac-studio/engine/internal/provisioner"
	"github.com/iac-studio/engine/internal/repository"
	"github.com/iac-studio/engine/internal/services"
	appErr "github.com/iac-studio/engine/pkg/errors"
	"github.com/iac-studio/engine/pkg/logger"
	"go.uber.org/zap"
)

// ProvisionPayload is the task payload for provision/destroy tasks.
type ProvisionPayload struct {
	DeploymentID string `json:"deployment_id"`
}

// ProvisionTaskHandler handles provisioning and destroy tasks.
type ProvisionTaskHandler struct {
	provisioner provisioner.Provisioner
	deploySvc   services.DeploymentService
	projectRepo repository.ProjectRepository
	graphRepo   repository.GraphRepository
	deployRepo  repository.DeploymentRepository
}

func NewProvisionTaskHandler(prov provisioner.Provisioner, deploySvc services.DeploymentService, projectRepo repository.ProjectRepository, graphRepo repository.GraphRepository, deployRepo repository.DeploymentRepository) *ProvisionTaskHandler {
	return &ProvisionTaskHandler{provisioner: prov, deploySvc: deploySvc, projectRepo: projectRepo, graphRepo: graphRepo, deployRepo: deployRepo}
}

func (h *ProvisionTaskHandler) HandleProvision(ctx context.Context, t *asynq.Task) error {
	var p ProvisionPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		logger.L().Error("invalid provision task payload", zap.Error(err))
		return err
	}
	id, err := uuid.Parse(p.DeploymentID)
	if err != nil {
		logger.L().Error("invalid deployment id in task", zap.Error(err))
		return err
	}

	logger.L().Info("handling provision task", zap.String("deployment_id", id.String()))

	// mark planning
	if err := h.deploySvc.UpdateDeploymentStatus(ctx, id, "planning"); err != nil {
		logger.L().Error("update status failed", zap.Error(err))
	}

	// load deployment, project, graph
	var d models.Deployment
	if err := h.deployRepo.GetByID(ctx, id, &d); err != nil {
		logger.L().Error("get deployment failed", zap.Error(err))
		_ = h.deploySvc.UpdateDeploymentStatus(ctx, id, "failed")
		return err
	}

	var proj models.Project
	if err := h.projectRepo.GetByID(ctx, d.ProjectID, &proj); err != nil {
		logger.L().Error("get project failed", zap.Error(err))
		_ = h.deploySvc.UpdateDeploymentStatus(ctx, id, "failed")
		return err
	}

	var g models.ProjectGraph
	if err := h.graphRepo.GetByID(ctx, d.GraphID, &g); err != nil {
		logger.L().Error("get graph failed", zap.Error(err))
		_ = h.deploySvc.UpdateDeploymentStatus(ctx, id, "failed")
		return err
	}

	// unmarshal nodes/edges into provisioner types
	var nodes []provisioner.Node
	var edges []provisioner.Edge
	if len(g.Nodes) > 0 {
		if err := json.Unmarshal(g.Nodes, &nodes); err != nil {
			logger.L().Error("unmarshal nodes failed", zap.Error(err))
			_ = h.deploySvc.UpdateDeploymentStatus(ctx, id, "failed")
			return appErr.Wrap(err, appErr.CodeInternal, "unmarshal nodes failed")
		}
	}
	if len(g.Edges) > 0 {
		if err := json.Unmarshal(g.Edges, &edges); err != nil {
			logger.L().Error("unmarshal edges failed", zap.Error(err))
			_ = h.deploySvc.UpdateDeploymentStatus(ctx, id, "failed")
			return appErr.Wrap(err, appErr.CodeInternal, "unmarshal edges failed")
		}
	}

	provGraph := provisioner.Graph{Nodes: nodes, Edges: edges}

	// build cloud config from project settings if present
	var settings map[string]interface{}
	if len(proj.Settings) > 0 {
		_ = json.Unmarshal(proj.Settings, &settings)
	}
	cloudCfg := provisioner.CloudConfig{Provider: proj.CloudProvider}
	if v, ok := settings["region"].(string); ok {
		cloudCfg.Region = v
	}
	if v, ok := settings["credentials"].(map[string]interface{}); ok {
		cloudCfg.Credentials = v
	}

	infra := &provisioner.InfraConfig{
		DeploymentID:  id,
		ProjectID:     proj.ID,
		GraphID:       g.ID,
		Graph:         provGraph,
		CloudProvider: proj.CloudProvider,
		CloudConfig:   cloudCfg,
		Variables:     map[string]interface{}{},
	}

	// apply
	if err := h.deploySvc.UpdateDeploymentStatus(ctx, id, "applying"); err != nil {
		logger.L().Warn("update status applying failed", zap.Error(err))
	}

	res, err := h.provisioner.Apply(ctx, infra)
	if err != nil {
		logger.L().Error("provision apply failed", zap.Error(err))
		_ = h.deploySvc.AppendLog(ctx, id, services.DeploymentLog{Timestamp: time.Now(), Level: "error", Message: fmt.Sprintf("apply error: %v", err)})
		_ = h.deploySvc.UpdateDeploymentStatus(ctx, id, "failed")
		return err
	}

	// persist outputs and state
	if res != nil {
		if res.Outputs != nil {
			_ = h.deploySvc.SaveDeploymentOutputs(ctx, id, res.Outputs)
		}
		if res.State != nil {
			_ = h.deploySvc.SaveTerraformState(ctx, id, res.State)
		}
	}

	_ = h.deploySvc.AppendLog(ctx, id, services.DeploymentLog{Timestamp: time.Now(), Level: "info", Message: "apply completed"})
	_ = h.deploySvc.UpdateDeploymentStatus(ctx, id, "applied")
	return nil
}

func (h *ProvisionTaskHandler) HandleDestroy(ctx context.Context, t *asynq.Task) error {
	var p ProvisionPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		logger.L().Error("invalid destroy task payload", zap.Error(err))
		return err
	}
	id, err := uuid.Parse(p.DeploymentID)
	if err != nil {
		logger.L().Error("invalid deployment id in task", zap.Error(err))
		return err
	}

	logger.L().Info("handling destroy task", zap.String("deployment_id", id.String()))
	_ = h.deploySvc.UpdateDeploymentStatus(ctx, id, "destroying")

	// fetch state
	var d models.Deployment
	if err := h.deployRepo.GetByID(ctx, id, &d); err != nil {
		logger.L().Error("get deployment failed", zap.Error(err))
		_ = h.deploySvc.UpdateDeploymentStatus(ctx, id, "failed")
		return err
	}

	state := []byte(nil)
	if len(d.TerraformState) > 0 {
		state = []byte(d.TerraformState)
	}

	res, err := h.provisioner.Destroy(ctx, id, state)
	if err != nil {
		logger.L().Error("destroy failed", zap.Error(err))
		_ = h.deploySvc.AppendLog(ctx, id, services.DeploymentLog{Timestamp: time.Now(), Level: "error", Message: fmt.Sprintf("destroy error: %v", err)})
		_ = h.deploySvc.UpdateDeploymentStatus(ctx, id, "failed")
		return err
	}

	if res != nil && res.State != nil {
		_ = h.deploySvc.SaveTerraformState(ctx, id, res.State)
	}
	_ = h.deploySvc.AppendLog(ctx, id, services.DeploymentLog{Timestamp: time.Now(), Level: "info", Message: "destroy completed"})
	_ = h.deploySvc.UpdateDeploymentStatus(ctx, id, "destroyed")
	return nil
}
