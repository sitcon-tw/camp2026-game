package playerevents

import "sync"

type RewardGrantedEvent struct {
	RewardID       string `json:"rewardId,omitempty"`
	Kind           string `json:"kind"`
	RefID          string `json:"refId,omitempty"`
	Name           string `json:"name"`
	Quantity       int    `json:"quantity,omitempty"`
	Amount         int    `json:"amount,omitempty"`
	ItemType       string `json:"itemType,omitempty"`
	SitoneType     string `json:"sitoneType,omitempty"`
	IconPath       string `json:"iconPath,omitempty"`
	Source         string `json:"source,omitempty"`
	StaffPlayerID  string `json:"staffPlayerId,omitempty"`
	StaffNickname  string `json:"staffNickname,omitempty"`
	SenderPlayerID string `json:"senderPlayerId,omitempty"`
	SenderNickname string `json:"senderNickname,omitempty"`
	OccurredAt     string `json:"occurredAt"`
	Delayed        bool   `json:"delayed,omitempty"`
}

type InventoryTrimmedEvent struct {
	TrimID      string `json:"trimId,omitempty"`
	Message     string `json:"message"`
	SitoneCount int    `json:"sitoneCount,omitempty"`
	OpenPower   int    `json:"openPower,omitempty"`
	OccurredAt  string `json:"occurredAt"`
	Delayed     bool   `json:"delayed,omitempty"`
}

type AchievementUnlockedEvent struct {
	AchievementID       string `json:"achievementId,omitempty"`
	Key                 string `json:"key"`
	Name                string `json:"name"`
	Tier                int    `json:"tier,omitempty"`
	RequiredSitoneCount int    `json:"requiredSitoneCount"`
	SitoneCount         int    `json:"sitoneCount"`
	OpenPowerReward     int    `json:"openPowerReward,omitempty"`
	OccurredAt          string `json:"occurredAt"`
	Delayed             bool   `json:"delayed,omitempty"`
}

type Event struct {
	Name                string
	Reward              *RewardGrantedEvent
	InventoryTrimmed    *InventoryTrimmedEvent
	AchievementUnlocked *AchievementUnlockedEvent
}

type Broker struct {
	mu          sync.Mutex
	subscribers map[string]map[chan Event]struct{}
}

func NewBroker() *Broker {
	return &Broker{
		subscribers: make(map[string]map[chan Event]struct{}),
	}
}

func (b *Broker) Subscribe(playerID string) (<-chan Event, func()) {
	ch := make(chan Event, 8)

	b.mu.Lock()
	if _, ok := b.subscribers[playerID]; !ok {
		b.subscribers[playerID] = make(map[chan Event]struct{})
	}
	b.subscribers[playerID][ch] = struct{}{}
	b.mu.Unlock()

	unsubscribe := func() {
		b.mu.Lock()
		defer b.mu.Unlock()

		if subscribers, ok := b.subscribers[playerID]; ok {
			delete(subscribers, ch)
			if len(subscribers) == 0 {
				delete(b.subscribers, playerID)
			}
		}
		close(ch)
	}

	return ch, unsubscribe
}

func (b *Broker) Publish(playerID string, event Event) int {
	b.mu.Lock()
	subscribers := make([]chan Event, 0, len(b.subscribers[playerID]))
	for ch := range b.subscribers[playerID] {
		subscribers = append(subscribers, ch)
	}
	b.mu.Unlock()

	delivered := 0
	for _, ch := range subscribers {
		select {
		case ch <- event:
			delivered++
		default:
		}
	}
	return delivered
}
