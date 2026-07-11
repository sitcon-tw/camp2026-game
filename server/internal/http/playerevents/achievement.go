package playerevents

import mongomodel "github.com/sitcon-tw/camp2026-game/internal/mongodb/model"

const AchievementUnlockedEventName = "achievement_unlocked"

func AchievementUnlocked(record mongomodel.Achievement, delayed bool) AchievementUnlockedEvent {
	return AchievementUnlockedEvent{
		AchievementID:       record.ID,
		Key:                 record.Key,
		Name:                record.Name,
		Tier:                record.Tier,
		RequiredSitoneCount: record.RequiredSitoneCount,
		SitoneCount:         record.SitoneCount,
		OpenPowerReward:     record.OpenPowerReward,
		OccurredAt:          rewardEventTime(record.CreatedAt),
		Delayed:             delayed,
	}
}
