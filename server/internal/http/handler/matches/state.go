package matches

import (
	"context"
	"errors"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/sitcon-tw/camp2026-game/internal/http/httpx"
	mongomodel "github.com/sitcon-tw/camp2026-game/internal/mongodb/model"
)

const matchSaveMaxAttempts = 5

var errMatchSaveConflict = errors.New("match save conflict")

func (h *Handler) findMatchByID(ctx context.Context, matchID string) (mongomodel.Match, error) {
	var match mongomodel.Match
	err := h.db.Collection(mongomodel.MatchesCollection).
		FindOne(ctx, bson.M{"_id": matchID}).
		Decode(&match)
	return match, err
}

func (h *Handler) findMatchByCode(ctx context.Context, code string) (mongomodel.Match, error) {
	var match mongomodel.Match
	err := h.db.Collection(mongomodel.MatchesCollection).
		FindOne(ctx, bson.M{
			"code":   code,
			"status": bson.M{"$ne": mongomodel.MatchStatusCompleted},
		}).
		Decode(&match)
	return match, err
}

func (h *Handler) findOpenParticipantMatch(ctx context.Context, playerID string) (mongomodel.Match, error) {
	var match mongomodel.Match
	err := h.db.Collection(mongomodel.MatchesCollection).
		FindOne(
			ctx,
			bson.M{
				"status": bson.M{
					"$in": bson.A{
						mongomodel.MatchStatusWaiting,
						mongomodel.MatchStatusActive,
					},
				},
				"players.player_id": playerID,
			},
			options.FindOne().SetSort(bson.D{
				{Key: "created_at", Value: -1},
				{Key: "_id", Value: -1},
			}),
		).
		Decode(&match)
	return match, err
}

func (h *Handler) saveMatch(ctx context.Context, match *mongomodel.Match) error {
	if match == nil {
		return errors.New("match is nil")
	}

	next := *match
	releaseOpenHostLockOnCompletion(&next)
	next.Revision++
	result, err := h.db.Collection(mongomodel.MatchesCollection).
		ReplaceOne(ctx, matchRevisionFilter(match.ID, match.Revision), next)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return errMatchSaveConflict
	}

	match.Revision = next.Revision
	match.OpenHostLock = next.OpenHostLock
	return nil
}

func releaseOpenHostLockOnCompletion(match *mongomodel.Match) {
	if match != nil && match.Status == mongomodel.MatchStatusCompleted {
		match.OpenHostLock = ""
	}
}

func matchRevisionFilter(matchID string, revision int64) bson.M {
	if revision == 0 {
		return bson.M{
			"_id": matchID,
			"$or": bson.A{
				bson.M{"revision": 0},
				bson.M{"revision": bson.M{"$exists": false}},
			},
		}
	}
	return bson.M{"_id": matchID, "revision": revision}
}

func (h *Handler) findAnswers(ctx context.Context, matchID string) ([]mongomodel.MatchAnswer, error) {
	cursor, err := h.db.Collection(mongomodel.MatchAnswersCollection).Find(
		ctx,
		bson.M{"match_id": matchID},
		options.Find().SetSort(bson.D{
			{Key: "question_id", Value: 1},
			{Key: "answered_at", Value: 1},
		}),
	)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = cursor.Close(ctx)
	}()

	var answers []mongomodel.MatchAnswer
	if err := cursor.All(ctx, &answers); err != nil {
		return nil, err
	}
	return answers, nil
}

func (h *Handler) findAnswer(ctx context.Context, matchID, playerID, questionID string) (mongomodel.MatchAnswer, error) {
	var answer mongomodel.MatchAnswer
	err := h.db.Collection(mongomodel.MatchAnswersCollection).
		FindOne(ctx, bson.M{
			"match_id":    matchID,
			"player_id":   playerID,
			"question_id": questionID,
		}).
		Decode(&answer)
	return answer, err
}

func matchAnswerRecordID(matchID string, playerID string, questionID string) string {
	return "answer_" + matchID + "_" + playerID + "_" + questionID
}

func (h *Handler) applyMatchAnswerScore(ctx context.Context, matchID string, playerID string, score int) error {
	inc := bson.M{"revision": 1}
	if score != 0 {
		inc["players.$.score"] = score
	}

	result, err := h.db.Collection(mongomodel.MatchesCollection).UpdateOne(
		ctx,
		bson.M{"_id": matchID, "players.player_id": playerID},
		bson.M{"$inc": inc},
	)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return errMatchSaveConflict
	}
	return nil
}

func (h *Handler) advanceMatch(ctx context.Context, match mongomodel.Match, now time.Time) (mongomodel.Match, []string, error) {
	events := []string{}
	for attempt := 0; attempt < matchSaveMaxAttempts; attempt++ {
		advanced, attemptEvents, err := h.advanceMatchOnce(ctx, match, now)
		events = append(events, attemptEvents...)
		if errors.Is(err, errMatchSaveConflict) {
			fresh, findErr := h.findMatchByID(ctx, match.ID)
			if findErr != nil {
				return advanced, events, findErr
			}
			match = fresh
			continue
		}
		return advanced, events, err
	}
	return match, events, errMatchSaveConflict
}

func (h *Handler) advanceMatchOnce(ctx context.Context, match mongomodel.Match, now time.Time) (mongomodel.Match, []string, error) {
	if match.Status == mongomodel.MatchStatusCompleted {
		return match, nil, h.writeMatchRewards(ctx, match)
	}
	if match.Status != mongomodel.MatchStatusActive {
		return match, nil, nil
	}

	var events []string
	for match.Status == mongomodel.MatchStatusActive {
		switch activeMatchPhase(match) {
		case mongomodel.MatchPhaseAnswering:
			computerAnswered, err := h.ensureComputerAnswer(ctx, &match, now)
			if err != nil {
				return match, events, err
			}
			if computerAnswered {
				events = append(events, "player_answered")
			}

			shouldReveal, err := h.shouldRevealRound(ctx, match, now)
			if err != nil {
				return match, events, err
			}
			if !shouldReveal {
				return match, events, nil
			}

			match.Phase = mongomodel.MatchPhaseRevealing
			match.RevealEndsAt = now.Add(revealDuration * time.Second)
			if err := h.saveMatch(ctx, &match); err != nil {
				return match, events, err
			}
			events = append(events, "round_revealed")

		case mongomodel.MatchPhaseRevealing:
			revealEndsAt := match.RevealEndsAt
			if revealEndsAt.IsZero() {
				revealEndsAt = now
			}
			if now.Before(revealEndsAt) {
				return match, events, nil
			}

			if match.CurrentQuestionIndex >= len(match.QuestionIDs)-1 {
				match.Status = mongomodel.MatchStatusCompleted
				match.Phase = ""
				match.CompletedAt = revealEndsAt
				if err := h.saveMatch(ctx, &match); err != nil {
					return match, events, err
				}
				if err := h.writeMatchRewards(ctx, match); err != nil {
					return match, events, err
				}
				events = append(events, "match_completed")
				return match, events, nil
			}

			match.CurrentQuestionIndex++
			match.Phase = mongomodel.MatchPhaseAnswering
			match.RoundStartedAt = revealEndsAt
			match.RoundEndsAt = revealEndsAt.Add(roundDuration * time.Second)
			match.RevealEndsAt = time.Time{}
			if err := h.ensureCurrentRoundEliminations(ctx, &match); err != nil {
				return match, events, err
			}
			if err := h.saveMatch(ctx, &match); err != nil {
				return match, events, err
			}
			events = append(events, "round_started")
		}
	}

	return match, events, nil
}

func (h *Handler) shouldRevealRound(ctx context.Context, match mongomodel.Match, now time.Time) (bool, error) {
	if len(match.QuestionIDs) == 0 || match.CurrentQuestionIndex >= len(match.QuestionIDs) {
		return false, nil
	}
	if !match.RoundEndsAt.IsZero() && !now.Before(match.RoundEndsAt) {
		return true, nil
	}

	questionID := match.QuestionIDs[match.CurrentQuestionIndex]
	answers, err := h.findAnswers(ctx, match.ID)
	if err != nil {
		return false, err
	}

	answered := make(map[string]struct{}, len(match.Players))
	for _, answer := range answers {
		if answer.QuestionID == questionID {
			answered[answer.PlayerID] = struct{}{}
		}
	}
	for _, player := range match.Players {
		if _, ok := answered[player.PlayerID]; !ok {
			return false, nil
		}
	}
	return len(match.Players) == 2, nil
}

func (h *Handler) findMatchOpenPowerRewards(ctx context.Context, matchID string) ([]mongomodel.OpenPowerRecord, error) {
	cursor, err := h.db.Collection(mongomodel.OpenPowerRecordsCollection).Find(
		ctx,
		bson.M{
			"reason": "quiz_match_completed",
			"source": bson.M{"$regex": "^quiz_match:" + matchID + ":player:"},
		},
		options.Find().SetSort(bson.D{{Key: "player_id", Value: 1}}),
	)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = cursor.Close(ctx)
	}()

	var records []mongomodel.OpenPowerRecord
	if err := cursor.All(ctx, &records); err != nil {
		return nil, err
	}
	return records, nil
}

func (h *Handler) writeMatchRewards(ctx context.Context, match mongomodel.Match) error {
	if h.client == nil {
		return h.writeMatchRewardsWithoutTransaction(ctx, match)
	}

	if err := h.writeMatchRewardsWithTransaction(ctx, match); err != nil {
		if transactionUnsupported(err) {
			return h.writeMatchRewardsWithoutTransaction(ctx, match)
		}
		return err
	}
	return nil
}

func (h *Handler) writeMatchRewardsWithTransaction(ctx context.Context, match mongomodel.Match) error {
	session, err := h.client.StartSession()
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

		if err := h.writeMatchRewardsWithoutTransaction(ctx, match); err != nil {
			return err
		}
		if err := session.CommitTransaction(ctx); err != nil {
			return err
		}
		committed = true
		return nil
	})
}

func (h *Handler) writeMatchRewardsWithoutTransaction(ctx context.Context, match mongomodel.Match) error {
	now := match.CompletedAt
	if now.IsZero() {
		now = time.Now()
	}

	for _, player := range match.Players {
		if isComputerPlayer(player) {
			continue
		}
		effects, err := h.matchPlayerBattleEffects(ctx, player)
		if err != nil {
			return err
		}
		if err := h.writeMatchItemDrop(ctx, match, player, effects, now); err != nil {
			return err
		}
		reward := matchOpenPowerReward(match, player, effects)
		if reward <= 0 {
			continue
		}
		record := mongomodel.OpenPowerRecord{
			ID:        matchRewardRecordID(match.ID, player.PlayerID),
			PlayerID:  player.PlayerID,
			Amount:    reward,
			Reason:    "quiz_match_completed",
			Source:    matchRewardSource(match.ID, player.PlayerID),
			CreatedAt: now,
		}
		_, err = h.db.Collection(mongomodel.OpenPowerRecordsCollection).UpdateOne(
			ctx,
			bson.M{"_id": record.ID},
			bson.M{"$setOnInsert": record},
			options.UpdateOne().SetUpsert(true),
		)
		if err != nil {
			return err
		}
	}
	return nil
}

func transactionUnsupported(err error) bool {
	var commandError mongo.CommandError
	return errors.As(err, &commandError) &&
		commandError.HasErrorCodeWithMessage(20, "Transaction numbers")
}

func (h *Handler) buildMatchState(ctx context.Context, match mongomodel.Match, viewerPlayerID string) (MatchStateResponse, error) {
	answers, err := h.findAnswers(ctx, match.ID)
	if err != nil {
		return MatchStateResponse{}, err
	}

	answerByQuestionPlayer := make(map[string]map[string]mongomodel.MatchAnswer)
	for _, answer := range answers {
		if _, ok := answerByQuestionPlayer[answer.QuestionID]; !ok {
			answerByQuestionPlayer[answer.QuestionID] = make(map[string]mongomodel.MatchAnswer)
		}
		answerByQuestionPlayer[answer.QuestionID][answer.PlayerID] = answer
	}
	eliminatedByQuestionPlayer := eliminatedChoicesByQuestionPlayer(match)
	dropByPlayer := make(map[string]mongomodel.MatchItemDrop)
	rewardByPlayer := make(map[string]int)
	if match.Status == mongomodel.MatchStatusCompleted {
		drops, err := h.findMatchItemDrops(ctx, match.ID)
		if err != nil {
			return MatchStateResponse{}, err
		}
		for _, drop := range drops {
			dropByPlayer[drop.PlayerID] = drop
		}
		rewards, err := h.findMatchOpenPowerRewards(ctx, match.ID)
		if err != nil {
			return MatchStateResponse{}, err
		}
		for _, reward := range rewards {
			rewardByPlayer[reward.PlayerID] = reward.Amount
		}
	}

	response := MatchStateResponse{
		MatchID:              match.ID,
		Code:                 match.Code,
		Mode:                 matchMode(match),
		Status:               match.Status,
		HostPlayerID:         match.HostPlayerID,
		Players:              make([]MatchPlayerResponse, 0, len(match.Players)),
		CurrentQuestionIndex: match.CurrentQuestionIndex,
		QuestionCount:        len(match.QuestionIDs),
		CreatedAt:            match.CreatedAt,
		StartedAt:            timePtr(match.StartedAt),
		CompletedAt:          timePtr(match.CompletedAt),
	}

	if match.Status == mongomodel.MatchStatusActive {
		phase := activeMatchPhase(match)
		response.Phase = phase
		response.RoundStartedAt = timePtr(match.RoundStartedAt)
		response.RoundEndsAt = timePtr(match.RoundEndsAt)
		response.RevealEndsAt = timePtr(match.RevealEndsAt)
		if match.CurrentQuestionIndex < len(match.QuestionIDs) {
			currentQuestionID := match.QuestionIDs[match.CurrentQuestionIndex]
			question, ok := h.content.GetQuizQuestion(currentQuestionID)
			if !ok {
				return MatchStateResponse{}, httpx.InternalServerError("match question is unavailable", "match_state_current_question_missing", errors.New("current quiz question not found in content store"))
			}
			currentQuestion := questionResponse(question)
			response.CurrentQuestion = &currentQuestion
			if phase == mongomodel.MatchPhaseRevealing {
				result, err := h.matchQuestionResult(match, currentQuestionID, answerByQuestionPlayer)
				if err != nil {
					return MatchStateResponse{}, err
				}
				response.CurrentQuestionResult = &result
			}
		}
	}

	currentQuestionID := ""
	if match.Status == mongomodel.MatchStatusActive && match.CurrentQuestionIndex < len(match.QuestionIDs) {
		currentQuestionID = match.QuestionIDs[match.CurrentQuestionIndex]
	}
	currentAnswers := answerByQuestionPlayer[currentQuestionID]
	currentEliminations := eliminatedByQuestionPlayer[currentQuestionID]
	for _, player := range match.Players {
		effects, err := h.matchPlayerBattleEffects(ctx, player)
		if err != nil {
			return MatchStateResponse{}, err
		}
		score := player.Score
		maxScore := maxScoreThroughCurrentQuestion(match, effects)
		playerResponse := MatchPlayerResponse{
			PlayerID:                 player.PlayerID,
			Nickname:                 player.Nickname,
			Kind:                     matchPlayerKind(player),
			Ready:                    player.Ready,
			SitoneIDs:                cloneStrings(player.SitoneIDs),
			Score:                    &score,
			MaxScore:                 &maxScore,
			AnswerScoreBonusPercent:  effects.AnswerScoreBonusPercent,
			OpenPowerBonusPercent:    effects.OpenPowerBonusPercent,
			MaterialDropBonusPercent: effects.MaterialDropBonusPercent,
			EliminateChancePercent:   effects.EliminateChancePercent,
			EliminateCount:           effects.EliminateCount,
		}
		if match.Status == mongomodel.MatchStatusActive {
			_, playerResponse.AnsweredCurrentQuestion = currentAnswers[player.PlayerID]
			playerResponse.EliminatedChoices, playerResponse.EliminatedBy = viewerEliminatedChoices(
				player.PlayerID,
				viewerPlayerID,
				currentEliminations,
			)
		}
		if match.Status == mongomodel.MatchStatusCompleted {
			baseReward := matchBaseOpenPowerReward(match, player)
			reward := rewardByPlayer[player.PlayerID]
			playerResponse.BaseOpenPowerReward = &baseReward
			playerResponse.OpenPowerReward = &reward
			if drop, ok := dropByPlayer[player.PlayerID]; ok {
				playerResponse.MaterialDrop = h.matchItemDropResponse(drop)
			}
		}
		response.Players = append(response.Players, playerResponse)
	}

	if match.Status == mongomodel.MatchStatusCompleted {
		results, err := h.matchResults(match, answerByQuestionPlayer)
		if err != nil {
			return MatchStateResponse{}, err
		}
		response.Results = results
	}

	return response, nil
}

func (h *Handler) matchResults(
	match mongomodel.Match,
	answerByQuestionPlayer map[string]map[string]mongomodel.MatchAnswer,
) ([]MatchQuestionResult, error) {
	results := make([]MatchQuestionResult, 0, len(match.QuestionIDs))
	for _, questionID := range match.QuestionIDs {
		result, err := h.matchQuestionResult(match, questionID, answerByQuestionPlayer)
		if err != nil {
			return nil, err
		}
		results = append(results, result)
	}
	return results, nil
}

func (h *Handler) matchQuestionResult(
	match mongomodel.Match,
	questionID string,
	answerByQuestionPlayer map[string]map[string]mongomodel.MatchAnswer,
) (MatchQuestionResult, error) {
	question, ok := h.content.GetQuizQuestion(questionID)
	if !ok {
		return MatchQuestionResult{}, httpx.InternalServerError("match question is unavailable", "match_results_question_missing", errors.New("result quiz question not found in content store"))
	}

	result := MatchQuestionResult{
		QuestionID:    question.ID,
		Prompt:        question.Prompt,
		ChoiceA:       question.ChoiceA,
		ChoiceB:       question.ChoiceB,
		ChoiceC:       question.ChoiceC,
		ChoiceD:       question.ChoiceD,
		CorrectChoice: question.CorrectChoice,
		Explanation:   question.Explanation,
		Answers:       make([]MatchAnswerResponse, 0, len(match.Players)),
	}
	for _, player := range match.Players {
		answer, ok := answerByQuestionPlayer[questionID][player.PlayerID]
		answerResponse := MatchAnswerResponse{
			PlayerID: player.PlayerID,
			Nickname: player.Nickname,
		}
		if ok {
			answeredAt := answer.AnsweredAt
			baseScore := answer.BaseScore
			if baseScore == 0 {
				baseScore = answer.Score - answer.BonusScore
			}
			answerResponse.Choice = answer.Choice
			answerResponse.Correct = answer.Correct
			answerResponse.BaseScore = baseScore
			answerResponse.BonusScore = answer.Score - baseScore
			answerResponse.Score = answer.Score
			answerResponse.ElapsedMillis = answer.ElapsedMillis
			answerResponse.AnsweredAt = &answeredAt
		}
		result.Answers = append(result.Answers, answerResponse)
	}
	return result, nil
}

func (h *Handler) publishState(_ context.Context, match mongomodel.Match, eventName string) {
	if h.broker == nil {
		return
	}
	h.broker.Publish(match.ID, Event{
		Name:  eventName,
		Match: match,
	})
}

func (h *Handler) writeAdvancedMatchState(w http.ResponseWriter, r *http.Request, match mongomodel.Match, playerID string) {
	match, events, err := h.advanceMatch(r.Context(), match, time.Now())
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("match state unavailable", "match_state_advance_failed", err))
		return
	}
	for _, event := range events {
		h.publishState(r.Context(), match, event)
	}

	state, err := h.buildMatchState(r.Context(), match, playerID)
	if err != nil {
		httpx.WriteProblem(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, state)
}

func writeMatchProblem(w http.ResponseWriter, r *http.Request, err error) {
	if errors.Is(err, mongo.ErrNoDocuments) {
		httpx.WriteProblem(w, r, httpx.NotFound("match not found"))
		return
	}
	if errors.Is(err, errMatchSaveConflict) {
		httpx.WriteProblem(w, r, httpx.NewError(http.StatusConflict, "match was updated; retry"))
		return
	}
	httpx.WriteProblem(w, r, err)
}
