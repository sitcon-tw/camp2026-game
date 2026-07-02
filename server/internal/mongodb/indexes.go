package mongodb

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	mongomodel "github.com/sitcon-tw/camp2026-game/internal/mongodb/model"
)

const indexTimeout = 2 * time.Minute
const shopPurchaseLocksCollection = "shop_purchase_locks"

type collectionIndexModels struct {
	collection string
	models     []mongo.IndexModel
}

func EnsureIndexes(ctx context.Context, db *mongo.Database) error {
	indexCtx, cancel := context.WithTimeout(ctx, indexTimeout)
	defer cancel()

	for _, collectionIndexes := range indexModelsByCollection() {
		if _, err := db.Collection(collectionIndexes.collection).Indexes().CreateMany(indexCtx, collectionIndexes.models); err != nil {
			return fmt.Errorf("ensure %s indexes: %w", collectionIndexes.collection, err)
		}
	}
	return nil
}

func indexModelsByCollection() []collectionIndexModels {
	return []collectionIndexModels{
		{collection: mongomodel.PlayersCollection, models: playerIndexModels()},
		{collection: mongomodel.MatchesCollection, models: matchIndexModels()},
		{collection: mongomodel.MatchAnswersCollection, models: matchAnswerIndexModels()},
		{collection: mongomodel.MatchItemDropsCollection, models: matchItemDropIndexModels()},
		{collection: mongomodel.OpenPowerRecordsCollection, models: openPowerRecordIndexModels()},
		{collection: mongomodel.PlayerItemsCollection, models: playerItemIndexModels()},
		{collection: mongomodel.PlayerSitonesCollection, models: playerSitoneIndexModels()},
		{collection: mongomodel.ShopPurchasesCollection, models: shopPurchaseIndexModels()},
		{collection: shopPurchaseLocksCollection, models: shopPurchaseLockIndexModels()},
	}
}

func playerIndexModels() []mongo.IndexModel {
	return []mongo.IndexModel{
		{
			Keys: bson.D{{Key: "telegram_user_id", Value: 1}},
			Options: options.Index().
				SetName("telegram_user_id_1").
				SetUnique(true).
				SetPartialFilterExpression(bson.M{"telegram_user_id": bson.M{"$exists": true}}),
		},
		{
			Keys: bson.D{{Key: "auth_token", Value: 1}},
			Options: options.Index().
				SetName("players_auth_token").
				SetUnique(true).
				SetPartialFilterExpression(bson.M{"auth_token": bson.M{"$gt": ""}}),
		},
		{
			Keys: bson.D{{Key: "qrcode_token", Value: 1}},
			Options: options.Index().
				SetName("players_qrcode_token").
				SetUnique(true).
				SetPartialFilterExpression(bson.M{"qrcode_token": bson.M{"$gt": ""}}),
		},
		{
			Keys: bson.D{
				{Key: "team_id", Value: 1},
				{Key: "nickname", Value: 1},
				{Key: "_id", Value: 1},
			},
			Options: options.Index().SetName("players_team_nickname"),
		},
	}
}

func matchIndexModels() []mongo.IndexModel {
	return []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "code", Value: 1},
				{Key: "status", Value: 1},
			},
			Options: options.Index().SetName("matches_code_status"),
		},
		{
			Keys: bson.D{
				{Key: "status", Value: 1},
				{Key: "players.player_id", Value: 1},
				{Key: "completed_at", Value: -1},
				{Key: "created_at", Value: -1},
			},
			Options: options.Index().SetName("matches_status_player_completed_created"),
		},
		{
			Keys: bson.D{{Key: "open_host_lock", Value: 1}},
			Options: options.Index().
				SetName("matches_open_host_lock").
				SetUnique(true).
				SetPartialFilterExpression(bson.M{"open_host_lock": bson.M{"$gt": ""}}),
		},
	}
}

func matchAnswerIndexModels() []mongo.IndexModel {
	return []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "match_id", Value: 1},
				{Key: "player_id", Value: 1},
				{Key: "question_id", Value: 1},
			},
			Options: options.Index().SetName("match_answers_match_player_question"),
		},
		{
			Keys: bson.D{
				{Key: "match_id", Value: 1},
				{Key: "question_id", Value: 1},
				{Key: "answered_at", Value: 1},
			},
			Options: options.Index().SetName("match_answers_match_question_answered_at"),
		},
	}
}

func matchItemDropIndexModels() []mongo.IndexModel {
	return []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "match_id", Value: 1},
				{Key: "player_id", Value: 1},
			},
			Options: options.Index().SetName("match_item_drops_match_player"),
		},
	}
}

func openPowerRecordIndexModels() []mongo.IndexModel {
	return []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "player_id", Value: 1}},
			Options: options.Index().SetName("open_power_records_player"),
		},
		{
			Keys: bson.D{
				{Key: "reason", Value: 1},
				{Key: "source", Value: 1},
				{Key: "player_id", Value: 1},
			},
			Options: options.Index().SetName("open_power_records_reason_source_player"),
		},
	}
}

func playerItemIndexModels() []mongo.IndexModel {
	return []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "player_id", Value: 1},
				{Key: "item_id", Value: 1},
			},
			Options: options.Index().SetName("player_items_player_item").SetUnique(true),
		},
	}
}

func playerSitoneIndexModels() []mongo.IndexModel {
	return []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "player_id", Value: 1},
				{Key: "sitone_id", Value: 1},
			},
			Options: options.Index().SetName("player_sitones_player_sitone").SetUnique(true),
		},
	}
}

func shopPurchaseIndexModels() []mongo.IndexModel {
	return []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "player_id", Value: 1},
				{Key: "item_id", Value: 1},
			},
			Options: options.Index().SetName("shop_purchases_player_item").SetUnique(true),
		},
	}
}

func shopPurchaseLockIndexModels() []mongo.IndexModel {
	return []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "expires_at", Value: 1}},
			Options: options.Index().SetName("shop_purchase_locks_expires_at_ttl").SetExpireAfterSeconds(0),
		},
	}
}
