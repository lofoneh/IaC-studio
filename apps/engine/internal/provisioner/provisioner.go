package provisioner

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/iac-studio/engine/internal/provisioner/compiler"
	"github.com/iac-studio/engine/internal/provisioner/terraform"
	"github.com/iac-studio/engine/pkg/logger"
	"go.uber.org/zap"
)

// Common errors
var (
	ErrInvalidInput = errors.New("invalid input")
	ErrInvalidState = errors.New("invalid terraform state")
)

// Provisioner handles infrastructure provisioning via Terraform
type Provisioner interface {
	// Plan generates an execution plan
	Plan(ctx context.Context, config *InfraConfig) (*Plan, error)

	// Apply executes the plan and provisions infrastructure
	Apply(ctx context.Context, config *InfraConfig) (*Result, error)

	// Destroy tears down infrastructure
	Destroy(ctx context.Context, deploymentID uuid.UUID, state []byte) (*Result, error)

	// GetState retrieves current Terraform state
	GetState(ctx context.Context, deploymentID uuid.UUID) ([]byte, error)
}

type InfraConfig struct {
	DeploymentID  uuid.UUID
	ProjectID     uuid.UUID
	GraphID       uuid.UUID
	Graph         Graph
	CloudProvider string
	CloudConfig   CloudConfig
	Variables     map[string]interface{}
}

type Graph struct {
	Nodes []Node `json:"nodes"`
	Edges []Edge `json:"edges"`
}

type Node struct {
	ID         string                 `json:"id"`
	Type       string                 `json:"type"` // e.g., "aws_instance"
	Properties map[string]interface{} `json:"properties"`
}

type Edge struct {
	ID   string `json:"id"`
	From string `json:"from"`
	To   string `json:"to"`
	Type string `json:"type"` // e.g., "depends_on"
}

type CloudConfig struct {
	Provider    string                 `json:"provider"`
	Region      string                 `json:"region"`
	Credentials map[string]interface{} `json:"credentials"`
}

type Plan struct {
	Changes      int    `json:"changes"`
	ResourceAdds int    `json:"resource_adds"`
	ResourceMods int    `json:"resource_mods"`
	ResourceDels int    `json:"resource_dels"`
	PlanOutput   string `json:"plan_output"`
}

type Result struct {
	Success      bool                   `json:"success"`
	Outputs      map[string]interface{} `json:"outputs"`
	State        []byte                 `json:"state"`
	Resources    []Resource             `json:"resources"`
	ErrorMessage string                 `json:"error_message,omitempty"`
}

type Resource struct {
	Type       string                 `json:"type"`
	Name       string                 `json:"name"`
	Attributes map[string]interface{} `json:"attributes"`
}

// TerraformProvisioner implements Provisioner using Terraform
type TerraformProvisioner struct {
	baseWorkingDir string
	compiler       *compiler.Compiler
	stateStore     terraform.StateStore
}

func NewTerraformProvisioner(workingDir string, stateStore terraform.StateStore) *TerraformProvisioner {
	return &TerraformProvisioner{
		baseWorkingDir: workingDir,
		compiler:       compiler.NewCompiler(),
		stateStore:     stateStore,
	}
}

// convert provisioner.Graph -> compiler.Graph
func convertGraph(g Graph) compiler.Graph {
	var cg compiler.Graph
	for _, n := range g.Nodes {
		cg.Nodes = append(cg.Nodes, compiler.Node{
			ID:         n.ID,
			Type:       n.Type,
			Properties: n.Properties,
		})
	}
	for _, e := range g.Edges {
		cg.Edges = append(cg.Edges, compiler.Edge{
			ID:   e.ID,
			From: e.From,
			To:   e.To,
			Type: e.Type,
		})
	}
	return cg
}

func (t *TerraformProvisioner) Plan(ctx context.Context, config *InfraConfig) (*Plan, error) {
	// 1. Compile graph to Terraform code
	tc, err := t.compiler.Compile(convertGraph(config.Graph), compiler.CloudConfig{
		Provider: config.CloudConfig.Provider,
		Region:   config.CloudConfig.Region,
	})
	if err != nil {
		return nil, fmt.Errorf("compile graph: %w", err)
	}

	// Convert compiler.TerraformCode -> terraform.TerraformCode
	code := terraform.TerraformCode{
		MainTF:      tc.MainTF,
		VariablesTF: tc.VariablesTF,
		OutputsTF:   tc.OutputsTF,
		ProviderTF:  tc.ProviderTF,
	}

	// Prepare a per-deployment working directory (unique for this run)
	depDir := filepath.Join(t.baseWorkingDir, config.DeploymentID.String(), strconv.FormatInt(time.Now().UnixNano(), 10))
	logger.L().Info("using working dir for plan", zap.String("dir", depDir))
	exec := terraform.NewExecutor(depDir)
	// ensure cleanup after plan
	defer func() {
		_ = exec.Cleanup()
	}()

	if err := exec.Initialize(ctx, &code); err != nil {
		return nil, fmt.Errorf("executor initialize: %w", err)
	}

	pr, err := exec.Plan(ctx)
	if err != nil {
		return nil, fmt.Errorf("executor plan: %w", err)
	}

	return &Plan{
		Changes:    0,
		PlanOutput: pr.PlanOutput,
	}, nil
}

func (t *TerraformProvisioner) Apply(ctx context.Context, config *InfraConfig) (*Result, error) {
	tc, err := t.compiler.Compile(convertGraph(config.Graph), compiler.CloudConfig{
		Provider: config.CloudConfig.Provider,
		Region:   config.CloudConfig.Region,
	})
	if err != nil {
		return nil, fmt.Errorf("compile graph: %w", err)
	}

	code := terraform.TerraformCode{
		MainTF:      tc.MainTF,
		VariablesTF: tc.VariablesTF,
		OutputsTF:   tc.OutputsTF,
		ProviderTF:  tc.ProviderTF,
	}

	// Per-deployment working directory
	depDir := filepath.Join(t.baseWorkingDir, config.DeploymentID.String(), strconv.FormatInt(time.Now().UnixNano(), 10))
	logger.L().Info("using working dir for apply", zap.String("dir", depDir))
	exec := terraform.NewExecutor(depDir)
	// cleanup working dir after apply to avoid disk bloat
	defer func() {
		_ = exec.Cleanup()
	}()

	if err := exec.Initialize(ctx, &code); err != nil {
		return nil, fmt.Errorf("executor initialize: %w", err)
	}

	ar, err := exec.Apply(ctx)
	if err != nil {
		return &Result{Success: false, ErrorMessage: err.Error()}, fmt.Errorf("executor apply: %w", err)
	}

	// persist state
	if t.stateStore != nil {
		_ = t.stateStore.SaveState(ctx, config.DeploymentID, ar.State)
	}

	return &Result{Success: true, Outputs: ar.Outputs, State: ar.State}, nil
}

func (t *TerraformProvisioner) Destroy(ctx context.Context, deploymentID uuid.UUID, state []byte) (*Result, error) {
	// For destroy, create a per-deployment dir and restore state if available
	depDir := filepath.Join(t.baseWorkingDir, deploymentID.String(), strconv.FormatInt(time.Now().UnixNano(), 10))
	logger.L().Info("using working dir for destroy", zap.String("dir", depDir))
	exec := terraform.NewExecutor(depDir)
	defer func() {
		_ = exec.Cleanup()
	}()

	// If state not provided, attempt to load from state store
	if len(state) == 0 && t.stateStore != nil {
		if s, err := t.stateStore.GetState(ctx, deploymentID); err == nil {
			state = s
		}
	}

	// If we have state, write it to terraform.tfstate so terraform can use it
	if len(state) > 0 {
		if err := os.MkdirAll(depDir, 0755); err != nil {
			return nil, fmt.Errorf("create working dir: %w", err)
		}
		if err := os.WriteFile(filepath.Join(depDir, "terraform.tfstate"), state, 0644); err != nil {
			return nil, fmt.Errorf("write state file: %w", err)
		}
	}

	if err := exec.Initialize(ctx, &terraform.TerraformCode{MainTF: "", VariablesTF: "", OutputsTF: "", ProviderTF: ""}); err != nil {
		return nil, fmt.Errorf("executor initialize: %w", err)
	}
	if err := exec.Destroy(ctx); err != nil {
		return &Result{Success: false, ErrorMessage: err.Error()}, fmt.Errorf("executor destroy: %w", err)
	}
	// clear state
	if t.stateStore != nil {
		_ = t.stateStore.SaveState(ctx, deploymentID, nil)
	}
	return &Result{Success: true}, nil
}

func (t *TerraformProvisioner) GetState(ctx context.Context, deploymentID uuid.UUID) ([]byte, error) {
	if t.stateStore == nil {
		return nil, nil
	}
	return t.stateStore.GetState(ctx, deploymentID)
}
