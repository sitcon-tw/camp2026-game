package matches

import (
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/sitcon-tw/camp2026-game/internal/content"
	"github.com/sitcon-tw/camp2026-game/internal/http/httpx"
)

// Ready godoc
// @Summary Mark current player ready
// @Description Marks the authenticated player ready. The match starts automatically when both players are ready.
// @Tags matches
// @Produce json
// @Security AuthCookieAuth
// @Success 200 {object} ReadyMatchResponse
// @Failure 401 {object} httpx.ProblemDetails
// @Failure 403 {object} httpx.ProblemDetails
// @Failure 404 {object} httpx.ProblemDetails
// @Failure 409 {object} httpx.ProblemDetails
// @Failure 500 {object} httpx.ProblemDetails
// @Failure 503 {object} httpx.ProblemDetails
// @Router /matches/{matchID}/ready [post]
func (h *Handler) Ready(w http.ResponseWriter, r *http.Request) {
	player, ok := currentPlayer(w, r)
	if !ok || !h.requireDatabase(w, r) || !h.requireContent(w, r) {
		return
	}
	if err := h.ensureBattleOpeningAllowed(r.Context(), "match ready failed"); err != nil {
		httpx.WriteProblem(w, r, err)
		return
	}

	matchID := chi.URLParam(r, "matchID")
	session, err := h.sessions.GetOrLoad(r.Context(), matchID)
	if err != nil {
		writeMatchProblem(w, r, err)
		return
	}

	state, err := session.Ready(r.Context(), player)
	if err != nil {
		if errors.Is(err, errMatchSaveConflict) || errors.Is(err, errOpenParticipantMatchExists) {
			writeMatchProblem(w, r, err)
			return
		}
		httpx.WriteProblem(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, state)
}

func (h *Handler) pickQuestionIDs() ([]string, error) {
	questions := h.content.ListQuizQuestions()
	if len(questions) < matchQuestionCount {
		return nil, httpx.InternalServerError("quiz questions are unavailable", "match_questions_insufficient", errors.New("not enough quiz questions"))
	}

	if err := shuffleQuizQuestions(questions, rand.Reader); err != nil {
		return nil, err
	}

	ids := make([]string, 0, matchQuestionCount)
	for _, question := range questions[:matchQuestionCount] {
		ids = append(ids, question.ID)
	}
	return ids, nil
}

func shuffleQuizQuestions(questions []content.QuizQuestion, random io.Reader) error {
	for i := len(questions) - 1; i > 0; i-- {
		n, err := rand.Int(random, big.NewInt(int64(i+1)))
		if err != nil {
			return fmt.Errorf("shuffle quiz questions: %w", err)
		}

		j := int(n.Int64())
		questions[i], questions[j] = questions[j], questions[i]
	}
	return nil
}
