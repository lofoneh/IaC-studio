package handlers

import (
	"encoding/json"
	"github.com/google/uuid"
	"github.com/iac-studio/engine/internal/api/middleware"
	"github.com/iac-studio/engine/internal/api/types"
	"github.com/iac-studio/engine/internal/api/validators"
	"github.com/iac-studio/engine/internal/models"
	"github.com/iac-studio/engine/internal/repository"
	"net/http"
	"strconv"
)

type ProjectsHandler struct {
	repo     repository.ProjectRepository
	validate interface{ Struct(any) error }
}

func NewProjectsHandler(repo repository.ProjectRepository, v interface{ Struct(any) error }) *ProjectsHandler {
	return &ProjectsHandler{repo: repo, validate: v}
}

func (h *ProjectsHandler) List(w http.ResponseWriter, r *http.Request) {
	uidStr := middleware.GetUserID(r.Context())
	userID, _ := uuid.Parse(uidStr)
	items, err := h.repo.ListByUser(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
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
	if start > len(items) {
		start = len(items)
	}
	if end > len(items) {
		end = len(items)
	}
	resp := types.APIResponse{Success: true, Data: items[start:end], Meta: &types.Meta{Page: page, PageSize: size, Total: int64(len(items))}}
	writeJSON(w, http.StatusOK, resp)
}

func (h *ProjectsHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name          string `json:"name" validate:"required"`
		Description   string `json:"description"`
		CloudProvider string `json:"cloud_provider" validate:"required,oneof=aws gcp azure do"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorStr(w, http.StatusBadRequest, "invalid json")
		return
	}
	if err := validators.New().Struct(req); err != nil {
		writeErrorStr(w, http.StatusBadRequest, err.Error())
		return
	}
	uid, _ := uuid.Parse(middleware.GetUserID(r.Context()))
	p := models.Project{UserID: uid, Name: req.Name, Description: req.Description, CloudProvider: req.CloudProvider}
	if err := h.repo.Create(r.Context(), &p); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusCreated, types.APIResponse{Success: true, Data: p})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, types.APIResponse{Success: false, Error: types.FromAppError(err)})
}

func writeErrorStr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, types.APIResponse{Success: false, Error: &types.APIError{Code: "invalid", Message: msg}})
}
