package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/sitcon-tw/camp2026-game/internal/config"
	"github.com/sitcon-tw/camp2026-game/internal/mongodb"
	mongomodel "github.com/sitcon-tw/camp2026-game/internal/mongodb/model"
)

func main() {
	if err := run(context.Background()); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "seed failed: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	client, err := mongodb.NewClient(ctx, cfg.MongoURI)
	if err != nil {
		return err
	}
	defer func() {
		_ = client.Disconnect(context.Background())
	}()

	db := client.Database(cfg.MongoDatabase)
	now := time.Now().UTC()

	if err := upsertMany(ctx, db.Collection(mongomodel.TeamsCollection), []any{
		mongomodel.Team{ID: "team-a", Name: "松鼠小隊"},
		mongomodel.Team{ID: "team-b", Name: "山羌小隊"},
	}); err != nil {
		return err
	}

	if err := upsertPlayers(ctx, db.Collection(mongomodel.PlayersCollection), []mongomodel.Player{
		mongomodel.Player{
			ID:          "player-a",
			AuthToken:   "auth_token_123456",
			QRCodeToken: "qr-token-player-a",
			Nickname:    "阿洛",
			TeamID:      "team-a",
		},
		mongomodel.Player{
			ID:          "player-b",
			AuthToken:   "auth_token_abcdef",
			QRCodeToken: "qr-token-player-b",
			Nickname:    "小白",
			TeamID:      "team-b",
		},
		mongomodel.Player{
			ID:        "staff-a",
			AuthToken: "staff_token_2026",
			Nickname:  "工作人員",
			Role:      "staff",
		},
	}); err != nil {
		return err
	}

	if err := upsertMany(ctx, db.Collection(mongomodel.OpenPowerRecordsCollection), []any{
		mongomodel.OpenPowerRecord{
			ID:        "seed-player-a-open-power",
			PlayerID:  "player-a",
			Amount:    800,
			Reason:    "dev_seed",
			Source:    "seed",
			CreatedAt: now,
		},
		mongomodel.OpenPowerRecord{
			ID:        "seed-player-b-open-power",
			PlayerID:  "player-b",
			Amount:    620,
			Reason:    "dev_seed",
			Source:    "seed",
			CreatedAt: now,
		},
	}); err != nil {
		return err
	}

	if err := upsertMany(ctx, db.Collection(mongomodel.PlayerSitonesCollection), []any{
		mongomodel.PlayerSitone{
			ID:       "seed-player-a-stone_engineering_base",
			PlayerID: "player-a",
			SitoneID: "stone_engineering_base",
			Quantity: 2,
		},
		mongomodel.PlayerSitone{
			ID:       "seed-player-a-stone_explorer_base",
			PlayerID: "player-a",
			SitoneID: "stone_explorer_base",
			Quantity: 1,
		},
		mongomodel.PlayerSitone{
			ID:       "seed-player-b-stone_resonance_base",
			PlayerID: "player-b",
			SitoneID: "stone_resonance_base",
			Quantity: 2,
		},
	}); err != nil {
		return err
	}

	if err := upsertMany(ctx, db.Collection(mongomodel.PlayerItemsCollection), []any{
		mongomodel.PlayerItem{
			ID:       "seed-player-a-item-adventure-backpack",
			PlayerID: "player-a",
			ItemID:   "item_adventure_backpack",
			Quantity: 6,
		},
		mongomodel.PlayerItem{
			ID:       "seed-player-b-item-adventure-backpack",
			PlayerID: "player-b",
			ItemID:   "item_adventure_backpack",
			Quantity: 3,
		},
	}); err != nil {
		return err
	}

	fmt.Println("seed complete")
	fmt.Println("player-a token: auth_token_123456")
	fmt.Println("player-b token: auth_token_abcdef")
	fmt.Println("staff token: staff_token_2026")
	return nil
}

func upsertPlayers(ctx context.Context, collection *mongo.Collection, players []mongomodel.Player) error {
	for _, player := range players {
		set := bson.M{
			"auth_token": player.AuthToken,
			"nickname":   player.Nickname,
		}
		unset := bson.M{}
		if player.QRCodeToken == "" {
			unset["qrcode_token"] = ""
		} else {
			set["qrcode_token"] = player.QRCodeToken
		}
		if player.TeamID == "" {
			unset["team_id"] = ""
		} else {
			set["team_id"] = player.TeamID
		}
		if player.Role == "" {
			unset["role"] = ""
		} else {
			set["role"] = player.Role
		}

		update := bson.M{"$set": set}
		if len(unset) > 0 {
			update["$unset"] = unset
		}

		_, err := collection.UpdateOne(
			ctx,
			bson.M{"_id": player.ID},
			update,
			options.UpdateOne().SetUpsert(true),
		)
		if err != nil {
			return fmt.Errorf("upsert %s/%s: %w", collection.Name(), player.ID, err)
		}
	}
	return nil
}

func upsertMany(ctx context.Context, collection *mongo.Collection, documents []any) error {
	for _, document := range documents {
		id, err := documentID(document)
		if err != nil {
			return err
		}
		_, err = collection.UpdateOne(
			ctx,
			bson.M{"_id": id},
			bson.M{"$set": document},
			options.UpdateOne().SetUpsert(true),
		)
		if err != nil {
			return fmt.Errorf("upsert %s/%s: %w", collection.Name(), id, err)
		}
	}
	return nil
}

func documentID(document any) (string, error) {
	data, err := bson.Marshal(document)
	if err != nil {
		return "", err
	}

	var value struct {
		ID string `bson:"_id"`
	}
	if err := bson.Unmarshal(data, &value); err != nil {
		return "", err
	}
	if value.ID == "" {
		return "", fmt.Errorf("document is missing _id")
	}
	return value.ID, nil
}
