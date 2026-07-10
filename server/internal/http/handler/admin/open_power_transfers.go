package admin

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/sitcon-tw/camp2026-game/internal/http/httpx"
	mongomodel "github.com/sitcon-tw/camp2026-game/internal/mongodb/model"
)

const (
	adminOpenPowerTransfersDefaultLimit int64 = 200
	adminOpenPowerTransfersMaxLimit     int64 = 1000
)

type OpenPowerTransferEntryResponse struct {
	TransferID        string    `json:"transferId" example:"open_power_transfer_507f1f77bcf86cd799439011"`
	TeamID            string    `json:"teamId" example:"8M4RXP"`
	TeamName          string    `json:"teamName,omitempty" example:"Blue Team"`
	SenderPlayerID    string    `json:"senderPlayerId" example:"7H9K2Q"`
	SenderNickname    string    `json:"senderNickname,omitempty" example:"Alice"`
	RecipientPlayerID string    `json:"recipientPlayerId" example:"2QK9H7"`
	RecipientNickname string    `json:"recipientNickname,omitempty" example:"Bob"`
	Amount            int       `json:"amount" example:"100"`
	CreatedAt         time.Time `json:"createdAt"`
}

type OpenPowerTransfersResponse struct {
	Transfers []OpenPowerTransferEntryResponse `json:"transfers"`
}

// ListOpenPowerTransfers godoc
// @Summary List open power transfers as admin
// @Description Admin-only endpoint. Returns recent same-team open power transfers with sender, recipient, and team metadata.
// @Tags admin
// @Produce json
// @Param limit query int false "Maximum number of transfer records to return"
// @Success 200 {object} OpenPowerTransfersResponse
// @Failure 401 {object} httpx.ProblemDetails
// @Failure 500 {object} httpx.ProblemDetails
// @Failure 503 {object} httpx.ProblemDetails
// @Router /admin/open-power-transfers [get]
func (h *Handler) ListOpenPowerTransfers(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdmin(w, r) || !h.requireDatabase(w, r) {
		return
	}

	limit := adminOpenPowerTransfersLimit(r.URL.Query().Get("limit"))
	transfers, err := h.openPowerTransferResponses(r.Context(), limit)
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("open power transfers unavailable", "admin_open_power_transfers_lookup_failed", err))
		return
	}

	httpx.WriteJSON(w, http.StatusOK, OpenPowerTransfersResponse{Transfers: transfers})
}

func adminOpenPowerTransfersLimit(value string) int64 {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return adminOpenPowerTransfersDefaultLimit
	}
	limit, err := strconv.ParseInt(trimmed, 10, 64)
	if err != nil || limit <= 0 {
		return adminOpenPowerTransfersDefaultLimit
	}
	if limit > adminOpenPowerTransfersMaxLimit {
		return adminOpenPowerTransfersMaxLimit
	}
	return limit
}

func (h *Handler) openPowerTransferResponses(ctx context.Context, limit int64) ([]OpenPowerTransferEntryResponse, error) {
	records, err := findAllDashboard[mongomodel.OpenPowerTransfer](
		ctx,
		h.db,
		mongomodel.OpenPowerTransfersCollection,
		bson.M{},
		options.Find().
			SetSort(bson.D{{Key: "created_at", Value: -1}, {Key: "_id", Value: -1}}).
			SetLimit(limit),
	)
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return []OpenPowerTransferEntryResponse{}, nil
	}

	players, err := h.openPowerTransferPlayers(ctx, records)
	if err != nil {
		return nil, err
	}
	teams, err := h.openPowerTransferTeams(ctx, records)
	if err != nil {
		return nil, err
	}

	responses := make([]OpenPowerTransferEntryResponse, 0, len(records))
	for _, record := range records {
		responses = append(responses, openPowerTransferResponse(record, players, teams))
	}
	return responses, nil
}

func openPowerTransferResponse(
	record mongomodel.OpenPowerTransfer,
	players map[string]mongomodel.Player,
	teams map[string]mongomodel.Team,
) OpenPowerTransferEntryResponse {
	response := OpenPowerTransferEntryResponse{
		TransferID:        record.ID,
		TeamID:            record.TeamID,
		SenderPlayerID:    record.SenderPlayerID,
		RecipientPlayerID: record.RecipientPlayerID,
		Amount:            record.Amount,
		CreatedAt:         record.CreatedAt,
	}
	if sender, ok := players[record.SenderPlayerID]; ok {
		response.SenderNickname = sender.Nickname
	}
	if recipient, ok := players[record.RecipientPlayerID]; ok {
		response.RecipientNickname = recipient.Nickname
	}
	if team, ok := teams[record.TeamID]; ok {
		response.TeamName = team.Name
	}
	return response
}

func (h *Handler) openPowerTransferPlayers(ctx context.Context, records []mongomodel.OpenPowerTransfer) (map[string]mongomodel.Player, error) {
	ids := make([]string, 0, len(records)*2)
	seen := make(map[string]struct{}, len(records)*2)
	for _, record := range records {
		for _, playerID := range []string{record.SenderPlayerID, record.RecipientPlayerID} {
			if playerID == "" {
				continue
			}
			if _, ok := seen[playerID]; ok {
				continue
			}
			seen[playerID] = struct{}{}
			ids = append(ids, playerID)
		}
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

func (h *Handler) openPowerTransferTeams(ctx context.Context, records []mongomodel.OpenPowerTransfer) (map[string]mongomodel.Team, error) {
	ids := make([]string, 0, len(records))
	seen := make(map[string]struct{}, len(records))
	for _, record := range records {
		if record.TeamID == "" {
			continue
		}
		if _, ok := seen[record.TeamID]; ok {
			continue
		}
		seen[record.TeamID] = struct{}{}
		ids = append(ids, record.TeamID)
	}
	if len(ids) == 0 {
		return map[string]mongomodel.Team{}, nil
	}

	teams, err := findAllDashboard[mongomodel.Team](
		ctx,
		h.db,
		mongomodel.TeamsCollection,
		bson.M{"_id": bson.M{"$in": ids}},
	)
	if err != nil {
		return nil, err
	}
	out := make(map[string]mongomodel.Team, len(teams))
	for _, team := range teams {
		out[team.ID] = team
	}
	return out, nil
}
