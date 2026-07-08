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

	BattleOpeningOverrideSchedule    = "schedule"
	BattleOpeningOverrideForceOpen   = "force_open"
	BattleOpeningOverrideForceClosed = "force_closed"

	defaultClassTimeBattleLockStart = "09:00"
	defaultClassTimeBattleLockEnd   = "17:00"
)

var classTimeLocation = time.FixedZone("Asia/Taipei", 8*60*60)

type Settings struct {
	ID                         string    `bson:"_id"`
	ComputerBattlesEnabled     bool      `bson:"computer_battles_enabled"`
	SameTeamBattlesDisabled    bool      `bson:"same_team_battles_disabled"`
	ComputerEasyAccuracy       int       `bson:"computer_easy_accuracy"`
	ComputerNormalAccuracy     int       `bson:"computer_normal_accuracy"`
	ComputerHardAccuracy       int       `bson:"computer_hard_accuracy"`
	ClassTimeBattleLockEnabled bool      `bson:"class_time_battle_lock_enabled"`
	ClassTimeBattleLockStart   string    `bson:"class_time_battle_lock_start"`
	ClassTimeBattleLockEnd     string    `bson:"class_time_battle_lock_end"`
	BattleOpeningOverride      string    `bson:"battle_opening_override"`
	UpdatedAt                  time.Time `bson:"updated_at,omitempty"`
}

func DefaultSettings() Settings {
	return Settings{
		ID:                       SettingsID,
		ComputerEasyAccuracy:     35,
		ComputerNormalAccuracy:   55,
		ComputerHardAccuracy:     75,
		ClassTimeBattleLockStart: defaultClassTimeBattleLockStart,
		ClassTimeBattleLockEnd:   defaultClassTimeBattleLockEnd,
		BattleOpeningOverride:    BattleOpeningOverrideSchedule,
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
	if !ValidClockTime(settings.ClassTimeBattleLockStart) {
		settings.ClassTimeBattleLockStart = defaultClassTimeBattleLockStart
	}
	if !ValidClockTime(settings.ClassTimeBattleLockEnd) {
		settings.ClassTimeBattleLockEnd = defaultClassTimeBattleLockEnd
	}
	if !ValidBattleOpeningOverride(settings.BattleOpeningOverride) {
		settings.BattleOpeningOverride = BattleOpeningOverrideSchedule
	}
}

func (settings Settings) SameTeamBattlesEnabled() bool {
	return !settings.SameTeamBattlesDisabled
}

func (settings Settings) BattleOpeningLocked(now time.Time) bool {
	settings.Normalize()
	switch settings.BattleOpeningOverride {
	case BattleOpeningOverrideForceOpen:
		return false
	case BattleOpeningOverrideForceClosed:
		return true
	}
	if !settings.ClassTimeBattleLockEnabled {
		return false
	}
	return clockTimeInRange(settings.ClassTimeBattleLockStart, settings.ClassTimeBattleLockEnd, now)
}

func ValidBattleOpeningOverride(value string) bool {
	switch value {
	case BattleOpeningOverrideSchedule, BattleOpeningOverrideForceOpen, BattleOpeningOverrideForceClosed:
		return true
	default:
		return false
	}
}

func ValidClockTime(value string) bool {
	_, ok := parseClockMinutes(value)
	return ok
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

func clockTimeInRange(startValue string, endValue string, now time.Time) bool {
	start, startOK := parseClockMinutes(startValue)
	end, endOK := parseClockMinutes(endValue)
	if !startOK || !endOK || start == end {
		return false
	}
	localNow := now.In(classTimeLocation)
	current := localNow.Hour()*60 + localNow.Minute()
	if start < end {
		return current >= start && current < end
	}
	return current >= start || current < end
}

func parseClockMinutes(value string) (int, bool) {
	if len(value) != 5 || value[2] != ':' {
		return 0, false
	}
	hour := int(value[0]-'0')*10 + int(value[1]-'0')
	minute := int(value[3]-'0')*10 + int(value[4]-'0')
	if value[0] < '0' || value[0] > '9' ||
		value[1] < '0' || value[1] > '9' ||
		value[3] < '0' || value[3] > '9' ||
		value[4] < '0' || value[4] > '9' ||
		hour > 23 || minute > 59 {
		return 0, false
	}
	return hour*60 + minute, true
}
