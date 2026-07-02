package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/sitcon-tw/camp2026-game/tgbot/internal/domain"
)

type MongoStore struct {
	db *mongo.Database
}

func NewMongoStore(db *mongo.Database) *MongoStore {
	return &MongoStore{db: db}
}

func (s *MongoStore) EnsureIndexes(ctx context.Context) error {
	loginRequests := s.db.Collection(domain.LoginRequestsCollection)
	if _, err := loginRequests.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "expires_at", Value: 1}},
		Options: options.Index().SetExpireAfterSeconds(0),
	}); err != nil {
		return fmt.Errorf("create %s expires_at ttl index: %w", loginRequests.Name(), err)
	}

	players := s.db.Collection(domain.PlayersCollection)
	if _, err := players.Indexes().CreateMany(ctx, []mongo.IndexModel{
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
	}); err != nil {
		return fmt.Errorf("create %s token indexes: %w", players.Name(), err)
	}

	playerSitones := s.db.Collection(domain.PlayerSitonesCollection)
	if _, err := playerSitones.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{
			{Key: "player_id", Value: 1},
			{Key: "sitone_id", Value: 1},
		},
		Options: options.Index().SetName("player_sitones_player_sitone").SetUnique(true),
	}); err != nil {
		return fmt.Errorf("create %s player sitone index: %w", playerSitones.Name(), err)
	}
	return nil
}

func (s *MongoStore) CreateLoginRequest(ctx context.Context, request domain.LoginRequest) error {
	_, err := s.db.Collection(domain.LoginRequestsCollection).InsertOne(ctx, request)
	if err != nil {
		return fmt.Errorf("insert %s/%s: %w", domain.LoginRequestsCollection, request.ID, err)
	}
	return nil
}

func (s *MongoStore) RedeemLoginRequest(ctx context.Context, nonce string, telegramUserID int64, now time.Time) (domain.LoginRequest, error) {
	var request domain.LoginRequest
	err := s.db.Collection(domain.LoginRequestsCollection).FindOneAndDelete(ctx, bson.M{
		"_id":              nonce,
		"telegram_user_id": telegramUserID,
		"expires_at":       bson.M{"$gt": now},
	}).Decode(&request)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return domain.LoginRequest{}, domain.ErrLoginRequestNotFound
	}
	if err != nil {
		return domain.LoginRequest{}, fmt.Errorf("redeem %s/%s: %w", domain.LoginRequestsCollection, nonce, err)
	}
	return request, nil
}

func (s *MongoStore) GetOrCreatePlayer(ctx context.Context, input domain.CreatePlayerInput) (domain.Player, bool, error) {
	collection := s.db.Collection(domain.PlayersCollection)

	var existing domain.Player
	err := collection.FindOne(ctx, bson.M{"telegram_user_id": input.TelegramUserID}).Decode(&existing)
	if err == nil {
		set := bson.M{
			"telegram_username": input.TelegramUsername,
			"updated_at":        input.Now,
		}
		if existing.QRCodeToken == "" && input.QRCodeToken != "" {
			set["qrcode_token"] = input.QRCodeToken
			existing.QRCodeToken = input.QRCodeToken
		}
		if _, updateErr := collection.UpdateByID(ctx, existing.ID, bson.M{"$set": set}); updateErr != nil {
			return domain.Player{}, false, fmt.Errorf("update existing telegram player %s: %w", existing.ID, updateErr)
		}
		existing.TelegramUsername = input.TelegramUsername
		existing.UpdatedAt = input.Now
		return existing, false, nil
	}
	if err != nil && !errors.Is(err, mongo.ErrNoDocuments) {
		return domain.Player{}, false, fmt.Errorf("find telegram player %d: %w", input.TelegramUserID, err)
	}

	player := domain.Player{
		ID:               input.PlayerID,
		AuthToken:        input.AuthToken,
		QRCodeToken:      input.QRCodeToken,
		Nickname:         input.Nickname,
		TeamID:           input.TeamID,
		DefaultSitoneIDs: append([]string(nil), input.InitialSitoneIDs...),
		TelegramUserID:   input.TelegramUserID,
		TelegramUsername: input.TelegramUsername,
		TelegramChatID:   input.TelegramChatID,
		CreatedAt:        input.Now,
		UpdatedAt:        input.Now,
	}
	_, err = collection.InsertOne(ctx, player)
	if mongo.IsDuplicateKeyError(err) {
		var raced domain.Player
		findErr := collection.FindOne(ctx, bson.M{"telegram_user_id": input.TelegramUserID}).Decode(&raced)
		if findErr != nil {
			return domain.Player{}, false, fmt.Errorf("find duplicate telegram player %d: %w", input.TelegramUserID, findErr)
		}
		return raced, false, nil
	}
	if err != nil {
		return domain.Player{}, false, fmt.Errorf("insert telegram player %s: %w", input.PlayerID, err)
	}
	if err := s.grantInitialSitones(ctx, player.ID, input.InitialSitoneIDs); err != nil {
		return domain.Player{}, false, err
	}
	return player, true, nil
}

func (s *MongoStore) grantInitialSitones(ctx context.Context, playerID string, sitoneIDs []string) error {
	collection := s.db.Collection(domain.PlayerSitonesCollection)
	for _, sitoneID := range sitoneIDs {
		_, err := collection.UpdateOne(
			ctx,
			bson.M{
				"player_id": playerID,
				"sitone_id": sitoneID,
			},
			bson.M{
				"$setOnInsert": bson.M{
					"_id":       newID("player_sitone"),
					"player_id": playerID,
					"sitone_id": sitoneID,
				},
				"$inc": bson.M{"quantity": 1},
			},
			options.UpdateOne().SetUpsert(true),
		)
		if err != nil {
			return fmt.Errorf("grant initial sitone %s to %s: %w", sitoneID, playerID, err)
		}
	}
	return nil
}

func newID(prefix string) string {
	return prefix + "_" + bson.NewObjectID().Hex()
}
