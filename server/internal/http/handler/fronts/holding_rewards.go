package fronts

import (
	"context"
	"fmt"
	"hash/fnv"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	mongomodel "github.com/sitcon-tw/camp2026-game/internal/mongodb/model"
)

var frontHoldingRewardBases = [...]int{3, 7, 11, 15}

const frontHoldingRewardVariancePercent = 30

type frontHoldingRewardSettlement struct {
	TeamID     string
	X          int
	Y          int
	Period     int
	Amount     int
	OccupiedAt time.Time
	SettledAt  time.Time
}

func advanceFrontHoldingRewardState(front mongomodel.Front, now time.Time) (mongomodel.Front, []frontHoldingRewardSettlement, bool) {
	if front.Territory == nil {
		return front, nil, false
	}
	matrix, err := decodeTerritoryRows(front.Territory)
	if err != nil {
		return front, nil, false
	}
	bases := territoryBaseSet(front.Territory.Bases)
	settlements := make([]frontHoldingRewardSettlement, 0)
	changed := false
	for y := range matrix {
		for x := range matrix[y] {
			cell := &matrix[y][x]
			if !cell.Playable || cell.Owner == "" || cell.OccupiedAt.IsZero() || now.Before(cell.OccupiedAt) {
				continue
			}
			if _, base := bases[[2]int{x, y}]; base {
				continue
			}
			duePeriods := int(now.Sub(cell.OccupiedAt) / frontTerritoryRewardInterval)
			for period := cell.HoldingRewardPeriods + 1; period <= duePeriods; period++ {
				settledAt := cell.OccupiedAt.Add(time.Duration(period) * frontTerritoryRewardInterval)
				settlements = append(settlements, frontHoldingRewardSettlement{
					TeamID: cell.Owner, X: x, Y: y, Period: period,
					Amount:     frontHoldingRewardAmount(front.ID, cell.OccupiedAt, x, y, period),
					OccupiedAt: cell.OccupiedAt, SettledAt: settledAt,
				})
			}
			if duePeriods > cell.HoldingRewardPeriods {
				cell.HoldingRewardPeriods = duePeriods
				changed = true
			}
		}
	}
	if changed {
		front.Territory.Rows = encodeTerritoryRows(matrix)
	}
	return front, settlements, changed
}

func frontHoldingRewardAmount(frontID string, occupiedAt time.Time, x int, y int, period int) int {
	level := clampFrontInt(period/3+1, 1, frontTerritoryMaximumLevel)
	base := frontHoldingRewardBases[level-1]
	hash := fnv.New32a()
	_, _ = fmt.Fprintf(hash, "%s:%d:%d:%d:%d", frontID, occupiedAt.UnixNano(), x, y, period)
	span := frontHoldingRewardVariancePercent*2 + 1
	variance := int(hash.Sum32()%uint32(span)) - frontHoldingRewardVariancePercent
	return maxFrontInt(1, (base*(100+variance)+50)/100)
}

func (h *Handler) recordFrontHoldingRewards(ctx context.Context, frontID string, settlements []frontHoldingRewardSettlement) error {
	playersByTeam := make(map[string][]string)
	for _, settlement := range settlements {
		playerIDs, ok := playersByTeam[settlement.TeamID]
		if !ok {
			cursor, err := h.db.Collection(mongomodel.PlayersCollection).Find(
				ctx,
				bson.M{"team_id": settlement.TeamID},
				options.Find().SetProjection(bson.M{"_id": 1}),
			)
			if err != nil {
				return err
			}
			for cursor.Next(ctx) {
				var player struct {
					ID string `bson:"_id"`
				}
				if err := cursor.Decode(&player); err != nil {
					cursor.Close(ctx)
					return err
				}
				playerIDs = append(playerIDs, player.ID)
			}
			if err := cursor.Err(); err != nil {
				cursor.Close(ctx)
				return err
			}
			cursor.Close(ctx)
			playersByTeam[settlement.TeamID] = playerIDs
		}
		source := fmt.Sprintf("%s:%d:%d:%d:%d", frontID, settlement.X, settlement.Y, settlement.OccupiedAt.UnixNano(), settlement.Period)
		for _, playerID := range playerIDs {
			record := mongomodel.OpenPowerRecord{
				ID: fmt.Sprintf("front_holding_%s_%s", playerID, source), PlayerID: playerID,
				Amount: settlement.Amount, Reason: "front_territory_holding", Source: source, CreatedAt: settlement.SettledAt,
			}
			if _, err := h.db.Collection(mongomodel.OpenPowerRecordsCollection).InsertOne(ctx, record); err != nil {
				return err
			}
		}
	}
	return nil
}
