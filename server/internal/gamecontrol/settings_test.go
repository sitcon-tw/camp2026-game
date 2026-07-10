package gamecontrol

import (
	"testing"
	"time"
)

func TestBattleOpeningLockedUsesClassTimeWindow(t *testing.T) {
	settings := DefaultSettings()
	settings.ClassTimeBattleLockEnabled = true
	settings.ClassTimeBattleLockSessions = []ClockTimeRange{
		{Start: "09:00", End: "17:00"},
	}

	if !settings.BattleOpeningLocked(clockAt(10, 30)) {
		t.Fatal("expected battle opening to be locked during class time")
	}
	if settings.BattleOpeningLocked(clockAt(17, 0)) {
		t.Fatal("expected battle opening to unlock at the class time end")
	}
}

func TestBattleOpeningLockedSupportsOvernightWindow(t *testing.T) {
	settings := DefaultSettings()
	settings.ClassTimeBattleLockEnabled = true
	settings.ClassTimeBattleLockSessions = []ClockTimeRange{
		{Start: "22:00", End: "06:00"},
	}

	if !settings.BattleOpeningLocked(clockAt(23, 30)) {
		t.Fatal("expected battle opening to be locked before midnight")
	}
	if !settings.BattleOpeningLocked(clockAt(5, 30)) {
		t.Fatal("expected battle opening to be locked after midnight")
	}
	if settings.BattleOpeningLocked(clockAt(12, 0)) {
		t.Fatal("expected battle opening to be unlocked outside overnight window")
	}
}

func TestBattleOpeningLockedSupportsMultipleDailyWindows(t *testing.T) {
	settings := DefaultSettings()
	settings.ClassTimeBattleLockEnabled = true
	settings.ClassTimeBattleLockSessions = []ClockTimeRange{
		{Start: "09:00", End: "10:30"},
		{Start: "13:00", End: "14:30"},
		{Start: "19:30", End: "21:00"},
	}

	for _, tt := range []struct {
		hour   int
		minute int
		locked bool
	}{
		{hour: 9, minute: 30, locked: true},
		{hour: 11, minute: 0, locked: false},
		{hour: 13, minute: 45, locked: true},
		{hour: 18, minute: 0, locked: false},
		{hour: 20, minute: 0, locked: true},
		{hour: 21, minute: 0, locked: false},
	} {
		if got := settings.BattleOpeningLocked(clockAt(tt.hour, tt.minute)); got != tt.locked {
			t.Fatalf("expected %02d:%02d locked=%v, got %v", tt.hour, tt.minute, tt.locked, got)
		}
	}
}

func TestBattleOpeningLockedUsesTaipeiDailyClock(t *testing.T) {
	settings := DefaultSettings()
	settings.ClassTimeBattleLockEnabled = true
	settings.ClassTimeBattleLockSessions = []ClockTimeRange{
		{Start: "00:00", End: "00:30"},
	}

	taipeiMidnight := time.Date(2026, 7, 9, 0, 10, 0, 0, classTimeLocation)
	if !settings.BattleOpeningLocked(taipeiMidnight.UTC()) {
		t.Fatal("expected UTC timestamp to be evaluated against UTC+8 daily midnight")
	}
}

func TestNormalizeMigratesLegacyClassTimeWindow(t *testing.T) {
	settings := DefaultSettings()
	settings.ClassTimeBattleLockStart = "08:30"
	settings.ClassTimeBattleLockEnd = "09:30"
	settings.ClassTimeBattleLockSessions = nil
	settings.Normalize()

	if len(settings.ClassTimeBattleLockSessions) != 1 {
		t.Fatalf("expected one migrated class time window, got %#v", settings.ClassTimeBattleLockSessions)
	}
	if got := settings.ClassTimeBattleLockSessions[0]; got.Start != "08:30" || got.End != "09:30" {
		t.Fatalf("unexpected migrated class time window: %#v", got)
	}
}

func TestBattleOpeningOverrideWinsOverSchedule(t *testing.T) {
	settings := DefaultSettings()
	settings.ClassTimeBattleLockEnabled = true
	settings.ClassTimeBattleLockStart = "09:00"
	settings.ClassTimeBattleLockEnd = "17:00"

	settings.BattleOpeningOverride = BattleOpeningOverrideForceOpen
	if settings.BattleOpeningLocked(clockAt(10, 30)) {
		t.Fatal("expected force-open override to allow opening")
	}

	settings.BattleOpeningOverride = BattleOpeningOverrideForceClosed
	if !settings.BattleOpeningLocked(clockAt(20, 30)) {
		t.Fatal("expected force-closed override to block opening")
	}
}

func TestMaintenanceBlocksOpeningAndWrites(t *testing.T) {
	settings := DefaultSettings()
	settings.BattleOpeningOverride = BattleOpeningOverrideForceOpen
	settings.MaintenanceMode = MaintenanceModeDraining

	if !settings.BattleOpeningLocked(clockAt(20, 30)) {
		t.Fatal("expected maintenance to block battle opening")
	}
	if !settings.MaintenanceBlocksNewMatches() || !settings.MaintenanceBlocksWrites() {
		t.Fatal("expected draining maintenance to block new matches and writes")
	}

	settings.MaintenanceMode = MaintenanceModeOff
	if settings.MaintenanceBlocksNewMatches() || settings.MaintenanceBlocksWrites() {
		t.Fatal("expected disabled maintenance to allow new matches and writes")
	}
}

func TestNormalizeMaintenanceMode(t *testing.T) {
	settings := DefaultSettings()
	settings.MaintenanceMode = "broken"
	settings.MaintenanceStartedAt = time.Now()
	settings.Normalize()

	if settings.MaintenanceMode != MaintenanceModeOff {
		t.Fatalf("expected invalid maintenance mode to reset to off, got %q", settings.MaintenanceMode)
	}
	if !settings.MaintenanceStartedAt.IsZero() {
		t.Fatal("expected disabled maintenance to clear started time")
	}
}

func TestQRCodeScanCooldownDuration(t *testing.T) {
	settings := DefaultSettings()
	settings.QRCodeScanCooldownEnabled = true
	settings.QRCodeScanCooldownMinutes = 7

	if got := settings.QRCodeScanCooldownDuration(); got != 7*time.Minute {
		t.Fatalf("expected 7 minute cooldown, got %s", got)
	}

	settings.QRCodeScanCooldownEnabled = false
	if got := settings.QRCodeScanCooldownDuration(); got != 0 {
		t.Fatalf("expected disabled cooldown to be zero, got %s", got)
	}
}

func TestNormalizeQRCodeScanCooldownMinutes(t *testing.T) {
	settings := DefaultSettings()
	settings.QRCodeScanCooldownMinutes = 0
	settings.Normalize()

	if settings.QRCodeScanCooldownMinutes != DefaultQRCodeScanCooldownMinutes {
		t.Fatalf("expected default QR code scan cooldown minutes, got %d", settings.QRCodeScanCooldownMinutes)
	}

	settings.QRCodeScanCooldownMinutes = MaxQRCodeScanCooldownMinutes + 1
	settings.Normalize()
	if settings.QRCodeScanCooldownMinutes != DefaultQRCodeScanCooldownMinutes {
		t.Fatalf("expected oversized QR code scan cooldown to reset, got %d", settings.QRCodeScanCooldownMinutes)
	}
}

func clockAt(hour int, minute int) time.Time {
	return time.Date(2026, 7, 8, hour, minute, 0, 0, classTimeLocation)
}
