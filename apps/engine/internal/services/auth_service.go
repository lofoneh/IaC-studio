// internal/services/auth_service.go
package services

import (
    "context"
    "fmt"
    "time"

    "github.com/golang-jwt/jwt/v5"
    "golang.org/x/crypto/bcrypt"
    "github.com/iac-studio/engine/internal/models"
    "github.com/iac-studio/engine/internal/repository"
)

type AuthService interface {
    Register(ctx context.Context, email, password, name string) (*models.User, error)
    Login(ctx context.Context, email, password string) (string, *models.User, error)
}

type authService struct {
    userRepo   repository.UserRepository
    hmacSecret []byte
}

func NewAuthService(userRepo repository.UserRepository, secret []byte) AuthService {
    return &authService{
        userRepo:   userRepo,
        hmacSecret: secret,
    }
}

func (s *authService) Register(ctx context.Context, email, password, name string) (*models.User, error) {
    // Hash password
    ph, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
    if err != nil {
        return nil, fmt.Errorf("hash password: %w", err)
    }

    user := &models.User{
        Email:        email,
        PasswordHash: string(ph),
        Name:         name,
    }

    if err := s.userRepo.Create(ctx, user); err != nil {
        return nil, fmt.Errorf("create user: %w", err)
    }

    return user, nil
}

func (s *authService) Login(ctx context.Context, email, password string) (string, *models.User, error) {
    var user models.User
    if err := s.userRepo.GetByEmail(ctx, email, &user); err != nil {
        return "", nil, fmt.Errorf("invalid credentials")
    }

    if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
        return "", nil, fmt.Errorf("invalid credentials")
    }

    // Generate JWT
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
        "sub": user.ID.String(),
        "exp": time.Now().Add(24 * time.Hour).Unix(),
    })

    tokenString, err := token.SignedString(s.hmacSecret)
    if err != nil {
        return "", nil, fmt.Errorf("sign token: %w", err)
    }

    return tokenString, &user, nil
}