//go:build e2e

package e2e

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	httpserver "github.com/sitcon-tw/camp2026-game/internal/http"
	mongomodel "github.com/sitcon-tw/camp2026-game/internal/mongodb/model"
	"github.com/sitcon-tw/camp2026-game/internal/testcontent"
	"github.com/testcontainers/testcontainers-go"
	tcmongodb "github.com/testcontainers/testcontainers-go/modules/mongodb"
)

const (
	playerAID    = "player-a"
	playerBID    = "player-b"
	playerCID    = "player-c"
	playerAToken = "auth-token-player-a-123456"
	playerBToken = "auth-token-player-b-123456"
	playerCToken = "auth-token-player-c-123456"
)

func TestMatchFlowE2E(t *testing.T) {
	ctx := t.Context()
	mongoClient, db := startMongo(t, ctx)
	seedPlayersAndTeams(t, ctx, db)

	server := newE2EServer(t, mongoClient, db)
	defer server.Close()

	playerACookie := login(t, server.URL, playerAToken)
	playerBCookie := login(t, server.URL, playerBToken)
	playerCCookie := login(t, server.URL, playerCToken)

	assertShopPurchaseFlow(t, ctx, db, server.URL, playerACookie)

	var created matchState
	body := postJSON(t, server.URL+"/api/matches", nil, []*http.Cookie{playerACookie}, http.StatusCreated)
	decodeJSON(t, body, &created)
	if created.Status != "waiting" {
		t.Fatalf("expected created match status waiting, got %q", created.Status)
	}
	if created.MatchID == "" || created.Code == "" {
		t.Fatalf("expected match id and code, got %#v", created)
	}
	if len(created.Players) != 1 || created.Players[0].PlayerID != playerAID || created.Players[0].Ready {
		t.Fatalf("unexpected created players: %#v", created.Players)
	}

	var joined matchState
	body = postJSON(t, server.URL+"/api/matches/join", map[string]string{
		"code": created.Code,
	}, []*http.Cookie{playerBCookie}, http.StatusOK)
	decodeJSON(t, body, &joined)
	if joined.Status != "waiting" || len(joined.Players) != 2 {
		t.Fatalf("expected joined waiting match with 2 players, got %#v", joined)
	}

	assertInitialSSEEvent(t, server.URL, created.MatchID, playerACookie)

	var readyA matchState
	body = postJSON(t, server.URL+"/api/matches/"+created.MatchID+"/ready", nil, []*http.Cookie{playerACookie}, http.StatusOK)
	decodeJSON(t, body, &readyA)
	if readyA.Status != "waiting" {
		t.Fatalf("expected match to wait for second ready, got %q", readyA.Status)
	}
	if !readyA.player(playerAID).Ready || readyA.player(playerBID).Ready {
		t.Fatalf("unexpected ready state after player A ready: %#v", readyA.Players)
	}

	var readyB matchState
	body = postJSON(t, server.URL+"/api/matches/"+created.MatchID+"/ready", nil, []*http.Cookie{playerBCookie}, http.StatusOK)
	decodeJSON(t, body, &readyB)
	if readyB.Status != "active" {
		t.Fatalf("expected match active after both ready, got %q", readyB.Status)
	}
	if readyB.QuestionCount != 10 || readyB.CurrentQuestion == nil {
		t.Fatalf("expected active match with 10 questions and current question, got %#v", readyB)
	}
	if bytes.Contains(body, []byte("correctChoice")) || bytes.Contains(body, []byte("explanation")) {
		t.Fatalf("active state must not reveal answers, got %s", string(body))
	}

	body = getJSON(t, server.URL+"/api/matches/open", []*http.Cookie{playerACookie}, http.StatusOK)
	var openA matchState
	decodeJSON(t, body, &openA)
	if openA.MatchID != created.MatchID || openA.Status != "active" {
		t.Fatalf("expected player A open match to be active %s, got %#v", created.MatchID, openA)
	}

	body = getJSON(t, server.URL+"/api/matches/open", []*http.Cookie{playerBCookie}, http.StatusOK)
	var openB matchState
	decodeJSON(t, body, &openB)
	if openB.MatchID != created.MatchID || openB.Status != "active" {
		t.Fatalf("expected player B open match to be active %s, got %#v", created.MatchID, openB)
	}

	body = postJSON(t, server.URL+"/api/matches/join", map[string]string{
		"code": created.Code,
	}, []*http.Cookie{playerBCookie}, http.StatusOK)
	var rejoined matchState
	decodeJSON(t, body, &rejoined)
	if rejoined.MatchID != created.MatchID || rejoined.Status != "active" || len(rejoined.Players) != 2 {
		t.Fatalf("expected participant to rejoin active match, got %#v", rejoined)
	}

	postJSON(t, server.URL+"/api/matches/join", map[string]string{
		"code": created.Code,
	}, []*http.Cookie{playerCCookie}, http.StatusConflict)

	for i := 0; i < 10; i++ {
		var state matchState
		body = getJSON(t, server.URL+"/api/matches/"+created.MatchID, []*http.Cookie{playerACookie}, http.StatusOK)
		decodeJSON(t, body, &state)
		if state.Status != "active" || state.CurrentQuestion == nil {
			t.Fatalf("question %d: expected active state with current question, got %#v", i, state)
		}

		questionID := state.CurrentQuestion.QuestionID
		postJSON(t, server.URL+"/api/matches/"+created.MatchID+"/answers", map[string]string{
			"questionId": questionID,
			"choice":     "A",
		}, []*http.Cookie{playerACookie}, http.StatusAccepted)

		body = getJSON(t, server.URL+"/api/matches/"+created.MatchID, []*http.Cookie{playerBCookie}, http.StatusOK)
		decodeJSON(t, body, &state)
		if !state.player(playerAID).AnsweredCurrentQuestion {
			t.Fatalf("question %d: expected player B to see player A answered, got %#v", i, state.Players)
		}

		postJSON(t, server.URL+"/api/matches/"+created.MatchID+"/answers", map[string]string{
			"questionId": questionID,
			"choice":     "B",
		}, []*http.Cookie{playerBCookie}, http.StatusAccepted)
	}

	body = getJSON(t, server.URL+"/api/matches/"+created.MatchID, []*http.Cookie{playerACookie}, http.StatusOK)
	var completed matchState
	decodeJSON(t, body, &completed)
	if completed.Status != "completed" {
		t.Fatalf("expected completed match, got %#v", completed)
	}
	if len(completed.Results) != 10 {
		t.Fatalf("expected 10 results, got %d", len(completed.Results))
	}
	for _, player := range completed.Players {
		if player.Score == nil {
			t.Fatalf("expected completed player score, got %#v", player)
		}
	}
	for _, result := range completed.Results {
		if result.CorrectChoice == "" || result.Explanation == "" {
			t.Fatalf("expected result to reveal correct choice and explanation, got %#v", result)
		}
		if len(result.Answers) != 2 {
			t.Fatalf("expected two answer rows, got %#v", result)
		}
	}

	getJSON(t, server.URL+"/api/matches/open", []*http.Cookie{playerACookie}, http.StatusNotFound)

	assertDatabaseState(t, ctx, db, created.MatchID, completed)
}

func startMongo(t *testing.T, ctx context.Context) (*mongo.Client, *mongo.Database) {
	t.Helper()

	container, err := tcmongodb.Run(ctx, "mongo:7.0", tcmongodb.WithReplicaSet("rs0"))
	if err != nil {
		t.Fatalf("start mongodb container: %v", err)
	}
	t.Cleanup(func() {
		if err := testcontainers.TerminateContainer(container); err != nil {
			t.Logf("terminate mongodb container: %v", err)
		}
	})

	uri, err := container.ConnectionString(ctx)
	if err != nil {
		t.Fatalf("mongodb connection string: %v", err)
	}

	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		t.Fatalf("connect mongodb: %v", err)
	}
	t.Cleanup(func() {
		_ = client.Disconnect(context.Background())
	})
	if err := client.Ping(ctx, nil); err != nil {
		t.Fatalf("ping mongodb: %v", err)
	}

	dbName := "camp2026_e2e_" + strings.ReplaceAll(bson.NewObjectID().Hex(), "-", "")
	db := client.Database(dbName)
	t.Cleanup(func() {
		_ = db.Drop(context.Background())
	})

	return client, db
}

func seedPlayersAndTeams(t *testing.T, ctx context.Context, db *mongo.Database) {
	t.Helper()

	_, err := db.Collection(mongomodel.TeamsCollection).InsertMany(ctx, []any{
		mongomodel.Team{ID: "team-a", Name: "Team A"},
		mongomodel.Team{ID: "team-b", Name: "Team B"},
	})
	if err != nil {
		t.Fatalf("seed teams: %v", err)
	}

	_, err = db.Collection(mongomodel.PlayersCollection).InsertMany(ctx, []any{
		mongomodel.Player{
			ID:          playerAID,
			AuthToken:   playerAToken,
			QRCodeToken: "qr-token-player-a",
			Nickname:    "Alice",
			TeamID:      "team-a",
			AvatarURL:   "https://example.test/avatar/alice.png",
		},
		mongomodel.Player{
			ID:          playerBID,
			AuthToken:   playerBToken,
			QRCodeToken: "qr-token-player-b",
			Nickname:    "Bob",
			TeamID:      "team-b",
			AvatarURL:   "https://example.test/avatar/bob.png",
		},
		mongomodel.Player{
			ID:          playerCID,
			AuthToken:   playerCToken,
			QRCodeToken: "qr-token-player-c",
			Nickname:    "Carol",
			TeamID:      "team-b",
			AvatarURL:   "https://example.test/avatar/carol.png",
		},
	})
	if err != nil {
		t.Fatalf("seed players: %v", err)
	}

	_, err = db.Collection(mongomodel.PlayerItemsCollection).InsertOne(ctx, mongomodel.PlayerItem{
		ID:       "player-a-item-adventure-backpack",
		PlayerID: playerAID,
		ItemID:   "item_adventure_backpack",
		Quantity: 3,
	})
	if err != nil {
		t.Fatalf("seed player items: %v", err)
	}

	_, err = db.Collection(mongomodel.OpenPowerRecordsCollection).InsertOne(ctx, mongomodel.OpenPowerRecord{
		ID:        "player-a-open-power-seed",
		PlayerID:  playerAID,
		Amount:    500,
		Reason:    "e2e_seed",
		Source:    "e2e",
		CreatedAt: time.Now(),
	})
	if err != nil {
		t.Fatalf("seed open power: %v", err)
	}

	_, err = db.Collection(mongomodel.PlayerSitonesCollection).InsertMany(ctx, []any{
		mongomodel.PlayerSitone{
			ID:       "player-a-stone_engineering_base",
			PlayerID: playerAID,
			SitoneID: "stone_engineering_base",
			Quantity: 1,
		},
		mongomodel.PlayerSitone{
			ID:       "player-b-stone_resonance_base",
			PlayerID: playerBID,
			SitoneID: "stone_resonance_base",
			Quantity: 1,
		},
	})
	if err != nil {
		t.Fatalf("seed player sitones: %v", err)
	}
}

func newE2EServer(t *testing.T, mongoClient *mongo.Client, db *mongo.Database) *httptest.Server {
	t.Helper()

	router := httpserver.NewRouter(httpserver.Dependencies{
		Log:         slog.New(slog.NewTextHandler(io.Discard, nil)),
		Content:     testcontent.Load(t),
		MongoClient: mongoClient,
		MongoDB:     db,
	})
	return httptest.NewServer(router)
}

func login(t *testing.T, serverURL string, token string) *http.Cookie {
	t.Helper()

	reqBody, err := json.Marshal(map[string]string{"token": token})
	if err != nil {
		t.Fatalf("marshal login body: %v", err)
	}
	req, err := http.NewRequest(http.MethodPost, serverURL+"/api/auth/login", bytes.NewReader(reqBody))
	if err != nil {
		t.Fatalf("new login request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("login request: %v", err)
	}
	defer func() {
		_ = res.Body.Close()
	}()
	payload, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("read login response: %v", err)
	}
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected login status %d, got %d: %s", http.StatusOK, res.StatusCode, string(payload))
	}
	if !bytes.Contains(payload, []byte(`"player"`)) {
		t.Fatalf("expected login response to include player, got %s", string(payload))
	}

	for _, cookie := range res.Cookies() {
		if cookie.Name == "camp2026_auth" {
			return cookie
		}
	}
	t.Fatal("expected camp2026_auth cookie")
	return nil
}

func postJSON(t *testing.T, url string, body any, cookies []*http.Cookie, wantStatus int) []byte {
	t.Helper()
	return requestJSON(t, http.MethodPost, url, body, cookies, wantStatus)
}

func getJSON(t *testing.T, url string, cookies []*http.Cookie, wantStatus int) []byte {
	t.Helper()
	return requestJSON(t, http.MethodGet, url, nil, cookies, wantStatus)
}

func requestJSON(t *testing.T, method string, url string, body any, cookies []*http.Cookie, wantStatus int) []byte {
	t.Helper()

	var reader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal request body: %v", err)
		}
		reader = bytes.NewReader(payload)
	}

	req, err := http.NewRequest(method, url, reader)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, url, err)
	}
	defer func() {
		_ = res.Body.Close()
	}()

	payload, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("read response body: %v", err)
	}
	if res.StatusCode != wantStatus {
		t.Fatalf("%s %s: expected status %d, got %d: %s", method, url, wantStatus, res.StatusCode, string(payload))
	}
	return payload
}

func decodeJSON(t *testing.T, body []byte, out any) {
	t.Helper()

	if err := json.Unmarshal(body, out); err != nil {
		t.Fatalf("decode json %s: %v", string(body), err)
	}
}

func assertInitialSSEEvent(t *testing.T, serverURL string, matchID string, cookie *http.Cookie) {
	t.Helper()

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, serverURL+"/api/matches/"+matchID+"/events", nil)
	if err != nil {
		t.Fatalf("new sse request: %v", err)
	}
	req.AddCookie(cookie)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("sse request: %v", err)
	}
	defer func() {
		_ = res.Body.Close()
	}()
	if res.StatusCode != http.StatusOK {
		payload, _ := io.ReadAll(res.Body)
		t.Fatalf("expected sse status %d, got %d: %s", http.StatusOK, res.StatusCode, string(payload))
	}
	if contentType := res.Header.Get("Content-Type"); !strings.HasPrefix(contentType, "text/event-stream") {
		t.Fatalf("expected text/event-stream, got %q", contentType)
	}

	reader := bufio.NewReader(res.Body)
	line, err := reader.ReadString('\n')
	if err != nil {
		t.Fatalf("read sse event line: %v", err)
	}
	if strings.TrimSpace(line) != "event: match_updated" {
		t.Fatalf("expected initial match_updated event, got %q", line)
	}
}

func assertShopPurchaseFlow(t *testing.T, ctx context.Context, db *mongo.Database, serverURL string, cookie *http.Cookie) {
	t.Helper()

	body := getJSON(t, serverURL+"/api/shop/items", []*http.Cookie{cookie}, http.StatusOK)
	var list shopItemList
	decodeJSON(t, body, &list)
	if len(list.Items) != 49 {
		t.Fatalf("expected 49 shop items, got %#v", list.Items)
	}
	if list.Items[0].ID != "item_adventure_backpack" || list.Items[0].PriceOpenPower != 150 {
		t.Fatalf("unexpected first shop item: %#v", list.Items[0])
	}

	body = getJSON(t, serverURL+"/api/shop/items/item_adventure_backpack", []*http.Cookie{cookie}, http.StatusOK)
	var detail shopItemDetail
	decodeJSON(t, body, &detail)
	if detail.Item.ID != "item_adventure_backpack" || detail.Item.PriceOpenPower != 150 {
		t.Fatalf("unexpected shop item detail: %#v", detail)
	}

	body = postJSON(t, serverURL+"/api/shop/purchases", map[string]string{
		"itemId": "item_adventure_backpack",
	}, []*http.Cookie{cookie}, http.StatusCreated)
	var purchase shopPurchase
	decodeJSON(t, body, &purchase)
	if purchase.PurchaseID == "" ||
		purchase.ItemID != "item_adventure_backpack" ||
		purchase.Quantity != 1 ||
		purchase.PriceOpenPower != 150 ||
		purchase.OpenPower != 350 {
		t.Fatalf("unexpected purchase response: %#v", purchase)
	}

	var storedPurchase mongomodel.ShopPurchase
	if err := db.Collection(mongomodel.ShopPurchasesCollection).
		FindOne(ctx, bson.M{"_id": purchase.PurchaseID}).
		Decode(&storedPurchase); err != nil {
		t.Fatalf("find shop purchase: %v", err)
	}
	if storedPurchase.PlayerID != playerAID || storedPurchase.ItemID != "item_adventure_backpack" || storedPurchase.PriceOpenPower != 150 {
		t.Fatalf("unexpected stored purchase: %#v", storedPurchase)
	}

	var deduction mongomodel.OpenPowerRecord
	if err := db.Collection(mongomodel.OpenPowerRecordsCollection).
		FindOne(ctx, bson.M{"source": purchase.PurchaseID, "reason": "shop_purchase"}).
		Decode(&deduction); err != nil {
		t.Fatalf("find open power deduction: %v", err)
	}
	if deduction.PlayerID != playerAID || deduction.Amount != -150 {
		t.Fatalf("unexpected open power deduction: %#v", deduction)
	}

	var item mongomodel.PlayerItem
	if err := db.Collection(mongomodel.PlayerItemsCollection).
		FindOne(ctx, bson.M{"player_id": playerAID, "item_id": "item_adventure_backpack"}).
		Decode(&item); err != nil {
		t.Fatalf("find player item: %v", err)
	}
	if item.Quantity != 4 {
		t.Fatalf("expected purchased item quantity 4, got %#v", item)
	}

	body = getJSON(t, serverURL+"/api/me/items", []*http.Cookie{cookie}, http.StatusOK)
	var meItems playerItemList
	decodeJSON(t, body, &meItems)
	if len(meItems.Items) != 1 || meItems.Items[0].ItemID != "item_adventure_backpack" || meItems.Items[0].Quantity != 4 {
		t.Fatalf("expected me items to include purchased item, got %#v", meItems.Items)
	}
}

func assertDatabaseState(t *testing.T, ctx context.Context, db *mongo.Database, matchID string, completed matchState) {
	t.Helper()

	var match mongomodel.Match
	if err := db.Collection(mongomodel.MatchesCollection).FindOne(ctx, bson.M{"_id": matchID}).Decode(&match); err != nil {
		t.Fatalf("find completed match: %v", err)
	}
	if match.Status != mongomodel.MatchStatusCompleted {
		t.Fatalf("expected persisted match completed, got %#v", match)
	}

	answerCount, err := db.Collection(mongomodel.MatchAnswersCollection).CountDocuments(ctx, bson.M{"match_id": matchID})
	if err != nil {
		t.Fatalf("count match answers: %v", err)
	}
	if answerCount != 20 {
		t.Fatalf("expected 20 match answers, got %d", answerCount)
	}

	winner := match.Players[0]
	topCount := 1
	for _, player := range match.Players[1:] {
		switch {
		case player.Score > winner.Score:
			winner = player
			topCount = 1
		case player.Score == winner.Score:
			topCount++
		}
	}
	if topCount != 1 {
		t.Fatalf("expected completed match to have a clear winner, got %#v", match.Players)
	}

	for _, player := range completed.Players {
		if player.OpenPowerReward == nil {
			t.Fatalf("expected completed player reward field, got %#v", player)
		}
		if player.PlayerID == winner.PlayerID {
			if *player.OpenPowerReward <= 0 {
				t.Fatalf("expected winner positive open power reward, got %#v", player)
			}
			continue
		}
		if *player.OpenPowerReward != 0 {
			t.Fatalf("expected loser open power reward 0, got %#v", player)
		}
	}

	sourcePattern := "^quiz_match:" + regexp.QuoteMeta(matchID) + ":player:"
	cursor, err := db.Collection(mongomodel.OpenPowerRecordsCollection).Find(ctx, bson.M{
		"source": bson.M{"$regex": sourcePattern},
		"reason": "quiz_match_completed",
	})
	if err != nil {
		t.Fatalf("find open power records: %v", err)
	}
	defer func() {
		_ = cursor.Close(ctx)
	}()

	var records []mongomodel.OpenPowerRecord
	if err := cursor.All(ctx, &records); err != nil {
		t.Fatalf("decode open power records: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 open power record, got %#v", records)
	}
	record := records[0]
	if record.PlayerID != winner.PlayerID {
		t.Fatalf("expected reward record for winner %q, got %#v", winner.PlayerID, record)
	}
	if record.Source != "quiz_match:"+matchID+":player:"+winner.PlayerID {
		t.Fatalf("unexpected reward source: %#v", record)
	}
	if record.Amount <= 0 {
		t.Fatalf("expected positive open power amount, got %#v", record)
	}
}

type matchState struct {
	MatchID              string         `json:"matchId"`
	Code                 string         `json:"code"`
	Status               string         `json:"status"`
	Players              []matchPlayer  `json:"players"`
	CurrentQuestionIndex int            `json:"currentQuestionIndex"`
	QuestionCount        int            `json:"questionCount"`
	CurrentQuestion      *matchQuestion `json:"currentQuestion"`
	Results              []matchResult  `json:"results"`
}

func (s matchState) player(playerID string) matchPlayer {
	for _, player := range s.Players {
		if player.PlayerID == playerID {
			return player
		}
	}
	panic(fmt.Sprintf("player %q not found in state", playerID))
}

type matchPlayer struct {
	PlayerID                string `json:"playerId"`
	Ready                   bool   `json:"ready"`
	AnsweredCurrentQuestion bool   `json:"answeredCurrentQuestion"`
	Score                   *int   `json:"score"`
	OpenPowerReward         *int   `json:"openPowerReward"`
}

type shopItemList struct {
	Items []shopItem `json:"items"`
}

type shopItemDetail struct {
	Item shopItem `json:"item"`
}

type shopItem struct {
	ID             string `json:"id"`
	PriceOpenPower int    `json:"priceOpenPower"`
}

type shopPurchase struct {
	PurchaseID     string `json:"purchaseId"`
	ItemID         string `json:"itemId"`
	Quantity       int    `json:"quantity"`
	PriceOpenPower int    `json:"priceOpenPower"`
	OpenPower      int    `json:"openPower"`
}

type playerItemList struct {
	Items []playerItem `json:"items"`
}

type playerItem struct {
	ItemID   string `json:"itemId"`
	Quantity int    `json:"quantity"`
}

type matchQuestion struct {
	QuestionID string `json:"questionId"`
	Prompt     string `json:"prompt"`
	ChoiceA    string `json:"choiceA"`
	ChoiceB    string `json:"choiceB"`
	ChoiceC    string `json:"choiceC"`
	ChoiceD    string `json:"choiceD"`
}

type matchResult struct {
	QuestionID    string        `json:"questionId"`
	CorrectChoice string        `json:"correctChoice"`
	Explanation   string        `json:"explanation"`
	Answers       []matchAnswer `json:"answers"`
}

type matchAnswer struct {
	PlayerID      string `json:"playerId"`
	Choice        string `json:"choice"`
	Correct       bool   `json:"correct"`
	Score         int    `json:"score"`
	ElapsedMillis int64  `json:"elapsedMillis"`
}
