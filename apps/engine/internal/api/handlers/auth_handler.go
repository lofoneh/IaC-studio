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

// RegisterRequest represents the registration payload
type RegisterRequest struct {
	Email    string `json:"email" example:"user@example.com"`
	Password string `json:"password" example:"password123"`
	Name     string `json:"name" example:"John Doe"`
}

// LoginRequest represents the login payload
type LoginRequest struct {
	Email    string `json:"email" example:"user@example.com"`
	Password string `json:"password" example:"password123"`
}

// AuthResponse represents the authentication response
type AuthResponse struct {
	AccessToken string      `json:"access_token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
	TokenType   string      `json:"token_type" example:"Bearer"`
	ExpiresIn   int         `json:"expires_in" example:"86400"`
	User        models.User `json:"user"`
}

// Register godoc
// @Summary      Register a new user
// @Description  Create a new user account with email and password
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body RegisterRequest true "Registration details"
// @Success      201 {object} types.APIResponse{data=map[string]interface{}}
// @Failure      400 {object} types.APIResponse{error=types.APIError}
// @Failure      409 {object} types.APIResponse{error=types.APIError}
// @Failure      500 {object} types.APIResponse{error=types.APIError}
// @Router       /auth/register [post]
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorStr(w, 400, "invalid json")
		return
	}

	// Validate input
	if req.Email == "" || req.Password == "" || req.Name == "" {
		writeErrorStr(w, 400, "email, password, and name are required")
		return
	}

	if len(req.Password) < 8 {
		writeErrorStr(w, 400, "password must be at least 8 characters")
		return
	}

	ph, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		writeError(w, 500, err)
		return
	}

	u := models.User{
		Email:        req.Email,
		PasswordHash: string(ph),
		Name:         req.Name,
	}

	if err := h.users.Create(r.Context(), &u); err != nil {
		writeErrorStr(w, 409, "email already exists")
		return
	}

	writeJSON(w, 201, types.APIResponse{
		Success: true,
		Data: map[string]any{
			"id":    u.ID,
			"email": u.Email,
			"name":  u.Name,
		},
	})
}

// Login godoc
// @Summary      Login
// @Description  Authenticate user and return JWT token
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body LoginRequest true "Login credentials"
// @Success      200 {object} types.APIResponse{data=AuthResponse}
// @Failure      400 {object} types.APIResponse{error=types.APIError}
// @Failure      401 {object} types.APIResponse{error=types.APIError}
// @Failure      500 {object} types.APIResponse{error=types.APIError}
// @Router       /auth/login [post]
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorStr(w, 400, "invalid json")
		return
	}

	if req.Email == "" || req.Password == "" {
		writeErrorStr(w, 400, "email and password are required")
		return
	}

	var u models.User
	if err := h.users.GetByEmail(r.Context(), req.Email, &u); err != nil {
		writeErrorStr(w, 401, "invalid credentials")
		return
	}

	if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(req.Password)) != nil {
		writeErrorStr(w, 401, "invalid credentials")
		return
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": u.ID.String(),
		"exp": time.Now().Add(24 * time.Hour).Unix(),
	})

	s, err := token.SignedString(h.hmacSecret)
	if err != nil {
		writeError(w, 500, err)
		return
	}

	writeJSON(w, 200, types.APIResponse{
		Success: true,
		Data: map[string]any{
			"access_token": s,
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

// Logout godoc
// @Summary      Logout
// @Description  Logout user (client-side token removal)
// @Tags         auth
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} types.APIResponse
// @Router       /auth/logout [post]
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, types.APIResponse{Success: true})
}

// Refresh godoc
// @Summary      Refresh token
// @Description  Refresh JWT token (not implemented yet)
// @Tags         auth
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} types.APIResponse
// @Failure      501 {object} types.APIResponse
// @Router       /auth/refresh [post]
func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "not implemented", http.StatusNotImplemented)
}

// Helper functions
// func writeJSON(w http.ResponseWriter, status int, v interface{}) {
// 	w.Header().Set("Content-Type", "application/json")
// 	w.WriteHeader(status)
// 	json.NewEncoder(w).Encode(v)
// }

// func writeError(w http.ResponseWriter, status int, err error) {
// 	writeJSON(w, status, types.APIResponse{
// 		Success: false,
// 		Error: &types.APIError{
// 			Code:    http.StatusText(status),
// 			Message: err.Error(),
// 		},
// 	})
// }

// func writeErrorStr(w http.ResponseWriter, status int, message string) {
// 	writeJSON(w, status, types.APIResponse{
// 		Success: false,
// 		Error: &types.APIError{
// 			Code:    http.StatusText(status),
// 			Message: message,
// 		},
// 	})
// }