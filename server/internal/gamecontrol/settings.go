package gamecontrol

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const (
	SettingsCollection = "game_settings"
	SettingsID         = "global"
)

type Settings struct {
	ID                      string    `bson:"_id"`
	ComputerBattlesEnabled  bool      `bson:"computer_battles_enabled"`
	SameTeamBattlesDisabled bool      `bson:"same_team_battles_disabled"`
	ComputerEasyAccuracy    int       `bson:"computer_easy_accuracy"`
	ComputerNormalAccuracy  int       `bson:"computer_normal_accuracy"`
	ComputerHardAccuracy    int       `bson:"computer_hard_accuracy"`
	UpdatedAt               time.Time `bson:"updated_at,omitempty"`
}

func DefaultSettings() Settings {
	return Settings{
		ID:                     SettingsID,
		ComputerEasyAccuracy:   35,
		ComputerNormalAccuracy: 55,
		ComputerHardAccuracy:   75,
	}
}

func ReadSettings(ctx context.Context, db *mongo.Database) (Settings, error) {
	settings := DefaultSettings()
	if db == nil {
		return settings, nil
	}

	err := db.Collection(SettingsCollection).
		FindOne(ctx, bson.M{"_id": SettingsID}).
		Decode(&settings)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return DefaultSettings(), nil
	}
	if err != nil {
		return Settings{}, err
	}
	settings.Normalize()
	return settings, nil
}

func SaveSettings(ctx context.Context, db *mongo.Database, settings Settings) (Settings, error) {
	settings.ID = SettingsID
	settings.UpdatedAt = time.Now().UTC()
	settings.Normalize()

	_, err := db.Collection(SettingsCollection).UpdateOne(
		ctx,
		bson.M{"_id": SettingsID},
		bson.M{"$set": settings},
		options.UpdateOne().SetUpsert(true),
	)
	if err != nil {
		return Settings{}, err
	}
	return settings, nil
}

func (settings *Settings) Normalize() {
	if settings.ID == "" {
		settings.ID = SettingsID
	}
	settings.ComputerEasyAccuracy = clampPercent(settings.ComputerEasyAccuracy)
	settings.ComputerNormalAccuracy = clampPercent(settings.ComputerNormalAccuracy)
	settings.ComputerHardAccuracy = clampPercent(settings.ComputerHardAccuracy)
}

func (settings Settings) SameTeamBattlesEnabled() bool {
	return !settings.SameTeamBattlesDisabled
}

func clampPercent(value int) int {
	if value < 0 {
		return 0
	}
	if value > 100 {
		return 100
	}
	return value
}
