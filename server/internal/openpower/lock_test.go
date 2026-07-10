package openpower

import (
	"errors"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

func TestPlayerLockDocumentsUseSharedOpenPowerIdentity(t *testing.T) {
	now := time.Date(2026, time.July, 10, 12, 0, 0, 0, time.UTC)
	lockID := playerLockID("player-a")
	if lockID != "open_power:player-a" {
		t.Fatalf("unexpected lock id %q", lockID)
	}

	filter := playerLockFilter(lockID, "owner-a", now)
	orConditions, ok := filter["$or"].(bson.A)
	if !ok || len(orConditions) != 2 {
		t.Fatalf("unexpected lock filter %#v", filter)
	}

	update := playerLockUpdate(lockID, "player-a", "owner-a", now)
	set, ok := update["$set"].(bson.M)
	if !ok || set["owner_id"] != "owner-a" {
		t.Fatalf("unexpected lock update %#v", update)
	}
	expiresAt, ok := set["expires_at"].(time.Time)
	if !ok || !expiresAt.Equal(now.Add(playerLockTTL)) {
		t.Fatalf("unexpected expiry %#v", set["expires_at"])
	}

	insert := playerLockInsert(lockID, "player-a", "owner-a", now)
	if insert["player_id"] != "player-a" || insert["owner_id"] != "owner-a" {
		t.Fatalf("unexpected lock insert %#v", insert)
	}
}

func TestPlayerLockBusyRecognizesMongoDuplicateErrors(t *testing.T) {
	errorsToRetry := []error{
		mongo.ErrNoDocuments,
		mongo.CommandError{Code: 11000, Message: "duplicate key"},
		mongo.WriteException{WriteErrors: mongo.WriteErrors{{Code: 11000, Message: "duplicate key"}}},
		errors.New(`E11000 duplicate key error collection: camp2026.open_power_locks index: _id_`),
	}
	for _, err := range errorsToRetry {
		if !playerLockBusy(err) {
			t.Fatalf("expected retryable lock error, got %v", err)
		}
	}
	if playerLockBusy(errors.New("network timeout")) {
		t.Fatal("unrelated errors must not be treated as a busy lock")
	}
}
