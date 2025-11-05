package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/iac-studio/engine/internal/api/types"
	"github.com/iac-studio/engine/internal/models"
	"github.com/iac-studio/engine/internal/repository"
)

type AuthHandler struct {
	users      repository.UserRepository
	hmacSecret []byte
}

func NewAuthHandler(users repository.UserRepository, secret []byte) *AuthHandler {
	return &AuthHandler{users: users, hmacSecret: secret}
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		Name     string `json:"name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorStr(w, http.StatusBadRequest, "invalid json")
		return
	}

	// Validate input
	if req.Email == "" || req.Password == "" || req.Name == "" {
		writeErrorStr(w, http.StatusBadRequest, "email, password, and name are required")
		return
	}

	if len(req.Password) < 8 {
		writeErrorStr(w, http.StatusBadRequest, "password must be at least 8 characters")
		return
	}

	// Hash password
	ph, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	// Create user
	u := models.User{
		Email:        req.Email,
		PasswordHash: string(ph),
		Name:         req.Name,
	}

	if err := h.users.Create(r.Context(), &u); err != nil {
		writeErrorStr(w, http.StatusConflict, "email already exists")
		return
	}

	writeJSON(w, http.StatusCreated, types.APIResponse{
		Success: true,
		Data: map[string]any{
			"id":    u.ID,
			"email": u.Email,
			"name":  u.Name,
		},
	})
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorStr(w, http.StatusBadRequest, "invalid json")
		return
	}

	if req.Email == "" || req.Password == "" {
		writeErrorStr(w, http.StatusBadRequest, "email and password are required")
		return
	}

	// Get user
	var u models.User
	if err := h.users.GetByEmail(r.Context(), req.Email, &u); err != nil {
		writeErrorStr(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(req.Password)); err != nil {
		writeErrorStr(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	// Generate JWT
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": u.ID.String(),
		"exp": time.Now().Add(24 * time.Hour).Unix(),
	})

	tokenString, err := token.SignedString(h.hmacSecret)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(w, http.StatusOK, types.APIResponse{
		Success: true,
		Data: map[string]any{
			"access_token": tokenString,
			"token_type":   "Bearer",
			"expires_in":   86400,
			"user": map[string]any{
				"id":    u.ID,
				"email": u.Email,
				"name":  u.Name,
			},
		},
	})
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, types.APIResponse{Success: true})
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "not implemented", http.StatusNotImplemented)
}
