package matches

import (
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/sitcon-tw/camp2026-game/internal/http/httpx"
)

// Answer godoc
// @Summary Submit match answer
// @Description Accepts the authenticated player's answer for the current question. Correctness is revealed when the round enters the reveal phase.
// @Tags matches
// @Accept json
// @Produce json
// @Security AuthCookieAuth
// @Param request body AnswerRequest true "Answer request"
// @Success 202 {object} AnswerAcceptedResponse
// @Failure 400 {object} httpx.ProblemDetails
// @Failure 401 {object} httpx.ProblemDetails
// @Failure 404 {object} httpx.ProblemDetails
// @Failure 409 {object} httpx.ProblemDetails
// @Failure 422 {object} httpx.ProblemDetails
// @Failure 500 {object} httpx.ProblemDetails
// @Failure 503 {object} httpx.ProblemDetails
// @Router /matches/{matchID}/answers [post]
func (h *Handler) Answer(w http.ResponseWriter, r *http.Request) {
	player, ok := currentPlayer(w, r)
	if !ok || !h.requireDatabase(w, r) || !h.requireContent(w, r) {
		return
	}

	var body AnswerRequest
	if err := httpx.DecodeJSON(r, &body); err != nil {
		httpx.WriteProblem(w, r, err)
		return
	}
	body.QuestionID = strings.TrimSpace(body.QuestionID)
	body.Choice = strings.ToUpper(strings.TrimSpace(body.Choice))
	if err := httpx.ValidateStruct(body); err != nil {
		httpx.WriteProblem(w, r, err)
		return
	}

	session, err := h.sessions.GetOrLoad(r.Context(), chi.URLParam(r, "matchID"))
	if err != nil {
		writeMatchProblem(w, r, err)
		return
	}
	if err := session.Answer(r.Context(), player.ID, body.QuestionID, body.Choice); err != nil {
		if errors.Is(err, errMatchSaveConflict) || errors.Is(err, errOpenParticipantMatchExists) {
			writeMatchProblem(w, r, err)
			return
		}
		httpx.WriteProblem(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusAccepted, AnswerAcceptedResponse{Accepted: true})
}
