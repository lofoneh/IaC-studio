package handlers

import (
    "net/http"
    "github.com/iac-studio/engine/internal/api/types"
    "encoding/json"
)

type HealthHandler struct{}

func NewHealthHandler() *HealthHandler { return &HealthHandler{} }

func (h *HealthHandler) Liveness(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(types.APIResponse{Success: true, Data: map[string]string{"status": "ok"}})
}

func (h *HealthHandler) Readiness(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(types.APIResponse{Success: true, Data: map[string]string{"status": "ready"}})
}


