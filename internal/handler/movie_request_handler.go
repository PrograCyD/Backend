package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"nodosml-pc4/internal/models"
	"nodosml-pc4/internal/service"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type MovieRequestHandler struct {
	svc *service.MovieRequestService
}

func NewMovieRequestHandler(s *service.MovieRequestService) *MovieRequestHandler {
	return &MovieRequestHandler{svc: s}
}

// ===== USER: crear y listar mis requests =====

// @Summary Crear request de nueva película
// @Tags movie-requests
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param body body models.MovieCreateRequest true "Datos propuestos de película"
// @Success 201 {object} models.MovieRequest
// @Router /me/movie-requests [post]
func (h *MovieRequestHandler) Create(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	userID := UserIDFromContext(r.Context())
	if userID == 0 {
		http.Error(w, "no user in context", http.StatusUnauthorized)
		return
	}

	var req models.MovieCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Title == "" {
		http.Error(w, "body inválido (title requerido)", http.StatusBadRequest)
		return
	}

	mr, err := h.svc.CreateRequest(r.Context(), userID, &req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(mr)
}

// @Summary Listar mis requests de película
// @Tags movie-requests
// @Security BearerAuth
// @Produce json
// @Param status query string false "pending|approved|rejected|all (default: pending)"
// @Param limit query int false "límite (default: 20)"
// @Param offset query int false "offset (default: 0)"
// @Success 200 {array} models.MovieRequest
// @Router /me/movie-requests [get]
func (h *MovieRequestHandler) ListMine(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	userID := UserIDFromContext(r.Context())
	if userID == 0 {
		http.Error(w, "no user in context", http.StatusUnauthorized)
		return
	}

	status := r.URL.Query().Get("status")
	if status == "" {
		status = "pending"
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if limit <= 0 {
		limit = 20
	}

	items, err := h.svc.ListMine(r.Context(), userID, status, limit, offset)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_ = json.NewEncoder(w).Encode(items)
}

// ===== ADMIN: listar / aprobar / rechazar =====

// @Summary Listar requests de películas (admin)
// @Tags movie-requests
// @Security BearerAuth
// @Produce json
// @Param status query string false "pending|approved|rejected|all (default: pending)"
// @Param limit query int false "límite (default: 20)"
// @Param offset query int false "offset (default: 0)"
// @Success 200 {array} models.MovieRequest
// @Router /admin/movie-requests [get]
func (h *MovieRequestHandler) ListAll(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	status := r.URL.Query().Get("status")
	if status == "" {
		status = "pending"
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if limit <= 0 {
		limit = 20
	}

	items, err := h.svc.ListAll(r.Context(), status, limit, offset)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_ = json.NewEncoder(w).Encode(items)
}

// @Summary Aprobar request de película
// @Tags movie-requests
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "movieRequestId (ObjectID)"
// @Param body body models.MovieCreateRequest false "Override opcional de datos"
// @Success 200 {object} map[string]interface{}
// @Router /admin/movie-requests/{id}/approve [post]
func (h *MovieRequestHandler) Approve(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	idStr := chi.URLParam(r, "id")
	objID, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		http.Error(w, "id inválido", http.StatusBadRequest)
		return
	}

	var override models.MovieCreateRequest
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&override) // opcional
	}

	mr, movie, err := h.svc.Approve(r.Context(), objID, &override)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if mr == nil {
		http.NotFound(w, r)
		return
	}
	if mr.Status != models.MovieRequestStatusApproved || movie == nil {
		http.Error(w, "request no está en estado pending", http.StatusBadRequest)
		return
	}

	resp := map[string]any{
		"request": mr,
		"movie":   movie,
	}
	_ = json.NewEncoder(w).Encode(resp)
}

// @Summary Rechazar request de película
// @Tags movie-requests
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "movieRequestId (ObjectID)"
// @Param body body models.RejectMovieRequest true "Motivo de rechazo"
// @Success 200 {object} models.MovieRequest
// @Router /admin/movie-requests/{id}/reject [post]
func (h *MovieRequestHandler) Reject(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	idStr := chi.URLParam(r, "id")
	objID, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		http.Error(w, "id inválido", http.StatusBadRequest)
		return
	}

	var body models.RejectMovieRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "body inválido", http.StatusBadRequest)
		return
	}

	mr, err := h.svc.Reject(r.Context(), objID, body.Reason)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if mr == nil {
		http.NotFound(w, r)
		return
	}
	_ = json.NewEncoder(w).Encode(mr)
}
