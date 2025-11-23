package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"nodosml-pc4/internal/models"
	"nodosml-pc4/internal/service"

	"github.com/go-chi/chi/v5"
)

// AdminMaintenanceHandler expone endpoints de mantenimiento.
type AdminMaintenanceHandler struct {
	svc *service.AdminMaintenanceService
}

// NewAdminMaintenanceHandler crea el handler.
func NewAdminMaintenanceHandler(svc *service.AdminMaintenanceService) *AdminMaintenanceHandler {
	return &AdminMaintenanceHandler{svc: svc}
}

// @Summary Resumen de estado de similitudes
// @Description Devuelve conteos de películas con/sin iIdx y con/sin similitudes precalculadas.
// @Tags admin-maintenance
// @Security BearerAuth
// @Produce json
// @Param minRatings query int false "Mínimo de ratings para considerar una película (default 5)"
// @Success 200 {object} models.AdminSimilaritySummary
// @Failure 500 {string} string "error interno"
// @Router /admin/maintenance/similarities/summary [get]
// GET /admin/maintenance/similarities/summary
func (h *AdminMaintenanceHandler) GetSummary(w http.ResponseWriter, r *http.Request) {
	minRatings := int64(5)
	if v := r.URL.Query().Get("minRatings"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 {
			minRatings = n
		}
	}

	summary, err := h.svc.GetSimilaritySummary(r.Context(), minRatings)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, summary)
}

// @Summary Películas pendientes de mapeo/similitudes
// @Description Lista películas sin iIdx y películas con iIdx pero sin documento en similarities.
// @Tags admin-maintenance
// @Security BearerAuth
// @Produce json
// @Param minRatings query int false "Mínimo de ratings para considerar una película (default 5)"
// @Param limitWithoutIdx query int false "Límite de películas sin iIdx (default 50)"
// @Param limitWithoutSims query int false "Límite de películas sin similitudes (default 50)"
// @Success 200 {object} models.AdminPendingSimilarities
// @Failure 500 {string} string "error interno"
// @Router /admin/maintenance/similarities/pending [get]
// GET /admin/maintenance/similarities/pending
func (h *AdminMaintenanceHandler) GetPending(w http.ResponseWriter, r *http.Request) {
	minRatings := int64(5)
	if v := r.URL.Query().Get("minRatings"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 {
			minRatings = n
		}
	}
	limitWithoutIdx := int64(50)
	if v := r.URL.Query().Get("limitWithoutIdx"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 {
			limitWithoutIdx = n
		}
	}
	limitWithoutSims := int64(50)
	if v := r.URL.Query().Get("limitWithoutSims"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 {
			limitWithoutSims = n
		}
	}

	resp, err := h.svc.GetPendingSimilarities(r.Context(), minRatings, limitWithoutIdx, limitWithoutSims)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// @Summary Remapear películas sin iIdx
// @Description Asigna nuevos valores de iIdx a películas que aún no lo tienen y tienen suficientes ratings.
// @Tags admin-maintenance
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param body body models.RemapMissingRequest true "Parámetros de remapeo"
// @Success 200 {object} models.RemapMissingResult
// @Failure 400 {string} string "body inválido"
// @Failure 500 {string} string "error interno"
// @Router /admin/maintenance/similarities/remap-missing [post]
// POST /admin/maintenance/similarities/remap-missing
func (h *AdminMaintenanceHandler) PostRemapMissing(w http.ResponseWriter, r *http.Request) {
	var req models.RemapMissingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "body inválido", http.StatusBadRequest)
		return
	}
	if req.MinRatings <= 0 {
		req.MinRatings = 5
	}
	if req.Limit <= 0 {
		req.Limit = 1000
	}

	res, err := h.svc.RemapMissingMovies(r.Context(), req.MinRatings, req.Limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

// @Summary Recalcular similitudes pendientes
// @Description Lanza el recálculo de similitudes en batches contra los nodos ML para películas sin entry en similarities.
// @Tags admin-maintenance
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param body body models.RebuildSimilaritiesRequest true "Parámetros de reconstrucción"
// @Success 200 {object} models.RebuildSimilaritiesResult
// @Failure 400 {string} string "body inválido"
// @Failure 500 {string} string "error interno"
// @Router /admin/maintenance/similarities/rebuild [post]
// POST /admin/maintenance/similarities/rebuild
func (h *AdminMaintenanceHandler) PostRebuild(w http.ResponseWriter, r *http.Request) {
	var req models.RebuildSimilaritiesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "body inválido", http.StatusBadRequest)
		return
	}
	if req.MinRatings <= 0 {
		req.MinRatings = 5
	}
	if req.BatchSize <= 0 {
		req.BatchSize = 50
	}
	if req.Parallelism <= 0 {
		req.Parallelism = 4
	}
	if req.K <= 0 {
		req.K = 20
	}
	if req.MinCommonUsers <= 0 {
		req.MinCommonUsers = 3
	}
	if req.Shrink < 0 {
		req.Shrink = 20
	}

	res, err := h.svc.RebuildSimilarities(r.Context(), &req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

// Utilidad pequeña para respuestas JSON.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// Helper para montar rutas en main.go
func MountAdminMaintenanceRoutes(r chi.Router, h *AdminMaintenanceHandler) {
	r.Route("/admin/maintenance", func(r chi.Router) {
		r.Get("/similarities/summary", h.GetSummary)
		r.Get("/similarities/pending", h.GetPending)
		r.Post("/similarities/remap-missing", h.PostRemapMissing)
		r.Post("/similarities/rebuild", h.PostRebuild)
	})
}
