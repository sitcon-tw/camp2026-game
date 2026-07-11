package matches

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/sitcon-tw/camp2026-game/internal/content"
	mongomodel "github.com/sitcon-tw/camp2026-game/internal/mongodb/model"
	"github.com/sitcon-tw/camp2026-game/internal/testcontent"
)

func TestScoreAnswer(t *testing.T) {
	question := content.QuizQuestion{CorrectChoice: "A"}
	answeredAt := time.Date(2026, 6, 2, 10, 0, 3, 0, time.UTC)
	roundEndsAt := time.Date(2026, 6, 2, 10, 0, 15, 0, time.UTC)

	correct, baseScore, score := scoreAnswer(question, "A", answeredAt, roundEndsAt, battleEffects{})
	if !correct {
		t.Fatal("expected answer to be correct")
	}
	if baseScore != 160 {
		t.Fatalf("expected base score 160, got %d", baseScore)
	}
	if score != 160 {
		t.Fatalf("expected score 160, got %d", score)
	}

	correct, baseScore, score = scoreAnswer(question, "B", answeredAt, roundEndsAt, battleEffects{AnswerScoreBonusPercent: 30})
	if correct {
		t.Fatal("expected answer to be incorrect")
	}
	if baseScore != 0 || score != 0 {
		t.Fatalf("expected incorrect score 0, got base=%d score=%d", baseScore, score)
	}
}

func TestScoreAnswerAppliesScoreBonus(t *testing.T) {
	question := content.QuizQuestion{CorrectChoice: "A"}
	answeredAt := time.Date(2026, 6, 2, 10, 0, 3, 0, time.UTC)
	roundEndsAt := time.Date(2026, 6, 2, 10, 0, 15, 0, time.UTC)

	correct, baseScore, score := scoreAnswer(question, "A", answeredAt, roundEndsAt, battleEffects{AnswerScoreBonusPercent: 30})
	if !correct {
		t.Fatal("expected answer to be correct")
	}
	if baseScore != 160 || score != 208 {
		t.Fatalf("expected base score 160 and boosted score 208, got base=%d score=%d", baseScore, score)
	}
}

func TestOpenPowerReward(t *testing.T) {
	if got := openPowerReward(850); got != 105 {
		t.Fatalf("expected open power reward 105, got %d", got)
	}
}

func TestMatchOpenPowerRewardRequiresClearWinner(t *testing.T) {
	match := mongomodel.Match{
		Players: []mongomodel.MatchPlayer{
			{PlayerID: "P1", Score: 340},
			{PlayerID: "P2", Score: 340},
		},
	}

	if matchHasClearWinner(match) {
		t.Fatal("expected tied match to have no clear winner")
	}
	for _, player := range match.Players {
		if got := matchOpenPowerReward(match, player, battleEffects{}); got != 0 {
			t.Fatalf("expected tied match reward 0 for %s, got %d", player.PlayerID, got)
		}
	}

	match.Players[1].Score = 330
	if !matchHasClearWinner(match) {
		t.Fatal("expected match with different scores to have a clear winner")
	}
	if got := matchOpenPowerReward(match, match.Players[0], battleEffects{}); got != 54 {
		t.Fatalf("expected winner reward 54, got %d", got)
	}
	if got := matchOpenPowerReward(match, match.Players[1], battleEffects{}); got != 0 {
		t.Fatalf("expected loser reward 0, got %d", got)
	}

	match.Players[0].Score = 320
	match.Players[1].Score = 340
	if got := matchOpenPowerReward(match, match.Players[0], battleEffects{}); got != 0 {
		t.Fatalf("expected lower-scoring player reward 0, got %d", got)
	}
	if got := matchOpenPowerReward(match, match.Players[1], battleEffects{}); got != 54 {
		t.Fatalf("expected winner reward 54, got %d", got)
	}
}

func TestMatchOpenPowerRewardAppliesBonus(t *testing.T) {
	match := mongomodel.Match{
		Players: []mongomodel.MatchPlayer{
			{PlayerID: "P1", Score: 340},
			{PlayerID: "P2", Score: 330},
		},
	}

	if got := matchOpenPowerReward(match, match.Players[0], battleEffects{OpenPowerBonusPercent: 40}); got != 75 {
		t.Fatalf("expected boosted winner reward 75, got %d", got)
	}
	if got := matchBaseOpenPowerReward(match, match.Players[0]); got != 54 {
		t.Fatalf("expected base winner reward 54, got %d", got)
	}
}

func TestMultiplayerOpenPowerRewardUsesPlayerScoreWithVariance(t *testing.T) {
	match := mongomodel.Match{
		ID:   "match_multiplayer_rewards",
		Mode: mongomodel.MatchModeMultiplayer,
		Players: []mongomodel.MatchPlayer{
			{PlayerID: "P1", Score: 340},
			{PlayerID: "P2", Score: 220},
			{PlayerID: "P3", Score: 220},
			{PlayerID: "P4", Score: 80},
		},
	}

	for _, player := range match.Players {
		base := openPowerReward(player.Score)
		got := matchOpenPowerReward(match, player, battleEffects{})
		minReward := applySignedPercentBonus(base, -openPowerRewardVariancePercent)
		maxReward := applySignedPercentBonus(base, openPowerRewardVariancePercent)
		if got < minReward || got > maxReward {
			t.Fatalf("expected multiplayer reward for %s in [%d,%d], got %d", player.PlayerID, minReward, maxReward, got)
		}
		if again := matchOpenPowerReward(match, player, battleEffects{}); again != got {
			t.Fatalf("expected deterministic multiplayer reward for %s, got %d then %d", player.PlayerID, got, again)
		}
		if baseGot := matchBaseOpenPowerReward(match, player); baseGot != base {
			t.Fatalf("expected multiplayer base reward %d for %s, got %d", base, player.PlayerID, baseGot)
		}
	}

	if matchBaseOpenPowerReward(match, match.Players[1]) != matchBaseOpenPowerReward(match, match.Players[2]) {
		t.Fatal("expected tied multiplayer players to share the same base reward")
	}
}

func TestMultiplayerOpenPowerRewardSkipsZeroScore(t *testing.T) {
	match := mongomodel.Match{
		ID:   "match_multiplayer_zero_score",
		Mode: mongomodel.MatchModeMultiplayer,
		Players: []mongomodel.MatchPlayer{
			{PlayerID: "P1", Score: 0},
		},
	}

	if got := matchOpenPowerReward(match, match.Players[0], battleEffects{}); got != 0 {
		t.Fatalf("expected zero-score multiplayer reward 0, got %d", got)
	}
	if got := matchBaseOpenPowerReward(match, match.Players[0]); got != 0 {
		t.Fatalf("expected zero-score multiplayer base reward 0, got %d", got)
	}
}

func TestMultiplayerOpenPowerRewardSkipsComputerPlayer(t *testing.T) {
	match := mongomodel.Match{
		ID:   "match_multiplayer_computer_player",
		Mode: mongomodel.MatchModeMultiplayer,
		Players: []mongomodel.MatchPlayer{
			{PlayerID: computerPlayerID, Kind: mongomodel.MatchPlayerKindComputer, Score: 340},
		},
	}

	if got := matchOpenPowerReward(match, match.Players[0], battleEffects{}); got != 0 {
		t.Fatalf("expected multiplayer computer reward 0, got %d", got)
	}
	if got := matchBaseOpenPowerReward(match, match.Players[0]); got != 0 {
		t.Fatalf("expected multiplayer computer base reward 0, got %d", got)
	}
}

func TestMultiplayerOpenPowerRewardAppliesBonusAfterVariance(t *testing.T) {
	match := mongomodel.Match{
		ID:   "match_multiplayer_bonus",
		Mode: mongomodel.MatchModeMultiplayer,
		Players: []mongomodel.MatchPlayer{
			{PlayerID: "P1", Score: 340},
		},
	}

	variedReward := multiplayerOpenPowerReward(match, match.Players[0])
	want := applyPercentBonus(variedReward, 40)
	if got := matchOpenPowerReward(match, match.Players[0], battleEffects{OpenPowerBonusPercent: 40}); got != want {
		t.Fatalf("expected boosted multiplayer reward %d, got %d", want, got)
	}
	if got := matchBaseOpenPowerReward(match, match.Players[0]); got != openPowerReward(340) {
		t.Fatalf("expected unvaried multiplayer base reward %d, got %d", openPowerReward(340), got)
	}
}

func TestBattleEffectsStackWithoutCaps(t *testing.T) {
	handler := New(Dependencies{Content: loadTestContent(t)})

	effects, err := handler.battleEffects(t.Context(), "P1", []string{
		"stone_2019_unboxed_algorithm",
		"stone_2019_unboxed_algorithm",
		"stone_2019_unboxed_algorithm",
	})
	if err != nil {
		t.Fatalf("battle effects: %v", err)
	}
	if effects.MaterialDropBonusPercent != 36 {
		t.Fatalf("expected material drop bonus 36, got %#v", effects)
	}

	effects, err = handler.battleEffects(t.Context(), "P1", []string{
		"stone_command_blind_trip",
		"stone_von_neumann",
	})
	if err != nil {
		t.Fatalf("battle effects: %v", err)
	}
	if effects.OpenPowerBonusPercent != 15 || effects.MaterialDropBonusPercent != 15 {
		t.Fatalf("expected open power bonus 15 and material drop bonus 15, got %#v", effects)
	}

	effects, err = handler.battleEffects(t.Context(), "P1", []string{
		"stone_fireside",
		"stone_tech_art",
		"stone_2020_sitcon_tour_group",
	})
	if err != nil {
		t.Fatalf("battle effects: %v", err)
	}
	if effects.OpenPowerBonusPercent != 144 {
		t.Fatalf("expected open power bonus 144, got %#v", effects)
	}

	effects, err = handler.battleEffects(t.Context(), "P1", []string{
		"stone_python_turtle",
		"stone_human_llm",
	})
	if err != nil {
		t.Fatalf("battle effects: %v", err)
	}
	if effects.EliminateChancePercent != 12 || effects.EliminateCount != 3 {
		t.Fatalf("expected eliminate bonus 12 and count 3, got %#v", effects)
	}
}

func TestMatchPlayerBattleEffectsPrefersSnapshot(t *testing.T) {
	handler := New(Dependencies{Content: loadTestContent(t)})
	player := mongomodel.MatchPlayer{
		PlayerID:  "P1",
		SitoneIDs: []string{"stone_engineering_base"},
		BattleEffects: &mongomodel.MatchBattleEffects{
			MaterialDropBonusPercent: 17,
			OpenPowerBonusPercent:    29,
			EliminateSourceNames:     []string{"snapshot"},
		},
	}

	effects, err := handler.matchPlayerBattleEffects(t.Context(), player)
	if err != nil {
		t.Fatalf("match player battle effects: %v", err)
	}
	if effects.MaterialDropBonusPercent != 17 || effects.OpenPowerBonusPercent != 29 {
		t.Fatalf("expected snapshotted effects, got %#v", effects)
	}
	if effects.AnswerScoreBonusPercent != 0 {
		t.Fatalf("expected snapshot to avoid live sitone lookup, got %#v", effects)
	}
	if len(effects.EliminateSourceNames) != 1 || effects.EliminateSourceNames[0] != "snapshot" {
		t.Fatalf("expected snapshot source names, got %#v", effects)
	}
}

func TestEliminatedChoicesNeverIncludesCorrectChoice(t *testing.T) {
	question := content.QuizQuestion{
		ID:            "quiz-001",
		CorrectChoice: "A",
	}
	player := mongomodel.MatchPlayer{PlayerID: "P1"}
	effects := battleEffects{
		EliminateChancePercent: 100,
		EliminateCount:         2,
		EliminateSourceNames:   []string{"靈光型小石"},
	}

	first := eliminatedChoicesForPlayer("match_123", question, player, effects)
	second := eliminatedChoicesForPlayer("match_123", question, player, effects)
	if len(first.Choices) != 2 {
		t.Fatalf("expected 2 eliminated choices, got %#v", first)
	}
	for _, choice := range first.Choices {
		if choice == "A" {
			t.Fatalf("eliminated correct choice: %#v", first)
		}
	}
	if len(second.Choices) != len(first.Choices) || second.Choices[0] != first.Choices[0] || second.Choices[1] != first.Choices[1] {
		t.Fatalf("expected deterministic eliminated choices, got first=%#v second=%#v", first, second)
	}
}

func TestMatchMaterialDropRateUsesWinnerAndLoserBaseRates(t *testing.T) {
	match := mongomodel.Match{
		Players: []mongomodel.MatchPlayer{
			{PlayerID: "P1", Score: 200},
			{PlayerID: "P2", Score: 100},
		},
	}

	if got := matchMaterialDropRate(match, match.Players[0], battleEffects{MaterialDropBonusPercent: 15}); got != 40 {
		t.Fatalf("expected winner drop rate 40, got %d", got)
	}
	if got := matchMaterialDropRate(match, match.Players[1], battleEffects{MaterialDropBonusPercent: 12}); got != 27 {
		t.Fatalf("expected loser drop rate 27, got %d", got)
	}
}

func TestLeastOwnedSitoneDropPoolPrefersLowestQuantity(t *testing.T) {
	pool := []content.Sitone{
		{ID: "stone_a"},
		{ID: "stone_b"},
		{ID: "stone_c"},
	}
	owned := map[string]int{
		"stone_a": 2,
		"stone_b": 0,
		"stone_c": 1,
	}

	got := leastOwnedSitoneDropPool(pool, owned)
	if len(got) != 1 || got[0].ID != "stone_b" {
		t.Fatalf("expected least-owned stone_b, got %#v", got)
	}
}

func TestLeastOwnedSitoneDropPoolIncludesTiedLowestQuantity(t *testing.T) {
	pool := []content.Sitone{
		{ID: "stone_a"},
		{ID: "stone_b"},
		{ID: "stone_c"},
	}
	owned := map[string]int{
		"stone_a": 1,
		"stone_b": 0,
		"stone_c": 0,
	}

	got := leastOwnedSitoneDropPool(pool, owned)
	if len(got) != 2 || got[0].ID != "stone_b" || got[1].ID != "stone_c" {
		t.Fatalf("expected tied lowest stones b and c, got %#v", got)
	}
}

func TestSecureRandomIntUsesReaderValue(t *testing.T) {
	got, err := secureRandomInt(strings.NewReader("*"), 100)
	if err != nil {
		t.Fatalf("secure random int: %v", err)
	}
	if got != 42 {
		t.Fatalf("expected reader value 42, got %d", got)
	}
}

func TestSecureRandomIntRejectsInvalidMax(t *testing.T) {
	if _, err := secureRandomInt(strings.NewReader("*"), 0); err == nil {
		t.Fatal("expected invalid max error")
	}
}

func TestSecureRandomIntReturnsRandomnessError(t *testing.T) {
	randomErr := errors.New("entropy unavailable")
	_, err := secureRandomInt(errReader{err: randomErr}, 100)
	if !errors.Is(err, randomErr) {
		t.Fatalf("expected random error, got %v", err)
	}
}

func TestMatchRewardKeysUseMatchAndPlayer(t *testing.T) {
	if got := matchRewardRecordID("match_123", "P1"); got != "open_power_reward_match_123_P1" {
		t.Fatalf("unexpected reward record id: %q", got)
	}
	if got := matchRewardSource("match_123", "P1"); got != "quiz_match:match_123:player:P1" {
		t.Fatalf("unexpected reward source: %q", got)
	}
}

func TestMatchRevisionFilterGuardsInitialAndExistingRevisions(t *testing.T) {
	initial := matchRevisionFilter("match_123", 0)
	if initial["_id"] != "match_123" {
		t.Fatalf("expected match id filter, got %#v", initial)
	}
	alternatives, ok := initial["$or"].(bson.A)
	if !ok || len(alternatives) != 2 {
		t.Fatalf("expected initial revision filter to allow zero or missing revision, got %#v", initial)
	}

	existing := matchRevisionFilter("match_123", 7)
	if existing["_id"] != "match_123" || existing["revision"] != int64(7) {
		t.Fatalf("expected existing revision guard, got %#v", existing)
	}
	if _, ok := existing["$or"]; ok {
		t.Fatalf("expected existing revision filter not to allow missing revision, got %#v", existing)
	}
}

func TestOpenHostedMatchFilterMatchesWaitingAndActiveHost(t *testing.T) {
	filter := openHostedMatchFilter("P1")
	if filter["host_player_id"] != "P1" {
		t.Fatalf("expected host player filter, got %#v", filter)
	}

	status, ok := filter["status"].(bson.M)
	if !ok {
		t.Fatalf("expected status filter, got %#v", filter)
	}
	values, ok := status["$in"].(bson.A)
	if !ok || len(values) != 2 {
		t.Fatalf("expected status $in filter, got %#v", status)
	}
	if values[0] != mongomodel.MatchStatusWaiting || values[1] != mongomodel.MatchStatusActive {
		t.Fatalf("expected waiting and active statuses, got %#v", values)
	}
}

func TestOpenParticipantMatchFilterMatchesWaitingAndActiveParticipant(t *testing.T) {
	filter := openParticipantMatchFilter("P1")
	if filter["players.player_id"] != "P1" {
		t.Fatalf("expected participant player filter, got %#v", filter)
	}

	status, ok := filter["status"].(bson.M)
	if !ok {
		t.Fatalf("expected status filter, got %#v", filter)
	}
	values, ok := status["$in"].(bson.A)
	if !ok || len(values) != 2 {
		t.Fatalf("expected status $in filter, got %#v", status)
	}
	if values[0] != mongomodel.MatchStatusWaiting || values[1] != mongomodel.MatchStatusActive {
		t.Fatalf("expected waiting and active statuses, got %#v", values)
	}
}

func TestMatchCanIssuePairingTokenRequiresWaitingHumanMatchHostOnly(t *testing.T) {
	match := mongomodel.Match{
		Mode:         mongomodel.MatchModePVP,
		Status:       mongomodel.MatchStatusWaiting,
		HostPlayerID: "P1",
		Players: []mongomodel.MatchPlayer{
			{PlayerID: "P1", Kind: mongomodel.MatchPlayerKindHuman},
		},
	}
	if !matchCanIssuePairingToken(match, "P1") {
		t.Fatal("expected waiting PVP host to issue pairing token")
	}

	if matchCanIssuePairingToken(match, "P2") {
		t.Fatal("expected non-host to be unable to issue pairing token")
	}

	match.Players = append(match.Players, mongomodel.MatchPlayer{PlayerID: "P2", Kind: mongomodel.MatchPlayerKindHuman})
	if matchCanIssuePairingToken(match, "P1") {
		t.Fatal("expected full human match to be unable to issue pairing token")
	}

	match.Players = match.Players[:1]
	match.Mode = mongomodel.MatchModeMultiplayer
	match.Players = append(match.Players,
		mongomodel.MatchPlayer{PlayerID: "P2", Kind: mongomodel.MatchPlayerKindHuman},
		mongomodel.MatchPlayer{PlayerID: "P3", Kind: mongomodel.MatchPlayerKindHuman},
	)
	if !matchCanIssuePairingToken(match, "P1") {
		t.Fatal("expected waiting multiplayer host to issue pairing token before room is full")
	}
	match.Players = append(match.Players, mongomodel.MatchPlayer{PlayerID: "P4", Kind: mongomodel.MatchPlayerKindHuman})
	if matchCanIssuePairingToken(match, "P1") {
		t.Fatal("expected full multiplayer match to be unable to issue pairing token")
	}

	match.Players = match.Players[:1]
	match.Status = mongomodel.MatchStatusActive
	if matchCanIssuePairingToken(match, "P1") {
		t.Fatal("expected active match to be unable to issue pairing token")
	}

	match.Status = mongomodel.MatchStatusWaiting
	match.Mode = mongomodel.MatchModeComputer
	if matchCanIssuePairingToken(match, "P1") {
		t.Fatal("expected computer match to be unable to issue pairing token")
	}
}

func TestMatchPairingTokenSingleUseOnlyForPVP(t *testing.T) {
	if !matchPairingTokenSingleUse(mongomodel.Match{Mode: mongomodel.MatchModePVP}) {
		t.Fatal("expected PVP pairing token to be single-use")
	}
	if matchPairingTokenSingleUse(mongomodel.Match{Mode: mongomodel.MatchModeMultiplayer}) {
		t.Fatal("expected multiplayer pairing token to remain reusable until it expires")
	}
}

func TestMatchPairingTokenHashDoesNotExposeToken(t *testing.T) {
	token := "7H9K2Q8M4RXPA3WZ"
	hash := matchPairingTokenHash(token)
	if hash == token {
		t.Fatal("expected pairing token hash not to expose raw token")
	}
	if hash != matchPairingTokenHash(token) {
		t.Fatal("expected pairing token hash to be stable")
	}
	if len(hash) != 64 {
		t.Fatalf("expected sha256 hex hash length 64, got %d", len(hash))
	}
}

func TestOpenMatchUnavailableDetectsStaleOpenRace(t *testing.T) {
	if !openMatchUnavailable(mongo.ErrNoDocuments) {
		t.Fatal("expected missing match to be treated as unavailable")
	}
	if !openMatchUnavailable(errMatchNotOpen) {
		t.Fatal("expected no-longer-open match to be treated as unavailable")
	}
	if openMatchUnavailable(errors.New("database unavailable")) {
		t.Fatal("expected unrelated errors to remain reportable")
	}
}

func TestWriteOpenMatchLookupProblemReturnsNotFoundForUnavailableOpenMatch(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/matches/open", nil)
	for _, err := range []error{mongo.ErrNoDocuments, errMatchNotOpen} {
		res := httptest.NewRecorder()
		writeOpenMatchLookupProblem(res, req, err)
		if res.Code != http.StatusNotFound {
			t.Fatalf("expected unavailable open match to return 404, got %d", res.Code)
		}
	}
}

func TestSyncOpenMatchLocks(t *testing.T) {
	match := mongomodel.Match{
		Status:       mongomodel.MatchStatusActive,
		HostPlayerID: "P1",
		Players: []mongomodel.MatchPlayer{
			{PlayerID: "P1", Kind: mongomodel.MatchPlayerKindHuman},
			{PlayerID: "computer", Kind: mongomodel.MatchPlayerKindComputer},
			{PlayerID: "P2", Kind: mongomodel.MatchPlayerKindHuman},
			{PlayerID: "P2", Kind: mongomodel.MatchPlayerKindHuman},
		},
	}
	syncOpenMatchLocks(&match)
	if match.OpenHostLock != "P1" {
		t.Fatalf("expected active match to keep open host lock, got %q", match.OpenHostLock)
	}
	if got := strings.Join(match.OpenPlayerLocks, ","); got != "P1,P2" {
		t.Fatalf("expected human participant locks, got %#v", match.OpenPlayerLocks)
	}

	match.Status = mongomodel.MatchStatusCompleted
	syncOpenMatchLocks(&match)
	if match.OpenHostLock != "" || match.OpenPlayerLocks != nil {
		t.Fatalf("expected completed match to release open locks, got host=%q players=%#v", match.OpenHostLock, match.OpenPlayerLocks)
	}
}

func TestNewRematchMatchResetsCompletedState(t *testing.T) {
	now := time.Date(2026, 7, 11, 12, 0, 0, 0, time.UTC)
	source := mongomodel.Match{
		ID:                   "match_done",
		Mode:                 mongomodel.MatchModeComputer,
		Status:               mongomodel.MatchStatusCompleted,
		Phase:                mongomodel.MatchPhaseRevealing,
		HostPlayerID:         "P1",
		QuestionIDs:          []string{"quiz-001"},
		CurrentQuestionIndex: 9,
		Players: []mongomodel.MatchPlayer{
			{PlayerID: "P1", Nickname: "Alice", Ready: true, Score: 900, SitoneIDs: []string{"stone_engineering_base"}},
			{PlayerID: computerPlayerID, Nickname: computerNickname, Kind: mongomodel.MatchPlayerKindComputer, Ready: true, Score: 500},
		},
		StartedAt:   now.Add(-time.Minute),
		CompletedAt: now,
	}

	rematch := newRematchMatch(source, "match_next", "P1", now.Add(time.Second))
	if rematch.ID != "match_next" || rematch.RematchSourceMatchID != source.ID {
		t.Fatalf("unexpected rematch identity: %#v", rematch)
	}
	if rematch.Status != mongomodel.MatchStatusWaiting || rematch.Phase != "" || rematch.StartedAt != (time.Time{}) || rematch.CompletedAt != (time.Time{}) {
		t.Fatalf("expected waiting rematch without completed state, got %#v", rematch)
	}
	if len(rematch.QuestionIDs) != 0 || rematch.CurrentQuestionIndex != 0 {
		t.Fatalf("expected rematch to clear question progress, got %#v", rematch)
	}
	if rematch.OpenHostLock != "P1" || strings.Join(rematch.OpenPlayerLocks, ",") != "P1" {
		t.Fatalf("expected open locks for human rematch player, got host=%q players=%#v", rematch.OpenHostLock, rematch.OpenPlayerLocks)
	}
	if len(rematch.Players) != 2 || rematch.Players[0].Ready || rematch.Players[0].Score != 0 {
		t.Fatalf("expected human rematch player reset, got %#v", rematch.Players)
	}
	if !rematch.Players[1].Ready || rematch.Players[1].Score != 0 || len(rematch.Players[1].SitoneIDs) == 0 {
		t.Fatalf("expected computer rematch player ready with default loadout, got %#v", rematch.Players[1])
	}
}

func TestMatchAnswerRecordIDUsesMatchPlayerAndQuestion(t *testing.T) {
	got := matchAnswerRecordID("match_123", "P1", "quiz-001")
	if got != "answer_match_123_P1_quiz-001" {
		t.Fatalf("unexpected answer record id: %q", got)
	}
}

func TestAllPlayersReady(t *testing.T) {
	match := mongomodel.Match{
		Players: []mongomodel.MatchPlayer{
			{PlayerID: "P1", Ready: true, SitoneIDs: []string{"stone_engineering_base"}},
			{PlayerID: "P2", Ready: true, SitoneIDs: []string{"stone_resonance_base"}},
		},
	}
	if !allPlayersReady(match) {
		t.Fatal("expected all players ready")
	}

	match.Players[1].Ready = false
	if allPlayersReady(match) {
		t.Fatal("expected not all players ready")
	}

	match.Players[1].Ready = true
	match.Players[1].SitoneIDs = nil
	if allPlayersReady(match) {
		t.Fatal("expected not all players ready without loadout")
	}

	match = mongomodel.Match{
		Mode: mongomodel.MatchModeMultiplayer,
		Players: []mongomodel.MatchPlayer{
			{PlayerID: "P1", Ready: true, SitoneIDs: []string{"stone_engineering_base"}},
			{PlayerID: "P2", Ready: true, SitoneIDs: []string{"stone_resonance_base"}},
			{PlayerID: "P3", Ready: true, SitoneIDs: []string{"stone_explorer_base"}},
			{PlayerID: "P4", Ready: true, SitoneIDs: []string{"stone_inspiration_base"}},
		},
	}
	if !allPlayersReady(match) {
		t.Fatal("expected four ready multiplayer players to start")
	}
	match.Players = match.Players[:3]
	if allPlayersReady(match) {
		t.Fatal("expected multiplayer match to wait for four players")
	}

	addMultiplayerComputerPlayer(&match)
	if !allPlayersReady(match) {
		t.Fatal("expected multiplayer match with ready humans and computer fill to start")
	}
}

func TestMultiplayerHostCanStartRequiresJoinedHumansReady(t *testing.T) {
	match := mongomodel.Match{
		Mode:         mongomodel.MatchModeMultiplayer,
		HostPlayerID: "P1",
		Players: []mongomodel.MatchPlayer{
			{PlayerID: "P1", Kind: mongomodel.MatchPlayerKindHuman, Ready: false, SitoneIDs: []string{"stone_engineering_base"}},
			{PlayerID: "P2", Kind: mongomodel.MatchPlayerKindHuman, Ready: false, SitoneIDs: []string{"stone_resonance_base"}},
		},
	}
	if multiplayerHostCanStart(match, "P1") {
		t.Fatal("expected host start to wait for joined player readiness")
	}

	match.Players[1].Ready = true
	if multiplayerHostCanStart(match, "P1") {
		t.Fatal("expected host start to wait for four occupied slots")
	}
	if multiplayerHostCanStart(match, "P2") {
		t.Fatal("expected non-host player to be unable to start multiplayer match")
	}

	addMultiplayerComputerPlayer(&match)
	addMultiplayerComputerPlayer(&match)
	if !multiplayerHostCanStart(match, "P1") {
		t.Fatal("expected host to start after empty slots are filled and joined humans are ready")
	}

	match.Players = []mongomodel.MatchPlayer{
		{PlayerID: "P1", Kind: mongomodel.MatchPlayerKindHuman, Ready: false, SitoneIDs: []string{"stone_engineering_base"}},
		{PlayerID: "P2", Kind: mongomodel.MatchPlayerKindHuman, Ready: true, SitoneIDs: []string{"stone_resonance_base"}},
		mongomodel.MatchPlayer{PlayerID: "P3", Kind: mongomodel.MatchPlayerKindHuman, Ready: true, SitoneIDs: []string{"stone_explorer_base"}},
		mongomodel.MatchPlayer{PlayerID: "P4", Kind: mongomodel.MatchPlayerKindHuman, Ready: true, SitoneIDs: []string{"stone_inspiration_base"}},
	}
	if !multiplayerHostCanStart(match, "P1") {
		t.Fatal("expected host to start full room after joined humans are ready")
	}
}

func TestActiveMatchPhaseDefaultsToAnswering(t *testing.T) {
	match := mongomodel.Match{Status: mongomodel.MatchStatusActive}
	if got := activeMatchPhase(match); got != mongomodel.MatchPhaseAnswering {
		t.Fatalf("expected missing active phase to default to answering, got %q", got)
	}

	match.Phase = mongomodel.MatchPhaseRevealing
	if got := activeMatchPhase(match); got != mongomodel.MatchPhaseRevealing {
		t.Fatalf("expected revealing phase, got %q", got)
	}

	match.Status = mongomodel.MatchStatusCompleted
	if got := activeMatchPhase(match); got != "" {
		t.Fatalf("expected completed match to have no active phase, got %q", got)
	}
}

func TestApplyRevealDeadlineTransitionStartsNextRound(t *testing.T) {
	revealEndsAt := time.Date(2026, 6, 2, 10, 0, 4, 0, time.UTC)
	match := mongomodel.Match{
		Status:               mongomodel.MatchStatusActive,
		Phase:                mongomodel.MatchPhaseRevealing,
		QuestionIDs:          []string{"quiz-001", "quiz-002"},
		CurrentQuestionIndex: 0,
		RevealEndsAt:         revealEndsAt,
	}

	event := applyRevealDeadlineTransition(&match, revealEndsAt)
	if event != "round_started" {
		t.Fatalf("expected round_started event, got %q", event)
	}
	if match.Status != mongomodel.MatchStatusActive || match.Phase != mongomodel.MatchPhaseAnswering {
		t.Fatalf("expected active answering phase, got status=%q phase=%q", match.Status, match.Phase)
	}
	if match.CurrentQuestionIndex != 1 {
		t.Fatalf("expected question index 1, got %d", match.CurrentQuestionIndex)
	}
	if !match.RoundStartedAt.Equal(revealEndsAt) {
		t.Fatalf("expected round start at reveal end, got %s", match.RoundStartedAt)
	}
	if !match.RoundEndsAt.Equal(revealEndsAt.Add(roundDuration * time.Second)) {
		t.Fatalf("expected round end at reveal end plus duration, got %s", match.RoundEndsAt)
	}
	if !match.RevealEndsAt.IsZero() {
		t.Fatalf("expected reveal end to be cleared, got %s", match.RevealEndsAt)
	}
}

func TestApplyRevealDeadlineTransitionCompletesLastRound(t *testing.T) {
	revealEndsAt := time.Date(2026, 6, 2, 10, 0, 4, 0, time.UTC)
	match := mongomodel.Match{
		Status:               mongomodel.MatchStatusActive,
		Phase:                mongomodel.MatchPhaseRevealing,
		QuestionIDs:          []string{"quiz-001", "quiz-002"},
		CurrentQuestionIndex: 1,
		RevealEndsAt:         revealEndsAt,
	}

	event := applyRevealDeadlineTransition(&match, revealEndsAt)
	if event != "match_completed" {
		t.Fatalf("expected match_completed event, got %q", event)
	}
	if match.Status != mongomodel.MatchStatusCompleted || match.Phase != "" {
		t.Fatalf("expected completed match without active phase, got status=%q phase=%q", match.Status, match.Phase)
	}
	if !match.CompletedAt.Equal(revealEndsAt) {
		t.Fatalf("expected completed at reveal end, got %s", match.CompletedAt)
	}
}

func TestShouldRevealRoundWithInMemoryAnswers(t *testing.T) {
	now := time.Date(2026, 6, 2, 10, 0, 3, 0, time.UTC)
	match := mongomodel.Match{
		Status:               mongomodel.MatchStatusActive,
		Phase:                mongomodel.MatchPhaseAnswering,
		QuestionIDs:          []string{"quiz-001"},
		CurrentQuestionIndex: 0,
		RoundEndsAt:          now.Add(10 * time.Second),
		Players: []mongomodel.MatchPlayer{
			{PlayerID: "P1"},
			{PlayerID: "P2"},
		},
	}

	answers := []mongomodel.MatchAnswer{
		{PlayerID: "P1", QuestionID: "quiz-001"},
	}
	if shouldRevealRoundWithAnswers(match, answers, now) {
		t.Fatal("expected one answer before deadline not to reveal")
	}

	answers = append(answers, mongomodel.MatchAnswer{PlayerID: "P2", QuestionID: "quiz-001"})
	if !shouldRevealRoundWithAnswers(match, answers, now) {
		t.Fatal("expected all players answered to reveal")
	}

	if !shouldRevealRoundWithAnswers(match, nil, match.RoundEndsAt) {
		t.Fatal("expected round deadline to reveal without answers")
	}

	match.Mode = mongomodel.MatchModeMultiplayer
	match.Players = []mongomodel.MatchPlayer{
		{PlayerID: "P1"},
		{PlayerID: "P2"},
		{PlayerID: "P3"},
		{PlayerID: "P4"},
	}
	answers = []mongomodel.MatchAnswer{
		{PlayerID: "P1", QuestionID: "quiz-001"},
		{PlayerID: "P2", QuestionID: "quiz-001"},
		{PlayerID: "P3", QuestionID: "quiz-001"},
	}
	if shouldRevealRoundWithAnswers(match, answers, now) {
		t.Fatal("expected multiplayer round to wait for the fourth answer")
	}
	answers = append(answers, mongomodel.MatchAnswer{PlayerID: "P4", QuestionID: "quiz-001"})
	if !shouldRevealRoundWithAnswers(match, answers, now) {
		t.Fatal("expected all multiplayer players answered to reveal")
	}
}

func TestMatchSessionTickBroadcastsActiveStateEverySecond(t *testing.T) {
	broker := NewBroker()
	handler := New(Dependencies{
		Content: loadTestContent(t),
		Broker:  broker,
	})
	match := mongomodel.Match{
		ID:                   "match_123",
		Status:               mongomodel.MatchStatusActive,
		Phase:                mongomodel.MatchPhaseAnswering,
		QuestionIDs:          []string{"quiz-001"},
		CurrentQuestionIndex: 0,
		RoundEndsAt:          time.Now().Add(10 * time.Second),
		Players: []mongomodel.MatchPlayer{
			{PlayerID: "P1", Nickname: "Alice"},
			{PlayerID: "P2", Nickname: "Bob"},
		},
	}
	session := newMatchSession(NewMatchSessionManager(handler), match, nil)
	events, unsubscribe := broker.Subscribe(match.ID)
	defer unsubscribe()

	session.tick(time.Now())

	select {
	case event := <-events:
		if event.Name != "match_updated" {
			t.Fatalf("expected match_updated event, got %q", event.Name)
		}
		if event.Match.ID != match.ID || event.Answers == nil {
			t.Fatalf("expected session event snapshot, got %#v", event)
		}
	case <-time.After(time.Second):
		t.Fatal("expected session tick to broadcast active state")
	}
}

func TestMatchModeAndPlayerKindDefaults(t *testing.T) {
	match := mongomodel.Match{}
	if got := matchMode(match); got != mongomodel.MatchModePVP {
		t.Fatalf("expected default match mode %q, got %q", mongomodel.MatchModePVP, got)
	}
	if got := matchPlayerKind(mongomodel.MatchPlayer{}); got != mongomodel.MatchPlayerKindHuman {
		t.Fatalf("expected default player kind %q, got %q", mongomodel.MatchPlayerKindHuman, got)
	}

	match.Mode = mongomodel.MatchModeComputer
	player := mongomodel.MatchPlayer{Kind: mongomodel.MatchPlayerKindComputer}
	if !isComputerMatch(match) || !isComputerPlayer(player) {
		t.Fatalf("expected computer match/player helpers to detect computer values")
	}

	match.Mode = mongomodel.MatchModeMultiplayer
	if got := matchMode(match); got != mongomodel.MatchModeMultiplayer {
		t.Fatalf("expected multiplayer match mode, got %q", got)
	}
}

func TestShouldCheckSameTeamBattleOnlyForNewPVPHumanOpponent(t *testing.T) {
	match := mongomodel.Match{
		Mode: mongomodel.MatchModePVP,
		Players: []mongomodel.MatchPlayer{
			{PlayerID: "P1", Kind: mongomodel.MatchPlayerKindHuman},
		},
	}
	player := mongomodel.Player{ID: "P2", TeamID: "team-a"}
	if !shouldCheckSameTeamBattle(match, player) {
		t.Fatal("expected PVP join with a human opponent to check same-team setting")
	}

	match.Mode = mongomodel.MatchModeComputer
	if shouldCheckSameTeamBattle(match, player) {
		t.Fatal("expected computer match to skip same-team setting")
	}

	match.Mode = mongomodel.MatchModePVP
	player.TeamID = ""
	if shouldCheckSameTeamBattle(match, player) {
		t.Fatal("expected ungrouped player to skip same-team setting")
	}

	player = mongomodel.Player{ID: "P1", TeamID: "team-a"}
	if shouldCheckSameTeamBattle(match, player) {
		t.Fatal("expected existing participant to skip same-team setting")
	}

	match.Players = []mongomodel.MatchPlayer{{PlayerID: computerPlayerID, Kind: mongomodel.MatchPlayerKindComputer}}
	player = mongomodel.Player{ID: "P2", TeamID: "team-a"}
	if shouldCheckSameTeamBattle(match, player) {
		t.Fatal("expected match without human opponent to skip same-team setting")
	}
}

func TestShouldCheckOpponentBattleLimitOnlyForNewPVPHumanOpponent(t *testing.T) {
	match := mongomodel.Match{
		Mode: mongomodel.MatchModePVP,
		Players: []mongomodel.MatchPlayer{
			{PlayerID: "P1", Kind: mongomodel.MatchPlayerKindHuman},
		},
	}
	player := mongomodel.Player{ID: "P2"}
	if !shouldCheckOpponentBattleLimit(match, player) {
		t.Fatal("expected PVP join with exactly one human opponent to check battle limit")
	}

	match.Mode = mongomodel.MatchModeComputer
	if shouldCheckOpponentBattleLimit(match, player) {
		t.Fatal("expected computer match to skip battle limit check")
	}

	match.Mode = mongomodel.MatchModePVP
	player = mongomodel.Player{ID: "P1"}
	if shouldCheckOpponentBattleLimit(match, player) {
		t.Fatal("expected existing participant to skip battle limit check")
	}

	match.Players = []mongomodel.MatchPlayer{
		{PlayerID: computerPlayerID, Kind: mongomodel.MatchPlayerKindComputer},
	}
	player = mongomodel.Player{ID: "P2"}
	if shouldCheckOpponentBattleLimit(match, player) {
		t.Fatal("expected match without a human opponent to skip battle limit check")
	}

	match.Players = []mongomodel.MatchPlayer{
		{PlayerID: "P1", Kind: mongomodel.MatchPlayerKindHuman},
		{PlayerID: "P3", Kind: mongomodel.MatchPlayerKindHuman},
	}
	if shouldCheckOpponentBattleLimit(match, player) {
		t.Fatal("expected match with more than one human opponent to skip battle limit check")
	}
}

func TestOpponentMatchLimitDayRangeUsesTaipeiMidnight(t *testing.T) {
	now := time.Date(2026, 7, 8, 15, 30, 0, 0, time.UTC)
	dayStart, dayEnd := opponentMatchLimitDayRange(now)

	wantStart := time.Date(2026, 7, 7, 16, 0, 0, 0, time.UTC)
	wantEnd := time.Date(2026, 7, 8, 16, 0, 0, 0, time.UTC)
	if !dayStart.Equal(wantStart) || !dayEnd.Equal(wantEnd) {
		t.Fatalf("unexpected day range: got %s - %s, want %s - %s", dayStart, dayEnd, wantStart, wantEnd)
	}
}

func TestCompletedPVPMatchCountBetweenPlayersFilterUsesCompletedAtDailyRange(t *testing.T) {
	dayStart := time.Date(2026, 7, 7, 16, 0, 0, 0, time.UTC)
	dayEnd := time.Date(2026, 7, 8, 16, 0, 0, 0, time.UTC)

	filter := completedPVPMatchCountBetweenPlayersFilter("P1", "P2", dayStart, dayEnd)
	if filter["mode"] != mongomodel.MatchModePVP || filter["status"] != mongomodel.MatchStatusCompleted {
		t.Fatalf("unexpected mode/status filter: %#v", filter)
	}

	playerFilter, ok := filter["players.player_id"].(bson.M)
	if !ok {
		t.Fatalf("expected player filter, got %#v", filter["players.player_id"])
	}
	players, ok := playerFilter["$all"].(bson.A)
	if !ok || len(players) != 2 || players[0] != "P1" || players[1] != "P2" {
		t.Fatalf("unexpected player $all filter: %#v", playerFilter)
	}

	completedAtFilter, ok := filter["completed_at"].(bson.M)
	if !ok {
		t.Fatalf("expected completed_at filter, got %#v", filter["completed_at"])
	}
	gte, ok := completedAtFilter["$gte"].(time.Time)
	if !ok || !gte.Equal(dayStart) {
		t.Fatalf("unexpected completed_at lower bound: %#v", completedAtFilter["$gte"])
	}
	lt, ok := completedAtFilter["$lt"].(time.Time)
	if !ok || !lt.Equal(dayEnd) {
		t.Fatalf("unexpected completed_at upper bound: %#v", completedAtFilter["$lt"])
	}
}

func TestShouldCheckSameTeamMatchRequiresPVPHumanParticipants(t *testing.T) {
	match := mongomodel.Match{
		Mode: mongomodel.MatchModePVP,
		Players: []mongomodel.MatchPlayer{
			{PlayerID: "P1", Kind: mongomodel.MatchPlayerKindHuman},
			{PlayerID: "P2", Kind: mongomodel.MatchPlayerKindHuman},
		},
	}
	if !shouldCheckSameTeamMatch(match) {
		t.Fatal("expected PVP match with two human participants to check same-team setting")
	}

	match.Mode = mongomodel.MatchModeComputer
	if shouldCheckSameTeamMatch(match) {
		t.Fatal("expected computer match to skip same-team setting")
	}

	match.Mode = mongomodel.MatchModePVP
	match.Players[1].Kind = mongomodel.MatchPlayerKindComputer
	if shouldCheckSameTeamMatch(match) {
		t.Fatal("expected match without two human participants to skip same-team setting")
	}
}

func TestHumanOpponentIDsExcludeCurrentAndComputerPlayers(t *testing.T) {
	match := mongomodel.Match{
		Players: []mongomodel.MatchPlayer{
			{PlayerID: "P1", Kind: mongomodel.MatchPlayerKindHuman},
			{PlayerID: "P2", Kind: mongomodel.MatchPlayerKindHuman},
			{PlayerID: computerPlayerID, Kind: mongomodel.MatchPlayerKindComputer},
		},
	}
	opponents := humanOpponentIDs(match, "P2")
	if len(opponents) != 1 || opponents[0] != "P1" {
		t.Fatalf("expected only P1 as human opponent, got %#v", opponents)
	}
}

func TestComputerDifficultyForPlayerUsesLeaderboardPercentile(t *testing.T) {
	entries := []computerRankEntry{
		{PlayerID: "p1"},
		{PlayerID: "p2"},
		{PlayerID: "p3"},
		{PlayerID: "p4"},
		{PlayerID: "p5"},
		{PlayerID: "p6"},
		{PlayerID: "p7"},
		{PlayerID: "p8"},
		{PlayerID: "p9"},
		{PlayerID: "p10"},
	}

	if got := computerDifficultyForPlayer(entries, "p1"); got != computerDifficultyHard {
		t.Fatalf("expected top percentile hard, got %q", got)
	}
	if got := computerDifficultyForPlayer(entries, "p3"); got != computerDifficultyNormal {
		t.Fatalf("expected middle percentile normal, got %q", got)
	}
	if got := computerDifficultyForPlayer(entries, "p7"); got != computerDifficultyEasy {
		t.Fatalf("expected lower percentile easy, got %q", got)
	}
	if got := computerDifficultyForPlayer(entries, "missing"); got != computerDifficultyEasy {
		t.Fatalf("expected missing player easy, got %q", got)
	}
}

func TestComputerAnswerChoiceRespectsAccuracyAndEliminations(t *testing.T) {
	question := content.QuizQuestion{
		ID:            "quiz-001",
		CorrectChoice: "A",
	}
	player := mongomodel.MatchPlayer{
		PlayerID: computerPlayerID,
		Kind:     mongomodel.MatchPlayerKindComputer,
	}
	match := mongomodel.Match{
		ID: "match_123",
		EliminatedChoices: []mongomodel.MatchEliminatedChoice{
			{QuestionID: "quiz-001", PlayerID: computerPlayerID, Choices: []string{"B"}},
		},
	}

	if got := computerAnswerChoice(match, question, player, 100); got != "A" {
		t.Fatalf("expected perfect accuracy to pick correct choice, got %q", got)
	}
	got := computerAnswerChoice(match, question, player, 0)
	if got == "A" || got == "B" || got == "" {
		t.Fatalf("expected wrong non-eliminated choice, got %q", got)
	}
}

func TestNormalizeSitoneLoadoutAllowsDuplicateSlots(t *testing.T) {
	got, err := normalizeSitoneLoadout([]string{
		" stone_engineering_base ",
		"stone_engineering_base",
		"",
	})
	if err != nil {
		t.Fatalf("normalize sitone loadout: %v", err)
	}

	want := []string{"stone_engineering_base", "stone_engineering_base"}
	if len(got) != len(want) {
		t.Fatalf("expected %d sitones, got %#v", len(want), got)
	}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("unexpected sitone at index %d: got %q want %q", index, got[index], want[index])
		}
	}
}

func TestMaxScoreThroughCurrentQuestion(t *testing.T) {
	match := mongomodel.Match{
		Status:               mongomodel.MatchStatusActive,
		QuestionIDs:          []string{"quiz-001", "quiz-002", "quiz-003"},
		CurrentQuestionIndex: 1,
	}

	if got := maxScorePerQuestion(); got != 175 {
		t.Fatalf("expected max score per question 175, got %d", got)
	}
	if got := maxScoreThroughCurrentQuestion(match, battleEffects{}); got != 350 {
		t.Fatalf("expected active max score 350, got %d", got)
	}

	match.Status = mongomodel.MatchStatusCompleted
	if got := maxScoreThroughCurrentQuestion(match, battleEffects{}); got != 525 {
		t.Fatalf("expected completed max score 525, got %d", got)
	}
	if got := maxScoreThroughCurrentQuestion(match, battleEffects{AnswerScoreBonusPercent: 30}); got != 681 {
		t.Fatalf("expected boosted completed max score 681, got %d", got)
	}
	if got := maxScoreThroughCurrentQuestion(match, battleEffects{AnswerScoreBonusPercent: 5}); got != 549 {
		t.Fatalf("expected per-question rounded completed max score 549, got %d", got)
	}
}

func TestPickQuestionIDs(t *testing.T) {
	handler := New(Dependencies{Content: loadTestContent(t)})

	ids, err := handler.pickQuestionIDs()
	if err != nil {
		t.Fatalf("pick question ids: %v", err)
	}
	if len(ids) != matchQuestionCount {
		t.Fatalf("expected %d question ids, got %d", matchQuestionCount, len(ids))
	}

	seen := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		if _, ok := seen[id]; ok {
			t.Fatalf("expected unique question ids, got duplicate %q", id)
		}
		seen[id] = struct{}{}
		if _, ok := handler.content.GetQuizQuestion(id); !ok {
			t.Fatalf("expected question %q to exist", id)
		}
	}
}

func TestShuffleQuizQuestionsReturnsRandomnessError(t *testing.T) {
	randomErr := errors.New("entropy unavailable")
	questions := []content.QuizQuestion{
		{ID: "quiz-001"},
		{ID: "quiz-002"},
	}

	err := shuffleQuizQuestions(questions, errReader{err: randomErr})
	if !errors.Is(err, randomErr) {
		t.Fatalf("expected random error, got %v", err)
	}
}

func TestMatchResultsRevealCompletedAnswers(t *testing.T) {
	handler := New(Dependencies{Content: loadTestContent(t)})
	match := mongomodel.Match{
		ID:          "match_123",
		Status:      mongomodel.MatchStatusCompleted,
		QuestionIDs: []string{"quiz-001"},
		Players: []mongomodel.MatchPlayer{
			{PlayerID: "P1", Nickname: "Alice", Score: 160},
			{PlayerID: "P2", Nickname: "Bob", Score: 0},
		},
	}
	answeredAt := time.Date(2026, 6, 2, 10, 0, 3, 0, time.UTC)
	answers := map[string]map[string]mongomodel.MatchAnswer{
		"quiz-001": {
			"P1": {
				PlayerID:      "P1",
				QuestionID:    "quiz-001",
				Choice:        "A",
				Correct:       true,
				Score:         160,
				ElapsedMillis: 3000,
				AnsweredAt:    answeredAt,
			},
		},
	}

	results, err := handler.matchResults(match, answers, map[string]string{"P1": "/avatar/alice.png"})
	if err != nil {
		t.Fatalf("match results: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].CorrectChoice == "" || results[0].Explanation == "" {
		t.Fatalf("expected completed result to reveal correct choice and explanation: %#v", results[0])
	}
	if len(results[0].Answers) != 2 {
		t.Fatalf("expected 2 player answer rows, got %d", len(results[0].Answers))
	}
	if results[0].Answers[0].Choice != "A" || !results[0].Answers[0].Correct {
		t.Fatalf("expected first answer to be revealed, got %#v", results[0].Answers[0])
	}
	if results[0].Answers[0].Nickname != "Alice" || results[0].Answers[1].Nickname != "Bob" {
		t.Fatalf("expected answer rows to include nicknames, got %#v", results[0].Answers)
	}
	if results[0].Answers[0].AvatarURL != "/avatar/alice.png" {
		t.Fatalf("expected answer row to include avatar URL, got %#v", results[0].Answers[0])
	}
}

func TestMatchQuestionResultRevealsCurrentRoundAnswers(t *testing.T) {
	handler := New(Dependencies{Content: loadTestContent(t)})
	match := mongomodel.Match{
		ID:          "match_123",
		Status:      mongomodel.MatchStatusActive,
		Phase:       mongomodel.MatchPhaseRevealing,
		QuestionIDs: []string{"quiz-001"},
		Players: []mongomodel.MatchPlayer{
			{PlayerID: "P1", Nickname: "Alice", Score: 160},
			{PlayerID: "P2", Nickname: "Bob", Score: 0},
		},
	}
	answeredAt := time.Date(2026, 6, 2, 10, 0, 3, 0, time.UTC)
	answers := map[string]map[string]mongomodel.MatchAnswer{
		"quiz-001": {
			"P1": {
				PlayerID:      "P1",
				QuestionID:    "quiz-001",
				Choice:        "A",
				Correct:       true,
				Score:         160,
				ElapsedMillis: 3000,
				AnsweredAt:    answeredAt,
			},
		},
	}

	result, err := handler.matchQuestionResult(match, "quiz-001", answers, map[string]string{"P1": "/avatar/alice.png"})
	if err != nil {
		t.Fatalf("match question result: %v", err)
	}
	if result.CorrectChoice == "" || result.Explanation == "" {
		t.Fatalf("expected reveal result to include correct answer and explanation: %#v", result)
	}
	if len(result.Answers) != 2 {
		t.Fatalf("expected 2 player answer rows, got %d", len(result.Answers))
	}
	if result.Answers[0].Choice != "A" || !result.Answers[0].Correct || result.Answers[0].Score != 160 {
		t.Fatalf("expected first player answer to be revealed, got %#v", result.Answers[0])
	}
	if result.Answers[1].Choice != "" || result.Answers[1].Correct || result.Answers[1].Score != 0 {
		t.Fatalf("expected unanswered player row with zero score, got %#v", result.Answers[1])
	}
	if result.Answers[0].AvatarURL != "/avatar/alice.png" {
		t.Fatalf("expected answer row to include avatar URL, got %#v", result.Answers[0])
	}
}

func TestViewerEliminatedChoicesOnlyExposesViewerChoices(t *testing.T) {
	match := mongomodel.Match{
		Players: []mongomodel.MatchPlayer{
			{PlayerID: "P1", Nickname: "Alice", Score: 0},
			{PlayerID: "P2", Nickname: "Bob", Score: 0},
		},
		EliminatedChoices: []mongomodel.MatchEliminatedChoice{
			{
				QuestionID:        "quiz-001",
				PlayerID:          "P1",
				Choices:           []string{"B"},
				SourceSitoneNames: []string{"靈光型小石"},
			},
			{
				QuestionID:        "quiz-001",
				PlayerID:          "P2",
				Choices:           []string{"C"},
				SourceSitoneNames: []string{"烏龜小石"},
			},
		},
	}
	currentEliminations := eliminatedChoicesByQuestionPlayer(match)["quiz-001"]

	choices, sources := viewerEliminatedChoices("P1", "P1", currentEliminations)
	if len(choices) != 1 || choices[0] != "B" {
		t.Fatalf("expected viewer eliminated choice B, got %#v", choices)
	}
	if len(sources) != 1 || sources[0] != "靈光型小石" {
		t.Fatalf("expected viewer eliminated source, got %#v", sources)
	}

	choices, sources = viewerEliminatedChoices("P2", "P1", currentEliminations)
	if len(choices) != 0 || len(sources) != 0 {
		t.Fatalf("expected opponent eliminated choices to be hidden, got choices=%#v sources=%#v", choices, sources)
	}
}

func TestBrokerPublishesEvents(t *testing.T) {
	broker := NewBroker()
	events, unsubscribe := broker.Subscribe("match_123")
	defer unsubscribe()

	broker.Publish("match_123", Event{Name: "player_answered", Match: mongomodel.Match{ID: "match_123"}})

	select {
	case event := <-events:
		if event.Name != "player_answered" {
			t.Fatalf("expected player_answered event, got %q", event.Name)
		}
	case <-time.After(time.Second):
		t.Fatal("expected event")
	}
}

func loadTestContent(t *testing.T) *content.Store {
	t.Helper()

	return testcontent.Load(t)
}

type errReader struct {
	err error
}

func (r errReader) Read(_ []byte) (int, error) {
	return 0, r.err
}
