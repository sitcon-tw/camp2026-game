package communitystands

import (
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	mongomodel "github.com/sitcon-tw/camp2026-game/internal/mongodb/model"
)

func TestEnabledStandByQRTokenFilterRequiresUnexpiredToken(t *testing.T) {
	now := time.Date(2026, 7, 8, 12, 0, 0, 0, time.UTC)
	filter := enabledStandByQRTokenFilter("cst_token", now)

	if filter["qr_token"] != "cst_token" || filter["enabled"] != true {
		t.Fatalf("unexpected token filter: %#v", filter)
	}
	expiresAt, ok := filter["qr_token_expires_at"].(bson.M)
	if !ok || !expiresAt["$gt"].(time.Time).Equal(now) {
		t.Fatalf("expected qr_token_expires_at $gt now filter, got %#v", filter["qr_token_expires_at"])
	}
}

func TestStandQRTokenActiveRequiresFutureExpiry(t *testing.T) {
	now := time.Date(2026, 7, 8, 12, 0, 0, 0, time.UTC)
	stand := mongomodel.CommunityStand{
		QRToken:          "cst_token",
		QRTokenExpiresAt: now.Add(time.Second),
	}
	if !standQRTokenActive(stand, now) {
		t.Fatal("expected future token expiry to be active")
	}

	stand.QRTokenExpiresAt = now
	if standQRTokenActive(stand, now) {
		t.Fatal("expected token expiring at now to be inactive")
	}

	stand.QRTokenExpiresAt = now.Add(time.Second)
	stand.QRToken = ""
	if standQRTokenActive(stand, now) {
		t.Fatal("expected empty token to be inactive")
	}
}

func TestCommunityStandQRTokenTTLIsTwoMinutes(t *testing.T) {
	if communityStandQRTokenTTL != 2*time.Minute {
		t.Fatalf("expected community stand QR token TTL to be 2 minutes, got %s", communityStandQRTokenTTL)
	}
}
