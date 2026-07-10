package fronts

import (
	"context"
	"errors"
	"hash/fnv"
	"sort"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	mongomodel "github.com/sitcon-tw/camp2026-game/internal/mongodb/model"
)

const (
	frontTradeLoopInterval = time.Second
	frontTradeMinDuration  = 10 * time.Second
	frontTradeMaxDuration  = 60 * time.Second
	frontTradeHistoryLimit = 64
)

type frontTradeSettlement struct {
	RouteID      string
	SourcePlayer string
	TargetPlayer string
	SourceReward int
	TargetReward int
	SettledAt    time.Time
}

func (h *Handler) runFrontTradeLoop(ctx context.Context) {
	ticker := time.NewTicker(frontTradeLoopInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			if err := h.advanceCurrentFrontTrades(ctx, now.UTC()); err != nil && !errors.Is(err, context.Canceled) && h.log != nil {
				h.log.Error("advance front trades", "error", err)
			}
		}
	}
}

func (h *Handler) advanceCurrentFrontTrades(ctx context.Context, now time.Time) error {
	cursor, err := h.db.Collection(mongomodel.FrontsCollection).Find(
		ctx,
		bson.M{
			"current":     true,
			"status":      bson.M{"$in": bson.A{mongomodel.FrontStatusOpenPlay, "surge", "booth_window"}},
			"garrisons.0": bson.M{"$exists": true},
		},
		options.Find().SetProjection(bson.M{"_id": 1}),
	)
	if err != nil {
		return err
	}
	defer cursor.Close(ctx)
	for cursor.Next(ctx) {
		var record struct {
			ID string `bson:"_id"`
		}
		if err := cursor.Decode(&record); err != nil {
			return err
		}
		name, changed, err := h.advanceFrontTrades(ctx, record.ID, now)
		if err != nil {
			return err
		}
		if changed {
			h.broker.Publish(record.ID, FrontEvent{Name: name})
		}
	}
	return cursor.Err()
}

func (h *Handler) advanceFrontTrades(ctx context.Context, frontID string, now time.Time) (string, bool, error) {
	for attempt := 0; attempt < 4; attempt++ {
		name, changed, conflict, err := h.advanceFrontTradesOnce(ctx, frontID, now)
		if err != nil {
			return "", false, err
		}
		if !conflict {
			return name, changed, nil
		}
	}
	return "", false, errFrontRevisionConflict
}

func (h *Handler) advanceFrontTradesOnce(ctx context.Context, frontID string, now time.Time) (string, bool, bool, error) {
	session, err := h.db.Client().StartSession()
	if err != nil {
		return "", false, false, err
	}
	defer session.EndSession(ctx)

	eventName := "front_updated"
	changed := false
	conflict := false
	_, err = session.WithTransaction(ctx, func(ctx context.Context) (any, error) {
		var front mongomodel.Front
		if err := h.db.Collection(mongomodel.FrontsCollection).FindOne(ctx, bson.M{"_id": frontID}).Decode(&front); err != nil {
			return nil, err
		}
		front = h.withFrontDefaults(front)
		previousRevision := front.Revision
		next, settlements, started, advanced := advanceFrontTradeState(front, now)
		if !advanced {
			return nil, nil
		}
		changed = true
		if len(settlements) > 0 {
			eventName = "trade_completed"
		} else if started > 0 {
			eventName = "trade_started"
		}
		for _, settlement := range settlements {
			if err := h.recordFrontTradeRewards(ctx, frontID, settlement); err != nil {
				return nil, err
			}
		}
		result, err := h.db.Collection(mongomodel.FrontsCollection).UpdateOne(
			ctx,
			frontRevisionFilter(frontID, previousRevision),
			bson.M{"$set": bson.M{
				"teams": next.Teams, "leaderboard": next.Leaderboard,
				"trade_routes": next.TradeRoutes, "revision": next.Revision,
				"tick": next.Tick, "updated_at": next.UpdatedAt,
			}},
		)
		if err != nil {
			return nil, err
		}
		if result.MatchedCount == 0 {
			conflict = true
			return nil, errFrontRevisionConflict
		}
		return nil, nil
	})
	if errors.Is(err, errFrontRevisionConflict) {
		return "", false, true, nil
	}
	return eventName, changed, conflict, err
}

func advanceFrontTradeState(front mongomodel.Front, now time.Time) (mongomodel.Front, []frontTradeSettlement, int, bool) {
	front = cloneFront(front)
	changed := resetFrontTradeWindows(&front, now)
	garrisons := make(map[string]mongomodel.FrontGarrison, len(front.Garrisons))
	for _, garrison := range front.Garrisons {
		garrisons[garrison.ID] = garrison
	}
	settlements := make([]frontTradeSettlement, 0)
	for i := range front.TradeRoutes {
		route := &front.TradeRoutes[i]
		if route.Status != frontTradeRouteActive || route.ArrivesAt.After(now) {
			continue
		}
		source, sourceOK := garrisons[route.SourceGarrisonID]
		target, targetOK := garrisons[route.TargetGarrisonID]
		if !sourceOK || !targetOK || source.TeamID == target.TeamID {
			route.Status = frontTradeRouteCancelled
			route.CancellationReason = "trade endpoint unavailable"
			route.SettledAt = timePtr(now)
			changed = true
			continue
		}
		sourceReward := applyFrontTradeReward(&front, source.TeamID, route.PotentialReward)
		targetReward := applyFrontTradeReward(&front, target.TeamID, route.PotentialReward)
		route.Status = frontTradeRouteCompleted
		route.SourceReward = sourceReward
		route.TargetReward = targetReward
		route.SettledAt = timePtr(now)
		settlements = append(settlements, frontTradeSettlement{
			RouteID: route.ID, SourcePlayer: source.PlayerID, TargetPlayer: target.PlayerID,
			SourceReward: sourceReward, TargetReward: targetReward, SettledAt: now,
		})
		changed = true
	}

	activeSource := make(map[string]struct{})
	for _, route := range front.TradeRoutes {
		if route.Status == frontTradeRouteActive {
			activeSource[route.SourceGarrisonID] = struct{}{}
		}
	}
	started := 0
	for _, source := range front.Garrisons {
		if _, active := activeSource[source.ID]; active || frontTradeRemaining(front, source.TeamID) <= 0 {
			continue
		}
		candidates := eligibleFrontTradeTargets(front, source)
		if len(candidates) == 0 {
			continue
		}
		target := candidates[stableTradeTargetIndex(source.ID, front.Revision, len(candidates))]
		distance := absFrontInt(source.X-target.X) + absFrontInt(source.Y-target.Y)
		reward := frontTradeReward(distance, source.TradeBonusPercent)
		front.TradeRoutes = append(front.TradeRoutes, mongomodel.FrontTradeRoute{
			ID:               "front_trade_" + bson.NewObjectID().Hex(),
			SourceGarrisonID: source.ID, TargetGarrisonID: target.ID,
			SourcePlayerID: source.PlayerID, TargetPlayerID: target.PlayerID,
			SourceTeamID: source.TeamID, TargetTeamID: target.TeamID,
			SourceX: source.X, SourceY: source.Y, TargetX: target.X, TargetY: target.Y,
			Distance: distance, PotentialReward: reward, Status: frontTradeRouteActive,
			StartedAt: now, ArrivesAt: now.Add(frontTradeDuration(front, distance)),
		})
		started++
		changed = true
	}
	if !changed {
		return front, nil, 0, false
	}
	front.TradeRoutes = pruneFrontTradeRoutes(front.TradeRoutes)
	front.Revision++
	front.Tick++
	front.UpdatedAt = now
	syncTerritoryTeamRanks(front.Teams, front.Territory)
	front.Leaderboard = deriveLeaderboard(front)
	return front, settlements, started, true
}

func resetFrontTradeWindows(front *mongomodel.Front, now time.Time) bool {
	window := now.Truncate(time.Hour)
	changed := false
	for i := range front.Teams {
		team := &front.Teams[i]
		if team.TradeHourlyLimit <= 0 {
			team.TradeHourlyLimit = frontTradeHourlyLimit
			changed = true
		}
		if team.TradeWindowStartedAt.IsZero() || !team.TradeWindowStartedAt.Equal(window) {
			team.TradeWindowStartedAt = window
			team.TradeHourlyEarned = 0
			changed = true
		}
	}
	return changed
}

func eligibleFrontTradeTargets(front mongomodel.Front, source mongomodel.FrontGarrison) []mongomodel.FrontGarrison {
	out := make([]mongomodel.FrontGarrison, 0, len(front.Garrisons))
	for _, target := range front.Garrisons {
		if target.ID == source.ID || target.TeamID == source.TeamID || frontTradeRemaining(front, target.TeamID) <= 0 {
			continue
		}
		out = append(out, target)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func frontTradeRemaining(front mongomodel.Front, teamID string) int {
	index := frontTeamIndex(front.Teams, teamID)
	if index < 0 {
		return 0
	}
	limit := front.Teams[index].TradeHourlyLimit
	if limit <= 0 {
		limit = frontTradeHourlyLimit
	}
	return maxFrontInt(0, limit-front.Teams[index].TradeHourlyEarned)
}

func applyFrontTradeReward(front *mongomodel.Front, teamID string, reward int) int {
	index := frontTeamIndex(front.Teams, teamID)
	if index < 0 {
		return 0
	}
	awarded := minFrontInt(reward, frontTradeRemaining(*front, teamID))
	front.Teams[index].TradeHourlyEarned += awarded
	front.Teams[index].TradeScore += awarded
	front.Teams[index].Score += awarded
	return awarded
}

func frontTradeDuration(front mongomodel.Front, distance int) time.Duration {
	maxDistance := 1
	if front.Territory != nil {
		maxDistance = maxFrontInt(1, front.Territory.Width+front.Territory.Height-2)
	}
	span := frontTradeMaxDuration - frontTradeMinDuration
	return frontTradeMinDuration + time.Duration(distance)*span/time.Duration(maxDistance)
}

func frontTradeReward(distance int, bonusPercent int) int {
	base := 5 + maxFrontInt(0, distance)/8
	return base + scaledFrontSitoneBonus(base, bonusPercent)
}

func stableTradeTargetIndex(sourceID string, revision int64, length int) int {
	if length <= 1 {
		return 0
	}
	hash := fnv.New32a()
	_, _ = hash.Write([]byte(sourceID))
	var revisionBytes [8]byte
	for i := range revisionBytes {
		revisionBytes[i] = byte(revision >> (8 * i))
	}
	_, _ = hash.Write(revisionBytes[:])
	return int(hash.Sum32() % uint32(length))
}

func pruneFrontTradeRoutes(routes []mongomodel.FrontTradeRoute) []mongomodel.FrontTradeRoute {
	active := make([]mongomodel.FrontTradeRoute, 0, len(routes))
	history := make([]mongomodel.FrontTradeRoute, 0, len(routes))
	for _, route := range routes {
		if route.Status == frontTradeRouteActive {
			active = append(active, route)
		} else {
			history = append(history, route)
		}
	}
	sort.Slice(history, func(i, j int) bool {
		return history[i].StartedAt.After(history[j].StartedAt)
	})
	if len(history) > frontTradeHistoryLimit {
		history = history[:frontTradeHistoryLimit]
	}
	return append(active, history...)
}

func (h *Handler) recordFrontTradeRewards(ctx context.Context, frontID string, settlement frontTradeSettlement) error {
	records := []mongomodel.OpenPowerRecord{
		{ID: settlement.RouteID + "_source", PlayerID: settlement.SourcePlayer, Amount: settlement.SourceReward, Reason: "front_trade", Source: frontID + ":" + settlement.RouteID, CreatedAt: settlement.SettledAt},
		{ID: settlement.RouteID + "_target", PlayerID: settlement.TargetPlayer, Amount: settlement.TargetReward, Reason: "front_trade", Source: frontID + ":" + settlement.RouteID, CreatedAt: settlement.SettledAt},
	}
	for _, record := range records {
		if record.PlayerID == "" || record.Amount <= 0 {
			continue
		}
		if _, err := h.db.Collection(mongomodel.OpenPowerRecordsCollection).InsertOne(ctx, record); err != nil {
			return err
		}
	}
	return nil
}
