package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"nodosml-pc4/internal/service"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"
)

type RecommendHandler struct {
	svc *service.RecommendService
}

func NewRecommendHandler(s *service.RecommendService) *RecommendHandler {
	return &RecommendHandler{svc: s}
}

// @Summary Recomendaciones para un usuario
// @Tags recommend
// @Produce json
// @Param id path int true "userId"
// @Param k query int false "cantidad de recomendaciones (máx 50)"
// @Param refresh query bool false "si true, ignora cache Redis"
// @Success 200 {array} models.RecItem
// @Router /users/{id}/recommendations [get]
func (h *RecommendHandler) GetRecommendations(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	userID, _ := strconv.Atoi(chi.URLParam(r, "id"))
	k, _ := strconv.Atoi(r.URL.Query().Get("k"))
	refresh := r.URL.Query().Get("refresh") == "true"

	items, err := h.svc.Recommend(r.Context(), service.RecRequest{
		UserID:  userID,
		K:       k,
		Refresh: refresh,
	})
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	_ = json.NewEncoder(w).Encode(items)
}

// upgrader global (no afecta a swagger)
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// @Summary Recomendaciones en tiempo real (WebSocket)
// @Tags recommend
// @Produce json
// @Param id path int true "userId"
// @Param k query int false "cantidad de recomendaciones (máx 50)"
// @Param refresh query bool false "si true, ignora cache Redis"
// @Success 200 {object} map[string]interface{}
// @Router /users/{id}/ws/recommendations [get]
func (h *RecommendHandler) GetRecommendationsWS(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, "No se pudo abrir WebSocket", 400)
		return
	}
	defer conn.Close()

	userID, _ := strconv.Atoi(chi.URLParam(r, "id"))
	k, _ := strconv.Atoi(r.URL.Query().Get("k"))
	refresh := r.URL.Query().Get("refresh") == "true"

	// Mensaje inicial
	conn.WriteJSON(map[string]any{
		"type": "start",
		"msg":  "Conexión WS abierta, iniciando cálculo…",
	})

	// Simular mensajes de progreso (uno por cada shard)
	for i := 1; i <= 4; i++ {
		time.Sleep(300 * time.Millisecond)
		conn.WriteJSON(map[string]any{
			"type":  "progress",
			"shard": i,
			"msg":   fmt.Sprintf("Nodo ML %d completó su parte", i),
		})
	}

	// Calcular recomendaciones reales
	items, err := h.svc.Recommend(r.Context(), service.RecRequest{
		UserID:  userID,
		K:       k,
		Refresh: refresh,
	})
	if err != nil {
		conn.WriteJSON(map[string]any{
			"type":  "error",
			"error": err.Error(),
		})
		return
	}

	// Mensaje final con recomendaciones
	conn.WriteJSON(map[string]any{
		"type":        "recommendations",
		"userId":      userID,
		"items":       items,
		"generatedAt": time.Now(),
	})
}
