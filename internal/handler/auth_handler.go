package handler

import (
	"encoding/json"
	"net/http"
	"nodosml-pc4/internal/models"
	"nodosml-pc4/internal/service"
	"strconv"

	"github.com/go-chi/chi/v5"
)

type AuthHandler struct {
	svc *service.AuthService
}

type userResponse struct {
	UserID          int      `json:"userId"`
	UIdx            *int     `json:"uIdx,omitempty"`
	FirstName       string   `json:"firstName,omitempty"`
	LastName        string   `json:"lastName,omitempty"`
	Username        string   `json:"username,omitempty"`
	Email           string   `json:"email"`
	Role            string   `json:"role"`
	About           string   `json:"about,omitempty"`
	PreferredGenres []string `json:"preferredGenres,omitempty"`
	CreatedAt       string   `json:"createdAt"`
	UpdatedAt       string   `json:"updatedAt"`
}

func toUserResponse(u *models.UserDoc) userResponse {
	return userResponse{
		UserID:          u.UserID,
		UIdx:            u.UIdx,
		FirstName:       u.FirstName,
		LastName:        u.LastName,
		Username:        u.Username,
		Email:           u.Email,
		Role:            u.Role,
		About:           u.About,
		PreferredGenres: u.PreferredGenres,
		CreatedAt:       u.CreatedAt,
		UpdatedAt:       u.UpdatedAt,
	}
}

func NewAuthHandler(s *service.AuthService) *AuthHandler {
	return &AuthHandler{svc: s}
}

type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     string `json:"role"`

	FirstName       string   `json:"firstName"`
	LastName        string   `json:"lastName"`
	Username        string   `json:"username"`
	About           string   `json:"about"`
	PreferredGenres []string `json:"preferredGenres"`
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

	u, err := h.svc.Register(r.Context(), service.RegisterUserData{
		Email:           req.Email,
		Password:        req.Password,
		Role:            req.Role,
		FirstName:       req.FirstName,
		LastName:        req.LastName,
		Username:        req.Username,
		About:           req.About,
		PreferredGenres: req.PreferredGenres,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(toUserResponse(u))
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
		"token": token,
		"user":  toUserResponse(u),
	})
}

type updateUserRequest struct {
	Email    *string `json:"email"`
	Role     *string `json:"role"`
	Password *string `json:"password"`

	FirstName       *string   `json:"firstName"`
	LastName        *string   `json:"lastName"`
	Username        *string   `json:"username"`
	About           *string   `json:"about"`
	PreferredGenres *[]string `json:"preferredGenres"`
}

// @Summary Actualizar usuario
// @Description Actualiza los datos de un usuario existente (email, role, password). Todos los campos son opcionales.
// @Tags auth
// @Security BearerAuth
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
		Email:           req.Email,
		Role:            req.Role,
		Password:        req.Password,
		FirstName:       req.FirstName,
		LastName:        req.LastName,
		Username:        req.Username,
		About:           req.About,
		PreferredGenres: req.PreferredGenres,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]any{"updated": true})
}

// @Summary Listar usuarios (ADMIN)
// @Tags users
// @Security BearerAuth
// @Produce json
// @Param role query string false "user|admin|all (default: all)"
// @Param q query string false "búsqueda por email/username/nombre"
// @Param limit query int false "límite (default: 20)"
// @Param offset query int false "offset (default: 0)"
// @Success 200 {array} userResponse
// @Router /users [get]
func (h *AuthHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	role := r.URL.Query().Get("role")
	if role == "" {
		role = "all"
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if limit <= 0 {
		limit = 20
	}
	q := r.URL.Query().Get("q")

	users, err := h.svc.ListUsers(r.Context(), role, q, limit, offset)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp := make([]userResponse, 0, len(users))
	for _, u := range users {
		uCopy := u
		resp = append(resp, toUserResponse(&uCopy))
	}
	_ = json.NewEncoder(w).Encode(resp)
}

// @Summary Obtener usuario por id (ADMIN)
// @Tags users
// @Security BearerAuth
// @Produce json
// @Param id path int true "userId"
// @Success 200 {object} userResponse
// @Router /users/{id} [get]
func (h *AuthHandler) GetUserByID(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	id, _ := strconv.Atoi(chi.URLParam(r, "id"))

	u, err := h.svc.GetUserByID(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if u == nil {
		http.NotFound(w, r)
		return
	}
	_ = json.NewEncoder(w).Encode(toUserResponse(u))
}
