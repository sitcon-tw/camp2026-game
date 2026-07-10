package fronts

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/sitcon-tw/camp2026-game/internal/http/httpx"
	mongomodel "github.com/sitcon-tw/camp2026-game/internal/mongodb/model"
	"github.com/sitcon-tw/camp2026-game/internal/openpower"
)

var frontCommandCosts = map[string]int{
	"expand":           10,
	"attack":           15,
	"reinforce":        8,
	"repair":           12,
	"rescue":           10,
	"support":          0,
	"answer_challenge": 0,
}

var (
	errFrontRevisionConflict      = errors.New("front revision conflict")
	errDuplicateFrontCommand      = errors.New("duplicate front command")
	errFrontChangedCommandInvalid = errors.New("front changed and command is no longer valid")
	errFrontSitoneNotOwned        = errors.New("sitone is not owned")
	errFrontInsufficientOpenPower = errors.New("insufficient open power")
)

// CreateCommand godoc
// @Summary Create front command
// @Description Accepts a front command for the authenticated player, applies it to the front snapshot, and records the command.
// @Tags fronts
// @Accept json
// @Produce json
// @Security AuthCookieAuth
// @Param frontID path string true "Front ID"
// @Param request body CreateCommandRequest true "Front command"
// @Success 202 {object} CreateCommandResponse
// @Failure 400 {object} httpx.ProblemDetails
// @Failure 401 {object} httpx.ProblemDetails
// @Failure 403 {object} httpx.ProblemDetails
// @Failure 404 {object} httpx.ProblemDetails
// @Failure 409 {object} httpx.ProblemDetails
// @Failure 422 {object} httpx.ProblemDetails
// @Failure 500 {object} httpx.ProblemDetails
// @Failure 503 {object} httpx.ProblemDetails
// @Router /fronts/{frontID}/commands [post]
func (h *Handler) CreateCommand(w http.ResponseWriter, r *http.Request) {
	player, ok := currentPlayer(w, r)
	if !ok || !h.requireDatabase(w, r) {
		return
	}

	front, err := h.frontByID(r.Context(), chi.URLParam(r, "frontID"))
	if errors.Is(err, errFrontNotFound) {
		httpx.WriteProblem(w, r, httpx.NotFound("front not found"))
		return
	}
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("front unavailable", "front_lookup_failed", err))
		return
	}
	front = withCurrentPlayerTeam(front, player)
	if !isParticipatingTerritoryTeam(player.TeamID) {
		httpx.WriteProblem(w, r, httpx.NewError(http.StatusForbidden, "this team can only observe the territory front"))
		return
	}

	var body CreateCommandRequest
	if err := httpx.DecodeJSON(r, &body); err != nil {
		httpx.WriteProblem(w, r, err)
		return
	}
	body = normalizeCommandRequest(body)
	if err := validateCommandRequest(body); err != nil {
		httpx.WriteProblem(w, r, err)
		return
	}

	if body.ClientCommandID != "" {
		existing, found, err := h.findExistingCommand(r.Context(), front.ID, player.ID, body.ClientCommandID)
		if err != nil {
			httpx.WriteProblem(w, r, httpx.InternalServerError("front command failed", "front_command_lookup_failed", err))
			return
		}
		if found {
			h.writeCommandResponse(w, r, http.StatusAccepted, front, player, existing)
			return
		}
	}
	if err := h.validateOwnedFrontSitones(r.Context(), player.ID, body.SitoneIDs); err != nil {
		if errors.Is(err, errFrontSitoneNotOwned) {
			httpx.WriteProblem(w, r, httpx.UnprocessableEntity("sitone loadout exceeds owned quantity"))
			return
		}
		httpx.WriteProblem(w, r, httpx.InternalServerError("front command failed", "front_sitone_lookup_failed", err))
		return
	}
	if body.ExpectedRevision != nil && *body.ExpectedRevision != front.Revision {
		httpx.WriteProblem(w, r, httpx.NewError(http.StatusConflict, "front revision is stale"))
		return
	}
	releaseOpenPowerLock, err := openpower.AcquirePlayerLock(r.Context(), h.db, player.ID)
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("front command failed", "front_open_power_lock_failed", err))
		return
	}
	defer releaseOpenPowerLock()

	if body.ClientCommandID != "" {
		existing, found, lookupErr := h.findExistingCommand(r.Context(), front.ID, player.ID, body.ClientCommandID)
		if lookupErr != nil {
			httpx.WriteProblem(w, r, httpx.InternalServerError("front command failed", "front_command_lookup_failed", lookupErr))
			return
		}
		if found {
			h.writeCommandResponse(w, r, http.StatusAccepted, front, player, existing)
			return
		}
	}
	playerOpenPower, err := openpower.TotalForPlayer(r.Context(), h.db, player.ID)
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("front command failed", "front_open_power_lookup_failed", err))
		return
	}

	command := mongomodel.FrontCommand{
		ID:               newID("front_command"),
		ClientCommandID:  body.ClientCommandID,
		FrontID:          front.ID,
		PlayerID:         player.ID,
		TeamID:           player.TeamID,
		Kind:             body.Kind,
		Type:             body.Kind,
		TargetX:          body.TargetX,
		TargetY:          body.TargetY,
		ExpectedRevision: body.ExpectedRevision,
		SitoneIDs:        append([]string(nil), body.SitoneIDs...),
		SitoneEffect:     frontSitoneEffect(h.content, body.Kind, body.SitoneIDs),
		Payload:          clonePayload(body.Payload),
		CreatedAt:        time.Now().UTC(),
	}
	previousRevision := front.Revision
	front, command, err = applyCommandToFront(front, command, playerOpenPower)
	if err != nil {
		if errors.Is(err, errFrontInsufficientOpenPower) {
			httpx.WriteProblem(w, r, httpx.NewError(http.StatusConflict, "insufficient open power"))
			return
		}
		httpx.WriteProblem(w, r, httpx.UnprocessableEntity(err.Error()))
		return
	}
	h.assignFrontCommandReward(&command)
	front, command, err = h.persistCommandWithRecompute(
		r.Context(), previousRevision, front, command, body.ExpectedRevision != nil, playerOpenPower,
	)
	if errors.Is(err, errFrontRevisionConflict) || errors.Is(err, errFrontChangedCommandInvalid) {
		httpx.WriteProblem(w, r, httpx.NewError(http.StatusConflict, "front changed while applying the command"))
		return
	} else if errors.Is(err, errFrontSitoneNotOwned) {
		httpx.WriteProblem(w, r, httpx.UnprocessableEntity("sitone is not owned"))
		return
	} else if errors.Is(err, errFrontInsufficientOpenPower) {
		httpx.WriteProblem(w, r, httpx.NewError(http.StatusConflict, "insufficient open power"))
		return
	} else if errors.Is(err, errDuplicateFrontCommand) {
		existing, found, lookupErr := h.findExistingCommand(r.Context(), front.ID, player.ID, command.ClientCommandID)
		if lookupErr != nil || !found {
			httpx.WriteProblem(w, r, httpx.NewError(http.StatusConflict, "front command was already submitted"))
			return
		}
		latest, lookupErr := h.frontByID(r.Context(), front.ID)
		if lookupErr != nil {
			httpx.WriteProblem(w, r, httpx.InternalServerError("front unavailable", "front_lookup_failed", lookupErr))
			return
		}
		h.writeCommandResponse(w, r, http.StatusAccepted, latest, player, existing)
		return
	} else if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("front command failed", "front_command_failed", err))
		return
	}
	h.writeCommandResponse(w, r, http.StatusAccepted, front, player, command)
}

func (h *Handler) persistCommandWithRecompute(ctx context.Context, previousRevision int64, front mongomodel.Front, command mongomodel.FrontCommand, allowRecompute bool, playerOpenPower int) (mongomodel.Front, mongomodel.FrontCommand, error) {
	err := h.recordCommand(ctx, previousRevision, front, command)
	for attempt := 0; allowRecompute && errors.Is(err, errFrontRevisionConflict) && attempt < 3; attempt++ {
		latest, lookupErr := h.frontByID(ctx, front.ID)
		if lookupErr != nil {
			return front, command, lookupErr
		}
		previousRevision = latest.Revision
		retryCommand := command
		resetFrontCommandDerivedFields(&retryCommand)
		next, applied, applyErr := applyCommandToFront(latest, retryCommand, playerOpenPower)
		if applyErr != nil {
			return latest, command, fmt.Errorf("%w: %v", errFrontChangedCommandInvalid, applyErr)
		}
		h.assignFrontCommandReward(&applied)
		front, command = next, applied
		err = h.recordCommand(ctx, previousRevision, front, command)
	}
	return front, command, err
}

func resetFrontCommandDerivedFields(command *mongomodel.FrontCommand) {
	if command == nil {
		return
	}
	command.Accepted = false
	command.Applied = false
	command.RejectReason = ""
	command.AffectedCells = nil
	command.CapturedCellCount = 0
	command.EnclosedCellCount = 0
	command.ScoreDelta = 0
	command.FrontOpenPowerDelta = 0
	command.FrontOpenPowerCost = 0
	command.RewardSitoneID = ""
	command.RewardSitoneQuantity = 0
	command.SitoneEffect.AffectedCellBonus = 0
	command.SitoneEffect.DefenseBonus = 0
	command.SitoneEffect.ScoreBonus = 0
}

func (h *Handler) writeCommandResponse(w http.ResponseWriter, r *http.Request, status int, front mongomodel.Front, player mongomodel.Player, command mongomodel.FrontCommand) {
	sitones, err := h.playerFrontSitones(r.Context(), player)
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("front sitones unavailable", "front_sitones_lookup_failed", err))
		return
	}
	playerOpenPower, err := openpower.TotalForPlayer(r.Context(), h.db, player.ID)
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("front open power unavailable", "front_open_power_lookup_failed", err))
		return
	}

	httpx.WriteJSON(w, status, CreateCommandResponse{
		Accepted: true,
		Command:  commandResponse(command),
		Front:    detailResponse(front, player.TeamID, sitones, playerOpenPower),
	})
}

func (h *Handler) findExistingCommand(ctx context.Context, frontID string, playerID string, clientCommandID string) (mongomodel.FrontCommand, bool, error) {
	var command mongomodel.FrontCommand
	err := h.db.Collection(mongomodel.FrontCommandsCollection).FindOne(ctx, bson.M{
		"front_id": frontID, "player_id": playerID, "client_command_id": clientCommandID,
	}).Decode(&command)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return mongomodel.FrontCommand{}, false, nil
	}
	return command, err == nil, err
}

func (h *Handler) recordCommand(ctx context.Context, previousRevision int64, front mongomodel.Front, command mongomodel.FrontCommand) error {
	session, err := h.db.Client().StartSession()
	if err != nil {
		return err
	}
	defer session.EndSession(ctx)

	_, err = session.WithTransaction(ctx, func(ctx context.Context) (any, error) {
		if err := h.validateOwnedFrontSitones(ctx, command.PlayerID, command.SitoneIDs); err != nil {
			return nil, err
		}
		if _, err := h.db.Collection(mongomodel.FrontCommandsCollection).InsertOne(ctx, command); err != nil {
			if mongo.IsDuplicateKeyError(err) {
				return nil, errDuplicateFrontCommand
			}
			return nil, err
		}
		if err := h.deductFrontCommandOpenPower(ctx, command); err != nil {
			return nil, err
		}
		if err := h.grantFrontCommandSitone(ctx, command); err != nil {
			return nil, err
		}
		if _, err := h.db.Collection(mongomodel.FrontsCollection).UpdateMany(
			ctx,
			bson.M{"_id": bson.M{"$ne": front.ID}, "current": true},
			bson.M{"$set": bson.M{"current": false, "updated_at": front.UpdatedAt}},
		); err != nil {
			return nil, err
		}
		if err := h.updateFrontAfterCommand(ctx, previousRevision, front, command); err != nil {
			return nil, err
		}
		return nil, nil
	})
	return err
}

func (h *Handler) deductFrontCommandOpenPower(ctx context.Context, command mongomodel.FrontCommand) error {
	if command.FrontOpenPowerCost <= 0 {
		return nil
	}
	balance, err := openpower.TotalForPlayer(ctx, h.db, command.PlayerID)
	if err != nil {
		return err
	}
	if balance < command.FrontOpenPowerCost {
		return errFrontInsufficientOpenPower
	}
	_, err = h.db.Collection(mongomodel.OpenPowerRecordsCollection).InsertOne(ctx, mongomodel.OpenPowerRecord{
		ID:        "front_open_power_" + command.ID,
		PlayerID:  command.PlayerID,
		Amount:    -command.FrontOpenPowerCost,
		Reason:    "front_command",
		Source:    command.ID,
		CreatedAt: command.CreatedAt,
	})
	return err
}

func (h *Handler) playerOwnsFrontSitone(ctx context.Context, playerID string, sitoneID string) (bool, error) {
	if h.content != nil {
		if _, ok := h.content.GetSitone(sitoneID); !ok {
			return false, nil
		}
	}
	err := h.db.Collection(mongomodel.PlayerSitonesCollection).FindOne(ctx, bson.M{
		"player_id": playerID,
		"sitone_id": sitoneID,
		"quantity":  bson.M{"$gt": 0},
	}).Err()
	if errors.Is(err, mongo.ErrNoDocuments) {
		return false, nil
	}
	return err == nil, err
}

func (h *Handler) validateOwnedFrontSitones(ctx context.Context, playerID string, sitoneIDs []string) error {
	required := sitoneCounts(sitoneIDs)
	if len(required) == 0 {
		return errFrontSitoneNotOwned
	}
	if h.content != nil {
		for sitoneID := range required {
			if _, ok := h.content.GetSitone(sitoneID); !ok {
				return errFrontSitoneNotOwned
			}
		}
	}
	owned, err := h.playerOwnedSitoneCounts(ctx, playerID)
	if err != nil {
		return err
	}
	for sitoneID, quantity := range required {
		if owned[sitoneID] < quantity {
			return errFrontSitoneNotOwned
		}
	}
	return nil
}

func (h *Handler) updateFrontAfterCommand(ctx context.Context, previousRevision int64, front mongomodel.Front, command mongomodel.FrontCommand) error {
	summary := commandSummary(command)
	result, err := h.db.Collection(mongomodel.FrontsCollection).UpdateOne(
		ctx,
		frontRevisionFilter(front.ID, previousRevision),
		bson.M{
			"$set": bson.M{
				"map_id":        front.MapID,
				"map_mode":      front.MapMode,
				"name":          front.Name,
				"status":        front.Status,
				"current":       front.Current,
				"tick":          front.Tick,
				"revision":      front.Revision,
				"teams":         front.Teams,
				"active_events": front.ActiveEvents,
				"leaderboard":   front.Leaderboard,
				"last_command":  summary,
				"territory":     front.Territory,
				"updated_at":    front.UpdatedAt,
			},
			"$setOnInsert": bson.M{
				"_id":        front.ID,
				"created_at": front.CreatedAt,
			},
		},
		options.UpdateOne().SetUpsert(true),
	)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return errFrontRevisionConflict
		}
		return err
	}
	if result.MatchedCount == 0 && result.UpsertedCount == 0 {
		return errFrontRevisionConflict
	}
	return nil
}

func frontRevisionFilter(frontID string, previousRevision int64) bson.M {
	filter := bson.M{"_id": frontID, "revision": previousRevision}
	if previousRevision == 1 {
		filter = bson.M{
			"_id": frontID,
			"$or": bson.A{
				bson.M{"revision": previousRevision},
				bson.M{"revision": bson.M{"$exists": false}},
			},
		}
	}
	return filter
}

func normalizeCommandRequest(body CreateCommandRequest) CreateCommandRequest {
	body.ClientCommandID = strings.TrimSpace(body.ClientCommandID)
	body.Kind = strings.TrimSpace(body.Kind)
	body.Type = strings.TrimSpace(body.Type)
	if body.Kind == "" {
		body.Kind = body.Type
	}
	body.SitoneIDs = normalizedSitoneLoadout(body.SitoneIDs)
	if body.Payload != nil {
		if body.TargetX == nil {
			body.TargetX = payloadInt(body.Payload, "targetX", "target_x", "x")
		}
		if body.TargetY == nil {
			body.TargetY = payloadInt(body.Payload, "targetY", "target_y", "y")
		}
		if body.ExpectedRevision == nil {
			body.ExpectedRevision = payloadInt64(body.Payload, "expectedRevision", "expected_revision", "revision")
		}
	}
	return body
}

func payloadInt(payload map[string]any, keys ...string) *int {
	value := payloadInt64(payload, keys...)
	if value == nil {
		return nil
	}
	converted := int(*value)
	if int64(converted) != *value {
		return nil
	}
	return &converted
}

func payloadInt64(payload map[string]any, keys ...string) *int64 {
	for _, key := range keys {
		value, ok := payload[key]
		if !ok {
			continue
		}
		var converted int64
		switch number := value.(type) {
		case int:
			converted = int64(number)
		case int64:
			converted = number
		case float64:
			converted = int64(number)
			if float64(converted) != number {
				continue
			}
		default:
			continue
		}
		return &converted
	}
	return nil
}

func validateCommandRequest(body CreateCommandRequest) error {
	var details []httpx.ErrorDetail
	if body.Kind == "" {
		details = append(details, httpx.ErrorDetail{Location: "body.kind", Message: "kind is required"})
	} else if _, ok := frontCommandCosts[body.Kind]; !ok {
		details = append(details, httpx.ErrorDetail{Location: "body.kind", Message: "kind is unsupported"})
	}
	if !territoryCommandIsSupported(body.Kind) {
		details = append(details, httpx.ErrorDetail{Location: "body.kind", Message: "kind is unsupported on the territory front"})
	}
	if body.ClientCommandID == "" {
		details = append(details, httpx.ErrorDetail{Location: "body.clientCommandId", Message: "clientCommandId is required"})
	}
	if len(body.SitoneIDs) < 1 || len(body.SitoneIDs) > maxFrontSitoneLoadout {
		details = append(details, httpx.ErrorDetail{Location: "body.sitoneIds", Message: "sitoneIds must contain between 1 and 5 sitones"})
	}
	if body.TargetX == nil || *body.TargetX < 0 || *body.TargetX >= 64 {
		details = append(details, httpx.ErrorDetail{Location: "body.targetX", Message: "targetX must be between 0 and 63"})
	}
	if body.TargetY == nil || *body.TargetY < 0 || *body.TargetY >= 57 {
		details = append(details, httpx.ErrorDetail{Location: "body.targetY", Message: "targetY must be between 0 and 56"})
	}
	if body.ExpectedRevision != nil && *body.ExpectedRevision < 0 {
		details = append(details, httpx.ErrorDetail{Location: "body.expectedRevision", Message: "expectedRevision must be non-negative"})
	}
	if len(details) > 0 {
		return httpx.UnprocessableEntity("invalid front command", details...)
	}
	return nil
}

func applyCommandToFront(front mongomodel.Front, command mongomodel.FrontCommand, playerOpenPower int) (mongomodel.Front, mongomodel.FrontCommand, error) {
	return applyTerritoryCommand(front, command, playerOpenPower)
}

func isPlayableFrontStatus(status string) bool {
	return status == "" || status == mongomodel.FrontStatusOpenPlay || status == "surge" || status == "booth_window"
}

func activeEventKindAtFrontCell(front mongomodel.Front, cellID string) string {
	for _, event := range front.ActiveEvents {
		if event.CellID == cellID {
			return event.Kind
		}
	}
	return ""
}

func removeActiveFrontEventAtCell(front *mongomodel.Front, cellID string) {
	active := front.ActiveEvents[:0]
	for _, event := range front.ActiveEvents {
		if event.CellID == cellID {
			continue
		}
		active = append(active, event)
	}
	front.ActiveEvents = active
}

func clampFrontInt(value int, minValue int, maxValue int) int {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}

func maxFrontInt(a int, b int) int {
	if a > b {
		return a
	}
	return b
}

func commandSummary(command mongomodel.FrontCommand) mongomodel.FrontCommandSummary {
	return mongomodel.FrontCommandSummary{
		ID:                   command.ID,
		ClientCommandID:      command.ClientCommandID,
		Kind:                 command.Kind,
		Type:                 command.Type,
		PlayerID:             command.PlayerID,
		TeamID:               command.TeamID,
		TargetX:              command.TargetX,
		TargetY:              command.TargetY,
		ExpectedRevision:     command.ExpectedRevision,
		AffectedCells:        append([]mongomodel.FrontCoordinate(nil), command.AffectedCells...),
		CapturedCellCount:    command.CapturedCellCount,
		EnclosedCellCount:    command.EnclosedCellCount,
		ScoreDelta:           command.ScoreDelta,
		FrontOpenPowerDelta:  command.FrontOpenPowerDelta,
		FrontOpenPowerCost:   command.FrontOpenPowerCost,
		RewardSitoneID:       command.RewardSitoneID,
		RewardSitoneQuantity: command.RewardSitoneQuantity,
		SitoneIDs:            append([]string(nil), command.SitoneIDs...),
		SitoneEffect:         command.SitoneEffect,
		Payload:              clonePayload(command.Payload),
		Accepted:             command.Accepted,
		Applied:              command.Applied,
		RejectReason:         command.RejectReason,
		CreatedAt:            command.CreatedAt,
	}
}

func newID(prefix string) string {
	return prefix + "_" + bson.NewObjectID().Hex()
}
