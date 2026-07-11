package mongodb

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	mongomodel "github.com/sitcon-tw/camp2026-game/internal/mongodb/model"
)

const indexTimeout = 2 * time.Minute

type collectionIndexModels struct {
	collection string
	models     []mongo.IndexModel
}

func EnsureIndexes(ctx context.Context, db *mongo.Database) error {
	indexCtx, cancel := context.WithTimeout(ctx, indexTimeout)
	defer cancel()

	for _, collectionIndexes := range allIndexModelsByCollection() {
		collection := db.Collection(collectionIndexes.collection)
		if _, err := collection.Indexes().CreateMany(indexCtx, collectionIndexes.models); err != nil {
			return fmt.Errorf("ensure %s indexes: %w", collectionIndexes.collection, err)
		}
		if collectionIndexes.collection == mongomodel.ShopPurchasesCollection {
			if err := dropLegacyShopPurchaseIndex(indexCtx, collection); err != nil {
				return fmt.Errorf("drop legacy shop purchase index: %w", err)
			}
		}
	}
	return nil
}

func dropLegacyShopPurchaseIndex(ctx context.Context, collection *mongo.Collection) error {
	err := collection.Indexes().DropOne(ctx, "shop_purchases_player_item")
	if err == nil || ignorableIndexDropError(err) {
		return nil
	}
	return err
}

func ignorableIndexDropError(err error) bool {
	var commandErr mongo.CommandError
	if !errors.As(err, &commandErr) {
		return false
	}
	return commandErr.Code == 26 || commandErr.Code == 27
}

func allIndexModelsByCollection() []collectionIndexModels {
	return append(indexModelsByCollection(), frontIndexModelsByCollection()...)
}

func indexModelsByCollection() []collectionIndexModels {
	return []collectionIndexModels{
		{collection: mongomodel.PlayersCollection, models: playerIndexModels()},
		{collection: mongomodel.MatchesCollection, models: matchIndexModels()},
		{collection: mongomodel.MatchPairingsCollection, models: matchPairingIndexModels()},
		{collection: mongomodel.MatchAnswersCollection, models: matchAnswerIndexModels()},
		{collection: mongomodel.MatchItemDropsCollection, models: matchItemDropIndexModels()},
		{collection: mongomodel.OpenPowerRecordsCollection, models: openPowerRecordIndexModels()},
		{collection: mongomodel.OpenPowerTransfersCollection, models: openPowerTransferIndexModels()},
		{collection: mongomodel.PlayerItemsCollection, models: playerItemIndexModels()},
		{collection: mongomodel.PlayerSitonesCollection, models: playerSitoneIndexModels()},
		{collection: mongomodel.ShopPurchasesCollection, models: shopPurchaseIndexModels()},
		{collection: mongomodel.OpenPowerLocksCollection, models: openPowerLockIndexModels()},
		{collection: mongomodel.AchievementsCollection, models: achievementIndexModels()},
		{collection: mongomodel.StaffRewardsCollection, models: staffRewardIndexModels()},
		{collection: mongomodel.StaffRewardTokensCollection, models: staffRewardTokenIndexModels()},
		{collection: mongomodel.StaffRewardTokenClaimsCollection, models: staffRewardTokenClaimIndexModels()},
		{collection: mongomodel.QRScanCooldownsCollection, models: qrScanCooldownIndexModels()},
		{collection: mongomodel.CommunityStandsCollection, models: communityStandIndexModels()},
		{collection: mongomodel.CommunityStandVisitsCollection, models: communityStandVisitIndexModels()},
		{collection: mongomodel.CommunityStandClaimsCollection, models: communityStandClaimIndexModels()},
		{collection: mongomodel.RoomTeamsCollection, models: roomTeamIndexModels()},
		{collection: mongomodel.RoomTeamMembershipsCollection, models: roomTeamMembershipIndexModels()},
	}
}

func frontIndexModelsByCollection() []collectionIndexModels {
	return []collectionIndexModels{
		{collection: mongomodel.FrontsCollection, models: frontIndexModels()},
		{collection: mongomodel.FrontCommandsCollection, models: frontCommandIndexModels()},
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
		{
			Keys: bson.D{{Key: "open_player_locks", Value: 1}},
			Options: options.Index().
				SetName("matches_open_player_locks").
				SetUnique(true).
				SetPartialFilterExpression(bson.M{"open_player_locks": bson.M{"$exists": true}}),
		},
	}
}

func matchPairingIndexModels() []mongo.IndexModel {
	return []mongo.IndexModel{
		{
			Keys: bson.D{{Key: "token_hash", Value: 1}},
			Options: options.Index().
				SetName("match_pairings_token_hash").
				SetUnique(true),
		},
		{
			Keys: bson.D{
				{Key: "match_id", Value: 1},
				{Key: "created_at", Value: -1},
			},
			Options: options.Index().SetName("match_pairings_match_created"),
		},
		{
			Keys:    bson.D{{Key: "expires_at", Value: 1}},
			Options: options.Index().SetName("match_pairings_expires_at_ttl").SetExpireAfterSeconds(0),
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

func openPowerTransferIndexModels() []mongo.IndexModel {
	return []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "created_at", Value: -1},
				{Key: "_id", Value: -1},
			},
			Options: options.Index().SetName("open_power_transfers_created"),
		},
		{
			Keys: bson.D{
				{Key: "team_id", Value: 1},
				{Key: "created_at", Value: -1},
			},
			Options: options.Index().SetName("open_power_transfers_team_created"),
		},
		{
			Keys: bson.D{
				{Key: "sender_player_id", Value: 1},
				{Key: "created_at", Value: -1},
			},
			Options: options.Index().SetName("open_power_transfers_sender_created"),
		},
		{
			Keys: bson.D{
				{Key: "recipient_player_id", Value: 1},
				{Key: "created_at", Value: -1},
			},
			Options: options.Index().SetName("open_power_transfers_recipient_created"),
		},
		{
			Keys: bson.D{
				{Key: "recipient_player_id", Value: 1},
				{Key: "created_at", Value: 1},
				{Key: "_id", Value: 1},
			},
			Options: options.Index().
				SetName("open_power_transfers_unnotified_recipient_created").
				SetPartialFilterExpression(bson.M{"notification_pending": true}),
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
			Options: options.Index().
				SetName("shop_purchases_non_repeatable_player_item").
				SetUnique(true).
				SetPartialFilterExpression(bson.M{"repeatable": bson.M{"$in": bson.A{false, nil}}}),
		},
	}
}

func openPowerLockIndexModels() []mongo.IndexModel {
	return []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "expires_at", Value: 1}},
			Options: options.Index().SetName("open_power_locks_expires_at_ttl").SetExpireAfterSeconds(0),
		},
	}
}

func achievementIndexModels() []mongo.IndexModel {
	return []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "player_id", Value: 1},
				{Key: "key", Value: 1},
			},
			Options: options.Index().
				SetName("achievements_player_key").
				SetUnique(true),
		},
		{
			Keys: bson.D{
				{Key: "player_id", Value: 1},
				{Key: "created_at", Value: 1},
				{Key: "sort_order", Value: 1},
				{Key: "_id", Value: 1},
			},
			Options: options.Index().
				SetName("achievements_unnotified_player_created").
				SetPartialFilterExpression(bson.M{"notification_pending": true}),
		},
	}
}

func staffRewardIndexModels() []mongo.IndexModel {
	return []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "recipient_player_id", Value: 1},
				{Key: "created_at", Value: -1},
				{Key: "_id", Value: -1},
			},
			Options: options.Index().SetName("staff_rewards_recipient_created"),
		},
		{
			Keys: bson.D{
				{Key: "recipient_player_id", Value: 1},
				{Key: "created_at", Value: 1},
				{Key: "_id", Value: 1},
			},
			Options: options.Index().
				SetName("staff_rewards_unnotified_recipient_created").
				SetPartialFilterExpression(bson.M{"notification_pending": true}),
		},
	}
}

func staffRewardTokenIndexModels() []mongo.IndexModel {
	return []mongo.IndexModel{
		{
			Keys: bson.D{{Key: "token", Value: 1}},
			Options: options.Index().
				SetName("staff_reward_tokens_token").
				SetUnique(true),
		},
		{
			Keys:    bson.D{{Key: "expires_at", Value: 1}},
			Options: options.Index().SetName("staff_reward_tokens_expires_at_ttl").SetExpireAfterSeconds(0),
		},
	}
}

func staffRewardTokenClaimIndexModels() []mongo.IndexModel {
	return []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "token_id", Value: 1},
				{Key: "player_id", Value: 1},
			},
			Options: options.Index().SetName("staff_reward_token_claims_token_player").SetUnique(true),
		},
		{
			Keys: bson.D{
				{Key: "player_id", Value: 1},
				{Key: "claimed_at", Value: -1},
			},
			Options: options.Index().SetName("staff_reward_token_claims_player_claimed"),
		},
	}
}

func qrScanCooldownIndexModels() []mongo.IndexModel {
	return []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "expires_at", Value: 1}},
			Options: options.Index().SetName("qr_scan_cooldowns_expires_at_ttl").SetExpireAfterSeconds(0),
		},
	}
}

func communityStandIndexModels() []mongo.IndexModel {
	return []mongo.IndexModel{
		{
			Keys: bson.D{{Key: "qr_token", Value: 1}},
			Options: options.Index().
				SetName("community_stands_qr_token").
				SetUnique(true).
				SetPartialFilterExpression(bson.M{"qr_token": bson.M{"$gt": ""}}),
		},
		{
			Keys: bson.D{
				{Key: "enabled", Value: -1},
				{Key: "created_at", Value: -1},
				{Key: "_id", Value: 1},
			},
			Options: options.Index().SetName("community_stands_enabled_created"),
		},
	}
}

func communityStandClaimIndexModels() []mongo.IndexModel {
	return []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "stand_id", Value: 1},
				{Key: "player_id", Value: 1},
			},
			Options: options.Index().SetName("community_stand_claims_stand_player").SetUnique(true),
		},
		{
			Keys: bson.D{
				{Key: "player_id", Value: 1},
				{Key: "created_at", Value: -1},
			},
			Options: options.Index().SetName("community_stand_claims_player_created"),
		},
	}
}

func roomTeamIndexModels() []mongo.IndexModel {
	return []mongo.IndexModel{
		{
			Keys: bson.D{{Key: "room_number", Value: 1}},
			Options: options.Index().
				SetName("room_teams_room_number").
				SetUnique(true),
		},
		{
			Keys: bson.D{{Key: "qr_token", Value: 1}},
			Options: options.Index().
				SetName("room_teams_qr_token").
				SetUnique(true).
				SetPartialFilterExpression(bson.M{"qr_token": bson.M{"$gt": ""}}),
		},
	}
}

func roomTeamMembershipIndexModels() []mongo.IndexModel {
	return []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "room_team_id", Value: 1},
				{Key: "player_id", Value: 1},
			},
			Options: options.Index().SetName("room_team_memberships_room_player").SetUnique(true),
		},
		{
			Keys:    bson.D{{Key: "player_id", Value: 1}},
			Options: options.Index().SetName("room_team_memberships_player").SetUnique(true),
		},
	}
}

func communityStandVisitIndexModels() []mongo.IndexModel {
	return []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "stand_id", Value: 1},
				{Key: "player_id", Value: 1},
			},
			Options: options.Index().SetName("community_stand_visits_stand_player").SetUnique(true),
		},
		{
			Keys: bson.D{
				{Key: "stand_id", Value: 1},
				{Key: "last_visited_at", Value: -1},
			},
			Options: options.Index().SetName("community_stand_visits_stand_last_visited"),
		},
	}
}

func frontIndexModels() []mongo.IndexModel {
	return []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "current", Value: -1},
				{Key: "updated_at", Value: -1},
				{Key: "_id", Value: 1},
			},
			Options: options.Index().SetName("fronts_current_updated"),
		},
	}
}

func frontCommandIndexModels() []mongo.IndexModel {
	return []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "front_id", Value: 1},
				{Key: "created_at", Value: -1},
				{Key: "_id", Value: -1},
			},
			Options: options.Index().SetName("front_commands_front_created"),
		},
		{
			Keys: bson.D{
				{Key: "player_id", Value: 1},
				{Key: "created_at", Value: -1},
			},
			Options: options.Index().SetName("front_commands_player_created"),
		},
		{
			Keys: bson.D{
				{Key: "front_id", Value: 1},
				{Key: "player_id", Value: 1},
				{Key: "client_command_id", Value: 1},
			},
			Options: options.Index().
				SetName("front_commands_front_player_client_command").
				SetUnique(true).
				SetPartialFilterExpression(bson.M{"client_command_id": bson.M{"$gt": ""}}),
		},
	}
}
