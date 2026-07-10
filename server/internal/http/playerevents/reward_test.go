package playerevents

import (
	"testing"
	"time"

	mongomodel "github.com/sitcon-tw/camp2026-game/internal/mongodb/model"
	"github.com/sitcon-tw/camp2026-game/internal/testcontent"
)

func TestStaffRewardGrantedEventBuildsDelayedItemEvent(t *testing.T) {
	createdAt := time.Date(2026, 7, 7, 9, 30, 0, 0, time.UTC)
	event, err := StaffRewardGrantedEvent(
		testcontent.Load(t),
		mongomodel.StaffReward{
			ID:                  "reward-1",
			StaffPlayerID:       "staff-a",
			RecipientPlayerID:   "player-a",
			Kind:                "item",
			RefID:               "item_adventure_backpack",
			Quantity:            2,
			CreatedAt:           createdAt,
			NotificationPending: true,
		},
		mongomodel.Player{ID: "staff-a", Nickname: "Staff One"},
		true,
	)
	if err != nil {
		t.Fatalf("build reward event: %v", err)
	}

	if event.RewardID != "reward-1" || event.Kind != "item" || event.RefID != "item_adventure_backpack" {
		t.Fatalf("unexpected reward identity: %#v", event)
	}
	if event.Name != "冒險背包" || event.Quantity != 2 || event.ItemType != "material" || event.IconPath == "" {
		t.Fatalf("unexpected item event fields: %#v", event)
	}
	if event.Source != StaffRewardSource || event.StaffNickname != "Staff One" || !event.Delayed {
		t.Fatalf("unexpected delivery fields: %#v", event)
	}
	if event.OccurredAt != createdAt.Format(time.RFC3339Nano) {
		t.Fatalf("expected occurredAt %q, got %q", createdAt.Format(time.RFC3339Nano), event.OccurredAt)
	}
}

func TestOpenPowerTransferReceivedEvent(t *testing.T) {
	createdAt := time.Date(2026, 7, 11, 8, 30, 0, 0, time.UTC)
	event := OpenPowerTransferReceivedEvent(
		mongomodel.OpenPowerTransfer{
			ID:                "transfer-a",
			SenderPlayerID:    "player-a",
			RecipientPlayerID: "player-b",
			Amount:            120,
			CreatedAt:         createdAt,
		},
		mongomodel.Player{ID: "player-a", Nickname: "Alice"},
		true,
	)

	if event.RewardID != "transfer-a" || event.Kind != "open_power" || event.Name != "開源力" || event.Amount != 120 {
		t.Fatalf("unexpected transfer reward fields: %#v", event)
	}
	if event.Source != OpenPowerTransferSource || event.SenderPlayerID != "player-a" || event.SenderNickname != "Alice" || !event.Delayed {
		t.Fatalf("unexpected transfer delivery fields: %#v", event)
	}
	if event.OccurredAt != createdAt.Format(time.RFC3339Nano) {
		t.Fatalf("expected occurredAt %q, got %q", createdAt.Format(time.RFC3339Nano), event.OccurredAt)
	}
}
