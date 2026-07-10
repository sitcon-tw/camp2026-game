package me

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/sitcon-tw/camp2026-game/internal/achievement"
	"github.com/sitcon-tw/camp2026-game/internal/http/httpx"
	"github.com/sitcon-tw/camp2026-game/internal/http/playerevents"
	mongomodel "github.com/sitcon-tw/camp2026-game/internal/mongodb/model"
)

// Events godoc
// @Summary Stream player events
// @Description Streams player reward notifications with Server-Sent Events.
// @Tags me
// @Produce text/event-stream
// @Security AuthCookieAuth
// @Success 200 {string} string "SSE event stream"
// @Failure 401 {object} httpx.ProblemDetails
// @Failure 500 {object} httpx.ProblemDetails
// @Router /me/events [get]
func (h *Handler) Events(w http.ResponseWriter, r *http.Request) {
	player, ok := currentPlayer(w, r)
	if !ok {
		return
	}
	if h.broker == nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("event stream is unavailable", "player_events_broker_unavailable", errors.New("player events broker is unavailable")))
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		httpx.WriteProblem(w, r, httpx.InternalServerError("event stream is unavailable", "player_events_flusher_unavailable", errors.New("response writer does not implement http.Flusher")))
		return
	}

	// Best-effort: clear write deadline so the SSE stream is not killed by WriteTimeout.
	// Ignored if the underlying ResponseWriter does not support it.
	_ = http.NewResponseController(w).SetWriteDeadline(time.Time{})

	eventCh, unsubscribe := h.broker.Subscribe(player.ID)
	defer unsubscribe()

	connectedAt := time.Now().UTC()
	if _, err := h.reconcileCodexAchievements(r.Context(), player.ID, connectedAt); err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("event stream is unavailable", "player_events_achievement_reconcile_failed", err))
		return
	}
	pendingEvents, err := h.pendingStaffRewardEvents(r.Context(), player.ID, connectedAt)
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("event stream is unavailable", "player_events_pending_rewards_failed", err))
		return
	}
	pendingTransferEvents, err := h.pendingOpenPowerTransferEvents(r.Context(), player.ID, connectedAt)
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("event stream is unavailable", "player_events_pending_transfers_failed", err))
		return
	}
	pendingTrimEvents, err := h.pendingInventoryTrimEvents(r.Context(), player.ID, connectedAt)
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("event stream is unavailable", "player_events_pending_trims_failed", err))
		return
	}
	pendingAchievementEvents, err := h.pendingAchievementEvents(r.Context(), player.ID, connectedAt, true)
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("event stream is unavailable", "player_events_pending_achievements_failed", err))
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	for _, event := range pendingEvents {
		if !writePlayerEventSSE(w, event.Event) {
			return
		}
		flusher.Flush()
		if err := h.markStaffRewardNotified(r.Context(), event.RewardID, time.Now().UTC()); err != nil {
			return
		}
	}
	for _, event := range pendingTransferEvents {
		if !writePlayerEventSSE(w, event.Event) {
			return
		}
		flusher.Flush()
		if err := h.markOpenPowerTransferNotified(r.Context(), event.TransferID, time.Now().UTC()); err != nil {
			return
		}
	}
	for _, event := range pendingTrimEvents {
		if !writePlayerEventSSE(w, event.Event) {
			return
		}
		flusher.Flush()
		if err := h.markInventoryTrimNotified(r.Context(), event.TrimID, time.Now().UTC()); err != nil {
			return
		}
	}
	if !h.writeAchievementEvents(r.Context(), w, flusher, pendingAchievementEvents) {
		return
	}

	heartbeat := time.NewTicker(20 * time.Second)
	defer heartbeat.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case event, ok := <-eventCh:
			if !ok {
				return
			}
			if !writePlayerEventSSE(w, event) {
				return
			}
			flusher.Flush()
		case <-heartbeat.C:
			now := time.Now().UTC()
			if _, err := h.reconcileCodexAchievements(r.Context(), player.ID, now); err == nil {
				pending, pendingErr := h.pendingAchievementEvents(r.Context(), player.ID, now, false)
				if pendingErr == nil && !h.writeAchievementEvents(r.Context(), w, flusher, pending) {
					return
				}
			}
			_, _ = fmt.Fprint(w, "event: keepalive\ndata: {}\n\n")
			flusher.Flush()
		}
	}
}

type pendingStaffRewardEvent struct {
	RewardID string
	Event    playerevents.Event
}

type pendingInventoryTrimEvent struct {
	TrimID string
	Event  playerevents.Event
}

type pendingOpenPowerTransferEvent struct {
	TransferID string
	Event      playerevents.Event
}

type pendingAchievementEvent struct {
	AchievementID string
	Event         playerevents.Event
}

func (h *Handler) reconcileCodexAchievements(ctx context.Context, playerID string, now time.Time) (int, error) {
	if h.db == nil || h.content == nil {
		return 0, nil
	}
	sitones := h.content.ListSitones()
	ids := make([]string, 0, len(sitones))
	for _, sitone := range sitones {
		ids = append(ids, sitone.ID)
	}
	return achievement.ReconcileCodex(ctx, h.db, playerID, ids, now)
}

func (h *Handler) pendingStaffRewardEvents(ctx context.Context, playerID string, connectedAt time.Time) ([]pendingStaffRewardEvent, error) {
	if h.db == nil || h.content == nil {
		return nil, nil
	}

	cursor, err := h.db.Collection(mongomodel.StaffRewardsCollection).Find(
		ctx,
		bson.M{
			"recipient_player_id":  playerID,
			"notification_pending": true,
			"created_at":           bson.M{"$lte": connectedAt},
		},
		options.Find().
			SetSort(bson.D{{Key: "created_at", Value: 1}, {Key: "sort_order", Value: 1}, {Key: "_id", Value: 1}}).
			SetLimit(100),
	)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = cursor.Close(ctx)
	}()

	var records []mongomodel.StaffReward
	if err := cursor.All(ctx, &records); err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, nil
	}

	staffPlayers, err := h.loadRewardStaffPlayers(ctx, records)
	if err != nil {
		return nil, err
	}

	events := make([]pendingStaffRewardEvent, 0, len(records))
	for _, record := range records {
		event, err := playerevents.StaffRewardGrantedEvent(h.content, record, staffPlayers[record.StaffPlayerID], true)
		if err != nil {
			return nil, err
		}
		events = append(events, pendingStaffRewardEvent{
			RewardID: record.ID,
			Event: playerevents.Event{
				Name:   "reward_granted",
				Reward: &event,
			},
		})
	}
	return events, nil
}

func (h *Handler) pendingOpenPowerTransferEvents(ctx context.Context, playerID string, connectedAt time.Time) ([]pendingOpenPowerTransferEvent, error) {
	if h.db == nil {
		return nil, nil
	}

	cursor, err := h.db.Collection(mongomodel.OpenPowerTransfersCollection).Find(
		ctx,
		bson.M{
			"recipient_player_id":  playerID,
			"notification_pending": true,
			"created_at":           bson.M{"$lte": connectedAt},
		},
		options.Find().
			SetSort(bson.D{{Key: "created_at", Value: 1}, {Key: "_id", Value: 1}}).
			SetLimit(100),
	)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = cursor.Close(ctx)
	}()

	var records []mongomodel.OpenPowerTransfer
	if err := cursor.All(ctx, &records); err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, nil
	}

	senders, err := h.loadOpenPowerTransferSenders(ctx, records)
	if err != nil {
		return nil, err
	}

	events := make([]pendingOpenPowerTransferEvent, 0, len(records))
	for _, record := range records {
		event := playerevents.OpenPowerTransferReceivedEvent(record, senders[record.SenderPlayerID], true)
		events = append(events, pendingOpenPowerTransferEvent{
			TransferID: record.ID,
			Event: playerevents.Event{
				Name:   "reward_granted",
				Reward: &event,
			},
		})
	}
	return events, nil
}

func (h *Handler) pendingInventoryTrimEvents(ctx context.Context, playerID string, connectedAt time.Time) ([]pendingInventoryTrimEvent, error) {
	if h.db == nil {
		return nil, nil
	}

	cursor, err := h.db.Collection(mongomodel.InventoryTrimsCollection).Find(
		ctx,
		bson.M{
			"player_id":            playerID,
			"notification_pending": true,
			"created_at":           bson.M{"$lte": connectedAt},
		},
		options.Find().
			SetSort(bson.D{{Key: "created_at", Value: 1}, {Key: "_id", Value: 1}}).
			SetLimit(100),
	)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = cursor.Close(ctx)
	}()

	var records []mongomodel.InventoryTrim
	if err := cursor.All(ctx, &records); err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, nil
	}

	events := make([]pendingInventoryTrimEvent, 0, len(records))
	for _, record := range records {
		event := playerevents.InventoryTrimmed(record, true)
		events = append(events, pendingInventoryTrimEvent{
			TrimID: record.ID,
			Event: playerevents.Event{
				Name:             playerevents.InventoryTrimmedEventName,
				InventoryTrimmed: &event,
			},
		})
	}
	return events, nil
}

func (h *Handler) pendingAchievementEvents(ctx context.Context, playerID string, connectedAt time.Time, delayed bool) ([]pendingAchievementEvent, error) {
	if h.db == nil {
		return nil, nil
	}

	cursor, err := h.db.Collection(mongomodel.AchievementsCollection).Find(
		ctx,
		bson.M{
			"player_id":            playerID,
			"notification_pending": true,
			"created_at":           bson.M{"$lte": connectedAt},
		},
		options.Find().
			SetSort(bson.D{{Key: "created_at", Value: 1}, {Key: "_id", Value: 1}}).
			SetLimit(100),
	)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = cursor.Close(ctx)
	}()

	var records []mongomodel.Achievement
	if err := cursor.All(ctx, &records); err != nil {
		return nil, err
	}

	events := make([]pendingAchievementEvent, 0, len(records))
	for _, record := range records {
		event := playerevents.AchievementUnlocked(record, delayed)
		events = append(events, pendingAchievementEvent{
			AchievementID: record.ID,
			Event: playerevents.Event{
				Name:                playerevents.AchievementUnlockedEventName,
				AchievementUnlocked: &event,
			},
		})
	}
	return events, nil
}

func (h *Handler) loadRewardStaffPlayers(ctx context.Context, records []mongomodel.StaffReward) (map[string]mongomodel.Player, error) {
	staffIDs := make(map[string]struct{}, len(records))
	for _, record := range records {
		if record.StaffPlayerID != "" {
			staffIDs[record.StaffPlayerID] = struct{}{}
		}
	}
	if len(staffIDs) == 0 {
		return map[string]mongomodel.Player{}, nil
	}

	ids := make([]string, 0, len(staffIDs))
	for id := range staffIDs {
		ids = append(ids, id)
	}
	cursor, err := h.db.Collection(mongomodel.PlayersCollection).Find(
		ctx,
		bson.M{"_id": bson.M{"$in": ids}},
		options.Find().SetProjection(bson.D{
			{Key: "auth_token", Value: 0},
			{Key: "qrcode_token", Value: 0},
			{Key: "default_sitone_ids", Value: 0},
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
	byID := make(map[string]mongomodel.Player, len(players))
	for _, player := range players {
		byID[player.ID] = player
	}
	return byID, nil
}

func (h *Handler) loadOpenPowerTransferSenders(ctx context.Context, records []mongomodel.OpenPowerTransfer) (map[string]mongomodel.Player, error) {
	senderIDs := make(map[string]struct{}, len(records))
	for _, record := range records {
		if record.SenderPlayerID != "" {
			senderIDs[record.SenderPlayerID] = struct{}{}
		}
	}
	if len(senderIDs) == 0 {
		return map[string]mongomodel.Player{}, nil
	}

	ids := make([]string, 0, len(senderIDs))
	for id := range senderIDs {
		ids = append(ids, id)
	}
	cursor, err := h.db.Collection(mongomodel.PlayersCollection).Find(
		ctx,
		bson.M{"_id": bson.M{"$in": ids}},
		options.Find().SetProjection(bson.D{
			{Key: "auth_token", Value: 0},
			{Key: "qrcode_token", Value: 0},
			{Key: "default_sitone_ids", Value: 0},
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
	byID := make(map[string]mongomodel.Player, len(players))
	for _, player := range players {
		byID[player.ID] = player
	}
	return byID, nil
}

func (h *Handler) markStaffRewardNotified(ctx context.Context, rewardID string, notifiedAt time.Time) error {
	if h.db == nil {
		return nil
	}
	_, err := h.db.Collection(mongomodel.StaffRewardsCollection).UpdateOne(
		ctx,
		bson.M{"_id": rewardID, "notification_pending": true},
		bson.M{
			"$set":   bson.M{"notified_at": notifiedAt},
			"$unset": bson.M{"notification_pending": ""},
		},
	)
	return err
}

func (h *Handler) markOpenPowerTransferNotified(ctx context.Context, transferID string, notifiedAt time.Time) error {
	if h.db == nil {
		return nil
	}
	_, err := h.db.Collection(mongomodel.OpenPowerTransfersCollection).UpdateOne(
		ctx,
		bson.M{"_id": transferID, "notification_pending": true},
		bson.M{
			"$set":   bson.M{"notified_at": notifiedAt},
			"$unset": bson.M{"notification_pending": ""},
		},
	)
	return err
}

func (h *Handler) markInventoryTrimNotified(ctx context.Context, trimID string, notifiedAt time.Time) error {
	if h.db == nil {
		return nil
	}
	_, err := h.db.Collection(mongomodel.InventoryTrimsCollection).UpdateOne(
		ctx,
		bson.M{"_id": trimID, "notification_pending": true},
		bson.M{
			"$set":   bson.M{"notified_at": notifiedAt},
			"$unset": bson.M{"notification_pending": ""},
		},
	)
	return err
}

func (h *Handler) writeAchievementEvents(ctx context.Context, w http.ResponseWriter, flusher http.Flusher, events []pendingAchievementEvent) bool {
	for _, event := range events {
		if !writePlayerEventSSE(w, event.Event) {
			return false
		}
		flusher.Flush()
		if err := h.markAchievementNotified(ctx, event.AchievementID, time.Now().UTC()); err != nil {
			return false
		}
	}
	return true
}

func (h *Handler) markAchievementNotified(ctx context.Context, achievementID string, notifiedAt time.Time) error {
	if h.db == nil {
		return nil
	}
	_, err := h.db.Collection(mongomodel.AchievementsCollection).UpdateOne(
		ctx,
		bson.M{"_id": achievementID, "notification_pending": true},
		bson.M{
			"$set":   bson.M{"notified_at": notifiedAt},
			"$unset": bson.M{"notification_pending": ""},
		},
	)
	return err
}

func writePlayerEventSSE(w http.ResponseWriter, event playerevents.Event) bool {
	var payload any
	switch event.Name {
	case "reward_granted":
		if event.Reward == nil {
			return true
		}
		payload = event.Reward
	case playerevents.InventoryTrimmedEventName:
		if event.InventoryTrimmed == nil {
			return true
		}
		payload = event.InventoryTrimmed
	case playerevents.AchievementUnlockedEventName:
		if event.AchievementUnlocked == nil {
			return true
		}
		payload = event.AchievementUnlocked
	default:
		return true
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return false
	}
	if _, err := fmt.Fprintf(w, "event: %s\n", event.Name); err != nil {
		return false
	}
	if _, err := fmt.Fprintf(w, "data: %s\n\n", encoded); err != nil {
		return false
	}
	return true
}
