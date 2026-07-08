package gamecontrol

import (
	"testing"
	"time"
)

func TestBattleOpeningLockedUsesClassTimeWindow(t *testing.T) {
	settings := DefaultSettings()
	settings.ClassTimeBattleLockEnabled = true
	settings.ClassTimeBattleLockStart = "09:00"
	settings.ClassTimeBattleLockEnd = "17:00"

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
	settings.ClassTimeBattleLockStart = "22:00"
	settings.ClassTimeBattleLockEnd = "06:00"

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

func clockAt(hour int, minute int) time.Time {
	return time.Date(2026, 7, 8, hour, minute, 0, 0, classTimeLocation)
}
