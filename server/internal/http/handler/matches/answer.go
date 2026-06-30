package matches

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/sitcon-tw/camp2026-game/internal/http/httpx"
	mongomodel "github.com/sitcon-tw/camp2026-game/internal/mongodb/model"
)

// Answer godoc
// @Summary Submit match answer
// @Description Accepts the authenticated player's answer for the current question. Correctness is revealed to both players when the round enters the reveal phase.
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

	match, err := h.findMatchByID(r.Context(), chi.URLParam(r, "matchID"))
	if err != nil {
		writeMatchProblem(w, r, err)
		return
	}
	if !isParticipant(match, player.ID) {
		httpx.WriteProblem(w, r, httpx.NotFound("match not found"))
		return
	}

	now := time.Now()

	if match.Status != mongomodel.MatchStatusActive {
		httpx.WriteProblem(w, r, httpx.NewError(http.StatusConflict, "match is not active"))
		return
	}
	if activeMatchPhase(match) != mongomodel.MatchPhaseAnswering {
		httpx.WriteProblem(w, r, httpx.NewError(http.StatusConflict, "round is not accepting answers"))
		return
	}
	if !match.RoundEndsAt.IsZero() && !now.Before(match.RoundEndsAt) {
		httpx.WriteProblem(w, r, httpx.NewError(http.StatusConflict, "round is not accepting answers"))
		return
	}
	if match.CurrentQuestionIndex >= len(match.QuestionIDs) || match.QuestionIDs[match.CurrentQuestionIndex] != body.QuestionID {
		httpx.WriteProblem(w, r, httpx.NewError(http.StatusConflict, "question is not current"))
		return
	}

	if _, err := h.findAnswer(r.Context(), match.ID, player.ID, body.QuestionID); err == nil {
		httpx.WriteProblem(w, r, httpx.NewError(http.StatusConflict, "question already answered"))
		return
	} else if !errors.Is(err, mongo.ErrNoDocuments) {
		httpx.WriteProblem(w, r, httpx.InternalServerError("answer failed", "match_answer_lookup_failed", err))
		return
	}

	question, ok := h.content.GetQuizQuestion(body.QuestionID)
	if !ok {
		httpx.WriteProblem(w, r, httpx.InternalServerError("match question is unavailable", "match_answer_question_missing", errors.New("quiz question not found in content store")))
		return
	}
	if choiceEliminated(match, body.QuestionID, player.ID, body.Choice) {
		httpx.WriteProblem(w, r, httpx.NewError(http.StatusUnprocessableEntity, "choice has been eliminated"))
		return
	}
	idx := playerIndex(match, player.ID)
	effects, err := h.matchPlayerBattleEffects(r.Context(), match.Players[idx])
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("answer failed", "match_answer_effects_failed", err))
		return
	}
	correct, baseScore, score := scoreAnswer(question, body.Choice, now, match.RoundEndsAt, effects)
	elapsedMillis := now.Sub(match.RoundStartedAt).Milliseconds()
	if elapsedMillis < 0 {
		elapsedMillis = 0
	}

	answer := mongomodel.MatchAnswer{
		ID:            matchAnswerRecordID(match.ID, player.ID, body.QuestionID),
		MatchID:       match.ID,
		PlayerID:      player.ID,
		QuestionID:    body.QuestionID,
		Choice:        body.Choice,
		Correct:       correct,
		BaseScore:     baseScore,
		BonusScore:    score - baseScore,
		Score:         score,
		ElapsedMillis: elapsedMillis,
		AnsweredAt:    now,
	}
	if _, err := h.db.Collection(mongomodel.MatchAnswersCollection).InsertOne(r.Context(), answer); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			httpx.WriteProblem(w, r, httpx.NewError(http.StatusConflict, "question already answered"))
			return
		}
		httpx.WriteProblem(w, r, httpx.InternalServerError("answer failed", "match_answer_insert_failed", err))
		return
	}

	if err := h.applyMatchAnswerScore(r.Context(), match.ID, player.ID, score); err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("answer failed", "match_answer_save_match_failed", err))
		return
	}
	match, err = h.findMatchByID(r.Context(), match.ID)
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("answer failed", "match_answer_reload_match_failed", err))
		return
	}

	h.publishState(r.Context(), match, "player_answered")
	var events []string
	match, events, err = h.advanceMatch(r.Context(), match, now)
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("answer failed", "match_answer_advance_after_failed", err))
		return
	}
	for _, event := range events {
		h.publishState(r.Context(), match, event)
	}

	httpx.WriteJSON(w, http.StatusAccepted, AnswerAcceptedResponse{Accepted: true})
}
