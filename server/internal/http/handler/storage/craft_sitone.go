package storage

import (
	"net/http"

	"github.com/sitcon-tw/camp2026-game/internal/http/apimodel"
	"github.com/sitcon-tw/camp2026-game/internal/http/httpx"
)

// CraftSitone godoc
// @Summary Craft sitone
// @Description Uses sitones and items to craft a new sitone.
// @Tags Crafting
// @Accept json
// @Produce json
// @Security AuthCookieAuth
// @Param request body apimodel.CraftRequest true "Craft request"
// @Success 201 {object} apimodel.CraftResponse
// @Failure 400 {object} httpx.ProblemDetails
// @Failure 401 {object} httpx.ProblemDetails
// @Failure 422 {object} httpx.ProblemDetails
// @Failure 501 {object} httpx.ProblemDetails
// @Router /crafting [post]
func (h *Handler) CraftSitone(w http.ResponseWriter, r *http.Request) {
	var body apimodel.CraftRequest
	if err := httpx.DecodeJSON(r, &body); err != nil {
		httpx.WriteProblem(w, r, err)
		return
	}
	if err := httpx.ValidateStruct(body); err != nil {
		httpx.WriteProblem(w, r, err)
		return
	}

	httpx.WriteProblem(w, r, httpx.NotImplemented("storage crafting is not implemented yet"))
}
