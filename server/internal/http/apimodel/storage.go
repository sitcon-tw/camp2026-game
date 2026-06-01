package apimodel

type SitoneListResponse struct {
	Sitones []SitoneSummary `json:"sitones"`
}

type ItemListResponse struct {
	Items []ItemSummary `json:"items"`
}

type StorageSummaryResponse struct {
	Sitones          []SitoneSummary `json:"sitones"`
	Items            []ItemSummary   `json:"items"`
	CraftableRecipes int             `json:"craftableRecipes" example:"1"`
}

type RecipeListResponse struct {
	Recipes []RecipeSummary `json:"recipes"`
}

type RecipeSummary struct {
	RecipeID                   string `json:"recipeId" example:"recipe_engineering_skin"`
	Name                       string `json:"name" example:"Engineering Skin"`
	RequiredSitoneType         string `json:"requiredSitoneType,omitempty" example:"engineering"`
	RequiredSitoneDefinitionID string `json:"requiredSitoneDefinitionId,omitempty" example:"sitone-engineering"`
	RequiredItemID             string `json:"requiredItemId" example:"item-camp-sticker"`
	RequiredItemQuantity       int    `json:"requiredItemQuantity" example:"1"`
	OutputDefinitionID         string `json:"outputDefinitionId" example:"sitone-engineering-skin"`
	Unlocked                   bool   `json:"unlocked" example:"true"`
	Craftable                  bool   `json:"craftable" example:"true"`
	AcquisitionHint            string `json:"acquisitionHint,omitempty" example:"Complete engineering missions."`
}

type CatalogSitoneListResponse struct {
	Sitones []CatalogSitoneSummary `json:"sitones"`
}

type CatalogSitoneSummary struct {
	DefinitionID    string `json:"definitionId" example:"sitone-engineering"`
	Name            string `json:"name" example:"Engineering Sitone"`
	Type            string `json:"type" example:"engineering"`
	Rarity          string `json:"rarity" example:"rare"`
	AcquisitionHint string `json:"acquisitionHint" example:"Complete engineering missions."`
}

type CatalogItemListResponse struct {
	Items []CatalogItemSummary `json:"items"`
}

type CatalogItemSummary struct {
	DefinitionID    string `json:"definitionId" example:"item-camp-sticker"`
	Name            string `json:"name" example:"Camp Sticker"`
	ItemType        string `json:"itemType" example:"craft_material"`
	AcquisitionHint string `json:"acquisitionHint" example:"Buy from the shop with open power."`
}

type CraftRequest struct {
	RecipeID  string   `json:"recipeId" validate:"required" example:"recipe_engineering_skin"`
	SitoneIDs []string `json:"sitoneIds" validate:"required,min=1,dive,required" example:"S9K2QA"`
	ItemIDs   []string `json:"itemIds" validate:"required,min=1,dive,required" example:"pit_01HR9Z7E2Z2VJ2QZ4P4Z"`
}

type CraftResponse struct {
	ConsumedSitoneIDs []string       `json:"consumedSitoneIds" example:"S9K2QA"`
	ConsumedItemIDs   []string       `json:"consumedItemIds" example:"pit_01HR9Z7E2Z2VJ2QZ4P4Z"`
	CreatedSitone     *SitoneSummary `json:"createdSitone,omitempty"`
	CreatedItem       *ItemSummary   `json:"createdItem,omitempty"`
}
