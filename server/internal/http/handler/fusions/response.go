package fusions

type RecipeListResponse struct {
	Recipes []FusionRecipeResponse `json:"recipes"`
}

type FusionRecipeResponse struct {
	ID          string                    `json:"id" example:"recipe_explore_2026_backpack_s1_s2"`
	BranchID    string                    `json:"branchId,omitempty" example:"branch_2026_camp"`
	Type        string                    `json:"type,omitempty" example:"exploration"`
	StageFrom   int                       `json:"stageFrom,omitempty" example:"1"`
	StageTo     int                       `json:"stageTo,omitempty" example:"2"`
	Name        string                    `json:"name" example:"營地背包小石"`
	Description string                    `json:"description" example:"牠把貼紙、名牌、備用線材都塞進包包，準備在 2026 營隊裡探索下一個任務。"`
	Story       string                    `json:"story,omitempty" example:"牠把貼紙、名牌、備用線材都塞進包包。"`
	ReviewTitle string                    `json:"reviewTitle,omitempty" example:"SITCON 2026 議程表"`
	ReviewURL   string                    `json:"reviewUrl,omitempty" example:"https://sitcon.org/2026/agenda/"`
	Enabled     bool                      `json:"enabled" example:"true"`
	Available   bool                      `json:"available" example:"true"`
	Inputs      []FusionComponentResponse `json:"inputs"`
	Outputs     []FusionComponentResponse `json:"outputs"`
}

type FusionComponentResponse struct {
	Kind               string `json:"kind" example:"item"`
	ID                 string `json:"id" example:"item_adventure_backpack"`
	Name               string `json:"name" example:"冒險背包"`
	Type               string `json:"type,omitempty" example:"material"`
	Rarity             string `json:"rarity,omitempty" example:"common"`
	IconPath           string `json:"iconPath,omitempty" example:"/game-icons/stones/basic_blue.png"`
	Source             string `json:"source,omitempty" example:"shop"`
	AbilityName        string `json:"abilityName,omitempty" example:"穩定輸出"`
	AbilityKind        string `json:"abilityKind,omitempty" example:"answer_score_bonus"`
	AbilityValue       int    `json:"abilityValue,omitempty" example:"5"`
	AbilityCount       int    `json:"abilityCount,omitempty" example:"0"`
	AbilityDescription string `json:"abilityDescription,omitempty" example:"答對時分數提高 5%。"`
	Quantity           int    `json:"quantity" example:"1"`
	OwnedQuantity      int    `json:"ownedQuantity,omitempty" example:"0"`
	MissingQuantity    int    `json:"missingQuantity,omitempty" example:"1"`
}

type CreateRequest struct {
	RecipeID string `json:"recipeId" validate:"required" example:"recipe_explore_2026_backpack_s1_s2"`
}

type CreateResponse struct {
	FusionID string               `json:"fusionId" example:"fusion_abc123"`
	Recipe   FusionRecipeResponse `json:"recipe"`
}

type FillMissingMaterialsResponse struct {
	Recipe              FusionRecipeResponse  `json:"recipe"`
	FilledMaterials     []FillMaterialResult  `json:"filledMaterials"`
	FailedMaterials     []FillMaterialFailure `json:"failedMaterials"`
	PriceOpenPowerSpent int                   `json:"priceOpenPowerSpent" example:"50"`
}

type FillMaterialResult struct {
	Kind     string `json:"kind" example:"item"`
	ID       string `json:"id" example:"item_adventure_backpack"`
	Name     string `json:"name" example:"冒險背包"`
	Quantity int    `json:"quantity" example:"1"`
}

type FillMaterialFailure struct {
	Kind     string `json:"kind" example:"item"`
	ID       string `json:"id" example:"item_adventure_backpack"`
	Name     string `json:"name" example:"冒險背包"`
	Quantity int    `json:"quantity" example:"1"`
	Reason   string `json:"reason" example:"material is not purchasable automatically"`
}
