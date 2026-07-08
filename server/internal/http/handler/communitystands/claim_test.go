package communitystands

import (
	"testing"
	"time"

	mongomodel "github.com/sitcon-tw/camp2026-game/internal/mongodb/model"
)

func TestNewCommunityStandClaimRecordsClaimSnapshot(t *testing.T) {
	createdAt := time.Date(2026, 7, 8, 12, 0, 0, 0, time.UTC)
	stand := mongomodel.CommunityStand{
		ID: "00000000-0000-4000-8000-000000000000",
		Reward: mongomodel.StandReward{
			Kind:     rewardKindItem,
			RefID:    "item_booth_sticker",
			Quantity: 2,
		},
	}

	claim := newCommunityStandClaim("community_claim_1", "player-a", stand, createdAt)

	if claim.ID != "community_claim_1" || claim.RewardID != "community_claim_1" {
		t.Fatalf("expected claim id and reward id to be recorded, got %#v", claim)
	}
	if claim.StandID != stand.ID || claim.PlayerID != "player-a" || !claim.CreatedAt.Equal(createdAt) {
		t.Fatalf("expected stand, player, and time to be recorded, got %#v", claim)
	}
	if claim.Reward != stand.Reward {
		t.Fatalf("expected reward snapshot to be recorded, got %#v", claim.Reward)
	}
}
