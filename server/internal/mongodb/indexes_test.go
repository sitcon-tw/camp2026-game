package mongodb

import (
	"reflect"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	mongomodel "github.com/sitcon-tw/camp2026-game/internal/mongodb/model"
)

func TestIndexModelsByCollection(t *testing.T) {
	models := flattenIndexModels(indexModelsByCollection())

	expected := []expectedIndexModel{
		{
			collection: mongomodel.PlayersCollection,
			name:       "telegram_user_id_1",
			keys:       bson.D{{Key: "telegram_user_id", Value: 1}},
			unique:     true,
			partial:    bson.M{"telegram_user_id": bson.M{"$exists": true}},
		},
		{
			collection: mongomodel.PlayersCollection,
			name:       "players_auth_token",
			keys:       bson.D{{Key: "auth_token", Value: 1}},
			unique:     true,
			partial:    bson.M{"auth_token": bson.M{"$gt": ""}},
		},
		{
			collection: mongomodel.PlayersCollection,
			name:       "players_qrcode_token",
			keys:       bson.D{{Key: "qrcode_token", Value: 1}},
			unique:     true,
			partial:    bson.M{"qrcode_token": bson.M{"$gt": ""}},
		},
		{
			collection: mongomodel.PlayersCollection,
			name:       "players_team_nickname",
			keys: bson.D{
				{Key: "team_id", Value: 1},
				{Key: "nickname", Value: 1},
				{Key: "_id", Value: 1},
			},
		},
		{
			collection: mongomodel.MatchesCollection,
			name:       "matches_code_status",
			keys: bson.D{
				{Key: "code", Value: 1},
				{Key: "status", Value: 1},
			},
		},
		{
			collection: mongomodel.MatchesCollection,
			name:       "matches_status_player_completed_created",
			keys: bson.D{
				{Key: "status", Value: 1},
				{Key: "players.player_id", Value: 1},
				{Key: "completed_at", Value: -1},
				{Key: "created_at", Value: -1},
			},
		},
		{
			collection: mongomodel.MatchesCollection,
			name:       "matches_open_host_lock",
			keys:       bson.D{{Key: "open_host_lock", Value: 1}},
			unique:     true,
			partial:    bson.M{"open_host_lock": bson.M{"$gt": ""}},
		},
		{
			collection: mongomodel.MatchAnswersCollection,
			name:       "match_answers_match_player_question",
			keys: bson.D{
				{Key: "match_id", Value: 1},
				{Key: "player_id", Value: 1},
				{Key: "question_id", Value: 1},
			},
		},
		{
			collection: mongomodel.MatchAnswersCollection,
			name:       "match_answers_match_question_answered_at",
			keys: bson.D{
				{Key: "match_id", Value: 1},
				{Key: "question_id", Value: 1},
				{Key: "answered_at", Value: 1},
			},
		},
		{
			collection: mongomodel.MatchItemDropsCollection,
			name:       "match_item_drops_match_player",
			keys: bson.D{
				{Key: "match_id", Value: 1},
				{Key: "player_id", Value: 1},
			},
		},
		{
			collection: mongomodel.OpenPowerRecordsCollection,
			name:       "open_power_records_player",
			keys:       bson.D{{Key: "player_id", Value: 1}},
		},
		{
			collection: mongomodel.OpenPowerRecordsCollection,
			name:       "open_power_records_reason_source_player",
			keys: bson.D{
				{Key: "reason", Value: 1},
				{Key: "source", Value: 1},
				{Key: "player_id", Value: 1},
			},
		},
		{
			collection: mongomodel.PlayerItemsCollection,
			name:       "player_items_player_item",
			keys: bson.D{
				{Key: "player_id", Value: 1},
				{Key: "item_id", Value: 1},
			},
			unique: true,
		},
		{
			collection: mongomodel.PlayerSitonesCollection,
			name:       "player_sitones_player_sitone",
			keys: bson.D{
				{Key: "player_id", Value: 1},
				{Key: "sitone_id", Value: 1},
			},
			unique: true,
		},
		{
			collection: mongomodel.ShopPurchasesCollection,
			name:       "shop_purchases_player_item",
			keys: bson.D{
				{Key: "player_id", Value: 1},
				{Key: "item_id", Value: 1},
			},
			unique: true,
		},
		{
			collection:         shopPurchaseLocksCollection,
			name:               "shop_purchase_locks_expires_at_ttl",
			keys:               bson.D{{Key: "expires_at", Value: 1}},
			expireAfterSeconds: int32Pointer(0),
		},
	}

	if len(models) != len(expected) {
		t.Fatalf("expected %d indexes, got %d", len(expected), len(models))
	}
	for _, want := range expected {
		got, ok := models[want.collection+"/"+want.name]
		if !ok {
			t.Fatalf("missing index %s/%s", want.collection, want.name)
		}
		assertIndexModel(t, got, want)
	}
}

type indexedModel struct {
	collection string
	keys       any
	options    options.IndexOptions
}

type expectedIndexModel struct {
	collection         string
	name               string
	keys               bson.D
	unique             bool
	partial            any
	expireAfterSeconds *int32
}

func flattenIndexModels(collections []collectionIndexModels) map[string]indexedModel {
	out := make(map[string]indexedModel)
	for _, collection := range collections {
		for _, model := range collection.models {
			opts := indexOptions(model.Options)
			out[collection.collection+"/"+*opts.Name] = indexedModel{
				collection: collection.collection,
				keys:       model.Keys,
				options:    opts,
			}
		}
	}
	return out
}

func assertIndexModel(t *testing.T, got indexedModel, want expectedIndexModel) {
	t.Helper()

	if got.collection != want.collection {
		t.Fatalf("unexpected index collection: got %q want %q", got.collection, want.collection)
	}
	if !reflect.DeepEqual(got.keys, want.keys) {
		t.Fatalf("unexpected index keys for %s/%s: got %#v want %#v", want.collection, want.name, got.keys, want.keys)
	}
	if got.options.Name == nil || *got.options.Name != want.name {
		t.Fatalf("unexpected index name for %s: got %#v want %q", want.collection, got.options.Name, want.name)
	}
	if gotUnique := got.options.Unique != nil && *got.options.Unique; gotUnique != want.unique {
		t.Fatalf("unexpected unique option for %s/%s: got %t want %t", want.collection, want.name, gotUnique, want.unique)
	}
	if !reflect.DeepEqual(got.options.PartialFilterExpression, want.partial) {
		t.Fatalf("unexpected partial filter for %s/%s: got %#v want %#v", want.collection, want.name, got.options.PartialFilterExpression, want.partial)
	}
	if !reflect.DeepEqual(got.options.ExpireAfterSeconds, want.expireAfterSeconds) {
		t.Fatalf("unexpected TTL for %s/%s: got %#v want %#v", want.collection, want.name, got.options.ExpireAfterSeconds, want.expireAfterSeconds)
	}
}

func indexOptions(builder *options.IndexOptionsBuilder) options.IndexOptions {
	var opts options.IndexOptions
	for _, apply := range builder.List() {
		if err := apply(&opts); err != nil {
			panic(err)
		}
	}
	return opts
}

func int32Pointer(value int32) *int32 {
	return &value
}
