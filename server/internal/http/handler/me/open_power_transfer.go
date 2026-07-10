package me

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/sitcon-tw/camp2026-game/internal/http/httpx"
	mongomodel "github.com/sitcon-tw/camp2026-game/internal/mongodb/model"
	"github.com/sitcon-tw/camp2026-game/internal/openpower"
)

var errTransferInsufficientOpenPower = errors.New("insufficient open power")

// CreateOpenPowerTransfer godoc
// @Summary Transfer open power to a teammate
// @Description Transfers open power from the authenticated player to another player in the same team and records the transfer.
// @Tags me
// @Accept json
// @Produce json
// @Security AuthCookieAuth
// @Param request body OpenPowerTransferRequest true "Open power transfer request"
// @Success 201 {object} OpenPowerTransferResponse
// @Failure 400 {object} httpx.ProblemDetails
// @Failure 401 {object} httpx.ProblemDetails
// @Failure 403 {object} httpx.ProblemDetails
// @Failure 404 {object} httpx.ProblemDetails
// @Failure 409 {object} httpx.ProblemDetails
// @Failure 422 {object} httpx.ProblemDetails
// @Failure 500 {object} httpx.ProblemDetails
// @Failure 503 {object} httpx.ProblemDetails
// @Router /me/open-power-transfers [post]
func (h *Handler) CreateOpenPowerTransfer(w http.ResponseWriter, r *http.Request) {
	player, ok := currentPlayer(w, r)
	if !ok {
		return
	}
	if h.db == nil {
		httpx.WriteProblem(w, r, httpx.ServiceUnavailable("database is unavailable"))
		return
	}

	var body OpenPowerTransferRequest
	if err := httpx.DecodeJSON(r, &body); err != nil {
		httpx.WriteProblem(w, r, err)
		return
	}
	body.RecipientPlayerID = strings.TrimSpace(body.RecipientPlayerID)
	if err := httpx.ValidateStruct(body); err != nil {
		httpx.WriteProblem(w, r, err)
		return
	}
	if player.TeamID == "" {
		httpx.WriteProblem(w, r, httpx.NewError(http.StatusBadRequest, "player has no team"))
		return
	}
	if body.RecipientPlayerID == player.ID {
		httpx.WriteProblem(w, r, httpx.NewError(http.StatusBadRequest, "cannot transfer open power to self"))
		return
	}

	recipient, err := h.findTransferRecipient(r.Context(), body.RecipientPlayerID)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			httpx.WriteProblem(w, r, httpx.NotFound("recipient player not found"))
			return
		}
		httpx.WriteProblem(w, r, httpx.InternalServerError("open power transfer failed", "open_power_transfer_recipient_lookup_failed", err))
		return
	}
	if recipient.TeamID == "" || recipient.TeamID != player.TeamID {
		httpx.WriteProblem(w, r, httpx.NewError(http.StatusForbidden, "open power transfers require same team"))
		return
	}

	result, err := h.createOpenPowerTransfer(r.Context(), player, recipient, body.Amount)
	if err != nil {
		if errors.Is(err, errTransferInsufficientOpenPower) {
			httpx.WriteProblem(w, r, httpx.NewError(http.StatusConflict, "insufficient open power for transfer"))
			return
		}
		httpx.WriteProblem(w, r, httpx.InternalServerError("open power transfer failed", "open_power_transfer_failed", err))
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, OpenPowerTransferResponse{
		TransferID:              result.transfer.ID,
		SenderPlayerID:          player.ID,
		SenderNickname:          player.Nickname,
		RecipientPlayerID:       recipient.ID,
		RecipientNickname:       recipient.Nickname,
		TeamID:                  result.transfer.TeamID,
		Amount:                  result.transfer.Amount,
		SenderOpenPowerAfter:    result.senderOpenPowerAfter,
		RecipientOpenPowerAfter: result.recipientOpenPowerAfter,
		CreatedAt:               result.transfer.CreatedAt,
	})
}

func (h *Handler) findTransferRecipient(ctx context.Context, playerID string) (mongomodel.Player, error) {
	var player mongomodel.Player
	err := h.db.Collection(mongomodel.PlayersCollection).
		FindOne(
			ctx,
			bson.M{"_id": playerID},
			options.FindOne().SetProjection(bson.D{
				{Key: "auth_token", Value: 0},
				{Key: "qrcode_token", Value: 0},
				{Key: "default_sitone_ids", Value: 0},
			}),
		).
		Decode(&player)
	return player, err
}

type openPowerTransferResult struct {
	transfer                mongomodel.OpenPowerTransfer
	senderOpenPowerAfter    int
	recipientOpenPowerAfter int
}

func (h *Handler) createOpenPowerTransfer(ctx context.Context, sender mongomodel.Player, recipient mongomodel.Player, amount int) (openPowerTransferResult, error) {
	releaseLocks, err := openpower.AcquirePlayerLocks(ctx, h.db, sender.ID, recipient.ID)
	if err != nil {
		return openPowerTransferResult{}, err
	}
	defer releaseLocks()

	if h.client == nil {
		return h.createOpenPowerTransferWithoutTransaction(ctx, sender, recipient, amount)
	}

	result, err := h.createOpenPowerTransferWithTransaction(ctx, sender, recipient, amount)
	if err != nil && openPowerTransferTransactionUnsupported(err) {
		return h.createOpenPowerTransferWithoutTransaction(ctx, sender, recipient, amount)
	}
	return result, err
}

func (h *Handler) createOpenPowerTransferWithTransaction(ctx context.Context, sender mongomodel.Player, recipient mongomodel.Player, amount int) (openPowerTransferResult, error) {
	session, err := h.client.StartSession()
	if err != nil {
		return openPowerTransferResult{}, err
	}
	defer session.EndSession(ctx)

	var result openPowerTransferResult
	err = mongo.WithSession(ctx, session, func(ctx context.Context) error {
		if err := session.StartTransaction(); err != nil {
			return err
		}
		committed := false
		defer func() {
			if !committed {
				_ = session.AbortTransaction(context.Background())
			}
		}()

		var err error
		result, err = h.createOpenPowerTransferWithoutTransaction(ctx, sender, recipient, amount)
		if err != nil {
			return err
		}

		if err := session.CommitTransaction(ctx); err != nil {
			return err
		}
		committed = true
		return nil
	})
	if err != nil {
		return openPowerTransferResult{}, err
	}
	return result, nil
}

func (h *Handler) createOpenPowerTransferWithoutTransaction(ctx context.Context, sender mongomodel.Player, recipient mongomodel.Player, amount int) (openPowerTransferResult, error) {
	senderOpenPower, err := openpower.TotalForPlayer(ctx, h.db, sender.ID)
	if err != nil {
		return openPowerTransferResult{}, err
	}
	if senderOpenPower < amount {
		return openPowerTransferResult{}, errTransferInsufficientOpenPower
	}
	recipientOpenPower, err := openpower.TotalForPlayer(ctx, h.db, recipient.ID)
	if err != nil {
		return openPowerTransferResult{}, err
	}

	now := time.Now().UTC()
	transferID := newOpenPowerID("open_power_transfer")
	senderRecordID := newOpenPowerID("open_power")
	recipientRecordID := newOpenPowerID("open_power")
	transfer := mongomodel.OpenPowerTransfer{
		ID:                transferID,
		SenderPlayerID:    sender.ID,
		RecipientPlayerID: recipient.ID,
		TeamID:            sender.TeamID,
		Amount:            amount,
		SenderRecordID:    senderRecordID,
		RecipientRecordID: recipientRecordID,
		CreatedAt:         now,
	}

	if err := h.insertOpenPowerTransferRecords(ctx, transfer); err != nil {
		return openPowerTransferResult{}, err
	}
	if err := h.insertOpenPowerTransfer(ctx, transfer); err != nil {
		return openPowerTransferResult{}, err
	}

	return openPowerTransferResult{
		transfer:                transfer,
		senderOpenPowerAfter:    senderOpenPower - amount,
		recipientOpenPowerAfter: recipientOpenPower + amount,
	}, nil
}

func (h *Handler) insertOpenPowerTransferRecords(ctx context.Context, transfer mongomodel.OpenPowerTransfer) error {
	_, err := h.db.Collection(mongomodel.OpenPowerRecordsCollection).InsertMany(ctx, []any{
		mongomodel.OpenPowerRecord{
			ID:        transfer.SenderRecordID,
			PlayerID:  transfer.SenderPlayerID,
			Amount:    -transfer.Amount,
			Reason:    "open_power_transfer_out",
			Source:    transfer.ID,
			CreatedAt: transfer.CreatedAt,
		},
		mongomodel.OpenPowerRecord{
			ID:        transfer.RecipientRecordID,
			PlayerID:  transfer.RecipientPlayerID,
			Amount:    transfer.Amount,
			Reason:    "open_power_transfer_in",
			Source:    transfer.ID,
			CreatedAt: transfer.CreatedAt,
		},
	})
	return err
}

func (h *Handler) insertOpenPowerTransfer(ctx context.Context, transfer mongomodel.OpenPowerTransfer) error {
	_, err := h.db.Collection(mongomodel.OpenPowerTransfersCollection).InsertOne(ctx, transfer)
	return err
}

func openPowerTransferTransactionUnsupported(err error) bool {
	var commandError mongo.CommandError
	return errors.As(err, &commandError) &&
		commandError.HasErrorCodeWithMessage(20, "Transaction numbers")
}

func newOpenPowerID(prefix string) string {
	return prefix + "_" + bson.NewObjectID().Hex()
}
