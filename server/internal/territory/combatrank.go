package territory

// Hidden combat standings (design §4). The score formula, weights and
// jitter range live server-side only: never expose scores or coefficients
// through any API response — callers may only surface tier and allowed
// actions.

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/sitcon-tw/camp2026-game/internal/content"
	mongomodel "github.com/sitcon-tw/camp2026-game/internal/mongodb/model"
)

// Tier is the hidden combat tier a team is assigned by rank.
type Tier int

const (
	// TierLeader: combat rank 1-3. Cannot attack, can be attacked.
	TierLeader Tier = 1
	// TierContested: combat rank 4-6. Can attack and be attacked.
	TierContested Tier = 2
	// TierChallenger: combat rank 7-9. Can attack, cannot be attacked.
	TierChallenger Tier = 3
)

func (t Tier) String() string {
	switch t {
	case TierLeader:
		return "leader"
	case TierContested:
		return "contested"
	case TierChallenger:
		return "challenger"
	default:
		return "unknown"
	}
}

// CanAttack reports whether a team in the given tier may initiate attacks.
func CanAttack(tier Tier) bool {
	return tier == TierContested || tier == TierChallenger
}

// CanBeAttacked reports whether a team in the given tier may be targeted.
func CanBeAttacked(tier Tier) bool {
	return tier == TierLeader || tier == TierContested
}

// TeamStanding is one team's position in the hidden combat ranking.
// Score is server-side only and must never reach the frontend.
type TeamStanding struct {
	TeamID      string
	TeamName    string
	PlayerCount int
	Rank        int
	Tier        Tier
	Score       float64
}

// CombatRankConfig holds the hidden scoring coefficients. Defaults are
// tunable later via admin config; none of this is exposed to the frontend.
type CombatRankConfig struct {
	// SitonePowerWeight scales the summed (attack+defense) value of every
	// sitone owned by team members.
	SitonePowerWeight float64
	// SitoneCountWeight scales raw sitone quantity for sitones without
	// combat stats.
	SitoneCountWeight float64
	// OpenPowerWeight scales the summed open power of team members.
	OpenPowerWeight float64
	// JitterFraction is the deterministic random wobble applied to each
	// team score (anti-precision-gaming, design §14).
	JitterFraction float64
	// BalanceCorrection, when set, adjusts a team's raw score before
	// jitter (catch-up / balancing hook).
	BalanceCorrection func(teamID string, score float64) float64
	// SeedRef seeds the deterministic jitter so repeated computations in
	// the same window agree. Empty disables jitter.
	SeedRef string
}

// DefaultCombatRankConfig returns the baseline hidden ranking coefficients.
func DefaultCombatRankConfig() CombatRankConfig {
	return CombatRankConfig{
		SitonePowerWeight: 1.0,
		SitoneCountWeight: 5.0,
		OpenPowerWeight:   0.5,
		JitterFraction:    0.05,
	}
}

// ComputeStandings computes the hidden combat standings with default
// coefficients. System teams (研三舍 / is_system) are excluded.
func ComputeStandings(ctx context.Context, db *mongo.Database, store *content.Store) ([]TeamStanding, error) {
	return ComputeStandingsWithConfig(ctx, db, store, DefaultCombatRankConfig())
}

// ComputeStandingsWithConfig computes the hidden combat standings using the
// given coefficients. Results are ordered best rank first with tiers
// assigned (rank 1-3 leader, 4-6 contested, 7-9+ challenger).
func ComputeStandingsWithConfig(ctx context.Context, db *mongo.Database, store *content.Store, config CombatRankConfig) ([]TeamStanding, error) {
	teams, err := findCombatTeams(ctx, db)
	if err != nil {
		return nil, fmt.Errorf("combat standings teams: %w", err)
	}
	teamByPlayer, playerCounts, err := teamMembership(ctx, db, teams)
	if err != nil {
		return nil, fmt.Errorf("combat standings players: %w", err)
	}
	sitonePower, sitoneCount, err := sitoneScoresByTeam(ctx, db, store, teamByPlayer)
	if err != nil {
		return nil, fmt.Errorf("combat standings sitones: %w", err)
	}
	openPower, err := openPowerByTeam(ctx, db, teamByPlayer)
	if err != nil {
		return nil, fmt.Errorf("combat standings open power: %w", err)
	}

	scores := make([]TeamScore, 0, len(teams))
	for _, team := range teams {
		scores = append(scores, TeamScore{
			TeamID:      team.ID,
			TeamName:    team.Name,
			PlayerCount: playerCounts[team.ID],
			SitonePower: sitonePower[team.ID],
			SitoneCount: sitoneCount[team.ID],
			OpenPower:   openPower[team.ID],
		})
	}
	return BuildStandings(scores, config), nil
}

// TeamScore is the raw per-team input to the hidden ranking formula.
type TeamScore struct {
	TeamID      string
	TeamName    string
	PlayerCount int
	SitonePower int
	SitoneCount int
	OpenPower   int
}

// BuildStandings turns raw team scores into ordered standings with tiers.
// It is pure and deterministic for a given config (including SeedRef).
func BuildStandings(scores []TeamScore, config CombatRankConfig) []TeamStanding {
	var roll *Roll
	if config.JitterFraction > 0 && config.SeedRef != "" {
		roll = NewRoll(config.SeedRef)
	}

	standings := make([]TeamStanding, 0, len(scores))
	for _, score := range scores {
		raw := config.SitonePowerWeight*float64(score.SitonePower) +
			config.SitoneCountWeight*float64(score.SitoneCount) +
			config.OpenPowerWeight*float64(score.OpenPower)
		if score.PlayerCount > 0 {
			raw /= float64(score.PlayerCount)
		}
		if config.BalanceCorrection != nil {
			raw = config.BalanceCorrection(score.TeamID, raw)
		}
		if roll != nil {
			raw *= roll.Jitter(config.JitterFraction)
		}
		standings = append(standings, TeamStanding{
			TeamID:      score.TeamID,
			TeamName:    score.TeamName,
			PlayerCount: score.PlayerCount,
			Score:       raw,
		})
	}

	sortStandings(standings)
	for i := range standings {
		standings[i].Rank = i + 1
		standings[i].Tier = TierForRank(standings[i].Rank)
	}
	return standings
}

// TierForRank maps a combat rank to its tier: 1-3 leader, 4-6 contested,
// 7 and beyond challenger.
func TierForRank(rank int) Tier {
	switch {
	case rank <= 3:
		return TierLeader
	case rank <= 6:
		return TierContested
	default:
		return TierChallenger
	}
}

func sortStandings(standings []TeamStanding) {
	for i := 1; i < len(standings); i++ {
		for j := i; j > 0 && standingLess(standings[j], standings[j-1]); j-- {
			standings[j], standings[j-1] = standings[j-1], standings[j]
		}
	}
}

func standingLess(left TeamStanding, right TeamStanding) bool {
	if left.Score != right.Score {
		return left.Score > right.Score
	}
	return left.TeamID < right.TeamID
}

func findCombatTeams(ctx context.Context, db *mongo.Database) ([]mongomodel.Team, error) {
	cursor, err := db.Collection(mongomodel.TeamsCollection).Find(
		ctx,
		bson.M{
			"_id":       bson.M{"$nin": bson.A{TeamYansanID, TeamStaffID}},
			"is_system": bson.M{"$ne": true},
		},
		options.Find().SetSort(bson.D{{Key: "_id", Value: 1}}),
	)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = cursor.Close(ctx)
	}()

	var teams []mongomodel.Team
	if err := cursor.All(ctx, &teams); err != nil {
		return nil, err
	}
	return teams, nil
}

func teamMembership(ctx context.Context, db *mongo.Database, teams []mongomodel.Team) (map[string]string, map[string]int, error) {
	teamIDs := make([]string, 0, len(teams))
	for _, team := range teams {
		teamIDs = append(teamIDs, team.ID)
	}

	cursor, err := db.Collection(mongomodel.PlayersCollection).Find(
		ctx,
		bson.M{"team_id": bson.M{"$in": teamIDs}},
		options.Find().SetProjection(bson.D{
			{Key: "_id", Value: 1},
			{Key: "team_id", Value: 1},
		}),
	)
	if err != nil {
		return nil, nil, err
	}
	defer func() {
		_ = cursor.Close(ctx)
	}()

	var players []mongomodel.Player
	if err := cursor.All(ctx, &players); err != nil {
		return nil, nil, err
	}

	teamByPlayer := make(map[string]string, len(players))
	playerCounts := make(map[string]int, len(teams))
	for _, player := range players {
		if player.ID == "" || player.TeamID == "" {
			continue
		}
		teamByPlayer[player.ID] = player.TeamID
		playerCounts[player.TeamID]++
	}
	return teamByPlayer, playerCounts, nil
}

func sitoneScoresByTeam(ctx context.Context, db *mongo.Database, store *content.Store, teamByPlayer map[string]string) (map[string]int, map[string]int, error) {
	cursor, err := db.Collection(mongomodel.PlayerSitonesCollection).Aggregate(ctx, mongo.Pipeline{
		{{Key: "$match", Value: bson.D{{Key: "quantity", Value: bson.D{{Key: "$gt", Value: 0}}}}}},
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: bson.D{
				{Key: "player_id", Value: "$player_id"},
				{Key: "sitone_id", Value: "$sitone_id"},
			}},
			{Key: "quantity", Value: bson.D{{Key: "$sum", Value: "$quantity"}}},
		}}},
	})
	if err != nil {
		return nil, nil, err
	}
	defer func() {
		_ = cursor.Close(ctx)
	}()

	var rows []struct {
		ID struct {
			PlayerID string `bson:"player_id"`
			SitoneID string `bson:"sitone_id"`
		} `bson:"_id"`
		Quantity int `bson:"quantity"`
	}
	if err := cursor.All(ctx, &rows); err != nil {
		return nil, nil, err
	}

	power := make(map[string]int)
	count := make(map[string]int)
	for _, row := range rows {
		teamID, ok := teamByPlayer[row.ID.PlayerID]
		if !ok {
			continue
		}
		count[teamID] += row.Quantity
		if sitone, ok := store.GetSitone(row.ID.SitoneID); ok {
			power[teamID] += row.Quantity * (sitone.Attack + sitone.Defense)
		}
	}
	return power, count, nil
}

func openPowerByTeam(ctx context.Context, db *mongo.Database, teamByPlayer map[string]string) (map[string]int, error) {
	cursor, err := db.Collection(mongomodel.OpenPowerRecordsCollection).Aggregate(ctx, mongo.Pipeline{
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$player_id"},
			{Key: "total", Value: bson.D{{Key: "$sum", Value: "$amount"}}},
		}}},
	})
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = cursor.Close(ctx)
	}()

	var rows []struct {
		ID    string `bson:"_id"`
		Total int    `bson:"total"`
	}
	if err := cursor.All(ctx, &rows); err != nil {
		return nil, err
	}

	totals := make(map[string]int)
	for _, row := range rows {
		if teamID, ok := teamByPlayer[row.ID]; ok {
			totals[teamID] += row.Total
		}
	}
	return totals, nil
}
