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
	expiresAt := now.Add(3 * time.Minute)
	update := ReserveUpdate("qr_scan_cooldown:player-a", "player-a", "owner-a", now, expiresAt)

	set, ok := update["$set"].(bson.M)
	if !ok {
		t.Fatalf("expected $set update, got %#v", update)
	}
	gotExpiresAt, ok := set["expires_at"].(time.Time)
	if !ok || !gotExpiresAt.Equal(expiresAt) {
		t.Fatalf("expected expiry three minutes later, got %#v", set["expires_at"])
	}
	setOnInsert, ok := update["$setOnInsert"].(bson.M)
	if !ok || setOnInsert["player_id"] != "player-a" {
		t.Fatalf("expected inserted player id, got %#v", update)
	}
}

func TestReserveInsertSetsExpiry(t *testing.T) {
	now := time.Date(2026, 7, 8, 12, 0, 0, 0, time.UTC)
	expiresAt := now.Add(5 * time.Minute)
	insert := ReserveInsert("qr_scan_cooldown:player-a", "player-a", "owner-a", now, expiresAt)

	if insert["_id"] != "qr_scan_cooldown:player-a" || insert["player_id"] != "player-a" {
		t.Fatalf("expected cooldown identity, got %#v", insert)
	}
	gotExpiresAt, ok := insert["expires_at"].(time.Time)
	if !ok || !gotExpiresAt.Equal(expiresAt) {
		t.Fatalf("expected inserted expiry, got %#v", insert["expires_at"])
	}
}

func TestCooldownID(t *testing.T) {
	if got := CooldownID("player-a"); got != "qr_scan_cooldown:player-a" {
		t.Fatalf("unexpected cooldown id %q", got)
	}
}
