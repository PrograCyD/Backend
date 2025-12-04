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

type RegisterUserData struct {
	Email    string
	Password string
	Role     string

	FirstName string
	LastName  string
	Username  string
	About     string

	PreferredGenres []string
}

type UpdateUserData struct {
	Email    *string
	Role     *string
	Password *string

	FirstName       *string
	LastName        *string
	Username        *string
	About           *string
	PreferredGenres *[]string
}

func NewAuthService(users *repository.UserRepository, secret string) *AuthService {
	return &AuthService{users: users, jwtSecret: []byte(secret)}
}

// ================== REGISTER & LOGIN ==================

// Register crea un usuario nuevo. El role viene del body, pero solo se permite "user" o "admin".
// Register crea un usuario nuevo.
func (s *AuthService) Register(ctx context.Context, data RegisterUserData) (*models.UserDoc, error) {
	existing, err := s.users.FindByEmail(ctx, data.Email)
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

	nextUIdx, err := s.users.GetNextUIdx(ctx)
	if err != nil {
		return nil, err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(data.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	role := data.Role
	if role == "" {
		role = "user"
	}
	if role != "user" && role != "admin" {
		return nil, fmt.Errorf("invalid role (must be user|admin)")
	}

	now := time.Now().UTC().Format(time.RFC3339)

	u := &models.UserDoc{
		UserID:       nextID,
		UIdx:         nextUIdx,
		Email:        data.Email,
		PasswordHash: string(hash),
		Role:         role,
		CreatedAt:    now,
		UpdatedAt:    now,

		FirstName:       data.FirstName,
		LastName:        data.LastName,
		Username:        data.Username,
		About:           data.About,
		PreferredGenres: data.PreferredGenres,
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

// UpdateUser actualiza campos opcionales de un usuario.
func (s *AuthService) UpdateUser(ctx context.Context, userID int, data UpdateUserData) error {
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

	// Campos de perfil
	if data.FirstName != nil {
		update["firstName"] = *data.FirstName
	}
	if data.LastName != nil {
		update["lastName"] = *data.LastName
	}
	if data.Username != nil {
		update["username"] = *data.Username
	}
	if data.About != nil {
		update["about"] = *data.About
	}
	if data.PreferredGenres != nil {
		update["preferredGenres"] = *data.PreferredGenres
	}

	if len(update) == 0 {
		return fmt.Errorf("no fields to update")
	}

	update["updatedAt"] = time.Now().UTC().Format(time.RFC3339)

	return s.users.UpdateByID(ctx, userID, update)
}

func (s *AuthService) ListUsers(ctx context.Context, role, q string, limit, offset int) ([]models.UserDoc, error) {
	return s.users.Search(ctx, role, q, limit, offset)
}

func (s *AuthService) GetUserByID(ctx context.Context, userID int) (*models.UserDoc, error) {
	return s.users.FindByID(ctx, userID)
}
