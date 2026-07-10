package fronts

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	mongomodel "github.com/sitcon-tw/camp2026-game/internal/mongodb/model"
)

func (h *Handler) applyFrontCommandSitoneEscrow(ctx context.Context, command mongomodel.FrontCommand) error {
	switch command.Kind {
	case "station":
		return h.removeEscrowedSitones(ctx, command.PlayerID, command.SitoneIDs)
	case "withdraw":
		return h.grantEscrowedSitones(ctx, command.PlayerID, command.SitoneIDs)
	default:
		captured := make([]string, 0)
		for _, garrison := range command.CapturedGarrisons {
			captured = append(captured, garrison.SitoneIDs...)
		}
		return h.grantEscrowedSitones(ctx, command.PlayerID, captured)
	}
}

func (h *Handler) removeEscrowedSitones(ctx context.Context, playerID string, sitoneIDs []string) error {
	for sitoneID, quantity := range sitoneCounts(sitoneIDs) {
		result, err := h.db.Collection(mongomodel.PlayerSitonesCollection).UpdateOne(
			ctx,
			bson.M{
				"player_id": playerID,
				"sitone_id": sitoneID,
				"quantity":  bson.M{"$gte": quantity},
			},
			bson.M{"$inc": bson.M{"quantity": -quantity}},
		)
		if err != nil {
			return err
		}
		if result.MatchedCount == 0 {
			return errFrontSitoneNotOwned
		}
	}
	return nil
}

func (h *Handler) grantEscrowedSitones(ctx context.Context, playerID string, sitoneIDs []string) error {
	for sitoneID, quantity := range sitoneCounts(sitoneIDs) {
		_, err := h.db.Collection(mongomodel.PlayerSitonesCollection).UpdateOne(
			ctx,
			bson.M{"player_id": playerID, "sitone_id": sitoneID},
			bson.M{
				"$inc": bson.M{"quantity": quantity},
				"$setOnInsert": bson.M{
					"_id":       fmt.Sprintf("front_escrow_%s_%s", playerID, sitoneID),
					"player_id": playerID,
					"sitone_id": sitoneID,
				},
			},
			options.UpdateOne().SetUpsert(true),
		)
		if err != nil {
			return err
		}
	}
	return nil
}
