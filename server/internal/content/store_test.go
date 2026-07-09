package content

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadSitones(t *testing.T) {
	dir := writeContent(t, `
[[sitones]]
id = "stone_engineering_base"
name = "工程型小石"
type = "engineering"
rarity = "base"
style = "default"
description = "完成技術任務、分享解法或協助除錯。"
icon_path = " /game-icons/stones/basic_blue.png "
ability_name = "穩定輸出"
ability_kind = "answer_score_bonus"
ability_value = 5
ability_count = 0
ability_description = "答對時分數提高 5%。"

[[sitones]]
id = "stone_explorer_base"
name = "探索型小石"
type = "exploration"
rarity = "base"
style = "default"
description = "逛攤位、問問題、參與社群事件。"
ability_name = "拾荒直覺"
ability_kind = "material_drop_rate"
ability_value = 5
ability_count = 0
ability_description = "對戰素材掉落率提高 5%。"
`)

	store, err := Load(dir)
	if err != nil {
		t.Fatalf("load content: %v", err)
	}

	sitones := store.ListSitones()
	if len(sitones) != 2 {
		t.Fatalf("expected 2 sitones, got %d", len(sitones))
	}
	if sitones[0].ID != "stone_engineering_base" || sitones[1].ID != "stone_explorer_base" {
		t.Fatalf("expected sitones sorted by id, got %#v", sitones)
	}

	sitone, ok := store.GetSitone("stone_engineering_base")
	if !ok {
		t.Fatal("expected stone_engineering_base to exist")
	}
	if sitone.Name != "工程型小石" {
		t.Fatalf("expected engineering sitone name, got %q", sitone.Name)
	}
	if sitone.IconPath != "/game-icons/stones/basic_blue.png" {
		t.Fatalf("expected engineering sitone icon path, got %q", sitone.IconPath)
	}
	if sitone.AbilityName != "穩定輸出" || sitone.AbilityKind != SitoneAbilityAnswerScoreBonus || sitone.AbilityValue != 5 || sitone.AbilityCount != 0 {
		t.Fatalf("unexpected engineering sitone ability: %#v", sitone)
	}
	if _, ok := store.GetSitone("missing"); ok {
		t.Fatal("expected missing sitone not to exist")
	}

	sitones[0].ID = "mutated"
	sitones = store.ListSitones()
	if sitones[0].ID != "stone_engineering_base" {
		t.Fatalf("expected ListSitones to return a copy, got %q", sitones[0].ID)
	}
}

func TestLoadItems(t *testing.T) {
	dir := writeContent(t, validSitonesTOML(), `
[[items]]
id = "item_maze_map"
name = "迷宮地圖"
type = "material"
rarity = "common"
description = "迷宮地圖，可用於小石合成。"
icon_path = " /game-icons/items/item_maze_map.png "
source = "shop"
purchasable = true
enabled = true
price_open_power = 50

[[items]]
id = "item_adventure_backpack"
name = "冒險背包"
type = "material"
rarity = "common"
description = "冒險背包，可用於小石合成。"
`)

	store, err := Load(dir)
	if err != nil {
		t.Fatalf("load content: %v", err)
	}

	items := store.ListItems()
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	if items[0].ID != "item_adventure_backpack" || items[1].ID != "item_maze_map" {
		t.Fatalf("expected items sorted by id, got %#v", items)
	}

	item, ok := store.GetItem("item_maze_map")
	if !ok {
		t.Fatal("expected item_maze_map to exist")
	}
	if item.Name != "迷宮地圖" {
		t.Fatalf("expected maze map name, got %q", item.Name)
	}
	if !item.Purchasable || !item.Enabled || item.PriceOpenPower != 50 {
		t.Fatalf("expected maze map shop metadata, got %#v", item)
	}
	if item.IconPath != "/game-icons/items/item_maze_map.png" {
		t.Fatalf("expected maze map icon path, got %q", item.IconPath)
	}
	if item.Source != "shop" {
		t.Fatalf("expected maze map source shop, got %q", item.Source)
	}
	if item, ok := store.GetItem("item_adventure_backpack"); !ok || item.Purchasable || item.Enabled || item.PriceOpenPower != 0 {
		t.Fatalf("expected adventure backpack to use default shop metadata, got %#v", item)
	}
	if _, ok := store.GetItem("missing"); ok {
		t.Fatal("expected missing item not to exist")
	}

	items[0].ID = "mutated"
	items = store.ListItems()
	if items[0].ID != "item_adventure_backpack" {
		t.Fatalf("expected ListItems to return a copy, got %q", items[0].ID)
	}
}

func TestLoadItemsAllowsCharmTypeAndDropSources(t *testing.T) {
	dir := writeContent(t, validSitonesTOML(), `
[[items]]
id = "item_charm_debug"
name = "順利除蟲御守"
type = "charm"
rarity = "rare"
description = "永久生效。"
source = "shop"
purchasable = true
enabled = true
price_open_power = 200

[[items]]
id = "item_clean_spec"
name = "整潔規格書"
type = "material"
rarity = "common"
description = "整潔規格書，可用於小石合成。"
source = "drop"
purchasable = false
enabled = true
price_open_power = 0

[[items]]
id = "item_shared_notes_link"
name = "共筆連結"
type = "material"
rarity = "common"
description = "共筆連結，可用於小石合成。"
source = "both"
purchasable = true
enabled = true
price_open_power = 50

[[items]]
id = "item_adventure_backpack"
name = "冒險背包"
type = "material"
rarity = "common"
description = "冒險背包，可用於小石合成。"
`)

	store, err := Load(dir)
	if err != nil {
		t.Fatalf("load content: %v", err)
	}

	charm, ok := store.GetItem("item_charm_debug")
	if !ok || charm.Type != "charm" || charm.Source != "shop" || charm.PriceOpenPower != 200 {
		t.Fatalf("unexpected charm item: %#v", charm)
	}
	dropOnly, ok := store.GetItem("item_clean_spec")
	if !ok || dropOnly.Source != "drop" || dropOnly.Purchasable {
		t.Fatalf("unexpected drop-only item: %#v", dropOnly)
	}
	dualSource, ok := store.GetItem("item_shared_notes_link")
	if !ok || dualSource.Source != "both" || !dualSource.Purchasable {
		t.Fatalf("unexpected dual-source item: %#v", dualSource)
	}
}

func TestLoadQuizQuestions(t *testing.T) {
	dir := writeContent(t, validSitonesTOML(), validItemsTOML(), validQuizQuestionsCSV())

	store, err := Load(dir)
	if err != nil {
		t.Fatalf("load content: %v", err)
	}

	questions := store.ListQuizQuestions()
	if len(questions) != minQuizQuestionCount {
		t.Fatalf("expected %d questions, got %d", minQuizQuestionCount, len(questions))
	}
	if questions[0].ID != "quiz-001" {
		t.Fatalf("expected questions sorted by id, got %#v", questions[0])
	}

	question, ok := store.GetQuizQuestion("quiz-001")
	if !ok {
		t.Fatal("expected quiz-001 to exist")
	}
	if question.CorrectChoice != "A" {
		t.Fatalf("expected correct choice A, got %q", question.CorrectChoice)
	}
	if _, ok := store.GetQuizQuestion("missing"); ok {
		t.Fatal("expected missing question not to exist")
	}

	questions[0].ID = "mutated"
	questions = store.ListQuizQuestions()
	if questions[0].ID != "quiz-001" {
		t.Fatalf("expected ListQuizQuestions to return a copy, got %q", questions[0].ID)
	}
}

func TestLoadQuizQuestionsSkipsDuplicateIDs(t *testing.T) {
	quizContent := validQuizQuestionsCSV() + "\nquiz-001,Duplicate Question,A,B,C,D,D,Duplicate explanation"
	dir := writeContent(t, validSitonesTOML(), validItemsTOML(), quizContent)

	store, err := Load(dir)
	if err != nil {
		t.Fatalf("load content: %v", err)
	}

	questions := store.ListQuizQuestions()
	if len(questions) != minQuizQuestionCount {
		t.Fatalf("expected %d questions, got %d", minQuizQuestionCount, len(questions))
	}

	question, ok := store.GetQuizQuestion("quiz-001")
	if !ok {
		t.Fatal("expected quiz-001 to exist")
	}
	if question.Prompt != "Question 1" || question.CorrectChoice != "A" {
		t.Fatalf("expected first quiz-001 to win, got %#v", question)
	}
}

func TestLoadFrontMaps(t *testing.T) {
	dir := writeContent(t, validSitonesTOML(), validItemsTOML(), validQuizQuestionsCSV(), validFusionRecipesTOML(), `
[[front_maps]]
id = " camp_day1_training "
name = " 開源戰線暖身 "
enabled = true
tick_seconds = 1
initial_front_open_power = 100
command_cooldown_ticks = 2
command_resolve_ticks = 1
resource_decay_per_tick = 1

[[front_maps.cells]]
id = "base_team_1"
x = 0
y = 0
terrain = "base"
zone = "base"
neighbors = ["system_a"]

[[front_maps.cells]]
id = "system_a"
x = 1
y = 0
terrain = "system"
zone = "system"
neighbors = ["base_team_1", "center_a"]
initial_defense = 5

[[front_maps.cells]]
id = "center_a"
x = 2
y = 0
terrain = "center"
zone = "frontier"
neighbors = ["system_a"]
initial_resource = 25

[[front_maps.events]]
id = "evt_config_bug"
cell_id = "system_a"
kind = "repair"
title = "設定檔讀不到"
starts_at_tick = 30
expires_after_ticks = 300
severity = 1
`)

	store, err := Load(dir)
	if err != nil {
		t.Fatalf("load content: %v", err)
	}

	frontMaps := store.ListFrontMaps()
	if len(frontMaps) != 1 {
		t.Fatalf("expected 1 front map, got %d", len(frontMaps))
	}
	if frontMaps[0].ID != "camp_day1_training" || frontMaps[0].Name != "開源戰線暖身" {
		t.Fatalf("unexpected front map metadata: %#v", frontMaps[0])
	}
	if !frontMaps[0].Enabled || frontMaps[0].TickSeconds != 1 || frontMaps[0].InitialFrontOpenPower != 100 {
		t.Fatalf("unexpected front map settings: %#v", frontMaps[0])
	}
	if len(frontMaps[0].Cells) != 3 || frontMaps[0].Cells[0].ID != "base_team_1" {
		t.Fatalf("expected sorted front cells, got %#v", frontMaps[0].Cells)
	}
	if len(frontMaps[0].Events) != 1 || frontMaps[0].Events[0].Kind != FrontEventRepair {
		t.Fatalf("unexpected front map events: %#v", frontMaps[0].Events)
	}

	frontMap, ok := store.GetFrontMap("camp_day1_training")
	if !ok {
		t.Fatal("expected camp_day1_training to exist")
	}
	systemCell := frontMapCellByID(t, frontMap, "system_a")
	if len(systemCell.Neighbors) != 2 || systemCell.Neighbors[1] != "center_a" {
		t.Fatalf("unexpected system_a neighbors: %#v", systemCell.Neighbors)
	}
	if _, ok := store.GetFrontMap("missing"); ok {
		t.Fatal("expected missing front map not to exist")
	}

	systemCellIndex := frontMapCellIndex(t, frontMaps[0], "system_a")
	frontMaps[0].Cells[systemCellIndex].Neighbors[0] = "mutated"
	frontMap, ok = store.GetFrontMap("camp_day1_training")
	if !ok {
		t.Fatal("expected camp_day1_training to exist")
	}
	systemCell = frontMapCellByID(t, frontMap, "system_a")
	if systemCell.Neighbors[0] != "base_team_1" {
		t.Fatalf("expected front map cells to be copied, got %#v", systemCell.Neighbors)
	}
}

func TestLoadFusionRecipes(t *testing.T) {
	dir := writeContent(t, validSitonesTOML(), validItemsTOML(), validQuizQuestionsCSV(), `
[[fusion_recipes]]
id = "recipe_engineering_terminal_s1_s2"
branch_id = "branch-engineering-route"
type = "engineering"
stage_from = 1
stage_to = 2
name = "Engineering Route Frame"
description = "Build a base display frame."
story = "Build the first route frame."
review_title = "Engineering Notes"
review_url = "https://example.test/engineering"
enabled = true

[[fusion_recipes.inputs]]
kind = "sitone"
id = "stone_engineering_base"
quantity = 1

[[fusion_recipes.inputs]]
kind = "item"
id = "item_adventure_backpack"
quantity = 3

[[fusion_recipes.outputs]]
kind = "item"
id = "item_adventure_backpack"
quantity = 1
`)

	store, err := Load(dir)
	if err != nil {
		t.Fatalf("load content: %v", err)
	}

	recipes := store.ListFusionRecipes()
	if len(recipes) != 1 {
		t.Fatalf("expected 1 recipe, got %d", len(recipes))
	}
	if recipes[0].ID != "recipe_engineering_terminal_s1_s2" || !recipes[0].Enabled {
		t.Fatalf("unexpected fusion recipe: %#v", recipes[0])
	}

	recipe, ok := store.GetFusionRecipe("recipe_engineering_terminal_s1_s2")
	if !ok {
		t.Fatal("expected fusion recipe to exist")
	}
	if recipe.BranchID != "branch-engineering-route" || recipe.Type != "engineering" || recipe.StageFrom != 1 || recipe.StageTo != 2 {
		t.Fatalf("unexpected fusion recipe metadata: %#v", recipe)
	}
	if recipe.Story != "Build the first route frame." || recipe.ReviewTitle != "Engineering Notes" || recipe.ReviewURL != "https://example.test/engineering" {
		t.Fatalf("unexpected fusion recipe story metadata: %#v", recipe)
	}
	if len(recipe.Inputs) != 2 || len(recipe.Outputs) != 1 {
		t.Fatalf("unexpected recipe components: %#v", recipe)
	}

	recipes[0].Inputs[0].ID = "mutated"
	recipe, ok = store.GetFusionRecipe("recipe_engineering_terminal_s1_s2")
	if !ok {
		t.Fatal("expected fusion recipe to exist")
	}
	if recipe.Inputs[0].ID != "stone_engineering_base" {
		t.Fatalf("expected fusion recipe components to be copied, got %#v", recipe.Inputs[0])
	}
}

func TestLoadAllowsDisabledFusionRecipeWithoutInputs(t *testing.T) {
	dir := writeContent(t, validSitonesTOML(), validItemsTOML(), validQuizQuestionsCSV(), `
[[fusion_recipes]]
id = "recipe_basic_blue"
branch_id = "basic"
type = "engineering"
name = "Engineering Base"
description = "Content-only base recipe."
enabled = false

[[fusion_recipes.outputs]]
kind = "sitone"
id = "stone_engineering_base"
quantity = 1
`)

	store, err := Load(dir)
	if err != nil {
		t.Fatalf("load content: %v", err)
	}

	recipe, ok := store.GetFusionRecipe("recipe_basic_blue")
	if !ok {
		t.Fatal("expected recipe_basic_blue to exist")
	}
	if recipe.Enabled || len(recipe.Inputs) != 0 || len(recipe.Outputs) != 1 {
		t.Fatalf("unexpected disabled recipe: %#v", recipe)
	}
}

func TestLoadRejectsEnabledFusionRecipeWithoutInputs(t *testing.T) {
	dir := writeContent(t, validSitonesTOML(), validItemsTOML(), validQuizQuestionsCSV(), `
[[fusion_recipes]]
id = "recipe_free_blue"
name = "Free Engineering"
description = "Enabled recipes must consume something."
enabled = true

[[fusion_recipes.outputs]]
kind = "sitone"
id = "stone_engineering_base"
quantity = 1
`)

	_, err := Load(dir)
	if err == nil {
		t.Fatal("expected missing inputs error")
	}
	if !strings.Contains(err.Error(), "fusion_recipes[0].inputs is required") {
		t.Fatalf("expected missing inputs error, got %v", err)
	}
}

func TestLoadRejectsDuplicateSitoneID(t *testing.T) {
	dir := writeContent(t, `
[[sitones]]
id = "stone_engineering_base"
name = "Engineering"
type = "engineering"
rarity = "base"

[[sitones]]
id = "stone_engineering_base"
name = "Engineering Again"
type = "engineering"
rarity = "base"
`)

	_, err := Load(dir)
	if err == nil {
		t.Fatal("expected duplicate id error")
	}
	if !strings.Contains(err.Error(), `duplicate sitone id "stone_engineering_base"`) {
		t.Fatalf("expected duplicate id error, got %v", err)
	}
}

func TestLoadRejectsDuplicateItemID(t *testing.T) {
	dir := writeContent(t, validSitonesTOML(), `
[[items]]
id = "item_adventure_backpack"
name = "Crafting Fragment"
type = "material"
rarity = "common"

[[items]]
id = "item_adventure_backpack"
name = "Crafting Fragment Again"
type = "material"
rarity = "common"
`)

	_, err := Load(dir)
	if err == nil {
		t.Fatal("expected duplicate id error")
	}
	if !strings.Contains(err.Error(), `duplicate item id "item_adventure_backpack"`) {
		t.Fatalf("expected duplicate id error, got %v", err)
	}
}

func TestLoadRejectsMissingRequiredSitoneFields(t *testing.T) {
	dir := writeContent(t, `
[[sitones]]
id = ""
name = ""
type = ""
rarity = ""
`)

	_, err := Load(dir)
	if err == nil {
		t.Fatal("expected required field error")
	}
	for _, want := range []string{
		"sitones[0].id is required",
		"sitones[0].name is required",
		"sitones[0].type is required",
		"sitones[0].rarity is required",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("expected error to contain %q, got %v", want, err)
		}
	}
}

func TestLoadRejectsMissingRequiredItemFields(t *testing.T) {
	dir := writeContent(t, validSitonesTOML(), `
[[items]]
id = ""
name = ""
type = ""
rarity = ""
`)

	_, err := Load(dir)
	if err == nil {
		t.Fatal("expected required field error")
	}
	for _, want := range []string{
		"items[0].id is required",
		"items[0].name is required",
		"items[0].type is required",
		"items[0].rarity is required",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("expected error to contain %q, got %v", want, err)
		}
	}
}

func TestLoadRejectsInvalidSitoneEnums(t *testing.T) {
	dir := writeContent(t, `
[[sitones]]
id = "sitone-unknown"
name = "Unknown"
type = "unknown"
rarity = "mythic"
`)

	_, err := Load(dir)
	if err == nil {
		t.Fatal("expected enum error")
	}
	for _, want := range []string{
		"sitones[0].type must be one of",
		"sitones[0].rarity must be one of",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("expected error to contain %q, got %v", want, err)
		}
	}
}

func TestLoadRejectsInvalidSitoneAbility(t *testing.T) {
	dir := writeContent(t, `
[[sitones]]
id = "stone_engineering_base"
name = "Engineering"
type = "engineering"
rarity = "base"
ability_name = "Broken"
ability_kind = "unknown"
ability_value = 0
ability_count = 1
ability_description = ""
`)

	_, err := Load(dir)
	if err == nil {
		t.Fatal("expected invalid ability error")
	}
	for _, want := range []string{
		"sitones[0].ability_kind must be one of",
		"sitones[0].ability_value must be greater than 0",
		"sitones[0].ability_count must be 0 unless ability_kind is eliminate_wrong_choice",
		"sitones[0].ability_description is required",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("expected error to contain %q, got %v", want, err)
		}
	}
}

func TestLoadRejectsInvalidItemEnums(t *testing.T) {
	dir := writeContent(t, validSitonesTOML(), `
[[items]]
id = "item-unknown"
name = "Unknown"
type = "unknown"
rarity = "mythic"
`)

	_, err := Load(dir)
	if err == nil {
		t.Fatal("expected enum error")
	}
	for _, want := range []string{
		"items[0].type must be one of",
		"items[0].rarity must be one of",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("expected error to contain %q, got %v", want, err)
		}
	}
}

func TestLoadRejectsInvalidItemSourceRules(t *testing.T) {
	dir := writeContent(t, validSitonesTOML(), `
[[items]]
id = "item_clean_spec"
name = "Clean Spec"
type = "material"
rarity = "common"
source = "drop"
purchasable = true
enabled = true
price_open_power = 50

[[items]]
id = "item_shared_notes_link"
name = "Shared Notes"
type = "material"
rarity = "common"
source = "both"
purchasable = false
enabled = true
price_open_power = 0
`)

	_, err := Load(dir)
	if err == nil {
		t.Fatal("expected invalid source rule error")
	}
	for _, want := range []string{
		"items[0].purchasable must be false when source is drop",
		"items[1].purchasable must be true when source is both",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("expected error to contain %q, got %v", want, err)
		}
	}
}

func TestLoadRejectsPurchasableItemWithoutPrice(t *testing.T) {
	dir := writeContent(t, validSitonesTOML(), `
[[items]]
id = "item_adventure_backpack"
name = "Crafting Fragment"
type = "material"
rarity = "common"
purchasable = true
enabled = true
price_open_power = 0
`)

	_, err := Load(dir)
	if err == nil {
		t.Fatal("expected invalid price error")
	}
	if !strings.Contains(err.Error(), "items[0].price_open_power must be greater than 0 when purchasable is true") {
		t.Fatalf("expected price error, got %v", err)
	}
}

func TestLoadRejectsInvalidQuizChoice(t *testing.T) {
	dir := writeContent(t, validSitonesTOML(), validItemsTOML(), strings.Replace(validQuizQuestionsCSV(), "quiz-001,Question 1,A,B,C,D,A,Explanation 1", "quiz-001,Question 1,A,B,C,D,Z,Explanation 1", 1))

	_, err := Load(dir)
	if err == nil {
		t.Fatal("expected invalid quiz choice error")
	}
	if !strings.Contains(err.Error(), "correct_choice must be one of") {
		t.Fatalf("expected invalid correct choice error, got %v", err)
	}
}

func TestLoadRejectsInvalidFrontMap(t *testing.T) {
	dir := writeContent(t, validSitonesTOML(), validItemsTOML(), validQuizQuestionsCSV(), validFusionRecipesTOML(), `
[[front_maps]]
id = "broken"
name = "Broken"
enabled = true
tick_seconds = 1

[[front_maps.cells]]
id = "base_team_1"
x = 0
y = 0
terrain = "base"
zone = "base"
neighbors = ["missing"]
`)

	_, err := Load(dir)
	if err == nil {
		t.Fatal("expected invalid front map error")
	}
	if !strings.Contains(err.Error(), `neighbors references unknown cell "missing"`) {
		t.Fatalf("expected unknown neighbor error, got %v", err)
	}
}

func TestLoadRejectsTooFewQuizQuestions(t *testing.T) {
	dir := writeContent(t, validSitonesTOML(), validItemsTOML(), strings.TrimSpace(`
id,prompt,choice_a,choice_b,choice_c,choice_d,correct_choice,explanation
quiz-001,Question 1,A,B,C,D,A,Explanation 1
`))

	_, err := Load(dir)
	if err == nil {
		t.Fatal("expected too few quiz questions error")
	}
	if !strings.Contains(err.Error(), "at least 10 quiz questions are required") {
		t.Fatalf("expected too few questions error, got %v", err)
	}
}

func TestLoadResolvesServerContentFallback(t *testing.T) {
	root := t.TempDir()
	serverDir := filepath.Join(root, "server")
	contentDir := filepath.Join(serverDir, "content")
	if err := os.MkdirAll(contentDir, 0o755); err != nil {
		t.Fatalf("mkdir content dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(contentDir, sitonesFile), []byte(validSitonesTOML()), 0o644); err != nil {
		t.Fatalf("write sitones: %v", err)
	}
	if err := os.WriteFile(filepath.Join(contentDir, itemsFile), []byte(validItemsTOML()), 0o644); err != nil {
		t.Fatalf("write items: %v", err)
	}
	if err := os.WriteFile(filepath.Join(contentDir, fusionRecipesFile), []byte(validFusionRecipesTOML()), 0o644); err != nil {
		t.Fatalf("write fusion recipes: %v", err)
	}
	if err := os.WriteFile(filepath.Join(contentDir, quizQuestionsFile), []byte(validQuizQuestionsCSV()), 0o644); err != nil {
		t.Fatalf("write quiz questions: %v", err)
	}
	if err := os.WriteFile(filepath.Join(contentDir, frontMapsFile), []byte(validFrontMapsTOML()), 0o644); err != nil {
		t.Fatalf("write front maps: %v", err)
	}

	oldCWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("get cwd: %v", err)
	}
	if err := os.Chdir(serverDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(oldCWD); err != nil {
			t.Fatalf("restore cwd: %v", err)
		}
	})

	store, err := Load("server/content")
	if err != nil {
		t.Fatalf("load content through fallback: %v", err)
	}
	if len(store.ListSitones()) != 1 {
		t.Fatalf("expected 1 sitone, got %d", len(store.ListSitones()))
	}
	if len(store.ListItems()) != 1 {
		t.Fatalf("expected 1 item, got %d", len(store.ListItems()))
	}
	if len(store.ListQuizQuestions()) != minQuizQuestionCount {
		t.Fatalf("expected %d quiz questions, got %d", minQuizQuestionCount, len(store.ListQuizQuestions()))
	}
	if len(store.ListFrontMaps()) != 1 {
		t.Fatalf("expected 1 front map, got %d", len(store.ListFrontMaps()))
	}
}

func writeContent(t *testing.T, sitones string, values ...string) string {
	t.Helper()

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, sitonesFile), []byte(strings.TrimSpace(sitones)), 0o644); err != nil {
		t.Fatalf("write sitones: %v", err)
	}

	itemContent := validItemsTOML()
	if len(values) > 0 {
		itemContent = values[0]
	}
	if err := os.WriteFile(filepath.Join(dir, itemsFile), []byte(strings.TrimSpace(itemContent)), 0o644); err != nil {
		t.Fatalf("write items: %v", err)
	}

	fusionContent := validFusionRecipesTOML()
	if len(values) > 2 {
		fusionContent = values[2]
	}
	if err := os.WriteFile(filepath.Join(dir, fusionRecipesFile), []byte(strings.TrimSpace(fusionContent)), 0o644); err != nil {
		t.Fatalf("write fusion recipes: %v", err)
	}

	quizContent := validQuizQuestionsCSV()
	if len(values) > 1 {
		quizContent = values[1]
	}
	if err := os.WriteFile(filepath.Join(dir, quizQuestionsFile), []byte(strings.TrimSpace(quizContent)), 0o644); err != nil {
		t.Fatalf("write quiz questions: %v", err)
	}

	frontMapsContent := validFrontMapsTOML()
	if len(values) > 3 {
		frontMapsContent = values[3]
	}
	if err := os.WriteFile(filepath.Join(dir, frontMapsFile), []byte(strings.TrimSpace(frontMapsContent)), 0o644); err != nil {
		t.Fatalf("write front maps: %v", err)
	}
	return dir
}

func validSitonesTOML() string {
	return strings.TrimSpace(`
[[sitones]]
id = "stone_engineering_base"
name = "Engineering"
type = "engineering"
rarity = "base"
ability_name = "穩定輸出"
ability_kind = "answer_score_bonus"
ability_value = 5
ability_count = 0
ability_description = "答對時分數提高 5%。"
`)
}

func validItemsTOML() string {
	return strings.TrimSpace(`
[[items]]
id = "item_adventure_backpack"
name = "Crafting Fragment"
type = "material"
rarity = "common"
`)
}

func validFusionRecipesTOML() string {
	return strings.TrimSpace(`
[[fusion_recipes]]
id = "recipe_engineering_terminal_s1_s2"
name = "Engineering Route Frame"
description = "Build a base display frame."
enabled = true

[[fusion_recipes.inputs]]
kind = "sitone"
id = "stone_engineering_base"
quantity = 1

[[fusion_recipes.inputs]]
kind = "item"
id = "item_adventure_backpack"
quantity = 3

[[fusion_recipes.outputs]]
kind = "item"
id = "item_adventure_backpack"
quantity = 1
`)
}

func validQuizQuestionsCSV() string {
	return strings.TrimSpace(`
id,prompt,choice_a,choice_b,choice_c,choice_d,correct_choice,explanation
quiz-001,Question 1,A,B,C,D,A,Explanation 1
quiz-002,Question 2,A,B,C,D,B,Explanation 2
quiz-003,Question 3,A,B,C,D,C,Explanation 3
quiz-004,Question 4,A,B,C,D,D,Explanation 4
quiz-005,Question 5,A,B,C,D,A,Explanation 5
quiz-006,Question 6,A,B,C,D,B,Explanation 6
quiz-007,Question 7,A,B,C,D,C,Explanation 7
quiz-008,Question 8,A,B,C,D,D,Explanation 8
quiz-009,Question 9,A,B,C,D,A,Explanation 9
quiz-010,Question 10,A,B,C,D,B,Explanation 10
`)
}

func validFrontMapsTOML() string {
	return strings.TrimSpace(`
[[front_maps]]
id = "camp_day1_training"
name = "開源戰線暖身"
enabled = true
tick_seconds = 1
initial_front_open_power = 100
command_cooldown_ticks = 2
command_resolve_ticks = 1
resource_decay_per_tick = 1

[[front_maps.cells]]
id = "base_team_1"
x = 0
y = 0
terrain = "base"
zone = "base"
neighbors = ["system_a"]

[[front_maps.cells]]
id = "system_a"
x = 1
y = 0
terrain = "system"
zone = "system"
neighbors = ["base_team_1", "center_a"]
initial_defense = 5

[[front_maps.cells]]
id = "center_a"
x = 2
y = 0
terrain = "center"
zone = "frontier"
neighbors = ["system_a"]
initial_resource = 25

[[front_maps.events]]
id = "evt_config_bug"
cell_id = "system_a"
kind = "repair"
title = "設定檔讀不到"
starts_at_tick = 30
expires_after_ticks = 300
severity = 1
`)
}

func frontMapCellByID(t *testing.T, template FrontMapTemplate, id string) FrontMapCell {
	t.Helper()

	index := frontMapCellIndex(t, template, id)
	return template.Cells[index]
}

func frontMapCellIndex(t *testing.T, template FrontMapTemplate, id string) int {
	t.Helper()

	for i, cell := range template.Cells {
		if cell.ID == id {
			return i
		}
	}
	t.Fatalf("front map cell %q not found in %#v", id, template.Cells)
	return -1
}
