package types

type RegisterRequest struct {
    Email    string `json:"email" validate:"required,email"`
    Password string `json:"password" validate:"required,min=8"`
    Name     string `json:"name" validate:"required"`
}

type LoginRequest struct {
    Email    string `json:"email" validate:"required,email"`
    Password string `json:"password" validate:"required"`
}

type ProjectCreateRequest struct {
    Name          string `json:"name" validate:"required"`
    Description   string `json:"description"`
    CloudProvider string `json:"cloud_provider" validate:"required,oneof=aws gcp azure do"`
}

type ProjectUpdateRequest struct {
    Description   string `json:"description"`
    CloudProvider string `json:"cloud_provider" validate:"omitempty,oneof=aws gcp azure do"`
    Archived      *bool  `json:"archived"`
}

type DeploymentCreateRequest struct {
    GraphID string `json:"graph_id" validate:"required,uuid4"`
}


