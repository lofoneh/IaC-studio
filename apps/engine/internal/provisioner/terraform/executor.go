package terraform

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/hashicorp/terraform-exec/tfexec"
	"github.com/iac-studio/engine/pkg/logger"
	"go.uber.org/zap"
)

// Executor wraps terraform-exec for running Terraform commands
type Executor struct {
	workingDir string
	tf         *tfexec.Terraform
}

func NewExecutor(workingDir string) *Executor {
	return &Executor{
		workingDir: workingDir,
	}
}

// Initialize sets up Terraform in the working directory
func (e *Executor) Initialize(ctx context.Context, code *TerraformCode) error {
	// Create working directory
	if err := os.MkdirAll(e.workingDir, 0755); err != nil {
		return fmt.Errorf("create working dir: %w", err)
	}

	// Write Terraform files
	files := map[string]string{
		"main.tf":      code.MainTF,
		"variables.tf": code.VariablesTF,
		"outputs.tf":   code.OutputsTF,
		"provider.tf":  code.ProviderTF,
	}

	for filename, content := range files {
		path := filepath.Join(e.workingDir, filename)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			return fmt.Errorf("write %s: %w", filename, err)
		}
	}

	// Find terraform binary
	tfPath, err := exec.LookPath("terraform")
	if err != nil {
		return fmt.Errorf("terraform not found in PATH: %w", err)
	}

	// Create terraform executor
	tf, err := tfexec.NewTerraform(e.workingDir, tfPath)
	if err != nil {
		return fmt.Errorf("create terraform executor: %w", err)
	}

	e.tf = tf

	// Run terraform init
	logger.L().Info("running terraform init", zap.String("working_dir", e.workingDir))
	if err := tf.Init(ctx, tfexec.Upgrade(true)); err != nil {
		return fmt.Errorf("terraform init: %w", err)
	}

	return nil
}

// Plan runs terraform plan
func (e *Executor) Plan(ctx context.Context) (*PlanResult, error) {
	logger.L().Info("running terraform plan", zap.String("working_dir", e.workingDir))

	hasChanges, err := e.tf.Plan(ctx)
	if err != nil {
		return nil, fmt.Errorf("terraform plan: %w", err)
	}

	// Get plan output
	planOutput, err := e.tf.Show(ctx)
	var planOutStr string
	if err != nil {
		logger.L().Warn("failed to get plan output", zap.Error(err))
	} else {
		if b, err := json.Marshal(planOutput); err == nil {
			planOutStr = string(b)
		} else {
			planOutStr = fmt.Sprintf("%v", planOutput)
		}
	}

	return &PlanResult{
		HasChanges: hasChanges,
		PlanOutput: planOutStr,
	}, nil
}

// Apply runs terraform apply
func (e *Executor) Apply(ctx context.Context) (*ApplyResult, error) {
	logger.L().Info("running terraform apply", zap.String("working_dir", e.workingDir))

	if err := e.tf.Apply(ctx); err != nil {
		return nil, fmt.Errorf("terraform apply: %w", err)
	}

	// Get outputs
	outputs, err := e.tf.Output(ctx)
	if err != nil {
		logger.L().Warn("failed to get outputs", zap.Error(err))
	}

	// Get state
	state, err := e.tf.Show(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get state: %w", err)
	}
	var stateBytes []byte
	if b, err := json.Marshal(state); err == nil {
		stateBytes = b
	} else {
		stateBytes = []byte(fmt.Sprintf("%v", state))
	}

	return &ApplyResult{
		Outputs: convertOutputs(outputs),
		State:   stateBytes,
	}, nil
}

// Destroy runs terraform destroy
func (e *Executor) Destroy(ctx context.Context) error {
	logger.L().Info("running terraform destroy", zap.String("working_dir", e.workingDir))

	if err := e.tf.Destroy(ctx); err != nil {
		return fmt.Errorf("terraform destroy: %w", err)
	}

	return nil
}

// Cleanup removes the working directory
func (e *Executor) Cleanup() error {
	return os.RemoveAll(e.workingDir)
}

type TerraformCode struct {
	MainTF      string
	VariablesTF string
	OutputsTF   string
	ProviderTF  string
}

type PlanResult struct {
	HasChanges bool
	PlanOutput string
}

type ApplyResult struct {
	Outputs map[string]interface{}
	State   []byte
}

func convertOutputs(tfOutputs map[string]tfexec.OutputMeta) map[string]interface{} {
	outputs := make(map[string]interface{})
	for key, output := range tfOutputs {
		outputs[key] = output.Value
	}
	return outputs
}
