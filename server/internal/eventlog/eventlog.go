package eventlog

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	mongomodel "github.com/sitcon-tw/camp2026-game/internal/mongodb/model"
)

const defaultListLimit = 100

// Insert writes a single event log record. Missing ID and CreatedAt are
// filled in automatically.
func Insert(ctx context.Context, db *mongo.Database, event mongomodel.EventLog) (mongomodel.EventLog, error) {
	if event.ID == "" {
		event.ID = "event_log_" + bson.NewObjectID().Hex()
	}
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now().UTC()
	}
	if _, err := db.Collection(mongomodel.EventLogsCollection).InsertOne(ctx, event); err != nil {
		return mongomodel.EventLog{}, err
	}
	return event, nil
}

type Filter struct {
	TeamID     string
	PlayerID   string
	EventTypes []string
	From       time.Time
	To         time.Time
	Limit      int64
	Skip       int64
}

// List returns event logs matching the filter, newest first.
func List(ctx context.Context, db *mongo.Database, filter Filter) ([]mongomodel.EventLog, error) {
	if filter.Limit <= 0 {
		filter.Limit = defaultListLimit
	}

	query := bson.M{}
	if filter.TeamID != "" {
		query["team_id"] = filter.TeamID
	}
	if filter.PlayerID != "" {
		query["$or"] = bson.A{
			bson.M{"actor_player_id": filter.PlayerID},
			bson.M{"target_player_id": filter.PlayerID},
		}
	}
	if len(filter.EventTypes) > 0 {
		query["event_type"] = bson.M{"$in": filter.EventTypes}
	}
	if createdAt := createdAtRange(filter.From, filter.To); len(createdAt) > 0 {
		query["created_at"] = createdAt
	}

	findOptions := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}, {Key: "_id", Value: -1}}).
		SetLimit(filter.Limit)
	if filter.Skip > 0 {
		findOptions.SetSkip(filter.Skip)
	}

	cursor, err := db.Collection(mongomodel.EventLogsCollection).Find(ctx, query, findOptions)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = cursor.Close(ctx)
	}()

	var events []mongomodel.EventLog
	if err := cursor.All(ctx, &events); err != nil {
		return nil, err
	}
	if events == nil {
		return []mongomodel.EventLog{}, nil
	}
	return events, nil
}

func createdAtRange(from time.Time, to time.Time) bson.M {
	createdAt := bson.M{}
	if !from.IsZero() {
		createdAt["$gte"] = from
	}
	if !to.IsZero() {
		createdAt["$lt"] = to
	}
	return createdAt
}
