package qrcooldown

import (
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestReserveFilterOnlyMatchesExpiredCooldown(t *testing.T) {
	now := time.Date(2026, 7, 8, 12, 0, 0, 0, time.UTC)
	filter := ReserveFilter("qr_scan_cooldown:player-a", "owner-a", now)

	if filter["_id"] != "qr_scan_cooldown:player-a" {
		t.Fatalf("expected cooldown id filter, got %#v", filter)
	}
	expiresAt, ok := filter["expires_at"].(bson.M)
	if !ok || !expiresAt["$lte"].(time.Time).Equal(now) {
		t.Fatalf("expected expires_at <= now filter, got %#v", filter["expires_at"])
	}
	if _, ok := filter["owner_id"]; ok {
		t.Fatalf("expected owner not to bypass active cooldown, got %#v", filter)
	}
}

func TestReserveUpdateSetsExpiry(t *testing.T) {
	now := time.Date(2026, 7, 8, 12, 0, 0, 0, time.UTC)
	update := ReserveUpdate("qr_scan_cooldown:player-a", "player-a", "owner-a", now, 3*time.Minute)

	set, ok := update["$set"].(bson.M)
	if !ok {
		t.Fatalf("expected $set update, got %#v", update)
	}
	expiresAt, ok := set["expires_at"].(time.Time)
	if !ok || !expiresAt.Equal(now.Add(3*time.Minute)) {
		t.Fatalf("expected expiry three minutes later, got %#v", set["expires_at"])
	}
	setOnInsert, ok := update["$setOnInsert"].(bson.M)
	if !ok || setOnInsert["player_id"] != "player-a" {
		t.Fatalf("expected inserted player id, got %#v", update)
	}
}

func TestCooldownID(t *testing.T) {
	if got := CooldownID("player-a"); got != "qr_scan_cooldown:player-a" {
		t.Fatalf("unexpected cooldown id %q", got)
	}
}
