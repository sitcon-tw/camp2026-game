package matches

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/sitcon-tw/camp2026-game/internal/content"
	"github.com/sitcon-tw/camp2026-game/internal/gamecontrol"
	"github.com/sitcon-tw/camp2026-game/internal/http/httpx"
	mongomodel "github.com/sitcon-tw/camp2026-game/internal/mongodb/model"
)

const (
	matchSessionTickInterval = time.Second
	matchSessionDBTimeout    = 5 * time.Second
)

type MatchSessionManager struct {
	h *Handler

	mu       sync.Mutex
	sessions map[string]*MatchSession
}

func NewMatchSessionManager(h *Handler) *MatchSessionManager {
	return &MatchSessionManager{
		h:        h,
		sessions: make(map[string]*MatchSession),
	}
}

func (m *MatchSessionManager) RecoverOpenMatches() {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), matchSessionDBTimeout)
		defer cancel()

		cursor, err := m.h.db.Collection(mongomodel.MatchesCollection).Find(
			ctx,
			bson.M{"status": openMatchStatusFilter()},
		)
		if err != nil {
			return
		}
		defer func() {
			_ = cursor.Close(context.Background())
		}()

		for cursor.Next(ctx) {
			var match mongomodel.Match
			if err := cursor.Decode(&match); err != nil {
				continue
			}
			answers, err := m.h.findAnswers(ctx, match.ID)
			if err != nil {
				continue
			}

			m.mu.Lock()
			if _, ok := m.sessions[match.ID]; !ok {
				session := newMatchSession(m, match, answers)
				m.sessions[match.ID] = session
				session.start()
			}
			m.mu.Unlock()
		}
	}()
}

func (m *MatchSessionManager) Start(match mongomodel.Match) *MatchSession {
	m.mu.Lock()
	defer m.mu.Unlock()

	if session, ok := m.sessions[match.ID]; ok {
		return session
	}
	session := newMatchSession(m, match, nil)
	m.sessions[match.ID] = session
	session.start()
	return session
}

func (m *MatchSessionManager) GetOrLoad(ctx context.Context, matchID string) (*MatchSession, error) {
	m.mu.Lock()
	if session, ok := m.sessions[matchID]; ok {
		m.mu.Unlock()
		return session, nil
	}
	m.mu.Unlock()

	match, err := m.h.findMatchByID(ctx, matchID)
	if err != nil {
		return nil, err
	}
	if !matchIsOpen(match) {
		return nil, httpx.NewError(http.StatusConflict, "match is not open")
	}
	answers, err := m.h.findAnswers(ctx, match.ID)
	if err != nil {
		return nil, err
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	if session, ok := m.sessions[matchID]; ok {
		return session, nil
	}
	session := newMatchSession(m, match, answers)
	m.sessions[match.ID] = session
	session.start()
	return session, nil
}

func (m *MatchSessionManager) Remove(matchID string, session *MatchSession) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if current, ok := m.sessions[matchID]; ok && current == session {
		delete(m.sessions, matchID)
	}
}

type MatchSession struct {
	manager *MatchSessionManager
	h       *Handler

	done     chan struct{}
	stopOnce sync.Once

	mu      sync.Mutex
	match   mongomodel.Match
	answers []mongomodel.MatchAnswer
}

func newMatchSession(manager *MatchSessionManager, match mongomodel.Match, answers []mongomodel.MatchAnswer) *MatchSession {
	return &MatchSession{
		manager: manager,
		h:       manager.h,
		done:    make(chan struct{}),
		match:   match,
		answers: cloneMatchAnswers(answers),
	}
}

func (s *MatchSession) start() {
	go s.run()
}

func (s *MatchSession) stop() {
	s.stopOnce.Do(func() {
		s.mu.Lock()
		matchID := s.match.ID
		s.mu.Unlock()

		close(s.done)
		s.manager.Remove(matchID, s)
	})
}

func (s *MatchSession) run() {
	ticker := time.NewTicker(matchSessionTickInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.done:
			return
		case now := <-ticker.C:
			s.tick(now)
		}
	}
}

func (s *MatchSession) tick(now time.Time) {
	ctx, cancel := context.WithTimeout(context.Background(), matchSessionDBTimeout)
	defer cancel()

	events, shouldBroadcast, shouldStop, err := s.advanceAndSnapshot(ctx, now, true)
	if err != nil {
		return
	}
	if len(events) == 0 && shouldBroadcast {
		events = []Event{s.eventSnapshot("match_updated")}
	}
	for _, event := range events {
		s.h.publishEvent(ctx, event)
	}
	if shouldStop {
		s.stop()
	}
}

func (s *MatchSession) State(ctx context.Context, viewerPlayerID string) (MatchStateResponse, error) {
	match, answers := s.snapshot()
	return s.h.buildMatchStateWithAnswers(ctx, match, viewerPlayerID, answers)
}

func (s *MatchSession) Join(ctx context.Context, player mongomodel.Player) (MatchStateResponse, error) {
	var events []Event
	var match mongomodel.Match
	var answers []mongomodel.MatchAnswer

	s.mu.Lock()
	if isParticipant(s.match, player.ID) {
		match, answers = s.snapshotLocked()
		s.mu.Unlock()
		return s.h.buildMatchStateWithAnswers(ctx, match, player.ID, answers)
	}
	if err := s.h.ensureNoOpenParticipantMatch(ctx, player.ID); err != nil {
		s.mu.Unlock()
		return MatchStateResponse{}, err
	}
	if s.match.Status != mongomodel.MatchStatusWaiting {
		s.mu.Unlock()
		return MatchStateResponse{}, httpx.NewError(http.StatusConflict, "match is not joinable")
	}
	if len(s.match.Players) >= 2 {
		s.mu.Unlock()
		return MatchStateResponse{}, httpx.NewError(http.StatusConflict, "match is full")
	}
	if err := s.h.ensureSameTeamBattleAllowed(ctx, s.match, player); err != nil {
		s.mu.Unlock()
		return MatchStateResponse{}, err
	}

	sitoneIDs, err := s.h.defaultSitoneLoadout(ctx, player)
	if err != nil {
		s.mu.Unlock()
		return MatchStateResponse{}, httpx.InternalServerError("match join failed", "match_join_default_loadout_failed", err)
	}

	s.match.Players = append(s.match.Players, mongomodel.MatchPlayer{
		PlayerID:  player.ID,
		Nickname:  player.Nickname,
		Kind:      mongomodel.MatchPlayerKindHuman,
		Ready:     false,
		Score:     0,
		SitoneIDs: sitoneIDs,
	})
	s.match.OpenPlayerLocks = humanParticipantIDs(s.match)
	if err := s.h.saveMatch(ctx, &s.match); err != nil {
		s.mu.Unlock()
		return MatchStateResponse{}, err
	}
	events = append(events, s.eventSnapshotLocked("match_updated"))
	match, answers = s.snapshotLocked()
	s.mu.Unlock()

	for _, event := range events {
		s.h.publishEvent(ctx, event)
	}
	return s.h.buildMatchStateWithAnswers(ctx, match, player.ID, answers)
}

func (h *Handler) ensureSameTeamBattleAllowed(ctx context.Context, match mongomodel.Match, player mongomodel.Player) error {
	if !shouldCheckSameTeamBattle(match, player) {
		return nil
	}

	settings, err := gamecontrol.ReadSettings(ctx, h.db)
	if err != nil {
		return httpx.InternalServerError("match join failed", "match_join_settings_lookup_failed", err)
	}
	if settings.SameTeamBattlesEnabled() {
		return nil
	}

	sameTeamOpponent, err := h.matchHasSameTeamOpponent(ctx, match, player)
	if err != nil {
		return httpx.InternalServerError("match join failed", "match_join_team_lookup_failed", err)
	}
	if sameTeamOpponent {
		return httpx.NewError(http.StatusForbidden, "same-team battles are disabled")
	}
	return nil
}

func (h *Handler) ensureMatchSameTeamBattleAllowed(ctx context.Context, match mongomodel.Match) error {
	if !shouldCheckSameTeamMatch(match) {
		return nil
	}

	settings, err := gamecontrol.ReadSettings(ctx, h.db)
	if err != nil {
		return httpx.InternalServerError("match start failed", "match_start_settings_lookup_failed", err)
	}
	if settings.SameTeamBattlesEnabled() {
		return nil
	}

	sameTeamParticipants, err := h.matchHasSameTeamParticipants(ctx, humanParticipantIDs(match))
	if err != nil {
		return httpx.InternalServerError("match start failed", "match_start_team_lookup_failed", err)
	}
	if sameTeamParticipants {
		return httpx.NewError(http.StatusForbidden, "same-team battles are disabled")
	}
	return nil
}

func shouldCheckSameTeamMatch(match mongomodel.Match) bool {
	return matchMode(match) == mongomodel.MatchModePVP && len(humanParticipantIDs(match)) > 1
}

func (h *Handler) matchHasSameTeamParticipants(ctx context.Context, playerIDs []string) (bool, error) {
	cursor, err := h.db.Collection(mongomodel.PlayersCollection).Find(ctx, bson.M{
		"_id":     bson.M{"$in": playerIDs},
		"team_id": bson.M{"$exists": true, "$ne": ""},
	})
	if err != nil {
		return false, err
	}
	defer func() {
		_ = cursor.Close(ctx)
	}()

	var players []struct {
		TeamID string `bson:"team_id"`
	}
	if err := cursor.All(ctx, &players); err != nil {
		return false, err
	}
	seen := make(map[string]struct{}, len(players))
	for _, player := range players {
		if _, ok := seen[player.TeamID]; ok {
			return true, nil
		}
		seen[player.TeamID] = struct{}{}
	}
	return false, nil
}

func shouldCheckSameTeamBattle(match mongomodel.Match, player mongomodel.Player) bool {
	return matchMode(match) == mongomodel.MatchModePVP &&
		player.TeamID != "" &&
		!isParticipant(match, player.ID) &&
		len(humanOpponentIDs(match, player.ID)) > 0
}

func (h *Handler) matchHasSameTeamOpponent(ctx context.Context, match mongomodel.Match, player mongomodel.Player) (bool, error) {
	err := h.db.Collection(mongomodel.PlayersCollection).
		FindOne(ctx, bson.M{
			"_id":     bson.M{"$in": humanOpponentIDs(match, player.ID)},
			"team_id": player.TeamID,
		}).
		Err()
	if errors.Is(err, mongo.ErrNoDocuments) {
		return false, nil
	}
	return err == nil, err
}

func humanOpponentIDs(match mongomodel.Match, playerID string) []string {
	ids := humanParticipantIDs(match)
	opponents := ids[:0]
	for _, id := range ids {
		if id != playerID {
			opponents = append(opponents, id)
		}
	}
	return opponents
}

func (s *MatchSession) Ready(ctx context.Context, player mongomodel.Player) (MatchStateResponse, error) {
	var events []Event
	var match mongomodel.Match
	var answers []mongomodel.MatchAnswer

	s.mu.Lock()
	if !isParticipant(s.match, player.ID) {
		s.mu.Unlock()
		return MatchStateResponse{}, httpx.NotFound("match not found")
	}
	if s.match.Status != mongomodel.MatchStatusWaiting {
		s.mu.Unlock()
		return MatchStateResponse{}, httpx.NewError(http.StatusConflict, "match is not waiting for ready")
	}

	idx := playerIndex(s.match, player.ID)
	if len(s.match.Players[idx].SitoneIDs) == 0 {
		sitoneIDs, err := s.h.defaultSitoneLoadout(ctx, player)
		if err != nil {
			s.mu.Unlock()
			return MatchStateResponse{}, httpx.InternalServerError("ready failed", "match_ready_default_loadout_failed", err)
		}
		s.match.Players[idx].SitoneIDs = sitoneIDs
	}
	if len(s.match.Players[idx].SitoneIDs) == 0 {
		s.mu.Unlock()
		return MatchStateResponse{}, httpx.NewError(http.StatusConflict, "select at least one sitone before ready")
	}
	wasReady := s.match.Players[idx].Ready
	s.match.Players[idx].Ready = true
	events = append(events, s.eventSnapshotLocked("player_ready"))

	if allPlayersReady(s.match) {
		if err := s.h.ensureMatchSameTeamBattleAllowed(ctx, s.match); err != nil {
			s.match.Players[idx].Ready = wasReady
			s.mu.Unlock()
			return MatchStateResponse{}, err
		}

		questionIDs, err := s.h.pickQuestionIDs()
		if err != nil {
			s.mu.Unlock()
			return MatchStateResponse{}, httpx.InternalServerError("match start failed", "match_ready_pick_questions_failed", err)
		}
		now := time.Now()
		s.match.Status = mongomodel.MatchStatusActive
		s.match.Phase = mongomodel.MatchPhaseAnswering
		s.match.QuestionIDs = questionIDs
		s.match.CurrentQuestionIndex = 0
		s.match.StartedAt = now
		s.match.RoundStartedAt = now
		s.match.RoundEndsAt = now.Add(roundDuration * time.Second)
		if err := s.h.snapshotMatchBattleEffects(ctx, &s.match); err != nil {
			s.mu.Unlock()
			return MatchStateResponse{}, httpx.InternalServerError("match start failed", "match_ready_effects_failed", err)
		}
		if err := s.h.ensureCurrentRoundEliminations(ctx, &s.match); err != nil {
			s.mu.Unlock()
			return MatchStateResponse{}, httpx.InternalServerError("match start failed", "match_ready_eliminations_failed", err)
		}
		events = append(events, s.eventSnapshotLocked("round_started"))
	}

	if err := s.h.saveMatch(ctx, &s.match); err != nil {
		s.mu.Unlock()
		return MatchStateResponse{}, err
	}
	events[len(events)-1] = s.eventSnapshotLocked(events[len(events)-1].Name)
	match, answers = s.snapshotLocked()
	s.mu.Unlock()

	for _, event := range events {
		s.h.publishEvent(ctx, event)
	}
	return s.h.buildMatchStateWithAnswers(ctx, match, player.ID, answers)
}

func (s *MatchSession) UpdateLoadout(ctx context.Context, playerID string, sitoneIDs []string) (MatchStateResponse, error) {
	var event Event
	var match mongomodel.Match
	var answers []mongomodel.MatchAnswer

	s.mu.Lock()
	if !isParticipant(s.match, playerID) {
		s.mu.Unlock()
		return MatchStateResponse{}, httpx.NotFound("match not found")
	}
	if s.match.Status != mongomodel.MatchStatusWaiting {
		s.mu.Unlock()
		return MatchStateResponse{}, httpx.NewError(http.StatusConflict, "match loadout is locked")
	}
	idx := playerIndex(s.match, playerID)
	if s.match.Players[idx].Ready {
		s.mu.Unlock()
		return MatchStateResponse{}, httpx.NewError(http.StatusConflict, "player is already ready")
	}
	s.match.Players[idx].SitoneIDs = cloneStrings(sitoneIDs)
	if err := s.h.saveMatch(ctx, &s.match); err != nil {
		s.mu.Unlock()
		return MatchStateResponse{}, err
	}
	event = s.eventSnapshotLocked("match_updated")
	match, answers = s.snapshotLocked()
	s.mu.Unlock()

	s.h.publishEvent(ctx, event)
	return s.h.buildMatchStateWithAnswers(ctx, match, playerID, answers)
}

func (s *MatchSession) Leave(ctx context.Context, playerID string) (bool, error) {
	var event Event
	var shouldStop bool

	s.mu.Lock()
	if !isParticipant(s.match, playerID) {
		s.mu.Unlock()
		return false, httpx.NotFound("match not found")
	}
	if s.match.Status != mongomodel.MatchStatusWaiting {
		s.mu.Unlock()
		return false, httpx.NewError(http.StatusConflict, "match has already started")
	}

	if s.match.HostPlayerID == playerID {
		result, err := s.h.db.Collection(mongomodel.MatchesCollection).DeleteOne(
			ctx,
			bson.M{
				"_id":            s.match.ID,
				"status":         mongomodel.MatchStatusWaiting,
				"host_player_id": playerID,
			},
		)
		if err != nil {
			s.mu.Unlock()
			return false, httpx.InternalServerError("leave match failed", "match_leave_delete_failed", err)
		}
		if result.DeletedCount == 0 {
			s.mu.Unlock()
			return false, errMatchSaveConflict
		}
		event = s.eventSnapshotLocked("match_deleted")
		shouldStop = true
		s.mu.Unlock()
		s.h.publishEvent(ctx, event)
		if shouldStop {
			s.stop()
		}
		return true, nil
	}

	idx := playerIndex(s.match, playerID)
	s.match.Players = append(s.match.Players[:idx], s.match.Players[idx+1:]...)
	if err := s.h.saveMatch(ctx, &s.match); err != nil {
		s.mu.Unlock()
		return false, err
	}
	event = s.eventSnapshotLocked("match_updated")
	s.mu.Unlock()

	s.h.publishEvent(ctx, event)
	return false, nil
}

func (s *MatchSession) Answer(ctx context.Context, playerID string, questionID string, choice string) error {
	now := time.Now()
	var events []Event
	var shouldStop bool

	s.mu.Lock()
	advanceEvents, _, stopAfterAdvance, err := s.advanceLocked(ctx, now)
	if err != nil {
		s.mu.Unlock()
		return err
	}
	events = append(events, advanceEvents...)
	shouldStop = shouldStop || stopAfterAdvance

	finishWithError := func(err error) error {
		s.mu.Unlock()
		for _, event := range events {
			s.h.publishEvent(ctx, event)
		}
		if shouldStop {
			s.stop()
		}
		return err
	}

	if s.match.Status != mongomodel.MatchStatusActive {
		return finishWithError(httpx.NewError(http.StatusConflict, "match is not active"))
	}
	if activeMatchPhase(s.match) != mongomodel.MatchPhaseAnswering {
		return finishWithError(httpx.NewError(http.StatusConflict, "round is not accepting answers"))
	}
	if s.match.CurrentQuestionIndex >= len(s.match.QuestionIDs) || s.match.QuestionIDs[s.match.CurrentQuestionIndex] != questionID {
		return finishWithError(httpx.NewError(http.StatusConflict, "question is not current"))
	}
	if !isParticipant(s.match, playerID) {
		return finishWithError(httpx.NotFound("match not found"))
	}
	if hasAnswer(s.answers, playerID, questionID) {
		return finishWithError(httpx.NewError(http.StatusConflict, "question already answered"))
	}

	question, ok := s.h.content.GetQuizQuestion(questionID)
	if !ok {
		return finishWithError(httpx.InternalServerError("match question is unavailable", "match_answer_question_missing", errors.New("quiz question not found in content store")))
	}
	if choiceEliminated(s.match, questionID, playerID, choice) {
		return finishWithError(httpx.NewError(http.StatusUnprocessableEntity, "choice has been eliminated"))
	}
	idx := playerIndex(s.match, playerID)
	effects, err := s.h.matchPlayerBattleEffects(ctx, s.match.Players[idx])
	if err != nil {
		return finishWithError(httpx.InternalServerError("answer failed", "match_answer_effects_failed", err))
	}
	answer := buildMatchAnswer(s.match, question, s.match.Players[idx], questionID, choice, now, effects)
	if err := s.persistAnswerLocked(ctx, idx, answer); err != nil {
		return finishWithError(err)
	}
	events = append(events, s.eventSnapshotLocked("player_answered"))

	advanceEvents, _, stopAfterAdvance, err = s.advanceLocked(ctx, now)
	if err != nil {
		s.mu.Unlock()
		return err
	}
	events = append(events, advanceEvents...)
	shouldStop = shouldStop || stopAfterAdvance
	s.mu.Unlock()

	for _, event := range events {
		s.h.publishEvent(ctx, event)
	}
	if shouldStop {
		s.stop()
	}
	return nil
}

func (s *MatchSession) advanceAndSnapshot(ctx context.Context, now time.Time, periodic bool) ([]Event, bool, bool, error) {
	s.mu.Lock()
	events, shouldBroadcast, shouldStop, err := s.advanceLocked(ctx, now)
	if periodic && len(events) == 0 && s.match.Status == mongomodel.MatchStatusActive {
		shouldBroadcast = true
	}
	if periodic && len(events) == 0 && s.match.Status == mongomodel.MatchStatusWaiting {
		shouldBroadcast = true
	}
	s.mu.Unlock()
	return events, shouldBroadcast, shouldStop, err
}

func (s *MatchSession) advanceLocked(ctx context.Context, now time.Time) ([]Event, bool, bool, error) {
	if s.match.Status == mongomodel.MatchStatusCompleted {
		return nil, false, true, nil
	}
	if s.match.Status != mongomodel.MatchStatusActive {
		return nil, false, false, nil
	}

	var events []Event
	shouldStop := false
	for s.match.Status == mongomodel.MatchStatusActive {
		switch activeMatchPhase(s.match) {
		case mongomodel.MatchPhaseAnswering:
			computerAnswered, err := s.ensureComputerAnswerLocked(ctx, now)
			if err != nil {
				return events, false, shouldStop, err
			}
			if computerAnswered {
				events = append(events, s.eventSnapshotLocked("player_answered"))
			}
			if !shouldRevealRoundWithAnswers(s.match, s.answers, now) {
				return events, false, shouldStop, nil
			}

			s.match.Phase = mongomodel.MatchPhaseRevealing
			s.match.RevealEndsAt = now.Add(revealDuration * time.Second)
			if err := s.h.saveMatch(ctx, &s.match); err != nil {
				return events, false, shouldStop, err
			}
			events = append(events, s.eventSnapshotLocked("round_revealed"))

		case mongomodel.MatchPhaseRevealing:
			revealEndsAt := s.match.RevealEndsAt
			if revealEndsAt.IsZero() {
				revealEndsAt = now
			}
			if now.Before(revealEndsAt) {
				return events, false, shouldStop, nil
			}

			eventName := applyRevealDeadlineTransition(&s.match, revealEndsAt)
			if eventName == "match_completed" {
				if err := s.h.saveMatch(ctx, &s.match); err != nil {
					return events, false, shouldStop, err
				}
				if err := s.h.writeMatchRewards(ctx, s.match); err != nil {
					return events, false, shouldStop, err
				}
				events = append(events, s.eventSnapshotLocked(eventName))
				shouldStop = true
				return events, false, shouldStop, nil
			}

			if err := s.h.ensureCurrentRoundEliminations(ctx, &s.match); err != nil {
				return events, false, shouldStop, err
			}
			if err := s.h.saveMatch(ctx, &s.match); err != nil {
				return events, false, shouldStop, err
			}
			events = append(events, s.eventSnapshotLocked(eventName))
		}
	}
	return events, false, shouldStop, nil
}

func (s *MatchSession) ensureComputerAnswerLocked(ctx context.Context, now time.Time) (bool, error) {
	if !isComputerMatch(s.match) || s.match.Status != mongomodel.MatchStatusActive {
		return false, nil
	}
	if activeMatchPhase(s.match) != mongomodel.MatchPhaseAnswering {
		return false, nil
	}
	if s.match.CurrentQuestionIndex < 0 || s.match.CurrentQuestionIndex >= len(s.match.QuestionIDs) {
		return false, nil
	}

	computerIndex := -1
	var humanPlayer mongomodel.MatchPlayer
	for index, player := range s.match.Players {
		if isComputerPlayer(player) {
			computerIndex = index
			continue
		}
		humanPlayer = player
	}
	if computerIndex < 0 || humanPlayer.PlayerID == "" {
		return false, nil
	}

	questionID := s.match.QuestionIDs[s.match.CurrentQuestionIndex]
	if hasAnswer(s.answers, computerPlayerID, questionID) {
		return false, nil
	}
	if !hasAnswer(s.answers, humanPlayer.PlayerID, questionID) && now.Before(s.match.RoundEndsAt) {
		return false, nil
	}

	question, ok := s.h.content.GetQuizQuestion(questionID)
	if !ok {
		return false, nil
	}
	settings, err := gamecontrol.ReadSettings(ctx, s.h.db)
	if err != nil {
		return false, err
	}
	difficulty, err := s.h.computerDifficulty(ctx, humanPlayer.PlayerID)
	if err != nil {
		return false, err
	}
	accuracy := computerAccuracy(settings, difficulty)
	choice := computerAnswerChoice(s.match, question, s.match.Players[computerIndex], accuracy)
	effects, err := s.h.matchPlayerBattleEffects(ctx, s.match.Players[computerIndex])
	if err != nil {
		return false, err
	}
	answer := buildMatchAnswer(s.match, question, s.match.Players[computerIndex], questionID, choice, now, effects)
	if err := s.persistAnswerLocked(ctx, computerIndex, answer); err != nil {
		return false, err
	}
	return true, nil
}

func (s *MatchSession) persistAnswerLocked(ctx context.Context, playerIndex int, answer mongomodel.MatchAnswer) error {
	if _, err := s.h.db.Collection(mongomodel.MatchAnswersCollection).InsertOne(ctx, answer); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return httpx.NewError(http.StatusConflict, "question already answered")
		}
		return httpx.InternalServerError("answer failed", "match_answer_insert_failed", err)
	}
	if err := s.h.applyMatchAnswerScore(ctx, s.match.ID, answer.PlayerID, answer.Score); err != nil {
		return httpx.InternalServerError("answer failed", "match_answer_save_match_failed", err)
	}
	s.answers = append(s.answers, answer)
	s.match.Players[playerIndex].Score += answer.Score
	s.match.Revision++
	return nil
}

func (s *MatchSession) snapshot() (mongomodel.Match, []mongomodel.MatchAnswer) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.snapshotLocked()
}

func (s *MatchSession) snapshotLocked() (mongomodel.Match, []mongomodel.MatchAnswer) {
	match := s.match
	return match, cloneMatchAnswers(s.answers)
}

func (s *MatchSession) eventSnapshot(name string) Event {
	match, answers := s.snapshot()
	return Event{Name: name, Match: match, Answers: answers}
}

func (s *MatchSession) eventSnapshotLocked(name string) Event {
	match, answers := s.snapshotLocked()
	return Event{Name: name, Match: match, Answers: answers}
}

func cloneMatchAnswers(answers []mongomodel.MatchAnswer) []mongomodel.MatchAnswer {
	if answers == nil {
		return []mongomodel.MatchAnswer{}
	}
	out := make([]mongomodel.MatchAnswer, len(answers))
	copy(out, answers)
	return out
}

func hasAnswer(answers []mongomodel.MatchAnswer, playerID string, questionID string) bool {
	for _, answer := range answers {
		if answer.PlayerID == playerID && answer.QuestionID == questionID {
			return true
		}
	}
	return false
}

func shouldRevealRoundWithAnswers(match mongomodel.Match, answers []mongomodel.MatchAnswer, now time.Time) bool {
	if len(match.QuestionIDs) == 0 || match.CurrentQuestionIndex >= len(match.QuestionIDs) {
		return false
	}
	if !match.RoundEndsAt.IsZero() && !now.Before(match.RoundEndsAt) {
		return true
	}

	questionID := match.QuestionIDs[match.CurrentQuestionIndex]
	for _, player := range match.Players {
		if !hasAnswer(answers, player.PlayerID, questionID) {
			return false
		}
	}
	return len(match.Players) == 2
}

func buildMatchAnswer(
	match mongomodel.Match,
	question content.QuizQuestion,
	player mongomodel.MatchPlayer,
	questionID string,
	choice string,
	now time.Time,
	effects battleEffects,
) mongomodel.MatchAnswer {
	correct, baseScore, score := scoreAnswer(question, choice, now, match.RoundEndsAt, effects)
	elapsedMillis := now.Sub(match.RoundStartedAt).Milliseconds()
	if elapsedMillis < 0 {
		elapsedMillis = 0
	}

	return mongomodel.MatchAnswer{
		ID:            matchAnswerRecordID(match.ID, player.PlayerID, questionID),
		MatchID:       match.ID,
		PlayerID:      player.PlayerID,
		QuestionID:    questionID,
		Choice:        choice,
		Correct:       correct,
		BaseScore:     baseScore,
		BonusScore:    score - baseScore,
		Score:         score,
		ElapsedMillis: elapsedMillis,
		AnsweredAt:    now,
	}
}
