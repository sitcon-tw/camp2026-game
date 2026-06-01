package catalog

import (
	"net/http"

	"github.com/sitcon-tw/camp2026-game/internal/http/apimodel"
	"github.com/sitcon-tw/camp2026-game/internal/http/httpx"
)

// ListItems godoc
// @Summary List item catalog
// @Description Lists all collectible item definitions and acquisition hints.
// @Tags Catalog
// @Produce json
// @Security AuthCookieAuth
// @Success 200 {object} apimodel.CatalogItemListResponse
// @Failure 401 {object} httpx.ProblemDetails
// @Failure 500 {object} httpx.ProblemDetails
// @Router /catalog/items [get]
func (h *Handler) ListItems(w http.ResponseWriter, _ *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, apimodel.CatalogItemListResponse{
		Items: []apimodel.CatalogItemSummary{
			{
				DefinitionID:    "item-camp-sticker",
				Name:            "Camp Sticker",
				ItemType:        "craft_material",
				AcquisitionHint: "Buy from the shop with open power.",
			},
		},
	})
}
