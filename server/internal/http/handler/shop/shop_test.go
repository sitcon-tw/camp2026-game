package shop

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/sitcon-tw/camp2026-game/internal/content"
	"github.com/sitcon-tw/camp2026-game/internal/testcontent"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

func TestShopItemsIncludesAllEnabledPurchasableContentItems(t *testing.T) {
	store := loadTestContent(t)

	items := shopItems(store)
	expectedCount := 0
	for _, item := range store.ListItems() {
		if item.Purchasable && item.Enabled {
			expectedCount++
		}
	}
	if len(items) != expectedCount {
		t.Fatalf("expected %d shop items, got %#v", expectedCount, items)
	}
	if items[0].ID != "item_adventure_backpack" || items[0].PriceOpenPower != 150 {
		t.Fatalf("unexpected first shop item: %#v", items[0])
	}
	if item, ok := shopItemByID(store, "item_polaroid_film"); !ok || !item.Locked {
		t.Fatalf("expected polaroid film item to be listed as locked, got %#v", item)
	}
	if item, ok := shopItemByID(store, "item_wooden_plank"); !ok || !item.Locked {
		t.Fatalf("expected wooden plank item to be listed as locked, got %#v", item)
	}
	if _, ok := shopItemByID(store, "item_shared_notes_link"); !ok {
		t.Fatal("expected shared notes item to be listed")
	}
	if _, ok := shopItemByID(store, "item_charm_debug"); !ok {
		t.Fatal("expected charm item to be purchasable")
	}
	if _, ok := shopItemByID(store, "item_tshirt_2026"); ok {
		t.Fatal("expected disabled cosmetic item not to be listed")
	}
}

func TestShopItemResponse(t *testing.T) {
	response := shopItemResponse(content.Item{
		ID:             "item_adventure_backpack",
		Name:           "冒險背包",
		Type:           "material",
		Rarity:         "common",
		Description:    "冒險背包，可用於小石合成。",
		Source:         "shop",
		PriceOpenPower: 150,
		Locked:         true,
	}, true)

	if response.ID != "item_adventure_backpack" || response.Source != "shop" || response.PriceOpenPower != 150 || !response.Locked || !response.Redeemed {
		t.Fatalf("unexpected shop item response: %#v", response)
	}
}

func TestShopItemResponsesIncludesRedeemedState(t *testing.T) {
	responses := shopItemResponses([]content.Item{
		{ID: "item-a", Name: "A", Type: "material", Rarity: "common", PriceOpenPower: 10},
		{ID: "item-b", Name: "B", Type: "material", Rarity: "common", PriceOpenPower: 20},
	}, map[string]struct{}{"item-b": {}})

	if len(responses) != 2 {
		t.Fatalf("expected 2 responses, got %#v", responses)
	}
	if responses[0].Redeemed {
		t.Fatalf("expected first item not redeemed: %#v", responses[0])
	}
	if !responses[1].Redeemed {
		t.Fatalf("expected second item redeemed: %#v", responses[1])
	}
}

func TestOpenPowerTotalPipeline(t *testing.T) {
	pipeline := openPowerTotalPipeline("player-a")
	if len(pipeline) != 2 {
		t.Fatalf("expected 2 pipeline stages, got %#v", pipeline)
	}

	matchStage, ok := pipeline[0][0].Value.(bson.D)
	if !ok {
		t.Fatalf("expected match stage document, got %#v", pipeline[0][0].Value)
	}
	var got any
	for _, element := range matchStage {
		if element.Key == "player_id" {
			got = element.Value
			break
		}
	}
	if got != "player-a" {
		t.Fatalf("expected player id match, got %#v", got)
	}
}

func TestShopPurchaseLockDocumentsArePlayerScoped(t *testing.T) {
	now := time.Date(2026, 6, 29, 12, 0, 0, 0, time.UTC)
	lockID := shopPurchaseLockID("player-a")
	ownerID := "shop_purchase_lock_owner"

	if lockID != "shop_purchase:player-a" {
		t.Fatalf("unexpected lock id: %q", lockID)
	}

	filter := shopPurchaseLockFilter(lockID, ownerID, now)
	if filter["_id"] != lockID {
		t.Fatalf("expected lock id filter, got %#v", filter)
	}
	clauses, ok := filter["$or"].(bson.A)
	if !ok || len(clauses) != 2 {
		t.Fatalf("expected expired-or-owner filter, got %#v", filter["$or"])
	}
	expired, ok := clauses[0].(bson.M)
	if !ok {
		t.Fatalf("expected expired clause, got %#v", clauses[0])
	}
	expiresAt, ok := expired["expires_at"].(bson.M)
	if !ok || !expiresAt["$lte"].(time.Time).Equal(now) {
		t.Fatalf("expected expired lock comparison, got %#v", expired)
	}
	owner, ok := clauses[1].(bson.M)
	if !ok || owner["owner_id"] != ownerID {
		t.Fatalf("expected owner clause, got %#v", clauses[1])
	}

	update := shopPurchaseLockUpdate(lockID, "player-a", ownerID, now)
	set, ok := update["$set"].(bson.M)
	if !ok {
		t.Fatalf("expected set update, got %#v", update)
	}
	if set["owner_id"] != ownerID {
		t.Fatalf("expected owner update, got %#v", set)
	}
	lockExpiresAt, ok := set["expires_at"].(time.Time)
	if !ok || !lockExpiresAt.Equal(now.Add(shopPurchaseLockTTL)) {
		t.Fatalf("expected lock expiry, got %#v", set["expires_at"])
	}
	setOnInsert, ok := update["$setOnInsert"].(bson.M)
	if !ok {
		t.Fatalf("expected setOnInsert update, got %#v", update)
	}
	if setOnInsert["_id"] != lockID || setOnInsert["player_id"] != "player-a" {
		t.Fatalf("expected player-scoped insert fields, got %#v", setOnInsert)
	}
	createdAt, ok := setOnInsert["created_at"].(time.Time)
	if !ok || !createdAt.Equal(now) {
		t.Fatalf("expected created_at, got %#v", setOnInsert["created_at"])
	}
}

func TestShopPurchaseLockBusy(t *testing.T) {
	if !shopPurchaseLockBusy(mongo.ErrNoDocuments) {
		t.Fatal("expected no matching lock document to be treated as contention")
	}
	if !shopPurchaseLockBusy(mongo.WriteException{WriteErrors: mongo.WriteErrors{
		{Code: 11000, Message: "duplicate key"},
	}}) {
		t.Fatal("expected duplicate lock insert to be treated as contention")
	}

	for _, err := range []error{
		errors.New("database unavailable"),
		mongo.CommandError{Code: 20, Message: "not a duplicate key"},
	} {
		if shopPurchaseLockBusy(err) {
			t.Fatalf("expected %v not to be lock contention", err)
		}
	}
}

func TestTransactionUnsupported(t *testing.T) {
	err := fmt.Errorf("wrapped: %w", mongo.CommandError{
		Code:    20,
		Message: "Transaction numbers are only allowed on a replica set member or mongos",
	})
	if !transactionUnsupported(err) {
		t.Fatal("expected standalone transaction error to be unsupported")
	}

	for _, err := range []error{
		mongo.CommandError{Code: 19, Message: "Transaction numbers are only allowed"},
		mongo.CommandError{Code: 20, Message: "not a transaction error"},
		fmt.Errorf("plain error"),
	} {
		if transactionUnsupported(err) {
			t.Fatalf("expected %v not to be unsupported", err)
		}
	}
}

func loadTestContent(t *testing.T) *content.Store {
	t.Helper()

	return testcontent.Load(t)
}
