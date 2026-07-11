package fronts

import (
	"errors"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/drivertest"

	mongomodel "github.com/sitcon-tw/camp2026-game/internal/mongodb/model"
	"github.com/sitcon-tw/camp2026-game/internal/testcontent"
)

func TestPlayerFrontSitonesReturnsAllOwnedAndPrioritizesOwnedDefaults(t *testing.T) {
	ownedIDs := []string{
		"stone_2019_blackbox",
		"stone_engineering_base",
		"stone_2020_tour_flag",
		"stone_explorer_base",
		"stone_2021_abacus",
		"stone_2022_maze",
		"stone_2024_finite",
		"stone_2024_ribbon",
	}
	documents := make([]bson.D, 0, len(ownedIDs))
	for i, sitoneID := range ownedIDs {
		documents = append(documents, bson.D{
			{Key: "_id", Value: "owned-" + sitoneID},
			{Key: "player_id", Value: "player-1"},
			{Key: "sitone_id", Value: sitoneID},
			{Key: "quantity", Value: i + 1},
		})
	}
	db := startFrontMockDatabase(t, frontCursorResponse(documents...))
	handler := New(Dependencies{Content: testcontent.Load(t), MongoDB: db})

	inventory, err := handler.playerFrontSitones(t.Context(), mongomodel.Player{
		ID: "player-1",
		DefaultSitoneIDs: []string{
			"stone_explorer_base",
			"stone_inspiration_base",
			"stone_engineering_base",
		},
	})
	if err != nil {
		t.Fatalf("load front sitones: %v", err)
	}
	if len(inventory.Available) != len(ownedIDs) {
		t.Fatalf("owned inventory was truncated: %#v", inventory.Available)
	}
	if inventory.Available[0].SitoneID != "stone_explorer_base" || inventory.Available[1].SitoneID != "stone_engineering_base" {
		t.Fatalf("owned defaults were not prioritized: %#v", inventory.Available)
	}
	if len(inventory.Selected) != 2 || inventory.Selected[0].SitoneID != "stone_explorer_base" || inventory.Selected[1].SitoneID != "stone_engineering_base" {
		t.Fatalf("unowned default was selected: %#v", inventory.Selected)
	}
}

func TestPlayerOwnsFrontSitoneRejectsKnownButUnownedSitone(t *testing.T) {
	db := startFrontMockDatabase(t, frontCursorResponse())
	handler := New(Dependencies{Content: testcontent.Load(t), MongoDB: db})

	owned, err := handler.playerOwnsFrontSitone(t.Context(), "player-1", "stone_engineering_base")
	if err != nil {
		t.Fatalf("check sitone ownership: %v", err)
	}
	if owned {
		t.Fatal("catalog presence without a positive player inventory record must not count as ownership")
	}
}

func TestGrantFrontCommandSitoneIsSourceIdempotent(t *testing.T) {
	db := startFrontMockDatabase(t,
		frontUpdateResponse(0),
		frontCursorResponse(),
		bson.D{{Key: "ok", Value: 1}, {Key: "n", Value: int32(1)}},
		frontUpdateResponse(0),
		frontCursorResponse(bson.D{
			{Key: "_id", Value: "front_sitone_player-1_stone_engineering_base"},
			{Key: "player_id", Value: "player-1"},
			{Key: "sitone_id", Value: "stone_engineering_base"},
			{Key: "quantity", Value: 1},
			{Key: "drop_grant_sources", Value: bson.A{"front:front-1:player:player-1:command:command-1"}},
		}),
	)
	handler := New(Dependencies{MongoDB: db})
	command := mongomodel.FrontCommand{
		ID: "command-1", FrontID: "front-1", PlayerID: "player-1",
		RewardSitoneID: "stone_engineering_base", RewardSitoneQuantity: 1,
	}

	if err := handler.grantFrontCommandSitone(t.Context(), command); err != nil {
		t.Fatalf("first reward grant: %v", err)
	}
	if err := handler.grantFrontCommandSitone(t.Context(), command); err != nil {
		t.Fatalf("idempotent reward replay: %v", err)
	}
}

func TestDeductFrontCommandOpenPowerWritesPlayerLedger(t *testing.T) {
	db := startFrontMockDatabase(t,
		frontCursorResponse(bson.D{{Key: "_id", Value: nil}, {Key: "total", Value: 800}}),
		bson.D{{Key: "ok", Value: 1}, {Key: "n", Value: int32(1)}},
	)
	handler := New(Dependencies{MongoDB: db})
	command := mongomodel.FrontCommand{
		ID: "command-1", PlayerID: "player-1", FrontOpenPowerCost: 15,
		CreatedAt: time.Date(2026, time.July, 10, 12, 0, 0, 0, time.UTC),
	}

	if err := handler.deductFrontCommandOpenPower(t.Context(), command); err != nil {
		t.Fatalf("deduct command open power: %v", err)
	}
}

func TestDeductFrontCommandOpenPowerRejectsInsufficientBalance(t *testing.T) {
	db := startFrontMockDatabase(t,
		frontCursorResponse(bson.D{{Key: "_id", Value: nil}, {Key: "total", Value: 9}}),
	)
	handler := New(Dependencies{MongoDB: db})
	err := handler.deductFrontCommandOpenPower(t.Context(), mongomodel.FrontCommand{
		ID: "command-1", PlayerID: "player-1", FrontOpenPowerCost: 10,
	})
	if !errors.Is(err, errFrontInsufficientOpenPower) {
		t.Fatalf("expected insufficient open power, got %v", err)
	}
}

func TestGrantFrontCommandOpenPowerWritesCaptureReward(t *testing.T) {
	db := startFrontMockDatabase(t, bson.D{{Key: "ok", Value: 1}, {Key: "n", Value: int32(1)}})
	handler := New(Dependencies{MongoDB: db})
	command := mongomodel.FrontCommand{
		ID: "command-1", PlayerID: "player-1", FrontOpenPowerReward: 3,
		CreatedAt: time.Date(2026, time.July, 10, 12, 0, 0, 0, time.UTC),
	}
	if err := handler.grantFrontCommandOpenPower(t.Context(), command); err != nil {
		t.Fatalf("grant capture open power: %v", err)
	}
}

func TestDisplacedGarrisonSitonesReturnToOriginalPlayer(t *testing.T) {
	db := startFrontMockDatabase(t, frontUpdateResponse(1), frontUpdateResponse(1))
	handler := New(Dependencies{MongoDB: db})
	command := mongomodel.FrontCommand{
		Kind: "attack", PlayerID: "attacker",
		DisplacedGarrisons: []mongomodel.FrontDisplacedGarrison{
			{PlayerID: "defender-a", SitoneIDs: []string{"stone-a"}},
			{PlayerID: "defender-b", SitoneIDs: []string{"stone-b"}},
		},
	}
	if err := handler.applyFrontCommandSitoneEscrow(t.Context(), command); err != nil {
		t.Fatalf("return displaced sitones: %v", err)
	}
}

func TestHoldingRewardWritesEveryTeamMemberLedger(t *testing.T) {
	db := startFrontMockDatabase(t,
		frontCursorResponse(
			bson.D{{Key: "_id", Value: "player-a"}},
			bson.D{{Key: "_id", Value: "player-b"}},
		),
		bson.D{{Key: "ok", Value: 1}, {Key: "n", Value: int32(1)}},
		bson.D{{Key: "ok", Value: 1}, {Key: "n", Value: int32(1)}},
	)
	handler := New(Dependencies{MongoDB: db})
	occupiedAt := time.Date(2026, time.July, 11, 4, 0, 0, 0, time.UTC)
	err := handler.recordFrontHoldingRewards(t.Context(), "front-1", []frontHoldingRewardSettlement{{
		TeamID: "team-001", X: 3, Y: 4, Period: 1, Amount: 3,
		OccupiedAt: occupiedAt, SettledAt: occupiedAt.Add(frontTerritoryRewardInterval),
	}})
	if err != nil {
		t.Fatalf("record team holding rewards: %v", err)
	}
}

func startFrontMockDatabase(t *testing.T, responses ...bson.D) *mongo.Database {
	t.Helper()
	deployment := drivertest.NewMockDeployment(responses...)
	clientOptions := options.Client()
	clientOptions.Deployment = deployment
	client, err := mongo.Connect(clientOptions)
	if err != nil {
		t.Fatalf("connect mock mongodb client: %v", err)
	}
	t.Cleanup(func() {
		_ = client.Disconnect(t.Context())
	})
	return client.Database("camp2026_game_test")
}

func frontCursorResponse(batch ...bson.D) bson.D {
	values := make(bson.A, 0, len(batch))
	for _, document := range batch {
		values = append(values, document)
	}
	return bson.D{
		{Key: "ok", Value: 1},
		{Key: "cursor", Value: bson.D{
			{Key: "id", Value: int64(0)},
			{Key: "ns", Value: "camp2026_game_test.player_sitones"},
			{Key: "firstBatch", Value: values},
		}},
	}
}

func frontUpdateResponse(matched int32) bson.D {
	return bson.D{
		{Key: "ok", Value: 1},
		{Key: "n", Value: matched},
		{Key: "nModified", Value: matched},
	}
}
