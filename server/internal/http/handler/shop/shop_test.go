package shop

import (
	"fmt"
	"testing"

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
	if item, ok := shopItemByID(store, "item_polaroid_film"); !ok || item.Locked || item.PriceOpenPower != 650 {
		t.Fatalf("expected polaroid film item to be listed as unlocked at 650, got %#v", item)
	}
	if _, ok := shopItemByID(store, "item_wooden_plank"); ok {
		t.Fatal("expected event-only wooden plank item not to be listed")
	}
	if item, ok := shopItemByID(store, "item_shared_notes_link"); !ok || item.Locked || item.PriceOpenPower != 1650 {
		t.Fatalf("expected shared notes item to be listed as unlocked at 1650, got %#v", item)
	}
	if _, ok := shopItemByID(store, "item_charm_debug"); !ok {
		t.Fatal("expected charm item to be purchasable")
	}
	if item, ok := shopItemByID(store, "item_postcard_sitcon2024"); !ok || item.PriceOpenPower != 800 {
		t.Fatalf("expected SITCON 2024 postcard item to be listed at 800, got %#v", item)
	}
	if item, ok := shopItemByID(store, "stone_pebble"); !ok || item.Type != "sitone" || item.PriceOpenPower != 8000 {
		t.Fatalf("expected Pebble sitone item to be listed at 8000, got %#v", item)
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
	}, 5)

	if response.ID != "item_adventure_backpack" || response.Source != "shop" || response.PriceOpenPower != 150 || !response.Locked || !response.Redeemed || response.PurchaseCount != 5 || response.PurchaseLimit != 5 {
		t.Fatalf("unexpected shop item response: %#v", response)
	}
}

func TestShopItemResponsesIncludesRedeemedState(t *testing.T) {
	responses := shopItemResponses([]content.Item{
		{ID: "item-a", Name: "A", Type: "material", Rarity: "common", PriceOpenPower: 10},
		{ID: "item-b", Name: "B", Type: "material", Rarity: "common", PriceOpenPower: 20},
	}, map[string]int{"item-b": 5})

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
