package fusions

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/sitcon-tw/camp2026-game/internal/content"
	"github.com/sitcon-tw/camp2026-game/internal/testcontent"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

func TestRecipeAvailable(t *testing.T) {
	recipe := content.FusionRecipe{
		Inputs: []content.FusionComponent{
			{Kind: content.FusionKindSitone, ID: "stone_engineering_base", Quantity: 1},
			{Kind: content.FusionKindItem, ID: "item_adventure_backpack", Quantity: 3},
		},
	}
	inventory := inventoryCounts{
		sitones: map[string]int{"stone_engineering_base": 1},
		items:   map[string]int{"item_adventure_backpack": 3},
	}
	if !recipeAvailable(recipe, inventory) {
		t.Fatal("expected recipe to be available")
	}

	inventory.items["item_adventure_backpack"] = 2
	if recipeAvailable(recipe, inventory) {
		t.Fatal("expected recipe to be unavailable with insufficient items")
	}
}

func TestInventoryCollection(t *testing.T) {
	collection, field, err := inventoryCollection(content.FusionKindItem)
	if err != nil {
		t.Fatalf("item collection: %v", err)
	}
	if collection != "player_items" || field != "item_id" {
		t.Fatalf("unexpected item collection mapping: %s %s", collection, field)
	}

	collection, field, err = inventoryCollection(content.FusionKindSitone)
	if err != nil {
		t.Fatalf("sitone collection: %v", err)
	}
	if collection != "player_sitones" || field != "sitone_id" {
		t.Fatalf("unexpected sitone collection mapping: %s %s", collection, field)
	}
}

func TestModelComponents(t *testing.T) {
	components := modelComponents([]content.FusionComponent{
		{Kind: content.FusionKindItem, ID: "item_adventure_backpack", Quantity: 3},
	})
	if len(components) != 1 {
		t.Fatalf("expected 1 component, got %#v", components)
	}
	if components[0].RefID != "item_adventure_backpack" || components[0].Quantity != 3 {
		t.Fatalf("unexpected model component: %#v", components[0])
	}
}

func TestCreateFusionRequiresTransactionClient(t *testing.T) {
	handler := &Handler{}

	_, err := handler.createFusion(context.Background(), "player-a", content.FusionRecipe{ID: "fusion-a"})
	if !errors.Is(err, errFusionTransactionsUnavailable) {
		t.Fatalf("expected transaction unavailable error, got %v", err)
	}
}

func TestBuildFillPlanForCompleteRecipe(t *testing.T) {
	store := loadTestContent(t)
	handler := &Handler{content: store}
	recipe, ok := store.GetFusionRecipe("recipe_explore_2026_backpack_s1_s2")
	if !ok {
		t.Fatal("expected recipe")
	}

	plan, err := handler.buildFillPlan(context.Background(), "player-a", recipe, inventoryCounts{
		sitones: map[string]int{"stone_explorer_base": 1},
		items:   map[string]int{"item_adventure_backpack": 1},
	})
	if err != nil {
		t.Fatalf("build plan: %v", err)
	}
	if len(plan) != 0 {
		t.Fatalf("expected no missing materials, got %#v", plan)
	}
}

func TestBuildFillPlanForPartiallyMissingMaterials(t *testing.T) {
	store := loadTestContent(t)
	handler := &Handler{content: store}
	recipe, ok := store.GetFusionRecipe("recipe_explore_2026_backpack_s1_s2")
	if !ok {
		t.Fatal("expected recipe")
	}

	plan, err := handler.buildFillPlan(context.Background(), "player-a", recipe, inventoryCounts{
		sitones: map[string]int{"stone_explorer_base": 1},
		items:   map[string]int{},
	})
	if err != nil {
		t.Fatalf("build plan: %v", err)
	}
	if len(plan) != 1 {
		t.Fatalf("expected one missing material, got %#v", plan)
	}
	if plan[0].component.ID != "item_adventure_backpack" || !plan[0].fillable {
		t.Fatalf("expected missing backpack to be fillable, got %#v", plan[0])
	}
}

func TestBuildFillPlanForFullyMissingMaterials(t *testing.T) {
	store := loadTestContent(t)
	handler := &Handler{content: store}
	recipe, ok := store.GetFusionRecipe("recipe_explore_2026_backpack_s1_s2")
	if !ok {
		t.Fatal("expected recipe")
	}

	plan, err := handler.buildFillPlan(context.Background(), "player-a", recipe, inventoryCounts{})
	if err != nil {
		t.Fatalf("build plan: %v", err)
	}
	if len(plan) != 2 {
		t.Fatalf("expected two missing materials, got %#v", plan)
	}
	if plan[0].component.Kind != content.FusionKindSitone || plan[0].fillable {
		t.Fatalf("expected missing sitone to be unfillable, got %#v", plan[0])
	}
	if plan[1].component.Kind != content.FusionKindItem || !plan[1].fillable {
		t.Fatalf("expected missing item to be fillable, got %#v", plan[1])
	}
}

func TestBuildFillPlanForUnavailableMaterials(t *testing.T) {
	store := loadTestContent(t)
	handler := &Handler{content: store}
	recipe := content.FusionRecipe{
		ID:      "recipe-unavailable",
		Enabled: true,
		Inputs: []content.FusionComponent{
			{Kind: content.FusionKindItem, ID: "item_essence_timer", Quantity: 1},
		},
	}

	plan, err := handler.buildFillPlan(context.Background(), "player-a", recipe, inventoryCounts{})
	if err != nil {
		t.Fatalf("build plan: %v", err)
	}
	if len(plan) != 1 {
		t.Fatalf("expected one missing material, got %#v", plan)
	}
	if plan[0].fillable {
		t.Fatalf("expected locked material to be unfillable, got %#v", plan[0])
	}
	if plan[0].reason == "" {
		t.Fatalf("expected unavailable reason, got %#v", plan[0])
	}
}

func loadTestContent(t *testing.T) *content.Store {
	t.Helper()
	return testcontent.Load(t)
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
