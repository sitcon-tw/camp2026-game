package fronts

import (
	"context"
	"errors"
	"hash/fnv"
	"sort"
	"strconv"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	mongomodel "github.com/sitcon-tw/camp2026-game/internal/mongodb/model"
)

const (
	frontTradeLoopInterval       = time.Second
	frontTradeMinDuration        = 10 * time.Second
	frontTradeMaxDuration        = 60 * time.Second
	frontTradeHistoryLimit       = 64
	frontTrainCooldown           = 30 * time.Second
	frontTrainSpawnChanceDivisor = 10
)

type frontTradeSettlement struct {
	RouteID      string
	StopIndex    int
	SourcePlayer string
	TargetPlayer string
	SourceReward int
	TargetReward int
	SettledAt    time.Time
}

type frontTradeAdvance struct {
	Changed     bool
	RailChanged bool
	Started     int
	Arrived     int
	Cancelled   int
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
		next, settlements, advanced := advanceFrontTradeState(front, now)
		if !advanced.Changed {
			return nil, nil
		}
		changed = true
		eventName = frontTradeEventName(advanced)
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
				"garrisons": next.Garrisons, "rail_segments": next.RailSegments,
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

func advanceFrontTradeState(front mongomodel.Front, now time.Time) (mongomodel.Front, []frontTradeSettlement, frontTradeAdvance) {
	front = cloneFront(front)
	advance := frontTradeAdvance{}
	advance.Changed = resetFrontTradeWindows(&front, now)
	advance.RailChanged, advance.Cancelled = reconcileFrontRailNetwork(&front, now, "rail network changed")
	advance.Changed = advance.Changed || advance.RailChanged || advance.Cancelled > 0

	garrisons := make(map[string]mongomodel.FrontGarrison, len(front.Garrisons))
	for _, garrison := range front.Garrisons {
		garrisons[garrison.ID] = garrison
	}
	settlements := make([]frontTradeSettlement, 0)
	for routeIndex := range front.TradeRoutes {
		route := &front.TradeRoutes[routeIndex]
		if route.Status != frontTradeRouteActive {
			continue
		}
		if route.NextStopIndex < 1 {
			route.NextStopIndex = 1
		}
		for route.NextStopIndex < len(route.Waypoints) {
			stopIndex := route.NextStopIndex
			waypoint := &route.Waypoints[stopIndex]
			if waypoint.ArrivesAt.After(now) {
				break
			}
			source, sourceOK := garrisons[route.SourceGarrisonID]
			target, targetOK := garrisons[waypoint.GarrisonID]
			if !sourceOK || !targetOK {
				route.Status = frontTradeRouteCancelled
				route.CancellationReason = "train station unavailable"
				route.SettledAt = timePtr(now)
				advance.Cancelled++
				advance.Changed = true
				break
			}
			settledAt := waypoint.ArrivesAt
			waypoint.SettledAt = timePtr(settledAt)
			if source.TeamID != target.TeamID {
				waypoint.SourceReward = applyFrontTradeReward(&front, source.TeamID, waypoint.PotentialReward)
				waypoint.TargetReward = applyFrontTradeReward(&front, target.TeamID, waypoint.PotentialReward)
				route.SourceReward += waypoint.SourceReward
				route.TargetReward += waypoint.TargetReward
				settlements = append(settlements, frontTradeSettlement{
					RouteID: route.ID, StopIndex: stopIndex,
					SourcePlayer: source.PlayerID, TargetPlayer: target.PlayerID,
					SourceReward: waypoint.SourceReward, TargetReward: waypoint.TargetReward,
					SettledAt: settledAt,
				})
			}
			route.NextStopIndex++
			advance.Arrived++
			advance.Changed = true
		}
		if route.Status == frontTradeRouteActive && route.NextStopIndex >= len(route.Waypoints) {
			route.Status = frontTradeRouteCompleted
			route.SettledAt = timePtr(route.ArrivesAt)
		}
	}

	activeSource := make(map[string]struct{})
	for _, route := range front.TradeRoutes {
		if route.Status == frontTradeRouteActive {
			activeSource[route.SourceGarrisonID] = struct{}{}
		}
	}
	for garrisonIndex := range front.Garrisons {
		source := &front.Garrisons[garrisonIndex]
		if _, active := activeSource[source.ID]; active || !frontTrainReady(*source, now) || !frontTrainShouldDepart(front.ID, source.ID, now) {
			continue
		}
		candidates := eligibleFrontTrainDestinations(front, *source)
		if len(candidates) == 0 {
			continue
		}
		target := candidates[stableFrontTradeIndex(source.ID, now, len(candidates))]
		path := frontRailShortestPath(front.RailSegments, source.ID, target.ID)
		if len(path) < 2 {
			continue
		}
		route, ok := newFrontTrainRoute(front, *source, target, path, now)
		if !ok {
			continue
		}
		front.TradeRoutes = append(front.TradeRoutes, route)
		source.LastTrainAt = now
		advance.Started++
		advance.Changed = true
	}
	if !advance.Changed {
		return front, nil, advance
	}
	front.TradeRoutes = pruneFrontTradeRoutes(front.TradeRoutes)
	front.Revision++
	front.Tick++
	front.UpdatedAt = now
	syncTerritoryTeamRanks(front.Teams, front.Territory)
	front.Leaderboard = deriveLeaderboard(front)
	return front, settlements, advance
}

func newFrontTrainRoute(front mongomodel.Front, source mongomodel.FrontGarrison, target mongomodel.FrontGarrison, path []string, now time.Time) (mongomodel.FrontTradeRoute, bool) {
	garrisons := make(map[string]mongomodel.FrontGarrison, len(front.Garrisons))
	for _, garrison := range front.Garrisons {
		garrisons[garrison.ID] = garrison
	}
	waypoints := make([]mongomodel.FrontTradeWaypoint, 0, len(path))
	waypoints = append(waypoints, mongomodel.FrontTradeWaypoint{
		GarrisonID: source.ID, TeamID: source.TeamID, X: source.X, Y: source.Y, ArrivesAt: now,
	})
	totalDistance := 0
	totalReward := 0
	arrivesAt := now
	for index := 1; index < len(path); index++ {
		previous, previousOK := garrisons[path[index-1]]
		current, currentOK := garrisons[path[index]]
		if !previousOK || !currentOK {
			return mongomodel.FrontTradeRoute{}, false
		}
		distance := frontGarrisonDistance(previous, current)
		totalDistance += distance
		arrivesAt = arrivesAt.Add(frontTradeDuration(front, distance))
		reward := 0
		if source.TeamID != current.TeamID {
			reward = frontTradeReward(distance, source.TradeBonusPercent)
			totalReward += reward
		}
		waypoints = append(waypoints, mongomodel.FrontTradeWaypoint{
			GarrisonID: current.ID, TeamID: current.TeamID, X: current.X, Y: current.Y,
			ArrivesAt: arrivesAt, PotentialReward: reward,
		})
	}
	return mongomodel.FrontTradeRoute{
		ID:               "front_train_" + bson.NewObjectID().Hex(),
		SourceGarrisonID: source.ID, TargetGarrisonID: target.ID,
		SourcePlayerID: source.PlayerID, TargetPlayerID: target.PlayerID,
		SourceTeamID: source.TeamID, TargetTeamID: target.TeamID,
		SourceX: source.X, SourceY: source.Y, TargetX: target.X, TargetY: target.Y,
		Distance: totalDistance, PotentialReward: totalReward,
		Waypoints: waypoints, NextStopIndex: 1,
		Status: frontTradeRouteActive, StartedAt: now, ArrivesAt: arrivesAt,
	}, true
}

func eligibleFrontTrainDestinations(front mongomodel.Front, source mongomodel.FrontGarrison) []mongomodel.FrontGarrison {
	out := make([]mongomodel.FrontGarrison, 0, len(front.Garrisons))
	for _, target := range front.Garrisons {
		if target.ID == source.ID || target.TeamID == source.TeamID {
			continue
		}
		if len(frontRailShortestPath(front.RailSegments, source.ID, target.ID)) >= 2 {
			out = append(out, target)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func frontTrainReady(garrison mongomodel.FrontGarrison, now time.Time) bool {
	reference := garrison.LastTrainAt
	if reference.IsZero() {
		reference = garrison.StationedAt
	}
	return !now.Before(reference.Add(frontTrainCooldown))
}

func frontTrainShouldDepart(frontID string, garrisonID string, now time.Time) bool {
	hash := fnv.New32a()
	_, _ = hash.Write([]byte(frontID))
	_, _ = hash.Write([]byte{0})
	_, _ = hash.Write([]byte(garrisonID))
	_, _ = hash.Write([]byte{0})
	_, _ = hash.Write([]byte(strconv.FormatInt(now.Unix(), 10)))
	return hash.Sum32()%frontTrainSpawnChanceDivisor == 0
}

func stableFrontTradeIndex(sourceID string, now time.Time, length int) int {
	if length <= 1 {
		return 0
	}
	hash := fnv.New32a()
	_, _ = hash.Write([]byte(sourceID))
	_, _ = hash.Write([]byte(strconv.FormatInt(now.Unix(), 10)))
	return int(hash.Sum32() % uint32(length))
}

func frontTradeEventName(advance frontTradeAdvance) string {
	switch {
	case advance.Cancelled > 0:
		return "train_cancelled"
	case advance.Arrived > 0:
		return "train_arrived"
	case advance.Started > 0:
		return "train_started"
	case advance.RailChanged:
		return "rail_updated"
	default:
		return "front_updated"
	}
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
	stopID := settlement.RouteID + "_stop_" + strconv.Itoa(settlement.StopIndex)
	records := []mongomodel.OpenPowerRecord{
		{ID: stopID + "_source", PlayerID: settlement.SourcePlayer, Amount: settlement.SourceReward, Reason: "front_trade", Source: frontID + ":" + stopID, CreatedAt: settlement.SettledAt},
		{ID: stopID + "_target", PlayerID: settlement.TargetPlayer, Amount: settlement.TargetReward, Reason: "front_trade", Source: frontID + ":" + stopID, CreatedAt: settlement.SettledAt},
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
