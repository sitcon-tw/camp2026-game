package playerevents

import (
	"testing"
	"time"

	mongomodel "github.com/sitcon-tw/camp2026-game/internal/mongodb/model"
)

func TestAchievementUnlocked(t *testing.T) {
	createdAt := time.Date(2026, 7, 10, 8, 0, 0, 0, time.UTC)
	event := AchievementUnlocked(mongomodel.Achievement{
		ID:                  "achievement-player-a-codex-tier-01",
		Key:                 "codex_tier_01",
		Name:                "石來運轉",
		Tier:                1,
		SortOrder:           1,
		RequiredSitoneCount: 5,
		SitoneCount:         7,
		OpenPowerReward:     500,
		CreatedAt:           createdAt,
	}, true)

	if event.Name != "石來運轉" || event.Tier != 1 || event.RequiredSitoneCount != 5 || event.OpenPowerReward != 500 {
		t.Fatalf("unexpected achievement event: %#v", event)
	}
	if !event.Delayed || event.OccurredAt != createdAt.Format(time.RFC3339Nano) {
		t.Fatalf("unexpected delivery fields: %#v", event)
	}
}
