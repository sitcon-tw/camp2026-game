package matches

import (
	"context"
	"errors"
	"sort"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/sitcon-tw/camp2026-game/internal/content"
	"github.com/sitcon-tw/camp2026-game/internal/gamecontrol"
	"github.com/sitcon-tw/camp2026-game/internal/http/authctx"
	mongomodel "github.com/sitcon-tw/camp2026-game/internal/mongodb/model"
)

const (
	computerDifficultyEasy   = "easy"
	computerDifficultyNormal = "normal"
	computerDifficultyHard   = "hard"
)

type computerRankEntry struct {
	PlayerID    string
	Nickname    string
	SitoneCount int
	OpenPower   int
}

func (h *Handler) ensureComputerAnswer(ctx context.Context, match *mongomodel.Match, now time.Time) (bool, error) {
	if match == nil || !matchHasComputerPlayers(*match) || match.Status != mongomodel.MatchStatusActive {
		return false, nil
	}
	if activeMatchPhase(*match) != mongomodel.MatchPhaseAnswering {
		return false, nil
	}
	if match.CurrentQuestionIndex < 0 || match.CurrentQuestionIndex >= len(match.QuestionIDs) {
		return false, nil
	}

	computerIndexes := matchComputerPlayerIndexes(*match)
	humanPlayers := matchHumanPlayers(*match)
	if len(computerIndexes) == 0 || len(humanPlayers) == 0 {
		return false, nil
	}

	questionID := match.QuestionIDs[match.CurrentQuestionIndex]
	allHumansAnswered := true
	for _, humanPlayer := range humanPlayers {
		humanAnswered, err := h.hasPlayerAnswered(ctx, match.ID, humanPlayer.PlayerID, questionID)
		if err != nil {
			return false, err
		}
		if !humanAnswered {
			allHumansAnswered = false
			break
		}
	}
	if !allHumansAnswered && now.Before(match.RoundEndsAt) {
		return false, nil
	}

	question, ok := h.content.GetQuizQuestion(questionID)
	if !ok {
		return false, nil
	}
	settings, err := gamecontrol.ReadSettings(ctx, h.db)
	if err != nil {
		return false, err
	}
	difficulty, err := h.computerDifficulty(ctx, humanPlayers[0].PlayerID)
	if err != nil {
		return false, err
	}
	accuracy := computerAccuracy(settings, difficulty)

	answered := false
	for _, computerIndex := range computerIndexes {
		computerPlayer := match.Players[computerIndex]
		if _, err := h.findAnswer(ctx, match.ID, computerPlayer.PlayerID, questionID); err == nil {
			continue
		} else if !errors.Is(err, mongo.ErrNoDocuments) {
			return false, err
		}

		choice := computerAnswerChoice(*match, question, computerPlayer, accuracy)
		effects, err := h.matchPlayerBattleEffects(ctx, computerPlayer)
		if err != nil {
			return false, err
		}
		correct, baseScore, score := scoreAnswer(question, choice, now, match.RoundEndsAt, effects)
		elapsedMillis := now.Sub(match.RoundStartedAt).Milliseconds()
		if elapsedMillis < 0 {
			elapsedMillis = 0
		}

		answer := mongomodel.MatchAnswer{
			ID:            matchAnswerRecordID(match.ID, computerPlayer.PlayerID, questionID),
			MatchID:       match.ID,
			PlayerID:      computerPlayer.PlayerID,
			QuestionID:    questionID,
			Choice:        choice,
			Correct:       correct,
			BaseScore:     baseScore,
			BonusScore:    score - baseScore,
			Score:         score,
			ElapsedMillis: elapsedMillis,
			AnsweredAt:    now,
		}
		if _, err := h.db.Collection(mongomodel.MatchAnswersCollection).InsertOne(ctx, answer); err != nil {
			if mongo.IsDuplicateKeyError(err) {
				continue
			}
			return false, err
		}

		if err := h.applyMatchAnswerScore(ctx, match.ID, computerPlayer.PlayerID, score); err != nil {
			return false, err
		}
		match.Players[computerIndex].Score += score
		match.Revision++
		answered = true
	}
	return answered, nil
}

func (h *Handler) hasPlayerAnswered(ctx context.Context, matchID string, playerID string, questionID string) (bool, error) {
	if _, err := h.findAnswer(ctx, matchID, playerID, questionID); err == nil {
		return true, nil
	} else if !errors.Is(err, mongo.ErrNoDocuments) {
		return false, err
	}
	return false, nil
}

func computerAnswerChoice(match mongomodel.Match, question content.QuizQuestion, player mongomodel.MatchPlayer, accuracy int) string {
	if deterministicPercent("computer-correct", match.ID, question.ID, player.PlayerID) < accuracy {
		return question.CorrectChoice
	}

	wrongChoices := wrongChoiceLabels(question)
	availableWrongChoices := make([]string, 0, len(wrongChoices))
	for _, choice := range wrongChoices {
		if choiceEliminated(match, question.ID, player.PlayerID, choice) {
			continue
		}
		availableWrongChoices = append(availableWrongChoices, choice)
	}
	if len(availableWrongChoices) == 0 {
		return question.CorrectChoice
	}

	sort.Slice(availableWrongChoices, func(i, j int) bool {
		left := deterministicUint64("computer-wrong-choice", match.ID, question.ID, player.PlayerID, availableWrongChoices[i])
		right := deterministicUint64("computer-wrong-choice", match.ID, question.ID, player.PlayerID, availableWrongChoices[j])
		return left < right
	})
	return availableWrongChoices[0]
}

func computerAccuracy(settings gamecontrol.Settings, difficulty string) int {
	switch difficulty {
	case computerDifficultyHard:
		return settings.ComputerHardAccuracy
	case computerDifficultyNormal:
		return settings.ComputerNormalAccuracy
	default:
		return settings.ComputerEasyAccuracy
	}
}

func (h *Handler) computerDifficulty(ctx context.Context, playerID string) (string, error) {
	entries, err := h.computerRankEntries(ctx)
	if err != nil {
		return "", err
	}
	return computerDifficultyForPlayer(entries, playerID), nil
}

func computerDifficultyForPlayer(entries []computerRankEntry, playerID string) string {
	if len(entries) == 0 || playerID == "" {
		return computerDifficultyEasy
	}
	for index, entry := range entries {
		if entry.PlayerID != playerID {
			continue
		}
		percentile := float64(index) / float64(len(entries))
		switch {
		case percentile < 0.20:
			return computerDifficultyHard
		case percentile < 0.60:
			return computerDifficultyNormal
		default:
			return computerDifficultyEasy
		}
	}
	return computerDifficultyEasy
}

func (h *Handler) computerRankEntries(ctx context.Context) ([]computerRankEntry, error) {
	players, err := h.findComputerRankPlayers(ctx)
	if err != nil {
		return nil, err
	}
	sitoneCounts, err := h.computerScoreMap(ctx, mongomodel.PlayerSitonesCollection, computerPlayerSitoneCountsPipeline())
	if err != nil {
		return nil, err
	}
	openPower, err := h.computerScoreMap(ctx, mongomodel.OpenPowerRecordsCollection, computerOpenPowerScoresPipeline())
	if err != nil {
		return nil, err
	}

	entries := make([]computerRankEntry, 0, len(players))
	for _, player := range players {
		if player.ID == "" || player.TeamID == "" || player.Role == authctx.PlayerRoleStaff {
			continue
		}
		entries = append(entries, computerRankEntry{
			PlayerID:    player.ID,
			Nickname:    player.Nickname,
			SitoneCount: sitoneCounts[player.ID],
			OpenPower:   openPower[player.ID],
		})
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].SitoneCount != entries[j].SitoneCount {
			return entries[i].SitoneCount > entries[j].SitoneCount
		}
		if entries[i].OpenPower != entries[j].OpenPower {
			return entries[i].OpenPower > entries[j].OpenPower
		}
		if entries[i].Nickname != entries[j].Nickname {
			return entries[i].Nickname < entries[j].Nickname
		}
		return entries[i].PlayerID < entries[j].PlayerID
	})
	return entries, nil
}

func (h *Handler) findComputerRankPlayers(ctx context.Context) ([]mongomodel.Player, error) {
	cursor, err := h.db.Collection(mongomodel.PlayersCollection).Find(
		ctx,
		bson.M{
			"team_id": bson.M{"$exists": true, "$ne": ""},
			"role":    bson.M{"$ne": authctx.PlayerRoleStaff},
		},
		options.Find().
			SetProjection(bson.D{
				{Key: "auth_token", Value: 0},
				{Key: "qrcode_token", Value: 0},
				{Key: "default_sitone_ids", Value: 0},
			}).
			SetSort(bson.D{
				{Key: "nickname", Value: 1},
				{Key: "_id", Value: 1},
			}),
	)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = cursor.Close(ctx)
	}()

	var players []mongomodel.Player
	if err := cursor.All(ctx, &players); err != nil {
		return nil, err
	}
	if players == nil {
		return []mongomodel.Player{}, nil
	}
	return players, nil
}

func (h *Handler) computerScoreMap(ctx context.Context, collection string, pipeline mongo.Pipeline) (map[string]int, error) {
	cursor, err := h.db.Collection(collection).Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = cursor.Close(ctx)
	}()

	var rows []struct {
		ID    string `bson:"_id"`
		Score int    `bson:"score"`
	}
	if err := cursor.All(ctx, &rows); err != nil {
		return nil, err
	}

	out := make(map[string]int, len(rows))
	for _, row := range rows {
		if row.ID == "" {
			continue
		}
		out[row.ID] = row.Score
	}
	return out, nil
}

func computerPlayerSitoneCountsPipeline() mongo.Pipeline {
	return mongo.Pipeline{
		{{Key: "$match", Value: bson.D{{Key: "quantity", Value: bson.D{{Key: "$gt", Value: 0}}}}}},
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$player_id"},
			{Key: "score", Value: bson.D{{Key: "$sum", Value: "$quantity"}}},
		}}},
	}
}

func computerOpenPowerScoresPipeline() mongo.Pipeline {
	return mongo.Pipeline{
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$player_id"},
			{Key: "score", Value: bson.D{{Key: "$sum", Value: "$amount"}}},
		}}},
	}
}
