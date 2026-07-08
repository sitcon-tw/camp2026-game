package admin

import (
	"context"
	"net/http"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/sitcon-tw/camp2026-game/internal/http/httpx"
	mongomodel "github.com/sitcon-tw/camp2026-game/internal/mongodb/model"
)

const adminCommunityStandClaimLimit int64 = 200

type CommunityStandClaimResponse struct {
	ClaimID        string                       `json:"claimId" example:"community_claim_507f1f77bcf86cd799439011"`
	StandID        string                       `json:"standId" example:"00000000-0000-4000-8000-000000000000"`
	StandName      string                       `json:"standName,omitempty" example:"SITCON 社群攤位"`
	PlayerID       string                       `json:"playerId" example:"7H9K2Q"`
	PlayerNickname string                       `json:"playerNickname,omitempty" example:"Alice"`
	Reward         CommunityStandRewardResponse `json:"reward"`
	CreatedAt      time.Time                    `json:"createdAt"`
}

type CommunityStandClaimsResponse struct {
	Claims []CommunityStandClaimResponse `json:"claims"`
}

// ListCommunityStandClaims godoc
// @Summary List community stand claim records as admin
// @Description Admin-only endpoint. Returns recent community stand claim records, including claimant, stand, reward snapshot, and claim time.
// @Tags admin
// @Produce json
// @Param standId query string false "Filter by community stand ID"
// @Success 200 {object} CommunityStandClaimsResponse
// @Failure 401 {object} httpx.ProblemDetails
// @Failure 500 {object} httpx.ProblemDetails
// @Failure 503 {object} httpx.ProblemDetails
// @Router /admin/community-stand-claims [get]
func (h *Handler) ListCommunityStandClaims(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdmin(w, r) || !h.requireContent(w, r) || !h.requireDatabase(w, r) {
		return
	}

	standID := strings.TrimSpace(r.URL.Query().Get("standId"))
	claims, err := h.communityStandClaimResponses(r.Context(), standID)
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("community stand claims unavailable", "admin_community_stand_claims_lookup_failed", err))
		return
	}

	httpx.WriteJSON(w, http.StatusOK, CommunityStandClaimsResponse{Claims: claims})
}

func (h *Handler) communityStandClaimResponses(ctx context.Context, standID string) ([]CommunityStandClaimResponse, error) {
	filter := bson.M{}
	if standID != "" {
		filter["stand_id"] = standID
	}

	claims, err := findAllDashboard[mongomodel.CommunityStandClaim](
		ctx,
		h.db,
		mongomodel.CommunityStandClaimsCollection,
		filter,
		options.Find().
			SetSort(bson.D{{Key: "created_at", Value: -1}, {Key: "_id", Value: -1}}).
			SetLimit(adminCommunityStandClaimLimit),
	)
	if err != nil {
		return nil, err
	}

	players, err := h.communityStandClaimPlayers(ctx, claims)
	if err != nil {
		return nil, err
	}
	stands, err := h.communityStandClaimStands(ctx, claims)
	if err != nil {
		return nil, err
	}

	responses := make([]CommunityStandClaimResponse, 0, len(claims))
	for _, claim := range claims {
		responses = append(responses, h.communityStandClaimResponse(claim, players, stands))
	}
	return responses, nil
}

func (h *Handler) communityStandClaimResponse(
	claim mongomodel.CommunityStandClaim,
	players map[string]mongomodel.Player,
	stands map[string]mongomodel.CommunityStand,
) CommunityStandClaimResponse {
	reward, _ := h.communityStandRewardResponse(claim.Reward)
	response := CommunityStandClaimResponse{
		ClaimID:   claim.ID,
		StandID:   claim.StandID,
		PlayerID:  claim.PlayerID,
		Reward:    reward,
		CreatedAt: claim.CreatedAt,
	}
	if player, ok := players[claim.PlayerID]; ok {
		response.PlayerNickname = player.Nickname
	}
	if stand, ok := stands[claim.StandID]; ok {
		response.StandName = stand.Name
	}
	return response
}

func (h *Handler) communityStandClaimPlayers(ctx context.Context, claims []mongomodel.CommunityStandClaim) (map[string]mongomodel.Player, error) {
	ids := make([]string, 0, len(claims))
	seen := make(map[string]struct{}, len(claims))
	for _, claim := range claims {
		if claim.PlayerID == "" {
			continue
		}
		if _, ok := seen[claim.PlayerID]; ok {
			continue
		}
		seen[claim.PlayerID] = struct{}{}
		ids = append(ids, claim.PlayerID)
	}
	if len(ids) == 0 {
		return map[string]mongomodel.Player{}, nil
	}

	players, err := findAllDashboard[mongomodel.Player](
		ctx,
		h.db,
		mongomodel.PlayersCollection,
		bson.M{"_id": bson.M{"$in": ids}},
		options.Find().SetProjection(bson.D{
			{Key: "auth_token", Value: 0},
			{Key: "qrcode_token", Value: 0},
			{Key: "default_sitone_ids", Value: 0},
		}),
	)
	if err != nil {
		return nil, err
	}
	out := make(map[string]mongomodel.Player, len(players))
	for _, player := range players {
		out[player.ID] = player
	}
	return out, nil
}

func (h *Handler) communityStandClaimStands(ctx context.Context, claims []mongomodel.CommunityStandClaim) (map[string]mongomodel.CommunityStand, error) {
	ids := make([]string, 0, len(claims))
	seen := make(map[string]struct{}, len(claims))
	for _, claim := range claims {
		if claim.StandID == "" {
			continue
		}
		if _, ok := seen[claim.StandID]; ok {
			continue
		}
		seen[claim.StandID] = struct{}{}
		ids = append(ids, claim.StandID)
	}
	if len(ids) == 0 {
		return map[string]mongomodel.CommunityStand{}, nil
	}

	stands, err := findAllDashboard[mongomodel.CommunityStand](
		ctx,
		h.db,
		mongomodel.CommunityStandsCollection,
		bson.M{"_id": bson.M{"$in": ids}},
	)
	if err != nil {
		return nil, err
	}
	out := make(map[string]mongomodel.CommunityStand, len(stands))
	for _, stand := range stands {
		out[stand.ID] = stand
	}
	return out, nil
}
