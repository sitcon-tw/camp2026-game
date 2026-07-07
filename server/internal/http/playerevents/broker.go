package playerevents

import "sync"

type RewardGrantedEvent struct {
	Kind          string `json:"kind"`
	RefID         string `json:"refId,omitempty"`
	Name          string `json:"name"`
	Quantity      int    `json:"quantity,omitempty"`
	Amount        int    `json:"amount,omitempty"`
	ItemType      string `json:"itemType,omitempty"`
	SitoneType    string `json:"sitoneType,omitempty"`
	IconPath      string `json:"iconPath,omitempty"`
	Source        string `json:"source,omitempty"`
	StaffPlayerID string `json:"staffPlayerId,omitempty"`
	StaffNickname string `json:"staffNickname,omitempty"`
	OccurredAt    string `json:"occurredAt"`
}

type Event struct {
	Name   string
	Reward *RewardGrantedEvent
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

func (b *Broker) Publish(playerID string, event Event) {
	b.mu.Lock()
	subscribers := make([]chan Event, 0, len(b.subscribers[playerID]))
	for ch := range b.subscribers[playerID] {
		subscribers = append(subscribers, ch)
	}
	b.mu.Unlock()

	for _, ch := range subscribers {
		select {
		case ch <- event:
		default:
		}
	}
}
