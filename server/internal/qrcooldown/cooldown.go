package qrcooldown

import (
	"context"
	"errors"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	mongomodel "github.com/sitcon-tw/camp2026-game/internal/mongodb/model"
)

var ErrActive = errors.New("qr code scan cooldown active")

type Reservation struct {
	PlayerID string
	OwnerID  string
}

func Reserve(ctx context.Context, db *mongo.Database, playerID string, ownerID string, duration time.Duration, now time.Time) (Reservation, error) {
	if duration <= 0 {
		return Reservation{}, nil
	}

	reservation := Reservation{PlayerID: playerID, OwnerID: ownerID}
	collection := db.Collection(mongomodel.QRScanCooldownsCollection)
	lockID := CooldownID(playerID)

	updateResult, err := collection.UpdateOne(
		ctx,
		ReserveFilter(lockID, ownerID, now),
		ReserveUpdate(lockID, playerID, ownerID, now, duration),
	)
	if err != nil {
		return Reservation{}, err
	}
	if updateResult.MatchedCount > 0 {
		return reservation, nil
	}

	_, err = collection.InsertOne(ctx, ReserveInsert(lockID, playerID, ownerID, now, duration))
	if err == nil {
		return reservation, nil
	}
	if reserveConflict(err) {
		return Reservation{}, ErrActive
	}
	return Reservation{}, err
}

func Release(ctx context.Context, db *mongo.Database, reservation Reservation) error {
	if reservation.PlayerID == "" || reservation.OwnerID == "" {
		return nil
	}
	_, err := db.Collection(mongomodel.QRScanCooldownsCollection).DeleteOne(ctx, bson.M{
		"_id":      CooldownID(reservation.PlayerID),
		"owner_id": reservation.OwnerID,
	})
	return err
}

func CooldownID(playerID string) string {
	return "qr_scan_cooldown:" + playerID
}

func ReserveFilter(lockID string, ownerID string, now time.Time) bson.M {
	return bson.M{
		"_id":        lockID,
		"expires_at": bson.M{"$lte": now},
	}
}

func ReserveUpdate(lockID string, playerID string, ownerID string, now time.Time, duration time.Duration) bson.M {
	return bson.M{
		"$set": bson.M{
			"owner_id":   ownerID,
			"expires_at": now.Add(duration),
			"updated_at": now,
		},
		"$setOnInsert": bson.M{
			"_id":        lockID,
			"player_id":  playerID,
			"created_at": now,
		},
	}
}

func ReserveInsert(lockID string, playerID string, ownerID string, now time.Time, duration time.Duration) bson.M {
	return bson.M{
		"_id":        lockID,
		"player_id":  playerID,
		"owner_id":   ownerID,
		"expires_at": now.Add(duration),
		"created_at": now,
		"updated_at": now,
	}
}

func reserveConflict(err error) bool {
	return mongo.IsDuplicateKeyError(err) || duplicateKeyMessage(err)
}

func duplicateKeyMessage(err error) bool {
	if err == nil {
		return false
	}
	message := err.Error()
	return strings.Contains(message, "E11000") && strings.Contains(message, "duplicate key")
}
