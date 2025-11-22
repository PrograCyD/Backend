package handler

import "net/http"

// @Summary Healthcheck
// @Tags health
// @Success 200
// @Router /health [get]
func Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte("ok"))
}
