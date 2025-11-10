package compiler

import (
	"fmt"
	"strings"
)

// Compiler converts visual graphs to Terraform HCL
type Compiler struct {
	resourceCompilers map[string]ResourceCompiler
}

type ResourceCompiler interface {
	Compile(node Node) (string, error)
	Validate(node Node) error
}

type TerraformCode struct {
	MainTF      string
	VariablesTF string
	OutputsTF   string
	ProviderTF  string
}

func NewCompiler() *Compiler {
	c := &Compiler{
		resourceCompilers: make(map[string]ResourceCompiler),
	}

	// Register AWS resource compilers
	c.RegisterCompiler("aws_instance", &EC2Compiler{})
	c.RegisterCompiler("aws_s3_bucket", &S3Compiler{})
	c.RegisterCompiler("aws_security_group", &SecurityGroupCompiler{})
	c.RegisterCompiler("aws_db_instance", &RDSCompiler{})
	c.RegisterCompiler("aws_vpc", &VPCCompiler{})
	c.RegisterCompiler("aws_subnet", &SubnetCompiler{})

	return c
}

func (c *Compiler) RegisterCompiler(resourceType string, compiler ResourceCompiler) {
	c.resourceCompilers[resourceType] = compiler
}

type Graph struct {
	Nodes []Node `json:"nodes"`
	Edges []Edge `json:"edges"`
}

type Node struct {
	ID         string                 `json:"id"`
	Type       string                 `json:"type"`
	Properties map[string]interface{} `json:"properties"`
}

type Edge struct {
	ID   string `json:"id"`
	From string `json:"from"`
	To   string `json:"to"`
	Type string `json:"type"`
}

type CloudConfig struct {
	Provider string `json:"provider"`
	Region   string `json:"region"`
}

func (c *Compiler) Compile(graph Graph, cloudConfig CloudConfig) (*TerraformCode, error) {
	var mainTF strings.Builder
	var outputsTF strings.Builder

	// Generate provider configuration
	providerTF := c.generateProvider(cloudConfig)

	// Compile each node
	for _, node := range graph.Nodes {
		compiler, exists := c.resourceCompilers[node.Type]
		if !exists {
			return nil, fmt.Errorf("unsupported resource type: %s", node.Type)
		}

		if err := compiler.Validate(node); err != nil {
			return nil, fmt.Errorf("validation failed for %s: %w", node.ID, err)
		}

		hcl, err := compiler.Compile(node)
		if err != nil {
			return nil, fmt.Errorf("compilation failed for %s: %w", node.ID, err)
		}

		mainTF.WriteString(hcl)
		mainTF.WriteString("\n\n")

		// Generate output for this resource
		outputsTF.WriteString(c.generateOutput(node))
		outputsTF.WriteString("\n")
	}

	return &TerraformCode{
		MainTF:      mainTF.String(),
		VariablesTF: c.generateVariables(),
		OutputsTF:   outputsTF.String(),
		ProviderTF:  providerTF,
	}, nil
}

func (c *Compiler) generateProvider(config CloudConfig) string {
	switch config.Provider {
	case "aws":
		return fmt.Sprintf(`
terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

provider "aws" {
  region = "%s"
}
`, config.Region)
	default:
		return ""
	}
}

func (c *Compiler) generateOutput(node Node) string {
	return fmt.Sprintf(`
output "%s_id" {
  value       = %s.%s.id
  description = "ID of %s"
}
`, node.ID, node.Type, node.ID, node.ID)
}

func (c *Compiler) generateVariables() string {
	return `
variable "tags" {
  description = "Common tags for all resources"
  type        = map(string)
  default     = {
    ManagedBy = "IaC-Studio"
  }
}
`
}
