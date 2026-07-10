package fronts

import (
	"context"
	"errors"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	mongomodel "github.com/sitcon-tw/camp2026-game/internal/mongodb/model"
)

const frontSitoneRewardQuantity = 1

var frontBaseSitoneRewardIDs = []string{
	"stone_explorer_base",
	"stone_inspiration_base",
	"stone_resonance_base",
	"stone_engineering_base",
	"stone_entertainment_base",
}

func (h *Handler) assignFrontCommandReward(command *mongomodel.FrontCommand) {
	if command == nil || !command.Applied || h.content == nil {
		return
	}
	if command.RewardSitoneID != "" && command.RewardSitoneQuantity > 0 {
		return
	}
	quantity := command.RewardSitoneQuantity
	if command.Kind == "rescue" {
		quantity += frontSitoneRewardQuantity
	}
	if quantity <= 0 {
		return
	}
	pool := make([]string, 0, len(frontBaseSitoneRewardIDs))
	for _, sitoneID := range frontBaseSitoneRewardIDs {
		if _, ok := h.content.GetSitone(sitoneID); ok {
			pool = append(pool, sitoneID)
		}
	}
	if len(pool) == 0 {
		return
	}
	key := command.FrontID + ":" + command.PlayerID + ":" + command.ClientCommandID + ":" + command.ID
	command.RewardSitoneID = pool[stableTeamIndex(key, len(pool))]
	command.RewardSitoneQuantity = quantity
}

func (h *Handler) grantFrontCommandSitone(ctx context.Context, command mongomodel.FrontCommand) error {
	if command.RewardSitoneID == "" || command.RewardSitoneQuantity <= 0 {
		return nil
	}
	source := fmt.Sprintf("front:%s:player:%s:command:%s", command.FrontID, command.PlayerID, command.ID)
	recordID := fmt.Sprintf("front_sitone_%s_%s", command.PlayerID, command.RewardSitoneID)
	collection := h.db.Collection(mongomodel.PlayerSitonesCollection)
	result, err := collection.UpdateOne(
		ctx,
		bson.M{
			"player_id": command.PlayerID, "sitone_id": command.RewardSitoneID,
			"drop_grant_sources": bson.M{"$ne": source},
		},
		bson.M{
			"$inc":      bson.M{"quantity": command.RewardSitoneQuantity},
			"$addToSet": bson.M{"drop_grant_sources": source},
		},
	)
	if err != nil || result.MatchedCount > 0 {
		return err
	}

	err = collection.FindOne(ctx, bson.M{"player_id": command.PlayerID, "drop_grant_sources": source}).Err()
	if err == nil {
		return nil
	}
	if !errors.Is(err, mongo.ErrNoDocuments) {
		return err
	}

	_, err = collection.InsertOne(ctx, bson.M{
		"_id": recordID, "player_id": command.PlayerID, "sitone_id": command.RewardSitoneID,
		"quantity": command.RewardSitoneQuantity, "drop_grant_sources": []string{source},
	})
	if err == nil {
		return nil
	}
	if !mongo.IsDuplicateKeyError(err) {
		return err
	}
	_, err = collection.UpdateOne(
		ctx,
		bson.M{
			"player_id": command.PlayerID, "sitone_id": command.RewardSitoneID,
			"drop_grant_sources": bson.M{"$ne": source},
		},
		bson.M{
			"$inc":      bson.M{"quantity": command.RewardSitoneQuantity},
			"$addToSet": bson.M{"drop_grant_sources": source},
		},
		options.UpdateOne(),
	)
	return err
}
