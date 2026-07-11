package achievement

import (
	"fmt"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/drivertest"
)

func TestCodexDefinitions(t *testing.T) {
	definitions := CodexDefinitions(52)
	if len(definitions) != 11 {
		t.Fatalf("expected 11 achievements, got %d", len(definitions))
	}

	first := definitions[0]
	if first.Name != "石來運轉" || first.Tier != 1 || first.RequiredSitoneCount != 5 || first.OpenPowerReward != 500 {
		t.Fatalf("unexpected first tier: %#v", first)
	}

	lastTier := definitions[9]
	if lastTier.Name != "五石知天命" || lastTier.Tier != 10 || lastTier.RequiredSitoneCount != 50 || lastTier.OpenPowerReward != 950 {
		t.Fatalf("unexpected tenth tier: %#v", lastTier)
	}

	complete := definitions[10]
	if complete.Key != CompleteKey || complete.Name != "石全石美" || complete.RequiredSitoneCount != 52 || complete.OpenPowerReward != 1200 {
		t.Fatalf("unexpected completion achievement: %#v", complete)
	}
}

func TestCodexRewardsTotal8450OpenPower(t *testing.T) {
	total := 0
	for _, definition := range CodexDefinitions(52) {
		total += definition.OpenPowerReward
	}
	if total != 8450 {
		t.Fatalf("expected 8450 total open power, got %d", total)
	}
}

func TestReconcileCodexDoesNotRewriteUnlockedAchievement(t *testing.T) {
	db := startAchievementMockDatabase(t,
		achievementCursorResponse("camp2026_game_test.player_sitones", bson.D{{Key: "n", Value: int32(5)}}),
		achievementCursorResponse("camp2026_game_test.achievements"),
		achievementCursorResponse("camp2026_game_test.open_power_records"),
		achievementUpdateResponse(1),
		achievementUpdateResponse(1),
		achievementCursorResponse("camp2026_game_test.player_sitones", bson.D{{Key: "n", Value: int32(5)}}),
		achievementCursorResponse("camp2026_game_test.achievements", bson.D{
			{Key: "_id", Value: "achievement_player-a_codex_tier_01"},
			{Key: "key", Value: "codex_tier_01"},
			{Key: "open_power_reward", Value: 500},
		}),
		achievementCursorResponse("camp2026_game_test.open_power_records", bson.D{
			{Key: "source", Value: "codex_tier_01"},
		}),
	)
	ids := []string{"s1", "s2", "s3", "s4", "s5", "s6"}

	if _, err := ReconcileCodex(t.Context(), db, "player-a", ids, time.Now().UTC()); err != nil {
		t.Fatalf("first reconciliation: %v", err)
	}
	if _, err := ReconcileCodex(t.Context(), db, "player-a", ids, time.Now().UTC()); err != nil {
		t.Fatalf("second reconciliation: %v", err)
	}
}

func TestReconcileCodexBackfillsCompleteReward(t *testing.T) {
	definitions := CodexDefinitions(52)
	achievementRecords := make([]bson.D, 0, len(definitions))
	rewardRecords := make([]bson.D, 0, len(definitions)-1)
	for _, definition := range definitions {
		reward := definition.OpenPowerReward
		if definition.Key == CompleteKey {
			reward = 0
		} else {
			rewardRecords = append(rewardRecords, bson.D{{Key: "source", Value: definition.Key}})
		}
		achievementRecords = append(achievementRecords, bson.D{
			{Key: "_id", Value: "achievement_player-a_" + definition.Key},
			{Key: "key", Value: definition.Key},
			{Key: "open_power_reward", Value: reward},
		})
	}

	db := startAchievementMockDatabase(t,
		achievementCursorResponse("camp2026_game_test.player_sitones", bson.D{{Key: "n", Value: int32(52)}}),
		achievementCursorResponse("camp2026_game_test.achievements", achievementRecords...),
		achievementCursorResponse("camp2026_game_test.open_power_records", rewardRecords...),
		achievementUpdateResponse(1),
		achievementUpdateResponse(1),
	)
	ids := make([]string, 52)
	for index := range ids {
		ids[index] = fmt.Sprintf("stone-%d", index+1)
	}

	count, err := ReconcileCodex(t.Context(), db, "player-a", ids, time.Now().UTC())
	if err != nil {
		t.Fatalf("backfill complete reward: %v", err)
	}
	if count != 52 {
		t.Fatalf("expected 52 collected sitones, got %d", count)
	}
}

func startAchievementMockDatabase(t *testing.T, responses ...bson.D) *mongo.Database {
	t.Helper()
	deployment := drivertest.NewMockDeployment(responses...)
	clientOptions := options.Client()
	clientOptions.Deployment = deployment
	client, err := mongo.Connect(clientOptions)
	if err != nil {
		t.Fatalf("connect mock mongodb client: %v", err)
	}
	t.Cleanup(func() { _ = client.Disconnect(t.Context()) })
	return client.Database("camp2026_game_test")
}

func achievementCursorResponse(namespace string, batch ...bson.D) bson.D {
	values := make(bson.A, 0, len(batch))
	for _, document := range batch {
		values = append(values, document)
	}
	return bson.D{
		{Key: "ok", Value: 1},
		{Key: "cursor", Value: bson.D{
			{Key: "id", Value: int64(0)},
			{Key: "ns", Value: namespace},
			{Key: "firstBatch", Value: values},
		}},
	}
}

func achievementUpdateResponse(modified int32) bson.D {
	return bson.D{
		{Key: "ok", Value: 1},
		{Key: "n", Value: modified},
		{Key: "nModified", Value: modified},
	}
}
