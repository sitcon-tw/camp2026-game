package me

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/sitcon-tw/camp2026-game/internal/http/httpx"
	"github.com/sitcon-tw/camp2026-game/internal/http/playerevents"
)

// Events godoc
// @Summary Stream player events
// @Description Streams player reward notifications with Server-Sent Events.
// @Tags me
// @Produce text/event-stream
// @Security AuthCookieAuth
// @Success 200 {string} string "SSE event stream"
// @Failure 401 {object} httpx.ProblemDetails
// @Failure 500 {object} httpx.ProblemDetails
// @Router /me/events [get]
func (h *Handler) Events(w http.ResponseWriter, r *http.Request) {
	player, ok := currentPlayer(w, r)
	if !ok {
		return
	}
	if h.broker == nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("event stream is unavailable", "player_events_broker_unavailable", errors.New("player events broker is unavailable")))
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		httpx.WriteProblem(w, r, httpx.InternalServerError("event stream is unavailable", "player_events_flusher_unavailable", errors.New("response writer does not implement http.Flusher")))
		return
	}

	// Best-effort: clear write deadline so the SSE stream is not killed by WriteTimeout.
	// Ignored if the underlying ResponseWriter does not support it.
	_ = http.NewResponseController(w).SetWriteDeadline(time.Time{})

	eventCh, unsubscribe := h.broker.Subscribe(player.ID)
	defer unsubscribe()

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	heartbeat := time.NewTicker(20 * time.Second)
	defer heartbeat.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case event, ok := <-eventCh:
			if !ok {
				return
			}
			writePlayerEventSSE(w, event)
			flusher.Flush()
		case <-heartbeat.C:
			_, _ = fmt.Fprint(w, "event: keepalive\ndata: {}\n\n")
			flusher.Flush()
		}
	}
}

func writePlayerEventSSE(w http.ResponseWriter, event playerevents.Event) {
	if event.Reward == nil {
		return
	}
	payload, err := json.Marshal(event.Reward)
	if err != nil {
		return
	}
	_, _ = fmt.Fprintf(w, "event: %s\n", event.Name)
	_, _ = fmt.Fprintf(w, "data: %s\n\n", payload)
}
