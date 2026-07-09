package territory

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/sitcon-tw/camp2026-game/internal/content"
	mongomodel "github.com/sitcon-tw/camp2026-game/internal/mongodb/model"
)

const territoryLocksCollection = "territory_locks"

const (
	territoryLockTTL            = time.Minute
	territoryLockRetryDelay     = 25 * time.Millisecond
	territoryLockReleaseTimeout = 2 * time.Second
)

// ErrInsufficientOpenPower is returned when the executor cannot pay a cost.
var ErrInsufficientOpenPower = errors.New("insufficient open power")

// ErrInsufficientSitones is returned when a quantity lock cannot be taken.
var ErrInsufficientSitones = errors.New("insufficient sitone quantity")

// Service bundles the shared dependencies of all territory flows (handler
// packages and the sweeper).
type Service struct {
	Client  *mongo.Client
	DB      *mongo.Database
	Content *content.Store
	Costs   CostConfig
	Engine  EngineConfig
}

// NewService builds a Service with default tunables.
func NewService(client *mongo.Client, db *mongo.Database, store *content.Store) *Service {
	return &Service{
		Client:  client,
		DB:      db,
		Content: store,
		Costs:   DefaultCostConfig(),
		Engine:  DefaultEngineConfig(),
	}
}

// NewID mints a prefixed unique id.
func NewID(prefix string) string {
	return prefix + "_" + bson.NewObjectID().Hex()
}

// WithLock runs fn while holding a distributed lock on the given key,
// following the shop purchase lock pattern (design §19).
func (s *Service) WithLock(ctx context.Context, key string, fn func(ctx context.Context) error) error {
	release, err := s.acquireLock(ctx, key)
	if err != nil {
		return err
	}
	defer release()
	return fn(ctx)
}

// InTransaction runs fn inside a MongoDB transaction when the deployment
// supports it, falling back to plain execution on standalone servers
// (mirrors the shop purchase pattern).
func (s *Service) InTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	if s.Client == nil {
		return fn(ctx)
	}
	err := s.runTransaction(ctx, fn)
	if err != nil && transactionUnsupported(err) {
		return fn(ctx)
	}
	return err
}

func (s *Service) runTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	session, err := s.Client.StartSession()
	if err != nil {
		return err
	}
	defer session.EndSession(ctx)

	return mongo.WithSession(ctx, session, func(ctx context.Context) error {
		if err := session.StartTransaction(); err != nil {
			return err
		}
		committed := false
		defer func() {
			if !committed {
				_ = session.AbortTransaction(context.Background())
			}
		}()
		if err := fn(ctx); err != nil {
			return err
		}
		if err := session.CommitTransaction(ctx); err != nil {
			return err
		}
		committed = true
		return nil
	})
}

func transactionUnsupported(err error) bool {
	var commandError mongo.CommandError
	return errors.As(err, &commandError) &&
		commandError.HasErrorCodeWithMessage(20, "Transaction numbers")
}

func (s *Service) acquireLock(ctx context.Context, key string) (func(), error) {
	lockID := "territory:" + key
	ownerID := NewID("territory_lock")
	collection := s.DB.Collection(territoryLocksCollection)

	for {
		now := time.Now()
		updateResult, err := collection.UpdateOne(
			ctx,
			bson.M{
				"_id": lockID,
				"$or": bson.A{
					bson.M{"expires_at": bson.M{"$lte": now}},
					bson.M{"owner_id": ownerID},
				},
			},
			bson.M{
				"$set": bson.M{
					"owner_id":   ownerID,
					"expires_at": now.Add(territoryLockTTL),
					"updated_at": now,
				},
				"$setOnInsert": bson.M{
					"_id":        lockID,
					"created_at": now,
				},
			},
		)
		if err != nil {
			return nil, err
		}
		if updateResult.MatchedCount > 0 {
			return func() { s.releaseLock(lockID, ownerID) }, nil
		}

		_, err = collection.InsertOne(ctx, bson.M{
			"_id":        lockID,
			"owner_id":   ownerID,
			"expires_at": now.Add(territoryLockTTL),
			"created_at": now,
			"updated_at": now,
		})
		if err == nil {
			return func() { s.releaseLock(lockID, ownerID) }, nil
		}
		if !lockBusy(err) {
			return nil, err
		}
		if err := sleepContext(ctx, territoryLockRetryDelay); err != nil {
			return nil, err
		}
	}
}

func (s *Service) releaseLock(lockID string, ownerID string) {
	ctx, cancel := context.WithTimeout(context.Background(), territoryLockReleaseTimeout)
	defer cancel()
	_, _ = s.DB.Collection(territoryLocksCollection).DeleteOne(ctx, bson.M{
		"_id":      lockID,
		"owner_id": ownerID,
	})
}

func lockBusy(err error) bool {
	if mongo.IsDuplicateKeyError(err) {
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

// TeamMembers returns the players of a team.
func (s *Service) TeamMembers(ctx context.Context, teamID string) ([]mongomodel.Player, error) {
	cursor, err := s.DB.Collection(mongomodel.PlayersCollection).Find(
		ctx,
		bson.M{"team_id": teamID},
		options.Find().SetProjection(bson.D{
			{Key: "_id", Value: 1},
			{Key: "nickname", Value: 1},
			{Key: "team_id", Value: 1},
		}),
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = cursor.Close(ctx) }()

	var players []mongomodel.Player
	if err := cursor.All(ctx, &players); err != nil {
		return nil, err
	}
	return players, nil
}

// PersonalRank computes a player's position on the personal leaderboard
// using the same ordering as the players scope of GET /api/leaderboards
// (sitone count desc, open power desc, nickname asc, id asc). Rank 0 means
// the player was not found on the board.
func (s *Service) PersonalRank(ctx context.Context, playerID string) (int, error) {
	cursor, err := s.DB.Collection(mongomodel.PlayersCollection).Find(
		ctx,
		bson.M{"team_id": bson.M{"$exists": true, "$ne": ""}},
		options.Find().SetProjection(bson.D{
			{Key: "_id", Value: 1},
			{Key: "nickname", Value: 1},
			{Key: "team_id", Value: 1},
		}),
	)
	if err != nil {
		return 0, err
	}
	var players []mongomodel.Player
	err = cursor.All(ctx, &players)
	if err != nil {
		return 0, err
	}

	sitoneCounts, err := s.sumByPlayer(ctx, mongomodel.PlayerSitonesCollection, mongo.Pipeline{
		{{Key: "$match", Value: bson.D{{Key: "quantity", Value: bson.D{{Key: "$gt", Value: 0}}}}}},
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$player_id"},
			{Key: "score", Value: bson.D{{Key: "$sum", Value: "$quantity"}}},
		}}},
	})
	if err != nil {
		return 0, err
	}
	openPower, err := s.sumByPlayer(ctx, mongomodel.OpenPowerRecordsCollection, mongo.Pipeline{
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$player_id"},
			{Key: "score", Value: bson.D{{Key: "$sum", Value: "$amount"}}},
		}}},
	})
	if err != nil {
		return 0, err
	}

	type entry struct {
		id       string
		nickname string
		sitones  int
		power    int
	}
	entries := make([]entry, 0, len(players))
	for _, player := range players {
		if player.ID == "" || player.TeamID == "" {
			continue
		}
		entries = append(entries, entry{
			id:       player.ID,
			nickname: player.Nickname,
			sitones:  sitoneCounts[player.ID],
			power:    openPower[player.ID],
		})
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].sitones != entries[j].sitones {
			return entries[i].sitones > entries[j].sitones
		}
		if entries[i].power != entries[j].power {
			return entries[i].power > entries[j].power
		}
		if entries[i].nickname != entries[j].nickname {
			return entries[i].nickname < entries[j].nickname
		}
		return entries[i].id < entries[j].id
	})
	for i, e := range entries {
		if e.id == playerID {
			return i + 1, nil
		}
	}
	return 0, nil
}

func (s *Service) sumByPlayer(ctx context.Context, collection string, pipeline mongo.Pipeline) (map[string]int, error) {
	cursor, err := s.DB.Collection(collection).Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer func() { _ = cursor.Close(ctx) }()

	var rows []struct {
		ID    string `bson:"_id"`
		Score int    `bson:"score"`
	}
	if err := cursor.All(ctx, &rows); err != nil {
		return nil, err
	}
	out := make(map[string]int, len(rows))
	for _, row := range rows {
		if row.ID != "" {
			out[row.ID] = row.Score
		}
	}
	return out, nil
}

// PlayerQuantities returns sitone_id → quantity for one player.
func (s *Service) PlayerQuantities(ctx context.Context, playerID string) (map[string]int, error) {
	cursor, err := s.DB.Collection(mongomodel.PlayerSitonesCollection).Find(ctx, bson.M{
		"player_id": playerID,
		"quantity":  bson.M{"$gt": 0},
	})
	if err != nil {
		return nil, err
	}
	defer func() { _ = cursor.Close(ctx) }()

	var rows []mongomodel.PlayerSitone
	if err := cursor.All(ctx, &rows); err != nil {
		return nil, err
	}
	out := make(map[string]int, len(rows))
	for _, row := range rows {
		out[row.SitoneID] += row.Quantity
	}
	return out, nil
}

// TeamQuantities returns sitone_id → total quantity across team members.
func (s *Service) TeamQuantities(ctx context.Context, memberIDs []string) (map[string]int, error) {
	cursor, err := s.DB.Collection(mongomodel.PlayerSitonesCollection).Find(ctx, bson.M{
		"player_id": bson.M{"$in": memberIDs},
		"quantity":  bson.M{"$gt": 0},
	})
	if err != nil {
		return nil, err
	}
	defer func() { _ = cursor.Close(ctx) }()

	var rows []mongomodel.PlayerSitone
	if err := cursor.All(ctx, &rows); err != nil {
		return nil, err
	}
	out := make(map[string]int, len(rows))
	for _, row := range rows {
		out[row.SitoneID] += row.Quantity
	}
	return out, nil
}

// DecrementQuantity atomically locks n units of a player's sitone stock.
// Returns ErrInsufficientSitones when the free quantity is too low.
func (s *Service) DecrementQuantity(ctx context.Context, playerID string, sitoneID string, n int) error {
	result, err := s.DB.Collection(mongomodel.PlayerSitonesCollection).UpdateOne(
		ctx,
		bson.M{
			"player_id": playerID,
			"sitone_id": sitoneID,
			"quantity":  bson.M{"$gte": n},
		},
		bson.M{"$inc": bson.M{"quantity": -n}},
	)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return ErrInsufficientSitones
	}
	return nil
}

// IncrementQuantity releases n units back into a player's free stock.
func (s *Service) IncrementQuantity(ctx context.Context, playerID string, sitoneID string, n int) error {
	_, err := s.DB.Collection(mongomodel.PlayerSitonesCollection).UpdateOne(
		ctx,
		bson.M{
			"player_id": playerID,
			"sitone_id": sitoneID,
		},
		bson.M{
			"$setOnInsert": bson.M{
				"_id":       NewID("player_sitone"),
				"player_id": playerID,
				"sitone_id": sitoneID,
			},
			"$inc": bson.M{"quantity": n},
		},
		options.UpdateOne().SetUpsert(true),
	)
	return err
}

// SpendOpenPower charges the player, verifying the balance first. It must
// run inside a lock (and preferably a transaction).
func (s *Service) SpendOpenPower(ctx context.Context, playerID string, amount int, reason string, source string) error {
	if amount <= 0 {
		return nil
	}
	total, err := s.openPowerTotal(ctx, playerID)
	if err != nil {
		return err
	}
	if total < amount {
		return ErrInsufficientOpenPower
	}
	_, err = s.DB.Collection(mongomodel.OpenPowerRecordsCollection).InsertOne(ctx, mongomodel.OpenPowerRecord{
		ID:        NewID("open_power"),
		PlayerID:  playerID,
		Amount:    -amount,
		Reason:    reason,
		Source:    source,
		CreatedAt: time.Now().UTC(),
	})
	return err
}

// GrantOpenPower credits the player (refunds).
func (s *Service) GrantOpenPower(ctx context.Context, playerID string, amount int, reason string, source string) error {
	if amount <= 0 {
		return nil
	}
	_, err := s.DB.Collection(mongomodel.OpenPowerRecordsCollection).InsertOne(ctx, mongomodel.OpenPowerRecord{
		ID:        NewID("open_power"),
		PlayerID:  playerID,
		Amount:    amount,
		Reason:    reason,
		Source:    source,
		CreatedAt: time.Now().UTC(),
	})
	return err
}

func (s *Service) openPowerTotal(ctx context.Context, playerID string) (int, error) {
	cursor, err := s.DB.Collection(mongomodel.OpenPowerRecordsCollection).Aggregate(ctx, mongo.Pipeline{
		{{Key: "$match", Value: bson.D{{Key: "player_id", Value: playerID}}}},
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: nil},
			{Key: "total", Value: bson.D{{Key: "$sum", Value: "$amount"}}},
		}}},
	})
	if err != nil {
		return 0, err
	}
	defer func() { _ = cursor.Close(ctx) }()

	var rows []struct {
		Total int `bson:"total"`
	}
	if err := cursor.All(ctx, &rows); err != nil {
		return 0, err
	}
	if len(rows) == 0 {
		return 0, nil
	}
	return rows[0].Total, nil
}
