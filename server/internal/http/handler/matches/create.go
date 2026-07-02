package matches

import (
	"context"
	"errors"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/sitcon-tw/camp2026-game/internal/http/httpx"
	mongomodel "github.com/sitcon-tw/camp2026-game/internal/mongodb/model"
)

// Create godoc
// @Summary Create match room
// @Description Creates a two-player quiz match room for the authenticated player.
// @Tags matches
// @Produce json
// @Security AuthCookieAuth
// @Success 201 {object} CreateMatchResponse
// @Failure 401 {object} httpx.ProblemDetails
// @Failure 409 {object} httpx.ProblemDetails
// @Failure 500 {object} httpx.ProblemDetails
// @Failure 503 {object} httpx.ProblemDetails
// @Router /matches [post]
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	player, ok := currentPlayer(w, r)
	if !ok || !h.requireDatabase(w, r) || !h.requireContent(w, r) {
		return
	}
	if err := h.ensureNoOpenHostedMatch(r.Context(), player.ID); err != nil {
		writeCreateMatchProblem(w, r, err)
		return
	}

	matchID, err := newID("match")
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("match creation failed", "match_id_create_failed", err))
		return
	}
	code, err := h.uniqueMatchCode(r.Context())
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("match creation failed", "match_code_create_failed", err))
		return
	}
	sitoneIDs, err := h.defaultSitoneLoadout(r.Context(), player)
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("match creation failed", "match_default_loadout_failed", err))
		return
	}

	now := time.Now()
	match := mongomodel.Match{
		ID:           matchID,
		Code:         code,
		Mode:         mongomodel.MatchModePVP,
		Status:       mongomodel.MatchStatusWaiting,
		HostPlayerID: player.ID,
		OpenHostLock: player.ID,
		Players: []mongomodel.MatchPlayer{
			{
				PlayerID:  player.ID,
				Nickname:  player.Nickname,
				Kind:      mongomodel.MatchPlayerKindHuman,
				Ready:     false,
				Score:     0,
				SitoneIDs: sitoneIDs,
			},
		},
		CreatedAt: now,
	}

	if _, err := h.db.Collection(mongomodel.MatchesCollection).InsertOne(r.Context(), match); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			writeOpenHostedMatchConflict(w, r)
			return
		}
		httpx.WriteProblem(w, r, httpx.InternalServerError("match creation failed", "match_insert_failed", err))
		return
	}

	state, err := h.buildMatchState(r.Context(), match, player.ID)
	if err != nil {
		httpx.WriteProblem(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, state)
}

var errOpenHostedMatchExists = errors.New("player already has an open match")

func (h *Handler) ensureNoOpenHostedMatch(ctx context.Context, playerID string) error {
	err := h.db.Collection(mongomodel.MatchesCollection).
		FindOne(
			ctx,
			openHostedMatchFilter(playerID),
			options.FindOne().SetProjection(bson.D{{Key: "_id", Value: 1}}),
		).
		Err()
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil
	}
	if err != nil {
		return err
	}
	return errOpenHostedMatchExists
}

func openHostedMatchFilter(playerID string) bson.M {
	return bson.M{
		"host_player_id": playerID,
		"status": bson.M{
			"$in": bson.A{
				mongomodel.MatchStatusWaiting,
				mongomodel.MatchStatusActive,
			},
		},
	}
}

func writeCreateMatchProblem(w http.ResponseWriter, r *http.Request, err error) {
	if errors.Is(err, errOpenHostedMatchExists) {
		writeOpenHostedMatchConflict(w, r)
		return
	}
	httpx.WriteProblem(w, r, httpx.InternalServerError("match creation failed", "match_open_lookup_failed", err))
}

func writeOpenHostedMatchConflict(w http.ResponseWriter, r *http.Request) {
	httpx.WriteProblem(w, r, httpx.NewError(http.StatusConflict, "player already has an open match"))
}

func (h *Handler) uniqueMatchCode(ctx context.Context) (string, error) {
	for i := 0; i < 5; i++ {
		code, err := newMatchCode()
		if err != nil {
			return "", err
		}

		err = h.db.Collection(mongomodel.MatchesCollection).
			FindOne(ctx, bson.M{"code": code, "status": bson.M{"$ne": mongomodel.MatchStatusCompleted}}).
			Err()
		if errors.Is(err, mongo.ErrNoDocuments) {
			return code, nil
		}
		if err != nil {
			return "", err
		}
	}
	return "", errors.New("match code collision")
}
