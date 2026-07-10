package fronts

import "sync"

type FrontEvent struct {
	Name string
}

type FrontBroker struct {
	mu          sync.Mutex
	subscribers map[string]map[chan FrontEvent]struct{}
}

func NewFrontBroker() *FrontBroker {
	return &FrontBroker{subscribers: make(map[string]map[chan FrontEvent]struct{})}
}

func (b *FrontBroker) Subscribe(frontID string) (<-chan FrontEvent, func()) {
	ch := make(chan FrontEvent, 8)
	b.mu.Lock()
	if b.subscribers[frontID] == nil {
		b.subscribers[frontID] = make(map[chan FrontEvent]struct{})
	}
	b.subscribers[frontID][ch] = struct{}{}
	b.mu.Unlock()

	return ch, func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		if subscribers := b.subscribers[frontID]; subscribers != nil {
			delete(subscribers, ch)
			if len(subscribers) == 0 {
				delete(b.subscribers, frontID)
			}
		}
		close(ch)
	}
}

func (b *FrontBroker) Publish(frontID string, event FrontEvent) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for ch := range b.subscribers[frontID] {
		select {
		case ch <- event:
		default:
		}
	}
}
