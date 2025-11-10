package compiler

import (
	"fmt"
	"strings"
)

// EC2Compiler compiles aws_instance resources
type EC2Compiler struct{}

func (c *EC2Compiler) Validate(node Node) error {
	required := []string{"ami", "instance_type"}
	for _, field := range required {
		if _, ok := node.Properties[field]; !ok {
			return fmt.Errorf("missing required field: %s", field)
		}
	}
	return nil
}

func (c *EC2Compiler) Compile(node Node) (string, error) {
	var hcl strings.Builder

	hcl.WriteString(fmt.Sprintf(`resource "aws_instance" "%s" {`, node.ID))
	hcl.WriteString(fmt.Sprintf(`
  ami           = "%v"
  instance_type = "%v"
`, node.Properties["ami"], node.Properties["instance_type"]))

	// Optional fields
	if name, ok := node.Properties["name"].(string); ok {
		hcl.WriteString(fmt.Sprintf(`
  tags = merge(var.tags, {
    Name = "%s"
  })
`, name))
	}

	if sg, ok := node.Properties["security_group"].(string); ok {
		hcl.WriteString(fmt.Sprintf(`
  vpc_security_group_ids = [aws_security_group.%s.id]
`, sg))
	}

	if subnet, ok := node.Properties["subnet"].(string); ok {
		hcl.WriteString(fmt.Sprintf(`
  subnet_id = aws_subnet.%s.id
`, subnet))
	}

	hcl.WriteString("}\n")
	return hcl.String(), nil
}

// S3Compiler compiles aws_s3_bucket resources
type S3Compiler struct{}

func (c *S3Compiler) Validate(node Node) error {
	if _, ok := node.Properties["bucket_name"]; !ok {
		return fmt.Errorf("missing required field: bucket_name")
	}
	return nil
}

func (c *S3Compiler) Compile(node Node) (string, error) {
	bucketName := fmt.Sprintf("%v", node.Properties["bucket_name"])

	var hcl strings.Builder
	hcl.WriteString(fmt.Sprintf(`resource "aws_s3_bucket" "%s" {
  bucket = "%s"
  
  tags = var.tags
}
`, node.ID, bucketName))

	// Add versioning if specified
	if versioning, ok := node.Properties["versioning"].(bool); ok && versioning {
		hcl.WriteString(fmt.Sprintf(`
resource "aws_s3_bucket_versioning" "%s_versioning" {
  bucket = aws_s3_bucket.%s.id
  
  versioning_configuration {
    status = "Enabled"
  }
}
`, node.ID, node.ID))
	}

	return hcl.String(), nil
}

// SecurityGroupCompiler compiles aws_security_group resources
type SecurityGroupCompiler struct{}

func (c *SecurityGroupCompiler) Validate(node Node) error {
	required := []string{"name", "description"}
	for _, field := range required {
		if _, ok := node.Properties[field]; !ok {
			return fmt.Errorf("missing required field: %s", field)
		}
	}
	return nil
}

func (c *SecurityGroupCompiler) Compile(node Node) (string, error) {
	var hcl strings.Builder

	hcl.WriteString(fmt.Sprintf(`resource "aws_security_group" "%s" {
  name        = "%v"
  description = "%v"
`, node.ID, node.Properties["name"], node.Properties["description"]))

	if vpc, ok := node.Properties["vpc"].(string); ok {
		hcl.WriteString(fmt.Sprintf(`  vpc_id = aws_vpc.%s.id
`, vpc))
	}

	// Ingress rules
	if ingress, ok := node.Properties["ingress"].([]interface{}); ok {
		for _, rule := range ingress {
			r := rule.(map[string]interface{})
			hcl.WriteString(fmt.Sprintf(`
  ingress {
    from_port   = %v
    to_port     = %v
    protocol    = "%v"
    cidr_blocks = ["%v"]
  }
`, r["from_port"], r["to_port"], r["protocol"], r["cidr_blocks"]))
		}
	}

	// Egress rules
	hcl.WriteString(`
  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
`)

	hcl.WriteString(`
  tags = var.tags
}
`)

	return hcl.String(), nil
}

// Add stubs for other compilers
type RDSCompiler struct{}

func (c *RDSCompiler) Validate(node Node) error { return nil }
func (c *RDSCompiler) Compile(node Node) (string, error) {
	return "# RDS not implemented yet\n", nil
}

type VPCCompiler struct{}

func (c *VPCCompiler) Validate(node Node) error { return nil }
func (c *VPCCompiler) Compile(node Node) (string, error) {
	return "# VPC not implemented yet\n", nil
}

type SubnetCompiler struct{}

func (c *SubnetCompiler) Validate(node Node) error { return nil }
func (c *SubnetCompiler) Compile(node Node) (string, error) {
	return "# Subnet not implemented yet\n", nil
}
