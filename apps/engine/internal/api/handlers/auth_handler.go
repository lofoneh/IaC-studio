package handlers

import (
    "encoding/json"
    "net/http"
    "time"
    "golang.org/x/crypto/bcrypt"
    "github.com/golang-jwt/jwt/v5"
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
    var req struct{ Email, Password, Name string }
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil { writeErrorStr(w, 400, "invalid json"); return }
    ph, _ := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
    u := models.User{ Email: req.Email, PasswordHash: string(ph), Name: req.Name }
    if err := h.users.Create(r.Context(), &u); err != nil { writeError(w, 500, err); return }
    writeJSON(w, 201, types.APIResponse{ Success: true, Data: map[string]any{"id": u.ID} })
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
    var req struct{ Email, Password string }
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil { writeErrorStr(w, 400, "invalid json"); return }
    var u models.User
    if err := h.users.GetByEmail(r.Context(), req.Email, &u); err != nil { writeErrorStr(w, 401, "invalid credentials"); return }
    if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(req.Password)) != nil { writeErrorStr(w, 401, "invalid credentials"); return }
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"sub": u.ID.String(), "exp": time.Now().Add(24*time.Hour).Unix()})
    s, _ := token.SignedString(h.hmacSecret)
    writeJSON(w, 200, types.APIResponse{ Success: true, Data: map[string]string{"access_token": s} })
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) { writeJSON(w, 200, types.APIResponse{ Success: true }) }
func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) { http.Error(w, "not implemented", 501) }


