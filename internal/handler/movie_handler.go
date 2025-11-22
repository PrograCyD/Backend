// internal/handler/movie_handler.go
package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"nodosml-pc4/internal/service"

	"github.com/go-chi/chi/v5"
)

type MovieHandler struct {
	svc *service.MovieService
}

func NewMovieHandler(s *service.MovieService) *MovieHandler { return &MovieHandler{svc: s} }

// @Summary Get movie
// @Tags movies
// @Produce json
// @Param id path int true "movieId"
// @Success 200 {object} models.MovieDoc
// @Router /movies/{id} [get]
func (h *MovieHandler) GetMovie(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	id, _ := strconv.Atoi(chi.URLParam(r, "id"))
	m, err := h.svc.GetMovie(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	if m == nil {
		http.NotFound(w, r)
		return
	}
	_ = json.NewEncoder(w).Encode(m)
}

// @Summary Buscar / listar películas (paginado)
// @Tags movies
// @Produce json
// @Param q query string false "búsqueda por título"
// @Param genre query string false "filtrar por género"
// @Param year_from query int false "año desde"
// @Param year_to query int false "año hasta"
// @Param limit query int false "límite"
// @Param offset query int false "offset"
// @Success 200 {array} models.MovieDoc
// @Router /movies/search [get]
func (h *MovieHandler) Search(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	q := r.URL.Query().Get("q")
	genre := r.URL.Query().Get("genre")

	yearFrom, _ := strconv.Atoi(r.URL.Query().Get("year_from"))
	yearTo, _ := strconv.Atoi(r.URL.Query().Get("year_to"))

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if limit <= 0 {
		limit = 20
	}

	movies, err := h.svc.Search(r.Context(), q, genre, yearFrom, yearTo, limit, offset)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	_ = json.NewEncoder(w).Encode(movies)
}

// @Summary Top películas (popularidad o rating)
// @Tags movies
// @Produce json
// @Param metric query string false "popular|rating (default: popular)"
// @Param limit query int false "límite (default: 20)"
// @Success 200 {array} models.MovieDoc
// @Router /movies/top [get]
func (h *MovieHandler) Top(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	metric := r.URL.Query().Get("metric")
	if metric == "" {
		metric = "popular"
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 {
		limit = 20
	}

	movies, err := h.svc.Top(r.Context(), metric, limit)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	_ = json.NewEncoder(w).Encode(movies)
}
