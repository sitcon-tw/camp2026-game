package communitystand

import (
	"regexp"
	"testing"
)

func TestNewStandIDReturnsUUIDV4(t *testing.T) {
	standID, err := NewStandID()
	if err != nil {
		t.Fatalf("NewStandID failed: %v", err)
	}

	pattern := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	if !pattern.MatchString(standID) {
		t.Fatalf("expected UUID v4 stand id, got %q", standID)
	}
}

func TestNewQRTokenReturnsOpaqueCommunityStandToken(t *testing.T) {
	qrToken, err := NewQRToken()
	if err != nil {
		t.Fatalf("NewQRToken failed: %v", err)
	}

	pattern := regexp.MustCompile(`^cst_[A-Za-z0-9_-]{24}$`)
	if !pattern.MatchString(qrToken) {
		t.Fatalf("expected opaque community stand QR token, got %q", qrToken)
	}
}
