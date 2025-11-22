package service

import (
	"context"
	"fmt"
	"time"

	"nodosml-pc4/internal/models"
	"nodosml-pc4/internal/repository"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	users     *repository.UserRepository
	jwtSecret []byte
}

func NewAuthService(users *repository.UserRepository, secret string) *AuthService {
	return &AuthService{users: users, jwtSecret: []byte(secret)}
}

// ================== REGISTER & LOGIN ==================

// Register crea un usuario nuevo. El role viene del body, pero solo se permite "user" o "admin".
func (s *AuthService) Register(ctx context.Context, email, password, role string) (*models.UserDoc, error) {
	existing, err := s.users.FindByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, fmt.Errorf("email already registered")
	}

	nextID, err := s.users.GetNextUserID(ctx)
	if err != nil {
		return nil, err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	// normalizar role
	if role == "" {
		role = "user"
	}
	if role != "user" && role != "admin" {
		return nil, fmt.Errorf("invalid role (must be user|admin)")
	}

	u := &models.UserDoc{
		UserID:       nextID,
		Email:        email,
		PasswordHash: string(hash),
		Role:         role,
		CreatedAt:    time.Now().UTC().Format(time.RFC3339),
	}

	if err := s.users.Insert(ctx, u); err != nil {
		return nil, err
	}
	return u, nil
}

func (s *AuthService) Login(ctx context.Context, email, password string) (string, *models.UserDoc, error) {
	u, err := s.users.FindByEmail(ctx, email)
	if err != nil {
		return "", nil, err
	}
	if u == nil {
		return "", nil, fmt.Errorf("invalid credentials")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err != nil {
		return "", nil, fmt.Errorf("invalid credentials")
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":  u.UserID,
		"role": u.Role,
		"exp":  time.Now().Add(24 * time.Hour).Unix(),
	})
	sToken, err := token.SignedString(s.jwtSecret)
	if err != nil {
		return "", nil, err
	}
	return sToken, u, nil
}

// ================== UPDATE USER ==================

type UpdateUserData struct {
	Email    *string
	Role     *string
	Password *string
}

// UpdateUser actualiza campos opcionales de un usuario.
func (s *AuthService) UpdateUser(ctx context.Context, userID int, data UpdateUserData) error {
	// Verificar que el usuario exista
	u, err := s.users.FindByID(ctx, userID)
	if err != nil {
		return err
	}
	if u == nil {
		return fmt.Errorf("user not found")
	}

	update := map[string]any{}

	// Email
	if data.Email != nil {
		if *data.Email == "" {
			return fmt.Errorf("email cannot be empty")
		}
		// Revisar que no esté usado por otro usuario
		existing, err := s.users.FindByEmail(ctx, *data.Email)
		if err != nil {
			return err
		}
		if existing != nil && existing.UserID != userID {
			return fmt.Errorf("email already in use")
		}
		update["email"] = *data.Email
	}

	// Role
	if data.Role != nil {
		if *data.Role != "user" && *data.Role != "admin" {
			return fmt.Errorf("invalid role (must be user|admin)")
		}
		update["role"] = *data.Role
	}

	// Password
	if data.Password != nil {
		if *data.Password == "" {
			return fmt.Errorf("password cannot be empty")
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(*data.Password), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		update["passwordHash"] = string(hash)
	}

	if len(update) == 0 {
		return fmt.Errorf("no fields to update")
	}

	return s.users.UpdateByID(ctx, userID, update)
}
