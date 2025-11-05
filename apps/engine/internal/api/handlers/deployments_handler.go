package handlers

import (
    "net/http"
    "github.com/google/uuid"
    "github.com/iac-studio/engine/internal/api/types"
    "github.com/iac-studio/engine/internal/repository"
)

type DeploymentsHandler struct { repo repository.DeploymentRepository }

func NewDeploymentsHandler(repo repository.DeploymentRepository) *DeploymentsHandler { return &DeploymentsHandler{repo: repo} }

func (h *DeploymentsHandler) List(w http.ResponseWriter, r *http.Request) {
    pidStr := r.URL.Query().Get("project_id")
    pid, _ := uuid.Parse(pidStr)
    items, err := h.repo.ListByProject(r.Context(), pid)
    if err != nil { writeError(w, http.StatusInternalServerError, err); return }
    writeJSON(w, http.StatusOK, types.APIResponse{ Success: true, Data: items })
}

func (h *DeploymentsHandler) Create(w http.ResponseWriter, r *http.Request) {
    // placeholder: would create a record and enqueue job
    writeJSON(w, http.StatusCreated, types.APIResponse{ Success: true })
}




