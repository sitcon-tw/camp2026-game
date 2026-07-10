package fronts

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/sitcon-tw/camp2026-game/internal/http/httpx"
	mongomodel "github.com/sitcon-tw/camp2026-game/internal/mongodb/model"
	"github.com/sitcon-tw/camp2026-game/internal/openpower"
)

// Events godoc
// @Summary Stream front events
// @Description Streams personalized front snapshots when territory, garrisons, rails, trains, or rankings change.
// @Tags fronts
// @Produce text/event-stream
// @Security AuthCookieAuth
// @Param frontID path string true "Front ID"
// @Success 200 {string} string "SSE event stream"
// @Failure 401 {object} httpx.ProblemDetails
// @Failure 404 {object} httpx.ProblemDetails
// @Failure 500 {object} httpx.ProblemDetails
// @Failure 503 {object} httpx.ProblemDetails
// @Router /fronts/{frontID}/events [get]
func (h *Handler) Events(w http.ResponseWriter, r *http.Request) {
	player, ok := currentPlayer(w, r)
	if !ok || !h.requireDatabase(w, r) {
		return
	}
	frontID := chi.URLParam(r, "frontID")
	if _, err := h.frontByID(r.Context(), frontID); errors.Is(err, errFrontNotFound) {
		httpx.WriteProblem(w, r, httpx.NotFound("front not found"))
		return
	} else if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("event stream is unavailable", "front_events_lookup_failed", err))
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		httpx.WriteProblem(w, r, httpx.InternalServerError("event stream is unavailable", "front_events_flusher_unavailable", errors.New("response writer does not implement http.Flusher")))
		return
	}
	_ = http.NewResponseController(w).SetWriteDeadline(time.Time{})
	events, unsubscribe := h.broker.Subscribe(frontID)
	defer unsubscribe()

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	if !h.writeFrontSnapshotEvent(w, r, "front_updated", frontID, player.ID, player.TeamID) {
		return
	}
	flusher.Flush()

	heartbeat := time.NewTicker(20 * time.Second)
	defer heartbeat.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case event, open := <-events:
			if !open || !h.writeFrontSnapshotEvent(w, r, event.Name, frontID, player.ID, player.TeamID) {
				return
			}
			flusher.Flush()
		case <-heartbeat.C:
			_, _ = fmt.Fprint(w, "event: keepalive\ndata: {}\n\n")
			flusher.Flush()
		}
	}
}

func (h *Handler) writeFrontSnapshotEvent(w http.ResponseWriter, r *http.Request, name string, frontID string, playerID string, teamID string) bool {
	front, err := h.frontByID(r.Context(), frontID)
	if err != nil {
		return false
	}
	sitones, err := h.playerFrontSitones(r.Context(), mongomodel.Player{ID: playerID, TeamID: teamID})
	if err != nil {
		return false
	}
	power, err := openpower.TotalForPlayer(r.Context(), h.db, playerID)
	if err != nil {
		return false
	}
	payload, err := json.Marshal(detailResponse(front, playerID, teamID, sitones, power))
	if err != nil {
		return false
	}
	_, _ = fmt.Fprintf(w, "event: %s\ndata: %s\n\n", name, payload)
	return true
}
