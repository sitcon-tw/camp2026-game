package openpower

import (
	"context"
	"errors"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	mongomodel "github.com/sitcon-tw/camp2026-game/internal/mongodb/model"
)

const (
	playerLockTTL            = time.Minute
	playerLockRetryDelay     = 25 * time.Millisecond
	playerLockReleaseTimeout = 2 * time.Second
)

func AcquirePlayerLock(ctx context.Context, db *mongo.Database, playerID string) (func(), error) {
	lockID := playerLockID(playerID)
	ownerID := "open_power_lock_" + bson.NewObjectID().Hex()
	collection := db.Collection(mongomodel.OpenPowerLocksCollection)

	for {
		now := time.Now()
		result, err := collection.UpdateOne(
			ctx,
			playerLockFilter(lockID, ownerID, now),
			playerLockUpdate(lockID, playerID, ownerID, now),
		)
		if err != nil {
			return nil, err
		}
		if result.MatchedCount > 0 {
			return playerLockRelease(db, lockID, ownerID), nil
		}

		_, err = collection.InsertOne(ctx, playerLockInsert(lockID, playerID, ownerID, now))
		if err == nil {
			return playerLockRelease(db, lockID, ownerID), nil
		}
		if !playerLockBusy(err) {
			return nil, err
		}
		if err := sleepContext(ctx, playerLockRetryDelay); err != nil {
			return nil, err
		}
	}
}

func playerLockRelease(db *mongo.Database, lockID string, ownerID string) func() {
	return func() {
		ctx, cancel := context.WithTimeout(context.Background(), playerLockReleaseTimeout)
		defer cancel()
		_, _ = db.Collection(mongomodel.OpenPowerLocksCollection).DeleteOne(ctx, bson.M{
			"_id": lockID, "owner_id": ownerID,
		})
	}
}

func playerLockFilter(lockID string, ownerID string, now time.Time) bson.M {
	return bson.M{
		"_id": lockID,
		"$or": bson.A{
			bson.M{"expires_at": bson.M{"$lte": now}},
			bson.M{"owner_id": ownerID},
		},
	}
}

func playerLockUpdate(lockID string, playerID string, ownerID string, now time.Time) bson.M {
	return bson.M{
		"$set": bson.M{
			"owner_id": ownerID, "expires_at": now.Add(playerLockTTL), "updated_at": now,
		},
		"$setOnInsert": bson.M{
			"_id": lockID, "player_id": playerID, "created_at": now,
		},
	}
}

func playerLockInsert(lockID string, playerID string, ownerID string, now time.Time) bson.M {
	return bson.M{
		"_id": lockID, "player_id": playerID, "owner_id": ownerID,
		"expires_at": now.Add(playerLockTTL), "created_at": now, "updated_at": now,
	}
}

func playerLockID(playerID string) string {
	return "open_power:" + playerID
}

func playerLockBusy(err error) bool {
	if errors.Is(err, mongo.ErrNoDocuments) || mongo.IsDuplicateKeyError(err) {
		return true
	}
	if err == nil {
		return false
	}
	message := err.Error()
	return strings.Contains(message, "E11000") && strings.Contains(message, "duplicate key")
}

func sleepContext(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
