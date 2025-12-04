// internal/handler/movie_handler.go
package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"nodosml-pc4/internal/models"
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

// ====== ADMIN: crear / actualizar películas ======

// @Summary Crear nueva película
// @Tags movies
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param body body models.MovieCreateRequest true "Datos de la película"
// @Success 201 {object} models.MovieDoc
// @Router /admin/movies [post]
func (h *MovieHandler) CreateMovie(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req models.MovieCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Title == "" {
		http.Error(w, "body inválido (title requerido)", http.StatusBadRequest)
		return
	}

	movie, err := h.svc.CreateMovie(r.Context(), &req)
	if err != nil {
		if errors.Is(err, service.ErrMovieAlreadyExists) {
			http.Error(w, "ya existe una película con ese título y año", http.StatusConflict)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(movie)
}

// @Summary Actualizar película existente
// @Tags movies
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "movieId"
// @Param body body models.MovieUpdateRequest true "Campos a actualizar"
// @Success 200 {object} models.MovieDoc
// @Router /admin/movies/{id} [put]
func (h *MovieHandler) UpdateMovie(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	id, _ := strconv.Atoi(chi.URLParam(r, "id"))

	var req models.MovieUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "body inválido", http.StatusBadRequest)
		return
	}

	movie, err := h.svc.UpdateMovie(r.Context(), id, &req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if movie == nil {
		http.NotFound(w, r)
		return
	}
	_ = json.NewEncoder(w).Encode(movie)
}

// @Summary Obtener datos de película desde TMDB (para prellenar formulario)
// @Tags movies
// @Produce json
// @Param tmdbId query string true "ID de TMDB, por ejemplo 603"
// @Success 200 {object} models.ExternalData
// @Failure 400 {string} string "tmdbId requerido"
// @Router /movies/tmdb [get]
func (h *MovieHandler) FetchFromTMDB(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	tmdbID := r.URL.Query().Get("tmdbId")
	if tmdbID == "" {
		http.Error(w, "tmdbId es requerido", http.StatusBadRequest)
		return
	}

	ext, err := h.svc.FetchExternalFromTMDB(r.Context(), tmdbID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_ = json.NewEncoder(w).Encode(ext)
}

// @Summary Prefill de película completo desde TMDB (para formulario de alta)
// @Tags movies
// @Produce json
// @Param tmdbId query string true "ID de TMDB, por ejemplo 603"
// @Success 200 {object} models.MovieCreateRequest
// @Failure 400 {string} string "tmdbId requerido"
// @Router /movies/tmdb-prefill [get]
func (h *MovieHandler) PrefillMovieFromTMDB(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	tmdbID := r.URL.Query().Get("tmdbId")
	if tmdbID == "" {
		http.Error(w, "tmdbId es requerido", http.StatusBadRequest)
		return
	}

	req, err := h.svc.PrefillCreateFromTMDB(r.Context(), tmdbID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_ = json.NewEncoder(w).Encode(req)
}
