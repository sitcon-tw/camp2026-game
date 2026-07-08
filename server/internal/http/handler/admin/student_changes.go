package admin

import (
	"context"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/sitcon-tw/camp2026-game/internal/http/authctx"
	"github.com/sitcon-tw/camp2026-game/internal/http/httpx"
	mongomodel "github.com/sitcon-tw/camp2026-game/internal/mongodb/model"
)

const (
	adminStudentChangesDefaultLimit int64 = 300
	adminStudentChangesMaxLimit     int64 = 1000
	rewardKindItem                        = "item"
	rewardKindSitone                      = "sitone"
	rewardKindOpenPower                   = "open_power"
)

type StudentChangeEntryResponse struct {
	ChangeID       string    `json:"changeId" example:"staff_reward_507f1f77bcf86cd799439011"`
	Source         string    `json:"source" example:"staff_reward"`
	SourceLabel    string    `json:"sourceLabel" example:"Staff 發獎"`
	PlayerID       string    `json:"playerId" example:"7H9K2Q"`
	PlayerNickname string    `json:"playerNickname,omitempty" example:"Alice"`
	TeamID         string    `json:"teamId,omitempty" example:"team-a"`
	Kind           string    `json:"kind" example:"sitone"`
	RefID          string    `json:"refId,omitempty" example:"stone_booth"`
	Name           string    `json:"name" example:"Booth 小石"`
	IconPath       string    `json:"iconPath,omitempty"`
	Delta          int       `json:"delta" example:"1"`
	Note           string    `json:"note,omitempty"`
	CreatedAt      time.Time `json:"createdAt"`
}

type StudentChangesResponse struct {
	Entries []StudentChangeEntryResponse `json:"entries"`
}

// ListStudentChanges godoc
// @Summary List student resource change records as admin
// @Description Admin-only endpoint. Returns recent resource changes across non-staff players, including staff rewards, community stand claims, drops, shop purchases, fusions, balance trims, and direct open power records.
// @Tags admin
// @Produce json
// @Param limit query int false "Maximum number of merged records to return"
// @Success 200 {object} StudentChangesResponse
// @Failure 401 {object} httpx.ProblemDetails
// @Failure 500 {object} httpx.ProblemDetails
// @Failure 503 {object} httpx.ProblemDetails
// @Router /admin/student-changes [get]
func (h *Handler) ListStudentChanges(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdmin(w, r) || !h.requireContent(w, r) || !h.requireDatabase(w, r) {
		return
	}

	limit := adminStudentChangesLimit(r.URL.Query().Get("limit"))
	entries, err := h.studentChangeResponses(r.Context(), limit)
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("student changes unavailable", "admin_student_changes_lookup_failed", err))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, StudentChangesResponse{Entries: entries})
}

func adminStudentChangesLimit(value string) int64 {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return adminStudentChangesDefaultLimit
	}
	limit, err := strconv.ParseInt(trimmed, 10, 64)
	if err != nil || limit <= 0 {
		return adminStudentChangesDefaultLimit
	}
	if limit > adminStudentChangesMaxLimit {
		return adminStudentChangesMaxLimit
	}
	return limit
}

func (h *Handler) studentChangeResponses(ctx context.Context, limit int64) ([]StudentChangeEntryResponse, error) {
	players, playerIDs, err := h.studentChangePlayers(ctx)
	if err != nil {
		return nil, err
	}
	if len(playerIDs) == 0 {
		return []StudentChangeEntryResponse{}, nil
	}

	entries := make([]StudentChangeEntryResponse, 0, limit)
	if err := h.appendStaffRewardChanges(ctx, &entries, players, playerIDs, limit); err != nil {
		return nil, err
	}
	if err := h.appendCommunityStandClaimChanges(ctx, &entries, players, playerIDs, limit); err != nil {
		return nil, err
	}
	if err := h.appendMatchDropChanges(ctx, &entries, players, playerIDs, limit); err != nil {
		return nil, err
	}
	if err := h.appendShopPurchaseChanges(ctx, &entries, players, playerIDs, limit); err != nil {
		return nil, err
	}
	if err := h.appendFusionChanges(ctx, &entries, players, playerIDs, limit); err != nil {
		return nil, err
	}
	if err := h.appendInventoryTrimChanges(ctx, &entries, players, playerIDs, limit); err != nil {
		return nil, err
	}
	if err := h.appendOpenPowerRecordChanges(ctx, &entries, players, playerIDs, limit); err != nil {
		return nil, err
	}

	sort.SliceStable(entries, func(i, j int) bool {
		if !entries[i].CreatedAt.Equal(entries[j].CreatedAt) {
			return entries[i].CreatedAt.After(entries[j].CreatedAt)
		}
		return entries[i].ChangeID > entries[j].ChangeID
	})
	if int64(len(entries)) > limit {
		entries = entries[:limit]
	}
	return entries, nil
}

func (h *Handler) studentChangePlayers(ctx context.Context) (map[string]mongomodel.Player, []string, error) {
	players, err := findAllDashboard[mongomodel.Player](
		ctx,
		h.db,
		mongomodel.PlayersCollection,
		bson.M{"role": bson.M{"$ne": authctx.PlayerRoleStaff}},
		options.Find().SetProjection(bson.D{
			{Key: "_id", Value: 1},
			{Key: "nickname", Value: 1},
			{Key: "team_id", Value: 1},
			{Key: "role", Value: 1},
		}),
	)
	if err != nil {
		return nil, nil, err
	}
	byID := make(map[string]mongomodel.Player, len(players))
	ids := make([]string, 0, len(players))
	for _, player := range players {
		if player.ID == "" {
			continue
		}
		byID[player.ID] = player
		ids = append(ids, player.ID)
	}
	return byID, ids, nil
}

func (h *Handler) appendStaffRewardChanges(ctx context.Context, entries *[]StudentChangeEntryResponse, players map[string]mongomodel.Player, playerIDs []string, limit int64) error {
	records, err := findAllDashboard[mongomodel.StaffReward](
		ctx,
		h.db,
		mongomodel.StaffRewardsCollection,
		bson.M{"recipient_player_id": bson.M{"$in": playerIDs}},
		recentStudentChangeFindOptions(limit),
	)
	if err != nil {
		return err
	}
	for _, record := range records {
		if _, ok := players[record.RecipientPlayerID]; !ok {
			continue
		}
		name, iconPath := h.studentChangeResource(record.Kind, record.RefID)
		*entries = append(*entries, h.studentChangeEntry(record.ID, "staff_reward", "Staff 發獎", record.RecipientPlayerID, players, record.Kind, record.RefID, name, iconPath, positiveOrOne(record.Quantity), "", record.CreatedAt))
	}
	return nil
}

func (h *Handler) appendCommunityStandClaimChanges(ctx context.Context, entries *[]StudentChangeEntryResponse, players map[string]mongomodel.Player, playerIDs []string, limit int64) error {
	claims, err := findAllDashboard[mongomodel.CommunityStandClaim](
		ctx,
		h.db,
		mongomodel.CommunityStandClaimsCollection,
		bson.M{"player_id": bson.M{"$in": playerIDs}},
		recentStudentChangeFindOptions(limit),
	)
	if err != nil {
		return err
	}
	stands, err := h.communityStandClaimStands(ctx, claims)
	if err != nil {
		return err
	}
	for _, claim := range claims {
		if _, ok := players[claim.PlayerID]; !ok {
			continue
		}
		delta := claim.Reward.Quantity
		if claim.Reward.Kind == standRewardKindOpenPower {
			delta = claim.Reward.Amount
		}
		name, iconPath := h.studentChangeResource(claim.Reward.Kind, claim.Reward.RefID)
		note := claim.StandID
		if stand, ok := stands[claim.StandID]; ok && stand.Name != "" {
			note = stand.Name
		}
		*entries = append(*entries, h.studentChangeEntry(claim.ID, "community_stand", "攤位領取", claim.PlayerID, players, claim.Reward.Kind, claim.Reward.RefID, name, iconPath, positiveOrOne(delta), note, claim.CreatedAt))
	}
	return nil
}

func (h *Handler) appendMatchDropChanges(ctx context.Context, entries *[]StudentChangeEntryResponse, players map[string]mongomodel.Player, playerIDs []string, limit int64) error {
	drops, err := findAllDashboard[mongomodel.MatchItemDrop](
		ctx,
		h.db,
		mongomodel.MatchItemDropsCollection,
		bson.M{"player_id": bson.M{"$in": playerIDs}, "dropped": true, "granted": true},
		recentStudentChangeFindOptions(limit),
	)
	if err != nil {
		return err
	}
	for _, drop := range drops {
		if _, ok := players[drop.PlayerID]; !ok {
			continue
		}
		kind := rewardKindSitone
		refID := drop.SitoneID
		if refID == "" {
			kind = rewardKindItem
			refID = drop.ItemID
		}
		if refID == "" {
			continue
		}
		name, iconPath := h.studentChangeResource(kind, refID)
		*entries = append(*entries, h.studentChangeEntry(drop.ID, "match_drop", "對戰掉落", drop.PlayerID, players, kind, refID, name, iconPath, positiveOrOne(drop.Quantity), drop.MatchID, drop.CreatedAt))
	}
	return nil
}

func (h *Handler) appendShopPurchaseChanges(ctx context.Context, entries *[]StudentChangeEntryResponse, players map[string]mongomodel.Player, playerIDs []string, limit int64) error {
	purchases, err := findAllDashboard[mongomodel.ShopPurchase](
		ctx,
		h.db,
		mongomodel.ShopPurchasesCollection,
		bson.M{"player_id": bson.M{"$in": playerIDs}},
		recentStudentChangeFindOptions(limit),
	)
	if err != nil {
		return err
	}
	for _, purchase := range purchases {
		if _, ok := players[purchase.PlayerID]; !ok {
			continue
		}
		name, iconPath := h.studentChangeResource(rewardKindItem, purchase.ItemID)
		*entries = append(*entries, h.studentChangeEntry(purchase.ID+":item", "shop_purchase", "商店購買", purchase.PlayerID, players, rewardKindItem, purchase.ItemID, name, iconPath, positiveOrOne(purchase.Quantity), purchase.ID, purchase.CreatedAt))
		if purchase.PriceOpenPower > 0 {
			*entries = append(*entries, h.studentChangeEntry(purchase.ID+":open_power", "shop_purchase", "商店購買", purchase.PlayerID, players, rewardKindOpenPower, "", "開源力", "", -purchase.PriceOpenPower, purchase.ID, purchase.CreatedAt))
		}
	}
	return nil
}

func (h *Handler) appendFusionChanges(ctx context.Context, entries *[]StudentChangeEntryResponse, players map[string]mongomodel.Player, playerIDs []string, limit int64) error {
	records, err := findAllDashboard[mongomodel.FusionRecord](
		ctx,
		h.db,
		mongomodel.FusionRecordsCollection,
		bson.M{"player_id": bson.M{"$in": playerIDs}},
		recentStudentChangeFindOptions(limit),
	)
	if err != nil {
		return err
	}
	for _, record := range records {
		if _, ok := players[record.PlayerID]; !ok {
			continue
		}
		for index, input := range record.Inputs {
			h.appendFusionComponentChange(entries, record, players, input, -positiveOrOne(input.Quantity), "input", index)
		}
		for index, output := range record.Outputs {
			h.appendFusionComponentChange(entries, record, players, output, positiveOrOne(output.Quantity), "output", index)
		}
	}
	return nil
}

func (h *Handler) appendFusionComponentChange(entries *[]StudentChangeEntryResponse, record mongomodel.FusionRecord, players map[string]mongomodel.Player, component mongomodel.FusionComponent, delta int, direction string, index int) {
	if component.Kind == "" || component.RefID == "" {
		return
	}
	name, iconPath := h.studentChangeResource(component.Kind, component.RefID)
	*entries = append(*entries, h.studentChangeEntry(record.ID+":"+direction+":"+strconv.Itoa(index), "fusion", "小石合成", record.PlayerID, players, component.Kind, component.RefID, name, iconPath, delta, record.RecipeID, record.CreatedAt))
}

func (h *Handler) appendInventoryTrimChanges(ctx context.Context, entries *[]StudentChangeEntryResponse, players map[string]mongomodel.Player, playerIDs []string, limit int64) error {
	records, err := findAllDashboard[mongomodel.InventoryTrim](
		ctx,
		h.db,
		mongomodel.InventoryTrimsCollection,
		bson.M{"player_id": bson.M{"$in": playerIDs}},
		recentStudentChangeFindOptions(limit),
	)
	if err != nil {
		return err
	}
	for _, record := range records {
		if _, ok := players[record.PlayerID]; !ok {
			continue
		}
		if record.SitoneCount > 0 {
			*entries = append(*entries, h.studentChangeEntry(record.ID+":sitone", "admin_balance", "Admin 平衡", record.PlayerID, players, rewardKindSitone, "", "小石", "", -record.SitoneCount, record.Message, record.CreatedAt))
		}
		if record.OpenPower > 0 {
			*entries = append(*entries, h.studentChangeEntry(record.ID+":open_power", "admin_balance", "Admin 平衡", record.PlayerID, players, rewardKindOpenPower, "", "開源力", "", -record.OpenPower, record.Message, record.CreatedAt))
		}
	}
	return nil
}

func (h *Handler) appendOpenPowerRecordChanges(ctx context.Context, entries *[]StudentChangeEntryResponse, players map[string]mongomodel.Player, playerIDs []string, limit int64) error {
	records, err := findAllDashboard[mongomodel.OpenPowerRecord](
		ctx,
		h.db,
		mongomodel.OpenPowerRecordsCollection,
		bson.M{
			"player_id": bson.M{"$in": playerIDs},
			"reason": bson.M{"$nin": []string{
				"staff_reward",
				"community_stand",
				"shop_purchase",
				"inventory_trim",
			}},
		},
		recentStudentChangeFindOptions(limit),
	)
	if err != nil {
		return err
	}
	for _, record := range records {
		if _, ok := players[record.PlayerID]; !ok || record.Amount == 0 {
			continue
		}
		*entries = append(*entries, h.studentChangeEntry(record.ID, "open_power", studentChangeOpenPowerSourceLabel(record.Reason), record.PlayerID, players, rewardKindOpenPower, "", "開源力", "", record.Amount, record.Source, record.CreatedAt))
	}
	return nil
}

func recentStudentChangeFindOptions(limit int64) *options.FindOptionsBuilder {
	return options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}, {Key: "_id", Value: -1}}).
		SetLimit(limit)
}

func (h *Handler) studentChangeEntry(changeID string, source string, sourceLabel string, playerID string, players map[string]mongomodel.Player, kind string, refID string, name string, iconPath string, delta int, note string, createdAt time.Time) StudentChangeEntryResponse {
	player := players[playerID]
	entry := StudentChangeEntryResponse{
		ChangeID:       changeID,
		Source:         source,
		SourceLabel:    sourceLabel,
		PlayerID:       playerID,
		PlayerNickname: player.Nickname,
		TeamID:         player.TeamID,
		Kind:           kind,
		RefID:          refID,
		Name:           name,
		IconPath:       iconPath,
		Delta:          delta,
		Note:           note,
		CreatedAt:      createdAt,
	}
	if entry.Name == "" {
		entry.Name = refID
	}
	if entry.Name == "" && kind == rewardKindOpenPower {
		entry.Name = "開源力"
	}
	return entry
}

func (h *Handler) studentChangeResource(kind string, refID string) (string, string) {
	switch kind {
	case rewardKindOpenPower:
		return "開源力", ""
	case rewardKindItem:
		if item, ok := h.content.GetItem(refID); ok {
			return item.Name, item.IconPath
		}
	case rewardKindSitone:
		if sitone, ok := h.content.GetSitone(refID); ok {
			return sitone.Name, sitone.IconPath
		}
	}
	return refID, ""
}

func studentChangeOpenPowerSourceLabel(reason string) string {
	switch reason {
	case "quiz_match_completed":
		return "對戰開源力"
	default:
		if strings.TrimSpace(reason) == "" {
			return "開源力變動"
		}
		return reason
	}
}

func positiveOrOne(value int) int {
	if value > 0 {
		return value
	}
	return 1
}
