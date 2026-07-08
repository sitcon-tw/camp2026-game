package matches

import (
	"context"
	"errors"
	"net/http"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/sitcon-tw/camp2026-game/internal/http/httpx"
	mongomodel "github.com/sitcon-tw/camp2026-game/internal/mongodb/model"
)

var errOpenParticipantMatchExists = errors.New("player already has an open match")

func (h *Handler) ensureNoOpenParticipantMatch(ctx context.Context, playerID string) error {
	err := h.db.Collection(mongomodel.MatchesCollection).
		FindOne(
			ctx,
			openParticipantMatchFilter(playerID),
			options.FindOne().SetProjection(bson.D{{Key: "_id", Value: 1}}),
		).
		Err()
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil
	}
	if err != nil {
		return err
	}
	return errOpenParticipantMatchExists
}

func openHostedMatchFilter(playerID string) bson.M {
	return bson.M{
		"host_player_id": playerID,
		"status":         openMatchStatusFilter(),
	}
}

func (h *Handler) writeCreateMatchProblem(w http.ResponseWriter, r *http.Request, playerID string, err error) {
	if errors.Is(err, errOpenParticipantMatchExists) {
		if h.writeExistingOpenMatchState(w, r, playerID) {
			return
		}
		writeOpenParticipantMatchConflict(w, r)
		return
	}
	httpx.WriteProblem(w, r, httpx.InternalServerError("match creation failed", "match_open_lookup_failed", err))
}

func (h *Handler) writeExistingOpenParticipantMatch(w http.ResponseWriter, r *http.Request, playerID string) {
	if h.writeExistingOpenMatchState(w, r, playerID) {
		return
	}
	writeOpenParticipantMatchConflict(w, r)
}

func writeOpenParticipantMatchConflict(w http.ResponseWriter, r *http.Request) {
	httpx.WriteProblem(w, r, httpx.NewError(http.StatusConflict, "player already has an open match"))
}
