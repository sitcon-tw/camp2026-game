package matches

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/sitcon-tw/camp2026-game/internal/http/httpx"
)

// Events godoc
// @Summary Stream match events
// @Description Streams match state changes for participants with Server-Sent Events.
// @Tags matches
// @Produce text/event-stream
// @Security AuthCookieAuth
// @Success 200 {string} string "SSE event stream"
// @Failure 401 {object} httpx.ProblemDetails
// @Failure 404 {object} httpx.ProblemDetails
// @Failure 500 {object} httpx.ProblemDetails
// @Failure 503 {object} httpx.ProblemDetails
// @Router /matches/{matchID}/events [get]
func (h *Handler) Events(w http.ResponseWriter, r *http.Request) {
	player, ok := currentPlayer(w, r)
	if !ok || !h.requireDatabase(w, r) || !h.requireContent(w, r) {
		return
	}

	matchID := chi.URLParam(r, "matchID")
	match, err := h.findMatchByID(r.Context(), matchID)
	if err != nil {
		writeMatchProblem(w, r, err)
		return
	}
	if !isParticipant(match, player.ID) {
		httpx.WriteProblem(w, r, httpx.NotFound("match not found"))
		return
	}
	match, advanceEvents, err := h.advanceMatch(r.Context(), match, time.Now())
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("match events unavailable", "match_events_advance_failed", err))
		return
	}
	for _, event := range advanceEvents {
		h.publishState(r.Context(), match, event)
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		httpx.WriteProblem(w, r, httpx.InternalServerError("event stream is unavailable", "match_events_flusher_unavailable", errors.New("response writer does not implement http.Flusher")))
		return
	}

	eventCh, unsubscribe := h.broker.Subscribe(matchID)
	defer unsubscribe()

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	state, err := h.buildMatchState(r.Context(), match, player.ID)
	if err != nil {
		httpx.WriteProblem(w, r, err)
		return
	}
	writeSSE(w, "match_updated", state)
	flusher.Flush()

	heartbeat := time.NewTicker(15 * time.Second)
	defer heartbeat.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case event, ok := <-eventCh:
			if !ok {
				return
			}
			if event.Name == "match_deleted" {
				writeSSEDeleted(w, event.Match.ID)
				flusher.Flush()
				return
			}
			state, err := h.buildMatchState(r.Context(), event.Match, player.ID)
			if err != nil {
				return
			}
			writeSSE(w, event.Name, state)
			flusher.Flush()
		case <-heartbeat.C:
			advanced, err := h.advanceAndPublishMatch(r.Context(), matchID, time.Now())
			if err != nil {
				return
			}
			if advanced {
				continue
			}
			_, _ = fmt.Fprint(w, ": keepalive\n\n")
			flusher.Flush()
		}
	}
}

func (h *Handler) advanceAndPublishMatch(ctx context.Context, matchID string, now time.Time) (bool, error) {
	match, err := h.findMatchByID(ctx, matchID)
	if err != nil {
		return false, err
	}
	match, events, err := h.advanceMatch(ctx, match, now)
	if err != nil {
		return false, err
	}
	for _, event := range events {
		h.publishState(ctx, match, event)
	}
	return len(events) > 0, nil
}

func writeSSE(w http.ResponseWriter, name string, data MatchStateResponse) {
	payload, err := json.Marshal(data)
	if err != nil {
		return
	}
	_, _ = fmt.Fprintf(w, "event: %s\n", name)
	_, _ = fmt.Fprintf(w, "data: %s\n\n", payload)
}

func writeSSEDeleted(w http.ResponseWriter, matchID string) {
	payload, err := json.Marshal(map[string]string{"matchId": matchID})
	if err != nil {
		return
	}
	_, _ = fmt.Fprint(w, "event: match_deleted\n")
	_, _ = fmt.Fprintf(w, "data: %s\n\n", payload)
}
