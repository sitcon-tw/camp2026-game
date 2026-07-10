package roomteam

import (
	"regexp"
	"testing"
	"time"
)

func TestNewQRTokenReturnsOpaqueRoomTeamToken(t *testing.T) {
	token, err := NewQRToken()
	if err != nil {
		t.Fatalf("NewQRToken failed: %v", err)
	}

	pattern := regexp.MustCompile(`^rmt_[A-Za-z0-9_-]{24}$`)
	if !pattern.MatchString(token) {
		t.Fatalf("expected opaque room team QR token, got %q", token)
	}
}

func TestDefaultRoomNumbers(t *testing.T) {
	rooms := DefaultRoomNumbers()
	if len(rooms) != 13 {
		t.Fatalf("expected 13 rooms, got %d", len(rooms))
	}
	if !ValidRoomNumber("208") || !ValidRoomNumber("123") {
		t.Fatalf("expected provided room numbers to be valid")
	}
	if ValidRoomNumber("211") {
		t.Fatalf("expected unlisted room number to be invalid")
	}

	rooms[0] = "mutated"
	if DefaultRoomNumbers()[0] == "mutated" {
		t.Fatalf("DefaultRoomNumbers must return a defensive copy")
	}
}

func TestTokenTTL(t *testing.T) {
	if TokenTTL != 10*time.Minute {
		t.Fatalf("expected 10 minute token ttl, got %s", TokenTTL)
	}
}
