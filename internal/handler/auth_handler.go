package handler

import (
	"encoding/json"
	"net/http"
	"nodosml-pc4/internal/service"
	"strconv"

	"github.com/go-chi/chi/v5"
)

type AuthHandler struct {
	svc *service.AuthService
}

func NewAuthHandler(s *service.AuthService) *AuthHandler {
	return &AuthHandler{svc: s}
}

type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

// @Summary Register
// @Description Crea un usuario nuevo
// @Tags auth
// @Accept json
// @Produce json
// @Param body body registerRequest true "datos"
// @Success 201 {object} map[string]any
// @Failure 400 {object} map[string]string
// @Router /auth/register [post]
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	u, err := h.svc.Register(r.Context(), req.Email, req.Password, req.Role)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(u)
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// @Summary Login
// @Tags auth
// @Accept json
// @Produce json
// @Param body body loginRequest true "credenciales"
// @Success 200 {object} map[string]any
// @Router /auth/login [post]
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	token, u, err := h.svc.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]any{
		"token":  token,
		"userId": u.UserID,
		"role":   u.Role,
	})
}

type updateUserRequest struct {
	Email    *string `json:"email"`
	Role     *string `json:"role"`
	Password *string `json:"password"`
}

// @Summary Actualizar usuario
// @Description Actualiza los datos de un usuario existente (email, role, password). Todos los campos son opcionales.
// @Tags auth
// @Accept json
// @Produce json
// @Param id path int true "userId"
// @Param body body updateUserRequest true "datos a actualizar"
// @Success 200 {object} map[string]any
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /users/{id}/update [put]
func (h *AuthHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	id, _ := strconv.Atoi(chi.URLParam(r, "id"))

	var req updateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err := h.svc.UpdateUser(r.Context(), id, service.UpdateUserData{
		Email:    req.Email,
		Role:     req.Role,
		Password: req.Password,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]any{"updated": true})
}
