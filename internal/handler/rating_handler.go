package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"nodosml-pc4/internal/service"

	"github.com/go-chi/chi/v5"
)

type RatingHandler struct {
	svc *service.RatingService
}

func NewRatingHandler(s *service.RatingService) *RatingHandler { return &RatingHandler{svc: s} }

type ratingRequest struct {
	MovieID int     `json:"movieId"`
	Rating  float64 `json:"rating"`
}

// @Summary Crear/actualizar rating
// @Tags ratings
// @Accept json
// @Param id path int true "userId"
// @Param body body ratingRequest true "rating"
// @Success 204
// @Router /users/{id}/ratings [post]
func (h *RatingHandler) PostRating(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	userID, _ := strconv.Atoi(chi.URLParam(r, "id"))
	var req ratingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	if err := h.svc.AddOrUpdate(r.Context(), userID, req.MovieID, req.Rating); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// @Summary Listar ratings del usuario
// @Tags ratings
// @Produce json
// @Param id path int true "userId"
// @Router /users/{id}/ratings [get]
func (h *RatingHandler) GetRatings(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	userID, _ := strconv.Atoi(chi.URLParam(r, "id"))
	list, err := h.svc.GetByUser(r.Context(), userID, 100, 0)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	_ = json.NewEncoder(w).Encode(list)
}
