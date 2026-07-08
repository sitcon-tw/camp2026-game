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

	MaintenanceModeOff      = "off"
	MaintenanceModeDraining = "draining"
	MaintenanceModeActive   = "active"

	defaultClassTimeBattleLockStart = "09:00"
	defaultClassTimeBattleLockEnd   = "17:00"

	maxClassTimeBattleLockSessions = 12
)

var classTimeLocation = time.FixedZone("Asia/Taipei", 8*60*60)

type Settings struct {
	ID                          string           `bson:"_id"`
	ComputerBattlesEnabled      bool             `bson:"computer_battles_enabled"`
	SameTeamBattlesDisabled     bool             `bson:"same_team_battles_disabled"`
	ComputerEasyAccuracy        int              `bson:"computer_easy_accuracy"`
	ComputerNormalAccuracy      int              `bson:"computer_normal_accuracy"`
	ComputerHardAccuracy        int              `bson:"computer_hard_accuracy"`
	ClassTimeBattleLockEnabled  bool             `bson:"class_time_battle_lock_enabled"`
	ClassTimeBattleLockStart    string           `bson:"class_time_battle_lock_start"`
	ClassTimeBattleLockEnd      string           `bson:"class_time_battle_lock_end"`
	ClassTimeBattleLockSessions []ClockTimeRange `bson:"class_time_battle_lock_sessions,omitempty"`
	BattleOpeningOverride       string           `bson:"battle_opening_override"`
	MaintenanceMode             string           `bson:"maintenance_mode"`
	MaintenanceMessage          string           `bson:"maintenance_message,omitempty"`
	MaintenanceStartedAt        time.Time        `bson:"maintenance_started_at,omitempty"`
	UpdatedAt                   time.Time        `bson:"updated_at,omitempty"`
}

type ClockTimeRange struct {
	Start string `bson:"start"`
	End   string `bson:"end"`
}

func DefaultSettings() Settings {
	return Settings{
		ID:                          SettingsID,
		ComputerEasyAccuracy:        35,
		ComputerNormalAccuracy:      55,
		ComputerHardAccuracy:        75,
		ClassTimeBattleLockStart:    defaultClassTimeBattleLockStart,
		ClassTimeBattleLockEnd:      defaultClassTimeBattleLockEnd,
		ClassTimeBattleLockSessions: defaultClassTimeBattleLockSessions(),
		BattleOpeningOverride:       BattleOpeningOverrideSchedule,
		MaintenanceMode:             MaintenanceModeOff,
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
	settings.ClassTimeBattleLockSessions = normalizeClockTimeRanges(
		settings.ClassTimeBattleLockSessions,
		ClockTimeRange{
			Start: settings.ClassTimeBattleLockStart,
			End:   settings.ClassTimeBattleLockEnd,
		},
	)
	if len(settings.ClassTimeBattleLockSessions) > 0 {
		settings.ClassTimeBattleLockStart = settings.ClassTimeBattleLockSessions[0].Start
		settings.ClassTimeBattleLockEnd = settings.ClassTimeBattleLockSessions[0].End
	}
	if !ValidBattleOpeningOverride(settings.BattleOpeningOverride) {
		settings.BattleOpeningOverride = BattleOpeningOverrideSchedule
	}
	if !ValidMaintenanceMode(settings.MaintenanceMode) {
		settings.MaintenanceMode = MaintenanceModeOff
	}
	if !settings.MaintenanceActive() {
		settings.MaintenanceStartedAt = time.Time{}
	}
}

func (settings Settings) SameTeamBattlesEnabled() bool {
	return !settings.SameTeamBattlesDisabled
}

func (settings Settings) BattleOpeningLocked(now time.Time) bool {
	settings.Normalize()
	if settings.MaintenanceBlocksNewMatches() {
		return true
	}
	switch settings.BattleOpeningOverride {
	case BattleOpeningOverrideForceOpen:
		return false
	case BattleOpeningOverrideForceClosed:
		return true
	}
	if !settings.ClassTimeBattleLockEnabled {
		return false
	}
	for _, session := range settings.ClassTimeBattleLockSessions {
		if clockTimeInRange(session.Start, session.End, now) {
			return true
		}
	}
	return false
}

func (settings Settings) MaintenanceActive() bool {
	return settings.MaintenanceMode == MaintenanceModeDraining ||
		settings.MaintenanceMode == MaintenanceModeActive
}

func (settings Settings) MaintenanceBlocksNewMatches() bool {
	return settings.MaintenanceActive()
}

func (settings Settings) MaintenanceBlocksWrites() bool {
	return settings.MaintenanceActive()
}

func ValidBattleOpeningOverride(value string) bool {
	switch value {
	case BattleOpeningOverrideSchedule, BattleOpeningOverrideForceOpen, BattleOpeningOverrideForceClosed:
		return true
	default:
		return false
	}
}

func ValidMaintenanceMode(value string) bool {
	switch value {
	case MaintenanceModeOff, MaintenanceModeDraining, MaintenanceModeActive:
		return true
	default:
		return false
	}
}

func ValidClockTime(value string) bool {
	_, ok := parseClockMinutes(value)
	return ok
}

func ValidClockTimeRange(session ClockTimeRange) bool {
	start, startOK := parseClockMinutes(session.Start)
	end, endOK := parseClockMinutes(session.End)
	return startOK && endOK && start != end
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

func defaultClassTimeBattleLockSessions() []ClockTimeRange {
	return []ClockTimeRange{
		{
			Start: defaultClassTimeBattleLockStart,
			End:   defaultClassTimeBattleLockEnd,
		},
	}
}

func normalizeClockTimeRanges(sessions []ClockTimeRange, fallback ClockTimeRange) []ClockTimeRange {
	out := make([]ClockTimeRange, 0, len(sessions))
	for _, session := range sessions {
		if !ValidClockTimeRange(session) {
			continue
		}
		out = append(out, session)
		if len(out) >= maxClassTimeBattleLockSessions {
			return out
		}
	}
	if len(out) > 0 {
		return out
	}
	if ValidClockTimeRange(fallback) {
		return []ClockTimeRange{fallback}
	}
	return defaultClassTimeBattleLockSessions()
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
