package matches

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/sitcon-tw/camp2026-game/internal/gamecontrol"
	"github.com/sitcon-tw/camp2026-game/internal/http/httpx"
	mongomodel "github.com/sitcon-tw/camp2026-game/internal/mongodb/model"
)

// Rematch godoc
// @Summary Create a rematch room
// @Description Creates a new waiting room from a completed quiz match with the same participants.
// @Tags matches
// @Produce json
// @Security AuthCookieAuth
// @Success 201 {object} RematchResponse
// @Success 200 {object} RematchResponse
// @Failure 401 {object} httpx.ProblemDetails
// @Failure 403 {object} httpx.ProblemDetails
// @Failure 404 {object} httpx.ProblemDetails
// @Failure 409 {object} httpx.ProblemDetails
// @Failure 500 {object} httpx.ProblemDetails
// @Failure 503 {object} httpx.ProblemDetails
// @Router /matches/{matchID}/rematch [post]
func (h *Handler) Rematch(w http.ResponseWriter, r *http.Request) {
	player, ok := currentPlayer(w, r)
	if !ok || !h.requireDatabase(w, r) || !h.requireContent(w, r) {
		return
	}
	if err := h.ensureBattleOpeningAllowed(r.Context(), "match rematch failed"); err != nil {
		httpx.WriteProblem(w, r, err)
		return
	}

	source, err := h.findMatchByID(r.Context(), chi.URLParam(r, "matchID"))
	if err != nil {
		writeMatchProblem(w, r, err)
		return
	}
	if !isParticipant(source, player.ID) {
		httpx.WriteProblem(w, r, httpx.NotFound("match not found"))
		return
	}
	if source.Status != mongomodel.MatchStatusCompleted {
		httpx.WriteProblem(w, r, httpx.NewError(http.StatusConflict, "match is not completed"))
		return
	}
	if h.writeExistingOpenMatchState(w, r, player.ID) {
		return
	}
	if err := h.ensureRematchAllowed(r.Context(), source); err != nil {
		httpx.WriteProblem(w, r, err)
		return
	}
	if err := h.ensureRematchParticipantsAvailable(r.Context(), source, player.ID); err != nil {
		httpx.WriteProblem(w, r, err)
		return
	}

	matchID, err := newID("match")
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("match rematch failed", "match_id_create_failed", err))
		return
	}
	match := newRematchMatch(source, matchID, player.ID, time.Now())
	if _, err := h.db.Collection(mongomodel.MatchesCollection).InsertOne(r.Context(), match); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			if h.writeExistingOpenMatchState(w, r, player.ID) {
				return
			}
			writeOpenParticipantMatchConflict(w, r)
			return
		}
		httpx.WriteProblem(w, r, httpx.InternalServerError("match rematch failed", "match_insert_failed", err))
		return
	}

	session := h.sessions.Start(match)
	state, err := session.State(r.Context(), player.ID)
	if err != nil {
		httpx.WriteProblem(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, state)
}

func (h *Handler) ensureRematchAllowed(ctx context.Context, source mongomodel.Match) error {
	if matchMode(source) == mongomodel.MatchModeComputer {
		settings, err := gamecontrol.ReadSettings(ctx, h.db)
		if err != nil {
			return httpx.InternalServerError("match rematch failed", "computer_settings_lookup_failed", err)
		}
		if !settings.ComputerBattlesEnabled {
			return httpx.NewError(http.StatusConflict, "computer battles are disabled")
		}
	}

	if err := h.ensureMatchSameTeamBattleAllowed(ctx, source); err != nil {
		return err
	}
	return h.ensureRematchOpponentLimitAllowed(ctx, source)
}

func (h *Handler) ensureRematchOpponentLimitAllowed(ctx context.Context, source mongomodel.Match) error {
	if matchMode(source) != mongomodel.MatchModePVP {
		return nil
	}
	playerIDs := humanParticipantIDs(source)
	if len(playerIDs) != 2 {
		return nil
	}
	count, err := h.completedPVPMatchCountBetweenPlayers(ctx, playerIDs[0], playerIDs[1])
	if err != nil {
		return httpx.InternalServerError("match rematch failed", "match_rematch_limit_lookup_failed", err)
	}
	if count >= maxOpponentMatchCount {
		return httpx.NewError(http.StatusForbidden, "player pair match limit reached")
	}
	return nil
}

func (h *Handler) ensureRematchParticipantsAvailable(ctx context.Context, source mongomodel.Match, currentPlayerID string) error {
	for _, playerID := range humanParticipantIDs(source) {
		if playerID == currentPlayerID {
			continue
		}
		if err := h.ensureNoOpenParticipantMatch(ctx, playerID); err != nil {
			if errors.Is(err, errOpenParticipantMatchExists) {
				return httpx.NewError(http.StatusConflict, "rematch participant already has an open match")
			}
			return httpx.InternalServerError("match rematch failed", "match_open_lookup_failed", err)
		}
	}
	return nil
}

func newRematchMatch(source mongomodel.Match, matchID string, hostPlayerID string, now time.Time) mongomodel.Match {
	match := mongomodel.Match{
		ID:                   matchID,
		Mode:                 matchMode(source),
		Status:               mongomodel.MatchStatusWaiting,
		HostPlayerID:         hostPlayerID,
		RematchSourceMatchID: source.ID,
		Players:              rematchPlayers(source),
		CreatedAt:            now,
	}
	syncOpenMatchLocks(&match)
	return match
}

func rematchPlayers(source mongomodel.Match) []mongomodel.MatchPlayer {
	players := make([]mongomodel.MatchPlayer, 0, len(source.Players))
	for _, player := range source.Players {
		next := mongomodel.MatchPlayer{
			PlayerID:  player.PlayerID,
			Nickname:  player.Nickname,
			Kind:      matchPlayerKind(player),
			Ready:     isComputerPlayer(player),
			Score:     0,
			SitoneIDs: cloneStrings(player.SitoneIDs),
		}
		if isComputerPlayer(next) && len(next.SitoneIDs) == 0 {
			next.SitoneIDs = []string{computerDefaultSitone}
		}
		players = append(players, next)
	}
	return players
}
