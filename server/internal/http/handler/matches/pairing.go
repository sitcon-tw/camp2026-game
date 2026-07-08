package matches

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	mongomodel "github.com/sitcon-tw/camp2026-game/internal/mongodb/model"
)

const matchPairingTokenTTL = 15 * time.Second

func (h *Handler) createPairingToken(ctx context.Context, match mongomodel.Match, hostPlayerID string) (mongomodel.MatchPairing, error) {
	var lastErr error
	for i := 0; i < 5; i++ {
		token, err := newMatchPairingToken()
		if err != nil {
			return mongomodel.MatchPairing{}, err
		}
		id, err := newID("match_pairing")
		if err != nil {
			return mongomodel.MatchPairing{}, err
		}

		now := time.Now()
		pairing := mongomodel.MatchPairing{
			ID:           id,
			Token:        token,
			MatchID:      match.ID,
			HostPlayerID: hostPlayerID,
			CreatedAt:    now,
			ExpiresAt:    now.Add(matchPairingTokenTTL),
		}
		_, err = h.db.Collection(mongomodel.MatchPairingsCollection).InsertOne(ctx, pairing)
		if err == nil {
			return pairing, nil
		}
		if !mongo.IsDuplicateKeyError(err) {
			return mongomodel.MatchPairing{}, err
		}
		lastErr = err
	}
	return mongomodel.MatchPairing{}, lastErr
}

func (h *Handler) pairingResponse(ctx context.Context, match mongomodel.Match, playerID string) (MatchPairingResponse, error) {
	state, err := h.openMatchState(ctx, match.ID, playerID)
	if err != nil {
		return MatchPairingResponse{}, err
	}
	pairing, err := h.createPairingToken(ctx, match, playerID)
	if err != nil {
		return MatchPairingResponse{}, err
	}
	return MatchPairingResponse{
		Match:     state,
		Token:     pairing.Token,
		ExpiresAt: pairing.ExpiresAt,
	}, nil
}

func matchCanIssuePairingToken(match mongomodel.Match, playerID string) bool {
	return match.Status == mongomodel.MatchStatusWaiting &&
		match.HostPlayerID == playerID &&
		matchMode(match) == mongomodel.MatchModePVP &&
		isParticipant(match, playerID) &&
		len(humanParticipantIDs(match)) == 1
}

func (h *Handler) findActivePairingToken(ctx context.Context, token string, now time.Time) (mongomodel.MatchPairing, error) {
	var pairing mongomodel.MatchPairing
	err := h.db.Collection(mongomodel.MatchPairingsCollection).
		FindOne(ctx, bson.M{
			"token":       token,
			"expires_at":  bson.M{"$gt": now},
			"consumed_at": bson.M{"$exists": false},
		}).
		Decode(&pairing)
	return pairing, err
}

func (h *Handler) consumePairingToken(ctx context.Context, pairing mongomodel.MatchPairing, playerID string) error {
	_, err := h.db.Collection(mongomodel.MatchPairingsCollection).UpdateOne(
		ctx,
		bson.M{
			"_id":         pairing.ID,
			"consumed_at": bson.M{"$exists": false},
		},
		bson.M{
			"$set": bson.M{
				"consumed_by_player_id": playerID,
				"consumed_at":           time.Now(),
			},
		},
	)
	return err
}

func isMissingPairingToken(err error) bool {
	return errors.Is(err, mongo.ErrNoDocuments)
}
