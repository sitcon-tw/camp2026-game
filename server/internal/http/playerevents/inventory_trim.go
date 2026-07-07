package playerevents

import mongomodel "github.com/sitcon-tw/camp2026-game/internal/mongodb/model"

const InventoryTrimmedEventName = "inventory_trimmed"

func InventoryTrimmed(record mongomodel.InventoryTrim, delayed bool) InventoryTrimmedEvent {
	return InventoryTrimmedEvent{
		TrimID:      record.ID,
		Message:     record.Message,
		SitoneCount: record.SitoneCount,
		OpenPower:   record.OpenPower,
		OccurredAt:  rewardEventTime(record.CreatedAt),
		Delayed:     delayed,
	}
}
