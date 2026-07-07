package matches

import (
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

	flusher, ok := w.(http.Flusher)
	if !ok {
		httpx.WriteProblem(w, r, httpx.InternalServerError("event stream is unavailable", "match_events_flusher_unavailable", errors.New("response writer does not implement http.Flusher")))
		return
	}

	// Best-effort: clear write deadline so the SSE stream is not killed by WriteTimeout.
	// Ignored if the underlying ResponseWriter does not support it.
	_ = http.NewResponseController(w).SetWriteDeadline(time.Time{})

	var session *MatchSession
	if matchIsOpen(match) {
		session, err = h.sessions.GetOrLoad(r.Context(), match.ID)
		if err != nil {
			httpx.WriteProblem(w, r, err)
			return
		}
	}

	eventCh, unsubscribe := h.broker.Subscribe(matchID)
	defer unsubscribe()

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	var state MatchStateResponse
	if session != nil {
		state, err = session.State(r.Context(), player.ID)
	} else {
		state, err = h.buildMatchState(r.Context(), match, player.ID)
	}
	if err != nil {
		httpx.WriteProblem(w, r, err)
		return
	}
	writeSSE(w, "match_updated", state)
	flusher.Flush()

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
			if event.Name == "match_deleted" {
				writeSSEDeleted(w, event.Match.ID)
				flusher.Flush()
				return
			}
			var state MatchStateResponse
			if event.Answers != nil {
				state, err = h.buildMatchStateWithAnswers(r.Context(), event.Match, player.ID, event.Answers)
			} else {
				state, err = h.buildMatchState(r.Context(), event.Match, player.ID)
			}
			if err != nil {
				return
			}
			writeSSE(w, event.Name, state)
			flusher.Flush()
		case <-heartbeat.C:
			_, _ = fmt.Fprint(w, "event: keepalive\ndata: {}\n\n")
			flusher.Flush()
		}
	}
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
