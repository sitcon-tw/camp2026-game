package system

import (
	"context"
	"net/http"
	"time"

	"github.com/sitcon-tw/camp2026-game/internal/http/httpx"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
)

type StatusResponse struct {
	Status string            `json:"status" example:"ok"`
	Checks map[string]string `json:"checks,omitempty" example:"database:ok"`
}

// Health godoc
// @Summary Health check
// @Description Confirms the HTTP process is alive and the database is reachable.
// @Tags system
// @Produce json
// @Success 200 {object} StatusResponse
// @Failure 503 {object} httpx.ProblemDetails
// @Router /healthz [get]
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	if h.mongoClient == nil {
		httpx.WriteJSON(w, http.StatusOK, StatusResponse{Status: "ok"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	if err := h.mongoClient.Ping(ctx, readpref.Primary()); err != nil {
		httpx.WriteProblem(w, r, httpx.ServiceUnavailable("database check failed"))
		return
	}

	httpx.WriteJSON(w, http.StatusOK, StatusResponse{
		Status: "ok",
		Checks: map[string]string{
			"database": "ok",
		},
	})
}
