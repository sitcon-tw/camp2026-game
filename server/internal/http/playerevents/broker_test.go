package playerevents

import "testing"

func TestBrokerPublishReportsDeliveredSubscribers(t *testing.T) {
	broker := NewBroker()
	eventCh, unsubscribe := broker.Subscribe("player-a")
	defer unsubscribe()

	delivered := broker.Publish("player-a", Event{Name: "reward_granted"})
	if delivered != 1 {
		t.Fatalf("expected one delivered event, got %d", delivered)
	}

	select {
	case event := <-eventCh:
		if event.Name != "reward_granted" {
			t.Fatalf("unexpected event: %#v", event)
		}
	default:
		t.Fatal("expected subscriber to receive event")
	}

	if got := broker.Publish("offline-player", Event{Name: "reward_granted"}); got != 0 {
		t.Fatalf("expected no offline deliveries, got %d", got)
	}
}
