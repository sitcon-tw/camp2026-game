package achievement

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	mongomodel "github.com/sitcon-tw/camp2026-game/internal/mongodb/model"
)

const (
	ReasonCodexAchievement = "codex_achievement"
	CompleteKey            = "codex_complete"
)

type Definition struct {
	Key                 string
	Name                string
	Tier                int
	SortOrder           int
	RequiredSitoneCount int
	OpenPowerReward     int
}

var codexTierNames = [...]string{
	"石來運轉",
	"石在必得",
	"一石三鳥",
	"與石俱進",
	"石半功倍",
	"三石而立",
	"水滴石穿",
	"石破天驚",
	"他山之石",
	"五石知天命",
}

func CodexDefinitions(totalSitones int) []Definition {
	definitions := make([]Definition, 0, len(codexTierNames)+1)
	for index, name := range codexTierNames {
		tier := index + 1
		definitions = append(definitions, Definition{
			Key:                 fmt.Sprintf("codex_tier_%02d", tier),
			Name:                name,
			Tier:                tier,
			SortOrder:           tier,
			RequiredSitoneCount: tier * 5,
			OpenPowerReward:     500 + 50*(tier-1),
		})
	}
	if totalSitones > 0 {
		definitions = append(definitions, Definition{
			Key:                 CompleteKey,
			Name:                "石全石美",
			SortOrder:           len(codexTierNames) + 1,
			RequiredSitoneCount: totalSitones,
			OpenPowerReward:     1200,
		})
	}
	return definitions
}

func ReconcileCodex(ctx context.Context, db *mongo.Database, playerID string, catalogSitoneIDs []string, now time.Time) (int, error) {
	if db == nil || playerID == "" || len(catalogSitoneIDs) == 0 {
		return 0, nil
	}

	ownedCount, err := db.Collection(mongomodel.PlayerSitonesCollection).CountDocuments(ctx, bson.M{
		"player_id": playerID,
		"sitone_id": bson.M{"$in": catalogSitoneIDs},
	})
	if err != nil {
		return 0, fmt.Errorf("count collected sitones: %w", err)
	}
	if ownedCount < 5 {
		return int(ownedCount), nil
	}

	unlocked, err := unlockedCodexKeys(ctx, db, playerID)
	if err != nil {
		return int(ownedCount), err
	}
	rewarded, err := rewardedCodexKeys(ctx, db, playerID)
	if err != nil {
		return int(ownedCount), err
	}

	for _, definition := range CodexDefinitions(len(catalogSitoneIDs)) {
		if int(ownedCount) < definition.RequiredSitoneCount {
			continue
		}
		if definition.OpenPowerReward > 0 {
			if _, ok := rewarded[definition.Key]; !ok {
				if err := grantOpenPowerReward(ctx, db, playerID, definition, now); err != nil {
					return int(ownedCount), err
				}
			}
		}
		if record, ok := unlocked[definition.Key]; ok {
			if record.OpenPowerReward != definition.OpenPowerReward {
				if err := updateAchievementReward(ctx, db, record.ID, definition.OpenPowerReward); err != nil {
					return int(ownedCount), err
				}
			}
			continue
		}
		if err := recordAchievement(ctx, db, playerID, int(ownedCount), definition, now); err != nil {
			return int(ownedCount), err
		}
	}
	return int(ownedCount), nil
}

func rewardedCodexKeys(ctx context.Context, db *mongo.Database, playerID string) (map[string]struct{}, error) {
	cursor, err := db.Collection(mongomodel.OpenPowerRecordsCollection).Find(
		ctx,
		bson.M{
			"player_id": playerID,
			"reason":    ReasonCodexAchievement,
		},
		options.Find().SetProjection(bson.D{{Key: "source", Value: 1}}),
	)
	if err != nil {
		return nil, fmt.Errorf("find rewarded codex achievements: %w", err)
	}
	defer func() { _ = cursor.Close(ctx) }()

	var records []mongomodel.OpenPowerRecord
	if err := cursor.All(ctx, &records); err != nil {
		return nil, fmt.Errorf("decode rewarded codex achievements: %w", err)
	}
	rewarded := make(map[string]struct{}, len(records))
	for _, record := range records {
		rewarded[record.Source] = struct{}{}
	}
	return rewarded, nil
}

func unlockedCodexKeys(ctx context.Context, db *mongo.Database, playerID string) (map[string]mongomodel.Achievement, error) {
	cursor, err := db.Collection(mongomodel.AchievementsCollection).Find(
		ctx,
		bson.M{"player_id": playerID},
		options.Find().SetProjection(bson.D{
			{Key: "key", Value: 1},
			{Key: "open_power_reward", Value: 1},
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("find unlocked codex achievements: %w", err)
	}
	defer func() {
		_ = cursor.Close(ctx)
	}()

	var records []mongomodel.Achievement
	if err := cursor.All(ctx, &records); err != nil {
		return nil, fmt.Errorf("decode unlocked codex achievements: %w", err)
	}
	unlocked := make(map[string]mongomodel.Achievement, len(records))
	for _, record := range records {
		unlocked[record.Key] = record
	}
	return unlocked, nil
}

func grantOpenPowerReward(ctx context.Context, db *mongo.Database, playerID string, definition Definition, now time.Time) error {
	recordID := openPowerRecordID(playerID, definition.Key)
	_, err := db.Collection(mongomodel.OpenPowerRecordsCollection).UpdateOne(
		ctx,
		bson.M{"_id": recordID},
		bson.M{"$setOnInsert": mongomodel.OpenPowerRecord{
			ID:        recordID,
			PlayerID:  playerID,
			Amount:    definition.OpenPowerReward,
			Reason:    ReasonCodexAchievement,
			Source:    definition.Key,
			CreatedAt: now,
		}},
		options.UpdateOne().SetUpsert(true),
	)
	if err != nil && !mongo.IsDuplicateKeyError(err) {
		return fmt.Errorf("grant %s open power: %w", definition.Key, err)
	}
	return nil
}

func recordAchievement(ctx context.Context, db *mongo.Database, playerID string, sitoneCount int, definition Definition, now time.Time) error {
	recordID := achievementRecordID(playerID, definition.Key)
	_, err := db.Collection(mongomodel.AchievementsCollection).UpdateOne(
		ctx,
		bson.M{"_id": recordID},
		bson.M{"$setOnInsert": mongomodel.Achievement{
			ID:                  recordID,
			PlayerID:            playerID,
			Key:                 definition.Key,
			Name:                definition.Name,
			Tier:                definition.Tier,
			SortOrder:           definition.SortOrder,
			RequiredSitoneCount: definition.RequiredSitoneCount,
			SitoneCount:         sitoneCount,
			OpenPowerReward:     definition.OpenPowerReward,
			CreatedAt:           now,
			NotificationPending: true,
		}},
		options.UpdateOne().SetUpsert(true),
	)
	if err != nil && !mongo.IsDuplicateKeyError(err) {
		return fmt.Errorf("record %s achievement: %w", definition.Key, err)
	}
	return nil
}

func updateAchievementReward(ctx context.Context, db *mongo.Database, achievementID string, reward int) error {
	_, err := db.Collection(mongomodel.AchievementsCollection).UpdateOne(
		ctx,
		bson.M{"_id": achievementID, "open_power_reward": bson.M{"$ne": reward}},
		bson.M{
			"$set": bson.M{
				"open_power_reward":    reward,
				"notification_pending": true,
			},
			"$unset": bson.M{"notified_at": ""},
		},
	)
	if err != nil {
		return fmt.Errorf("update achievement %s reward: %w", achievementID, err)
	}
	return nil
}

func achievementRecordID(playerID, key string) string {
	return "achievement_" + playerID + "_" + key
}

func openPowerRecordID(playerID, key string) string {
	return "achievement_open_power_" + playerID + "_" + key
}
