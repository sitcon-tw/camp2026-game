package gifthistory

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/sitcon-tw/camp2026-game/internal/content"
	mongomodel "github.com/sitcon-tw/camp2026-game/internal/mongodb/model"
)

const rewardKindOpenPower = "open_power"

type Entry struct {
	RewardID          string    `json:"rewardId"`
	Kind              string    `json:"kind"`
	RefID             string    `json:"refId,omitempty"`
	Name              string    `json:"name"`
	Quantity          int       `json:"quantity,omitempty"`
	Amount            int       `json:"amount,omitempty"`
	StaffPlayerID     string    `json:"staffPlayerId"`
	StaffNickname     string    `json:"staffNickname,omitempty"`
	RecipientPlayerID string    `json:"recipientPlayerId"`
	RecipientNickname string    `json:"recipientNickname,omitempty"`
	CreatedAt         time.Time `json:"createdAt"`
}

type Response struct {
	Entries []Entry `json:"entries"`
}

type Filter struct {
	RecipientPlayerID string
	Limit             int64
}

func List(ctx context.Context, db *mongo.Database, store *content.Store, filter Filter) ([]Entry, error) {
	if filter.Limit <= 0 {
		filter.Limit = 100
	}

	query := bson.M{}
	if filter.RecipientPlayerID != "" {
		query["recipient_player_id"] = filter.RecipientPlayerID
	}

	cursor, err := db.Collection(mongomodel.StaffRewardsCollection).Find(
		ctx,
		query,
		options.Find().
			SetSort(bson.D{{Key: "created_at", Value: -1}, {Key: "_id", Value: -1}}).
			SetLimit(filter.Limit),
	)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = cursor.Close(ctx)
	}()

	var records []mongomodel.StaffReward
	if err := cursor.All(ctx, &records); err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return []Entry{}, nil
	}

	playerIDs := make(map[string]struct{}, len(records)*2)
	for _, record := range records {
		playerIDs[record.StaffPlayerID] = struct{}{}
		playerIDs[record.RecipientPlayerID] = struct{}{}
	}
	players, err := loadPlayersByID(ctx, db, playerIDs)
	if err != nil {
		return nil, err
	}

	entries := make([]Entry, 0, len(records))
	for _, record := range records {
		name, err := rewardName(store, record.Kind, record.RefID)
		if err != nil {
			return nil, err
		}
		entry := Entry{
			RewardID:          record.ID,
			Kind:              record.Kind,
			RefID:             record.RefID,
			Name:              name,
			StaffPlayerID:     record.StaffPlayerID,
			RecipientPlayerID: record.RecipientPlayerID,
			CreatedAt:         record.CreatedAt,
		}
		if record.Kind == rewardKindOpenPower {
			entry.Amount = record.Quantity
		} else {
			entry.Quantity = record.Quantity
		}
		if staff, ok := players[record.StaffPlayerID]; ok {
			entry.StaffNickname = staff.Nickname
		}
		if recipient, ok := players[record.RecipientPlayerID]; ok {
			entry.RecipientNickname = recipient.Nickname
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

func loadPlayersByID(ctx context.Context, db *mongo.Database, ids map[string]struct{}) (map[string]mongomodel.Player, error) {
	list := make([]string, 0, len(ids))
	for id := range ids {
		if id != "" {
			list = append(list, id)
		}
	}
	cursor, err := db.Collection(mongomodel.PlayersCollection).Find(
		ctx,
		bson.M{"_id": bson.M{"$in": list}},
		options.Find().SetProjection(bson.D{
			{Key: "auth_token", Value: 0},
			{Key: "qrcode_token", Value: 0},
			{Key: "default_sitone_ids", Value: 0},
		}),
	)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = cursor.Close(ctx)
	}()

	var players []mongomodel.Player
	if err := cursor.All(ctx, &players); err != nil {
		return nil, err
	}
	out := make(map[string]mongomodel.Player, len(players))
	for _, player := range players {
		out[player.ID] = player
	}
	return out, nil
}

func rewardName(store *content.Store, kind, refID string) (string, error) {
	switch kind {
	case "item":
		item, ok := store.GetItem(refID)
		if !ok {
			return "", fmt.Errorf("gift history item %q not found", refID)
		}
		return item.Name, nil
	case "sitone":
		sitone, ok := store.GetSitone(refID)
		if !ok {
			return "", fmt.Errorf("gift history sitone %q not found", refID)
		}
		return sitone.Name, nil
	case rewardKindOpenPower:
		return "開源力", nil
	default:
		return "", fmt.Errorf("unsupported gift history kind %q", kind)
	}
}
