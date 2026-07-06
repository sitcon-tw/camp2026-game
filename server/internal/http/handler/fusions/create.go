package fusions

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/sitcon-tw/camp2026-game/internal/content"
	"github.com/sitcon-tw/camp2026-game/internal/http/httpx"
	"github.com/sitcon-tw/camp2026-game/internal/http/playerevents"
	mongomodel "github.com/sitcon-tw/camp2026-game/internal/mongodb/model"
)

var (
	errInsufficientMaterials         = errors.New("insufficient fusion materials")
	errFusionTransactionsUnavailable = errors.New("fusion requires transaction support")
)

// Create godoc
// @Summary Create fusion
// @Description Consumes the recipe inputs and grants recipe outputs to the authenticated player.
// @Tags fusions
// @Accept json
// @Produce json
// @Security AuthCookieAuth
// @Param request body CreateRequest true "Fusion request"
// @Success 201 {object} CreateResponse
// @Failure 400 {object} httpx.ProblemDetails
// @Failure 401 {object} httpx.ProblemDetails
// @Failure 404 {object} httpx.ProblemDetails
// @Failure 409 {object} httpx.ProblemDetails
// @Failure 422 {object} httpx.ProblemDetails
// @Failure 500 {object} httpx.ProblemDetails
// @Failure 503 {object} httpx.ProblemDetails
// @Router /fusions [post]
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	player, ok := currentPlayer(w, r)
	if !ok || !h.requireContent(w, r) || !h.requireDatabase(w, r) || !h.requireMongoClient(w, r) {
		return
	}

	var body CreateRequest
	if err := httpx.DecodeJSON(r, &body); err != nil {
		httpx.WriteProblem(w, r, err)
		return
	}
	body.RecipeID = strings.TrimSpace(body.RecipeID)
	if err := httpx.ValidateStruct(body); err != nil {
		httpx.WriteProblem(w, r, err)
		return
	}

	recipe, ok := h.content.GetFusionRecipe(body.RecipeID)
	if !ok || !recipe.Enabled {
		httpx.WriteProblem(w, r, httpx.NotFound("fusion recipe not found"))
		return
	}

	fusionID, err := h.createFusion(r.Context(), player.ID, recipe)
	if err != nil {
		if errors.Is(err, errInsufficientMaterials) {
			httpx.WriteProblem(w, r, httpx.NewError(http.StatusConflict, "insufficient fusion materials"))
			return
		}
		if errors.Is(err, errFusionTransactionsUnavailable) {
			httpx.WriteProblem(w, r, &httpx.Error{
				Status: http.StatusServiceUnavailable,
				Detail: errFusionTransactionsUnavailable.Error(),
				Code:   "fusion_transactions_unavailable",
				Cause:  err,
			})
			return
		}
		httpx.WriteProblem(w, r, httpx.InternalServerError("fusion failed", "fusion_create_failed", err))
		return
	}
	h.publishFusionEvents(player.ID, recipe)

	inventory, err := h.playerInventory(r.Context(), player.ID)
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("fusion inventory unavailable", "fusion_inventory_lookup_failed", err))
		return
	}
	response, err := h.recipeResponse(recipe, inventory)
	if err != nil {
		httpx.WriteProblem(w, r, httpx.InternalServerError("fusion recipe is inconsistent", "fusion_recipe_response_failed", err))
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, CreateResponse{
		FusionID: fusionID,
		Recipe:   response,
	})
}

func (h *Handler) createFusion(ctx context.Context, playerID string, recipe content.FusionRecipe) (string, error) {
	if h.client == nil {
		return "", errFusionTransactionsUnavailable
	}

	fusionID, err := h.createFusionWithTransaction(ctx, playerID, recipe)
	if err != nil && transactionUnsupported(err) {
		return "", fmt.Errorf("%w: %w", errFusionTransactionsUnavailable, err)
	}
	return fusionID, err
}

func (h *Handler) createFusionWithTransaction(ctx context.Context, playerID string, recipe content.FusionRecipe) (string, error) {
	session, err := h.client.StartSession()
	if err != nil {
		return "", err
	}
	defer session.EndSession(ctx)

	fusionID := newID("fusion")
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

		for _, input := range recipe.Inputs {
			if err := h.consumeComponent(ctx, playerID, input); err != nil {
				return err
			}
		}
		for _, output := range recipe.Outputs {
			if err := h.grantComponent(ctx, playerID, output); err != nil {
				return err
			}
		}
		if err := h.insertFusionRecord(ctx, fusionID, playerID, recipe, time.Now()); err != nil {
			return err
		}

		if err := session.CommitTransaction(ctx); err != nil {
			return err
		}
		committed = true
		return nil
	})
	if err != nil {
		return "", err
	}
	return fusionID, nil
}

func (h *Handler) publishFusionEvents(playerID string, recipe content.FusionRecipe) {
	if h.broker == nil {
		return
	}
	occurredAt := time.Now().UTC().Format(time.RFC3339Nano)
	for _, output := range recipe.Outputs {
		event := playerevents.RewardGrantedEvent{
			Kind:       output.Kind,
			RefID:      output.ID,
			Quantity:   output.Quantity,
			Source:     "fusion",
			OccurredAt: occurredAt,
		}
		switch output.Kind {
		case content.FusionKindItem:
			item, ok := h.content.GetItem(output.ID)
			if !ok {
				continue
			}
			event.Name = item.Name
			event.ItemType = item.Type
			event.IconPath = item.IconPath
		case content.FusionKindSitone:
			sitone, ok := h.content.GetSitone(output.ID)
			if !ok {
				continue
			}
			event.Name = sitone.Name
			event.SitoneType = sitone.Type
			event.IconPath = sitone.IconPath
		default:
			continue
		}
		h.broker.Publish(playerID, playerevents.Event{
			Name:   "reward_granted",
			Reward: &event,
		})
	}
}

func (h *Handler) consumeComponent(ctx context.Context, playerID string, component content.FusionComponent) error {
	collection, field, err := inventoryCollection(component.Kind)
	if err != nil {
		return err
	}
	result, err := h.db.Collection(collection).UpdateOne(
		ctx,
		bson.M{
			"player_id": playerID,
			field:       component.ID,
			"quantity":  bson.M{"$gte": component.Quantity},
		},
		bson.M{"$inc": bson.M{"quantity": -component.Quantity}},
	)
	if err != nil {
		return err
	}
	if result.ModifiedCount != 1 {
		return errInsufficientMaterials
	}
	return nil
}

func (h *Handler) grantComponent(ctx context.Context, playerID string, component content.FusionComponent) error {
	collection, field, err := inventoryCollection(component.Kind)
	if err != nil {
		return err
	}
	_, err = h.db.Collection(collection).UpdateOne(
		ctx,
		bson.M{
			"player_id": playerID,
			field:       component.ID,
		},
		bson.M{
			"$setOnInsert": bson.M{
				"_id":       newID("player_" + component.Kind),
				"player_id": playerID,
				field:       component.ID,
			},
			"$inc": bson.M{"quantity": component.Quantity},
		},
		options.UpdateOne().SetUpsert(true),
	)
	return err
}

func (h *Handler) insertFusionRecord(ctx context.Context, fusionID string, playerID string, recipe content.FusionRecipe, createdAt time.Time) error {
	_, err := h.db.Collection(mongomodel.FusionRecordsCollection).InsertOne(ctx, mongomodel.FusionRecord{
		ID:        fusionID,
		PlayerID:  playerID,
		RecipeID:  recipe.ID,
		Inputs:    modelComponents(recipe.Inputs),
		Outputs:   modelComponents(recipe.Outputs),
		CreatedAt: createdAt,
	})
	return err
}

func inventoryCollection(kind string) (collection string, idField string, err error) {
	switch kind {
	case content.FusionKindItem:
		return mongomodel.PlayerItemsCollection, "item_id", nil
	case content.FusionKindSitone:
		return mongomodel.PlayerSitonesCollection, "sitone_id", nil
	default:
		return "", "", errors.New("unsupported fusion component kind")
	}
}

func transactionUnsupported(err error) bool {
	var commandError mongo.CommandError
	return errors.As(err, &commandError) &&
		commandError.HasErrorCodeWithMessage(20, "Transaction numbers")
}

func modelComponents(components []content.FusionComponent) []mongomodel.FusionComponent {
	out := make([]mongomodel.FusionComponent, 0, len(components))
	for _, component := range components {
		out = append(out, mongomodel.FusionComponent{
			Kind:     component.Kind,
			RefID:    component.ID,
			Quantity: component.Quantity,
		})
	}
	return out
}

func newID(prefix string) string {
	return prefix + "_" + bson.NewObjectID().Hex()
}
