package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/iac-studio/engine/internal/api/middleware"
	"github.com/iac-studio/engine/internal/api/types"
	"github.com/iac-studio/engine/internal/models"
	"github.com/iac-studio/engine/internal/repository"
)

type ProjectsHandler struct {
	repo repository.ProjectRepository
}

func NewProjectsHandler(repo repository.ProjectRepository, _ interface{}) *ProjectsHandler {
	return &ProjectsHandler{repo: repo}
}

// CreateProjectRequest represents project creation data
type CreateProjectRequest struct {
	Name          string `json:"name" example:"My AWS Infrastructure"`
	Description   string `json:"description" example:"Production infrastructure for web application"`
	CloudProvider string `json:"cloud_provider" example:"aws" enums:"aws,gcp,azure,do"`
}

// UpdateProjectRequest represents project update data
type UpdateProjectRequest struct {
	Description   *string `json:"description,omitempty" example:"Updated description"`
	CloudProvider *string `json:"cloud_provider,omitempty" example:"aws"`
	Archived      *bool   `json:"archived,omitempty" example:"false"`
}

// List godoc
// @Summary      List projects
// @Description  Get all projects for authenticated user with pagination
// @Tags         Projects
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        page query int false "Page number" default(1)
// @Param        page_size query int false "Items per page" default(20)
// @Success      200 {object} types.APIResponse{data=[]models.Project}
// @Failure      401 {object} types.APIResponse{error=types.APIError}
// @Failure      500 {object} types.APIResponse{error=types.APIError}
// @Router       /projects [get]
func (h *ProjectsHandler) List(w http.ResponseWriter, r *http.Request) {
	// Get user ID from auth middleware
	uidStr := middleware.GetUserID(r.Context())
	userID, err := uuid.Parse(uidStr)
	if err != nil {
		writeErrorStr(w, http.StatusUnauthorized, "invalid user id")
		return
	}

	// Get projects from database
	items, err := h.repo.ListByUser(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	// Pagination
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	size, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
	if page <= 0 {
		page = 1
	}
	if size <= 0 || size > 100 {
		size = 20
	}

	start := (page - 1) * size
	end := start + size
	total := len(items)

	if start > total {
		start = total
	}
	if end > total {
		end = total
	}

	paginatedItems := items[start:end]

	resp := types.APIResponse{
		Success: true,
		Data:    paginatedItems,
		Meta: &types.Meta{
			Page:     page,
			PageSize: size,
			Total:    int64(total),
		},
	}

	writeJSON(w, http.StatusOK, resp)
}

// Create godoc
// @Summary      Create project
// @Description  Create a new infrastructure project
// @Tags         Projects
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body CreateProjectRequest true "Project details"
// @Success      201 {object} types.APIResponse{data=models.Project}
// @Failure      400 {object} types.APIResponse{error=types.APIError}
// @Failure      401 {object} types.APIResponse{error=types.APIError}
// @Failure      500 {object} types.APIResponse{error=types.APIError}
// @Router       /projects [post]
func (h *ProjectsHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorStr(w, http.StatusBadRequest, "invalid json")
		return
	}

	// Validate required fields
	if req.Name == "" {
		writeErrorStr(w, http.StatusBadRequest, "name is required")
		return
	}
	if req.CloudProvider == "" {
		writeErrorStr(w, http.StatusBadRequest, "cloud_provider is required")
		return
	}

	// Validate cloud provider
	validProviders := map[string]bool{"aws": true, "gcp": true, "azure": true, "do": true}
	if !validProviders[req.CloudProvider] {
		writeErrorStr(w, http.StatusBadRequest, "cloud_provider must be one of: aws, gcp, azure, do")
		return
	}

	// Get user ID
	uidStr := middleware.GetUserID(r.Context())
	userID, err := uuid.Parse(uidStr)
	if err != nil {
		writeErrorStr(w, http.StatusUnauthorized, "invalid user id")
		return
	}

	// Create project
	project := models.Project{
		UserID:        userID,
		Name:          req.Name,
		Description:   req.Description,
		CloudProvider: req.CloudProvider,
	}

	if err := h.repo.Create(r.Context(), &project); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(w, http.StatusCreated, types.APIResponse{
		Success: true,
		Data:    project,
	})
}

// Get godoc
// @Summary      Get project
// @Description  Get a specific project by ID
// @Tags         Projects
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Project ID" format(uuid)
// @Success      200 {object} types.APIResponse{data=models.Project}
// @Failure      400 {object} types.APIResponse{error=types.APIError}
// @Failure      401 {object} types.APIResponse{error=types.APIError}
// @Failure      404 {object} types.APIResponse{error=types.APIError}
// @Router       /projects/{id} [get]
func (h *ProjectsHandler) Get(w http.ResponseWriter, r *http.Request) {
	// Get project ID from URL
	projectIDStr := chi.URLParam(r, "id")
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		writeErrorStr(w, http.StatusBadRequest, "invalid project id")
		return
	}

	// Get user ID
	uidStr := middleware.GetUserID(r.Context())
	userID, err := uuid.Parse(uidStr)
	if err != nil {
		writeErrorStr(w, http.StatusUnauthorized, "invalid user id")
		return
	}

	// Get project
	var project models.Project
	if err := h.repo.GetByID(r.Context(), projectID, &project); err != nil {
		writeErrorStr(w, http.StatusNotFound, "project not found")
		return
	}

	// Check ownership
	if project.UserID != userID {
		writeErrorStr(w, http.StatusForbidden, "access denied")
		return
	}

	writeJSON(w, http.StatusOK, types.APIResponse{
		Success: true,
		Data:    project,
	})
}

// Update godoc
// @Summary      Update project
// @Description  Update an existing project
// @Tags         Projects
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Project ID" format(uuid)
// @Param        request body UpdateProjectRequest true "Update details"
// @Success      200 {object} types.APIResponse{data=models.Project}
// @Failure      400 {object} types.APIResponse{error=types.APIError}
// @Failure      401 {object} types.APIResponse{error=types.APIError}
// @Failure      403 {object} types.APIResponse{error=types.APIError}
// @Failure      404 {object} types.APIResponse{error=types.APIError}
// @Router       /projects/{id} [put]
func (h *ProjectsHandler) Update(w http.ResponseWriter, r *http.Request) {
	// Get project ID
	projectIDStr := chi.URLParam(r, "id")
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		writeErrorStr(w, http.StatusBadRequest, "invalid project id")
		return
	}

	// Get user ID
	uidStr := middleware.GetUserID(r.Context())
	userID, err := uuid.Parse(uidStr)
	if err != nil {
		writeErrorStr(w, http.StatusUnauthorized, "invalid user id")
		return
	}

	// Get existing project
	var project models.Project
	if err := h.repo.GetByID(r.Context(), projectID, &project); err != nil {
		writeErrorStr(w, http.StatusNotFound, "project not found")
		return
	}

	// Check ownership
	if project.UserID != userID {
		writeErrorStr(w, http.StatusForbidden, "access denied")
		return
	}

	// Parse update request
	var req UpdateProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorStr(w, http.StatusBadRequest, "invalid json")
		return
	}

	// Apply updates
	if req.Description != nil {
		project.Description = *req.Description
	}
	if req.CloudProvider != nil {
		// Validate provider
		validProviders := map[string]bool{"aws": true, "gcp": true, "azure": true, "do": true}
		if !validProviders[*req.CloudProvider] {
			writeErrorStr(w, http.StatusBadRequest, "cloud_provider must be one of: aws, gcp, azure, do")
			return
		}
		project.CloudProvider = *req.CloudProvider
	}
	if req.Archived != nil {
		project.Archived = *req.Archived
	}

	// Save updates
	if err := h.repo.Update(r.Context(), &project); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(w, http.StatusOK, types.APIResponse{
		Success: true,
		Data:    project,
	})
}

// Delete godoc
// @Summary      Delete project
// @Description  Archive a project (soft delete)
// @Tags         Projects
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Project ID" format(uuid)
// @Success      200 {object} types.APIResponse
// @Failure      400 {object} types.APIResponse{error=types.APIError}
// @Failure      401 {object} types.APIResponse{error=types.APIError}
// @Failure      403 {object} types.APIResponse{error=types.APIError}
// @Failure      404 {object} types.APIResponse{error=types.APIError}
// @Router       /projects/{id} [delete]
func (h *ProjectsHandler) Delete(w http.ResponseWriter, r *http.Request) {
	// Get project ID
	projectIDStr := chi.URLParam(r, "id")
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		writeErrorStr(w, http.StatusBadRequest, "invalid project id")
		return
	}

	// Get user ID
	uidStr := middleware.GetUserID(r.Context())
	userID, err := uuid.Parse(uidStr)
	if err != nil {
		writeErrorStr(w, http.StatusUnauthorized, "invalid user id")
		return
	}

	// Get project to check ownership
	var project models.Project
	if err := h.repo.GetByID(r.Context(), projectID, &project); err != nil {
		writeErrorStr(w, http.StatusNotFound, "project not found")
		return
	}

	// Check ownership
	if project.UserID != userID {
		writeErrorStr(w, http.StatusForbidden, "access denied")
		return
	}

	// Archive project
	if err := h.repo.Archive(r.Context(), projectID); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(w, http.StatusOK, types.APIResponse{
		Success: true,
		Data:    map[string]string{"message": "project archived successfully"},
	})
}

// Helper functions
func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, types.APIResponse{
		Success: false,
		Error: &types.APIError{
			Code:    http.StatusText(status),
			Message: err.Error(),
		},
	})
}

func writeErrorStr(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, types.APIResponse{
		Success: false,
		Error: &types.APIError{
			Code:    http.StatusText(status),
			Message: message,
		},
	})
}