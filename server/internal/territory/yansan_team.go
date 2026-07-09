package territory

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	mongomodel "github.com/sitcon-tw/camp2026-game/internal/mongodb/model"
)

// EnsureYansanTeam idempotently seeds the special staff-controlled boss
// team (研三舍, decision 8). It only inserts when missing and never
// overwrites an existing document.
func EnsureYansanTeam(ctx context.Context, db *mongo.Database) error {
	_, err := db.Collection(mongomodel.TeamsCollection).UpdateOne(
		ctx,
		bson.M{"_id": TeamYansanID},
		bson.M{"$setOnInsert": bson.M{
			"name":      YansanTeamName,
			"is_system": true,
			"is_boss":   true,
		}},
		options.UpdateOne().SetUpsert(true),
	)
	return err
}
