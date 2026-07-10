package playerevents

import (
	"fmt"
	"time"

	"github.com/sitcon-tw/camp2026-game/internal/content"
	mongomodel "github.com/sitcon-tw/camp2026-game/internal/mongodb/model"
)

const (
	StaffRewardSource       = "staff_reward"
	OpenPowerTransferSource = "open_power_transfer"
)

func StaffRewardGrantedEvent(store *content.Store, reward mongomodel.StaffReward, staff mongomodel.Player, delayed bool) (RewardGrantedEvent, error) {
	event := RewardGrantedEvent{
		RewardID:      reward.ID,
		Kind:          reward.Kind,
		RefID:         reward.RefID,
		Source:        StaffRewardSource,
		StaffPlayerID: reward.StaffPlayerID,
		StaffNickname: staff.Nickname,
		OccurredAt:    rewardEventTime(reward.CreatedAt),
		Delayed:       delayed,
	}

	switch reward.Kind {
	case "open_power":
		event.Name = "開源力"
		event.Amount = reward.Quantity
	case "item":
		if store == nil {
			return RewardGrantedEvent{}, fmt.Errorf("reward content store is unavailable")
		}
		item, ok := store.GetItem(reward.RefID)
		if !ok {
			return RewardGrantedEvent{}, fmt.Errorf("reward item %q not found", reward.RefID)
		}
		event.Name = item.Name
		event.Quantity = reward.Quantity
		event.ItemType = item.Type
		event.IconPath = item.IconPath
	case "sitone":
		if store == nil {
			return RewardGrantedEvent{}, fmt.Errorf("reward content store is unavailable")
		}
		sitone, ok := store.GetSitone(reward.RefID)
		if !ok {
			return RewardGrantedEvent{}, fmt.Errorf("reward sitone %q not found", reward.RefID)
		}
		event.Name = sitone.Name
		event.Quantity = reward.Quantity
		event.SitoneType = sitone.Type
		event.IconPath = sitone.IconPath
	default:
		return RewardGrantedEvent{}, fmt.Errorf("unsupported reward kind %q", reward.Kind)
	}
	return event, nil
}

func OpenPowerTransferReceivedEvent(transfer mongomodel.OpenPowerTransfer, sender mongomodel.Player, delayed bool) RewardGrantedEvent {
	return RewardGrantedEvent{
		RewardID:       transfer.ID,
		Kind:           "open_power",
		Name:           "開源力",
		Amount:         transfer.Amount,
		Source:         OpenPowerTransferSource,
		SenderPlayerID: sender.ID,
		SenderNickname: sender.Nickname,
		OccurredAt:     rewardEventTime(transfer.CreatedAt),
		Delayed:        delayed,
	}
}

func rewardEventTime(value time.Time) string {
	if value.IsZero() {
		return time.Now().UTC().Format(time.RFC3339Nano)
	}
	return value.UTC().Format(time.RFC3339Nano)
}
