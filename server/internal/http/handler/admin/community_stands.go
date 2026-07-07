package admin

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/sitcon-tw/camp2026-game/internal/http/httpx"
	mongomodel "github.com/sitcon-tw/camp2026-game/internal/mongodb/model"
)

const (
	standRewardKindItem      = "item"
	standRewardKindSitone    = "sitone"
	standRewardKindOpenPower = "open_power"

	communityStandIDMaxLen          = 128
	communityStandNameMaxLen        = 96
	communityStandDescriptionMaxLen = 1000
	communityStandLogoURLMaxLen     = 512
	communityStandWebsiteURLMaxLen  = 512
	communityStandRewardQuantityMax = 1000
	communityStandRewardAmountMax   = 100000
)

type CommunityStandRewardRequest struct {
	Kind     string `json:"kind" example:"item"`
	RefID    string `json:"refId,omitempty" example:"item_booth_sticker"`
	Quantity int    `json:"quantity,omitempty" example:"1"`
	Amount   int    `json:"amount,omitempty" example:"50"`
}

type UpdateCommunityStandRequest struct {
	Name        string                      `json:"name" example:"SITCON 社群攤位"`
	Description string                      `json:"description" example:"介紹學生社群與開源參與方式。"`
	LogoURL     string                      `json:"logoUrl,omitempty" example:"/game-icons/features/team.png"`
	WebsiteURL  string                      `json:"websiteUrl,omitempty" example:"https://sitcon.org"`
	Enabled     *bool                       `json:"enabled" example:"true"`
	Reward      CommunityStandRewardRequest `json:"reward"`
}

type CommunityStandRewardResponse struct {
	Kind     string `json:"kind" example:"item"`
	RefID    string `json:"refId,omitempty" example:"item_booth_sticker"`
	Name     string `json:"name" example:"攤位貼紙"`
	Quantity int    `json:"quantity,omitempty" example:"1"`
	Amount   int    `json:"amount,omitempty" example:"50"`
	IconPath string `json:"iconPath,omitempty" example:"/game-icons/items/item_booth_sticker.png"`
}

type CommunityStandResponse struct {
	StandID     string                       `json:"standId" example:"q7m4x2v9"`
	Name        string                       `json:"name" example:"SITCON 社群攤位"`
	Description string                       `json:"description" example:"介紹學生社群與開源參與方式。"`
	LogoURL     string                       `json:"logoUrl,omitempty" example:"/game-icons/features/team.png"`
	WebsiteURL  string                       `json:"websiteUrl,omitempty" example:"https://sitcon.org"`
	Enabled     bool                         `json:"enabled" example:"true"`
	Reward      CommunityStandRewardResponse `json:"reward"`
	VisitCount  int64                        `json:"visitCount" example:"42"`
	ClaimCount  int64                        `json:"claimCount" example:"38"`
	CreatedAt   time.Time                    `json:"createdAt"`
	UpdatedAt   time.Time                    `json:"updatedAt"`
}

type CommunityStandsResponse struct {
	Stands []CommunityStandResponse `json:"stands"`
}

// ListCommunityStands godoc
// @Summary List community stands as admin
// @Description Admin-only endpoint. Returns all community stands, including disabled stands, with counters and editable settings.
// @Tags admin
// @Produce json
// @Success 200 {object} CommunityStandsResponse
// @Failure 401 {object} httpx.ProblemDetails
// @Failure 500 {object} httpx.ProblemDetails
// @Failure 503 {object} httpx.ProblemDetails
// @Router /admin/community-stands [get]
func (h *Handler) ListCommunityStands(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdmin(w, r) || !h.requireContent(w, r) || !h.requireDatabase(w, r) {
		return
	}

	stands, err := findAllDashboard[mongomodel.CommunityStand](
		r.Context(),
		h.db,
		mongomodel.CommunityStandsCollection,
		bson.M{},
		options.Find().SetSort(bson.D{{Key: "name", Value: 1}, {Key: "_id", Value: 1}}),
	)
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("community stands unavailable", "admin_community_stands_lookup_failed", err))
		return
	}

	responses := make([]CommunityStandResponse, 0, len(stands))
	for _, stand := range stands {
		response, err := h.communityStandResponse(r.Context(), stand)
		if err != nil {
			httpx.WriteProblem(w, r, httpx.InternalServerError("community stand counters unavailable", "admin_community_stand_counters_failed", err))
			return
		}
		responses = append(responses, response)
	}

	httpx.WriteJSON(w, http.StatusOK, CommunityStandsResponse{Stands: responses})
}

// UpdateCommunityStand godoc
// @Summary Update a community stand as admin
// @Description Admin-only endpoint. Updates display settings, enablement, and reward settings for a community stand.
// @Tags admin
// @Accept json
// @Produce json
// @Param standID path string true "Community stand ID"
// @Param request body UpdateCommunityStandRequest true "Community stand update request"
// @Success 200 {object} CommunityStandResponse
// @Failure 400 {object} httpx.ProblemDetails
// @Failure 401 {object} httpx.ProblemDetails
// @Failure 404 {object} httpx.ProblemDetails
// @Failure 422 {object} httpx.ProblemDetails
// @Failure 500 {object} httpx.ProblemDetails
// @Failure 503 {object} httpx.ProblemDetails
// @Router /admin/community-stands/{standID} [put]
func (h *Handler) UpdateCommunityStand(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdmin(w, r) || !h.requireContent(w, r) || !h.requireDatabase(w, r) {
		return
	}

	standID := strings.TrimSpace(chi.URLParam(r, "standID"))
	var body UpdateCommunityStandRequest
	if err := httpx.DecodeJSON(r, &body); err != nil {
		httpx.WriteProblem(w, r, err)
		return
	}
	body = normalizeUpdateCommunityStandRequest(body)
	if details := h.validateUpdateCommunityStandRequest(standID, body); len(details) > 0 {
		httpx.WriteProblem(w, r, httpx.UnprocessableEntity("invalid community stand update request", details...))
		return
	}

	stand, err := h.updateCommunityStand(r.Context(), standID, body)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			httpx.WriteProblem(w, r, httpx.NotFound("community stand not found"))
			return
		}
		httpx.WriteProblem(w, r, httpx.InternalServerError("community stand update failed", "admin_community_stand_update_failed", err))
		return
	}
	response, err := h.communityStandResponse(r.Context(), stand)
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("community stand counters unavailable", "admin_community_stand_counters_failed", err))
		return
	}

	httpx.WriteJSON(w, http.StatusOK, response)
}

func normalizeUpdateCommunityStandRequest(body UpdateCommunityStandRequest) UpdateCommunityStandRequest {
	body.Name = strings.TrimSpace(body.Name)
	body.Description = strings.TrimSpace(body.Description)
	body.LogoURL = strings.TrimSpace(body.LogoURL)
	body.WebsiteURL = strings.TrimSpace(body.WebsiteURL)
	body.Reward.Kind = strings.TrimSpace(body.Reward.Kind)
	body.Reward.RefID = strings.TrimSpace(body.Reward.RefID)
	return body
}

func (h *Handler) validateUpdateCommunityStandRequest(standID string, body UpdateCommunityStandRequest) []httpx.ErrorDetail {
	details := make([]httpx.ErrorDetail, 0, 8)
	if standID == "" {
		details = append(details, httpx.ErrorDetail{Location: "path.standId", Message: "standId is required"})
	} else if len([]rune(standID)) > communityStandIDMaxLen {
		details = append(details, httpx.ErrorDetail{Location: "path.standId", Message: "standId must be at most 128"})
	}

	if body.Enabled == nil {
		details = append(details, httpx.ErrorDetail{Location: "body.enabled", Message: "enabled is required"})
	}
	if body.Name == "" {
		details = append(details, httpx.ErrorDetail{Location: "body.name", Message: "name is required"})
	} else if len([]rune(body.Name)) > communityStandNameMaxLen {
		details = append(details, httpx.ErrorDetail{Location: "body.name", Message: "name must be at most 96"})
	}
	if body.Description == "" {
		details = append(details, httpx.ErrorDetail{Location: "body.description", Message: "description is required"})
	} else if len([]rune(body.Description)) > communityStandDescriptionMaxLen {
		details = append(details, httpx.ErrorDetail{Location: "body.description", Message: "description must be at most 1000"})
	}
	if len([]rune(body.LogoURL)) > communityStandLogoURLMaxLen {
		details = append(details, httpx.ErrorDetail{Location: "body.logoUrl", Message: "logoUrl must be at most 512"})
	} else if !validOptionalRootRelativeOrHTTPURL(body.LogoURL) {
		details = append(details, httpx.ErrorDetail{Location: "body.logoUrl", Message: "logoUrl must be an http(s) URL or root-relative path"})
	}
	if len([]rune(body.WebsiteURL)) > communityStandWebsiteURLMaxLen {
		details = append(details, httpx.ErrorDetail{Location: "body.websiteUrl", Message: "websiteUrl must be at most 512"})
	} else if !validOptionalHTTPURL(body.WebsiteURL) {
		details = append(details, httpx.ErrorDetail{Location: "body.websiteUrl", Message: "websiteUrl must be an http(s) URL"})
	}

	return append(details, h.validateCommunityStandRewardRequest(body.Reward)...)
}

func (h *Handler) validateCommunityStandRewardRequest(reward CommunityStandRewardRequest) []httpx.ErrorDetail {
	details := make([]httpx.ErrorDetail, 0, 3)
	switch reward.Kind {
	case standRewardKindItem:
		if reward.RefID == "" {
			details = append(details, httpx.ErrorDetail{Location: "body.reward.refId", Message: "refId is required for item rewards"})
		} else if item, ok := h.content.GetItem(reward.RefID); !ok || !item.Enabled {
			details = append(details, httpx.ErrorDetail{Location: "body.reward.refId", Message: "refId must reference an enabled item"})
		}
		if reward.Quantity <= 0 || reward.Quantity > communityStandRewardQuantityMax {
			details = append(details, httpx.ErrorDetail{Location: "body.reward.quantity", Message: "quantity must be between 1 and 1000"})
		}
	case standRewardKindSitone:
		if reward.RefID == "" {
			details = append(details, httpx.ErrorDetail{Location: "body.reward.refId", Message: "refId is required for sitone rewards"})
		} else if _, ok := h.content.GetSitone(reward.RefID); !ok {
			details = append(details, httpx.ErrorDetail{Location: "body.reward.refId", Message: "refId must reference an existing sitone"})
		}
		if reward.Quantity <= 0 || reward.Quantity > communityStandRewardQuantityMax {
			details = append(details, httpx.ErrorDetail{Location: "body.reward.quantity", Message: "quantity must be between 1 and 1000"})
		}
	case standRewardKindOpenPower:
		if reward.Amount <= 0 || reward.Amount > communityStandRewardAmountMax {
			details = append(details, httpx.ErrorDetail{Location: "body.reward.amount", Message: "amount must be between 1 and 100000"})
		}
	default:
		details = append(details, httpx.ErrorDetail{Location: "body.reward.kind", Message: "kind must be item, sitone, or open_power"})
	}
	return details
}

func validOptionalRootRelativeOrHTTPURL(value string) bool {
	if value == "" {
		return true
	}
	if strings.HasPrefix(value, "/") && !strings.HasPrefix(value, "//") {
		return true
	}
	return validHTTPURL(value)
}

func validOptionalHTTPURL(value string) bool {
	return value == "" || validHTTPURL(value)
}

func validHTTPURL(value string) bool {
	parsed, err := url.Parse(value)
	if err != nil || parsed.Host == "" {
		return false
	}
	return parsed.Scheme == "http" || parsed.Scheme == "https"
}

func (h *Handler) updateCommunityStand(ctx context.Context, standID string, body UpdateCommunityStandRequest) (mongomodel.CommunityStand, error) {
	setFields := bson.D{
		{Key: "name", Value: body.Name},
		{Key: "description", Value: body.Description},
		{Key: "enabled", Value: *body.Enabled},
		{Key: "reward", Value: communityStandRewardModel(body.Reward)},
		{Key: "updated_at", Value: time.Now().UTC()},
	}
	unsetFields := bson.D{}
	if body.LogoURL == "" {
		unsetFields = append(unsetFields, bson.E{Key: "logo_url", Value: ""})
	} else {
		setFields = append(setFields, bson.E{Key: "logo_url", Value: body.LogoURL})
	}
	if body.WebsiteURL == "" {
		unsetFields = append(unsetFields, bson.E{Key: "website_url", Value: ""})
	} else {
		setFields = append(setFields, bson.E{Key: "website_url", Value: body.WebsiteURL})
	}

	update := bson.D{{Key: "$set", Value: setFields}}
	if len(unsetFields) > 0 {
		update = append(update, bson.E{Key: "$unset", Value: unsetFields})
	}

	var stand mongomodel.CommunityStand
	err := h.db.Collection(mongomodel.CommunityStandsCollection).FindOneAndUpdate(
		ctx,
		bson.M{"_id": standID},
		update,
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	).Decode(&stand)
	if err != nil {
		return mongomodel.CommunityStand{}, err
	}
	return stand, nil
}

func communityStandRewardModel(reward CommunityStandRewardRequest) mongomodel.StandReward {
	switch reward.Kind {
	case standRewardKindOpenPower:
		return mongomodel.StandReward{Kind: standRewardKindOpenPower, Amount: reward.Amount}
	case standRewardKindSitone:
		return mongomodel.StandReward{Kind: standRewardKindSitone, RefID: reward.RefID, Quantity: reward.Quantity}
	default:
		return mongomodel.StandReward{Kind: standRewardKindItem, RefID: reward.RefID, Quantity: reward.Quantity}
	}
}

func (h *Handler) communityStandResponse(ctx context.Context, stand mongomodel.CommunityStand) (CommunityStandResponse, error) {
	visitCount, claimCount, err := h.communityStandCounters(ctx, stand.ID)
	if err != nil {
		return CommunityStandResponse{}, err
	}
	reward, _ := h.communityStandRewardResponse(stand.Reward)
	return CommunityStandResponse{
		StandID:     stand.ID,
		Name:        stand.Name,
		Description: stand.Description,
		LogoURL:     stand.LogoURL,
		WebsiteURL:  stand.WebsiteURL,
		Enabled:     stand.Enabled,
		Reward:      reward,
		VisitCount:  visitCount,
		ClaimCount:  claimCount,
		CreatedAt:   stand.CreatedAt,
		UpdatedAt:   stand.UpdatedAt,
	}, nil
}

func (h *Handler) communityStandCounters(ctx context.Context, standID string) (int64, int64, error) {
	visitCount, err := h.db.Collection(mongomodel.CommunityStandVisitsCollection).CountDocuments(ctx, bson.M{"stand_id": standID})
	if err != nil {
		return 0, 0, err
	}
	claimCount, err := h.db.Collection(mongomodel.CommunityStandClaimsCollection).CountDocuments(ctx, bson.M{"stand_id": standID})
	if err != nil {
		return 0, 0, err
	}
	return visitCount, claimCount, nil
}

func (h *Handler) communityStandRewardResponse(reward mongomodel.StandReward) (CommunityStandRewardResponse, bool) {
	response := CommunityStandRewardResponse{
		Kind:     reward.Kind,
		RefID:    reward.RefID,
		Quantity: reward.Quantity,
		Amount:   reward.Amount,
	}
	switch reward.Kind {
	case standRewardKindOpenPower:
		response.Name = "開源力"
		return response, true
	case standRewardKindItem:
		item, ok := h.content.GetItem(reward.RefID)
		if !ok || !item.Enabled {
			response.Name = "未知道具"
			return response, false
		}
		response.Name = item.Name
		response.IconPath = item.IconPath
		return response, true
	case standRewardKindSitone:
		sitone, ok := h.content.GetSitone(reward.RefID)
		if !ok {
			response.Name = "未知小石"
			return response, false
		}
		response.Name = sitone.Name
		response.IconPath = sitone.IconPath
		return response, true
	default:
		response.Name = "未知獎勵"
		return response, false
	}
}
