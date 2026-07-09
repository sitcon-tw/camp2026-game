# fusion_recipes.toml 欄位說明

本文件說明 `server/content/fusion_recipes.toml` 中每個屬性的意義。此檔案定義遊戲中的「合成配方」（fusion recipe），由 `server/internal/content/fusion_recipe.go` 載入與驗證。

## 檔案結構

檔案由多個 `[[fusion_recipes]]` 區塊組成，每個區塊代表一條配方。每條配方底下可以有多個 `[[fusion_recipes.inputs]]`（材料）與 `[[fusion_recipes.outputs]]`（產物）。

```toml
[[fusion_recipes]]
id = "recipe_explore_2022_maze_s1_s2"
branch_id = "branch_2022_maze"
type = "exploration"
stage_from = 1
stage_to = 2
name = "2022 迷宮小石"
description = "..."
story = "..."
review_title = "SITCON 2022 Cat in a Maze"
review_url = "https://sitcon.org/2022/"
enabled = true

[[fusion_recipes.inputs]]
kind = "sitone"
id = "stone_explorer_base"
quantity = 1

[[fusion_recipes.outputs]]
kind = "sitone"
id = "stone_2022_maze"
quantity = 1
```

## fusion_recipes 欄位

| 欄位 | 型別 | 必填 | 說明 |
| --- | --- | --- | --- |
| `id` | string | 是 | 配方的唯一識別碼，全檔案不可重複。慣例格式為 `recipe_<類型>_<主題>_s<起始階段>_s<目標階段>`，例如 `recipe_explore_2022_maze_s1_s2`。 |
| `branch_id` | string | 否 | 進化分支識別碼。同一條進化路線的配方共用同一個 `branch_id`（例如 `branch_2022_maze` 的 s1→s2 與 s2→s3 兩條配方）。特殊值：`basic`（基礎小石）、`event_special` / `event_fireside` / `event_checkpoint`（活動獎勵）。 |
| `type` | string | 否 | 小石屬性類型，必須是下列之一：`exploration`（探索）、`inspiration`（靈光）、`resonance`（共鳴）、`engineering`（工程）、`entertainment`（娛樂）。與 sitones 的類型定義共用（見 `sitone.go:12`）。 |
| `stage_from` | int | 是* | 進化的起始階段，必須 ≥ 0。`0` 表示此配方不屬於階段進化（例如純道具合成或活動發放）。 |
| `stage_to` | int | 是* | 進化的目標階段，必須 ≥ 0。若 `stage_from` > 0，則 `stage_to` 必須大於 `stage_from`。兩者必須同時為 0 或同時大於 0（不可只填一個）。 |
| `name` | string | 是 | 配方／產出小石的顯示名稱，例如「2022 迷宮小石」。 |
| `description` | string | 否 | 配方的描述文字，用於介面顯示。目前內容與 `story` 相同。 |
| `story` | string | 否 | 產出小石背後的故事文案。目前內容與 `description` 相同，保留為獨立欄位以便未來區分用途。 |
| `review_title` | string | 否 | 延伸閱讀的標題，通常是相關的 SITCON 年會主題或議程名稱。 |
| `review_url` | string | 否 | 延伸閱讀的連結，通常指向 SITCON 官網或議程頁面。 |
| `enabled` | bool | 是 | 是否啟用此配方。`true` 表示玩家可以透過合成取得；`false` 表示不能主動合成（如基礎小石與活動獎勵小石，由系統直接發放）。啟用的配方必須至少有一個 `inputs`。 |

\* `stage_from` 與 `stage_to` 必須成對出現：同時為 0，或同時大於 0。

## fusion_recipes.inputs / fusion_recipes.outputs 欄位

`inputs` 是合成所需的材料，`outputs` 是合成後獲得的產物。兩者共用相同的欄位結構（`FusionComponent`）：

| 欄位 | 型別 | 必填 | 說明 |
| --- | --- | --- | --- |
| `kind` | string | 是 | 材料／產物的種類，必須是 `sitone`（小石）或 `item`（道具）。 |
| `id` | string | 是 | 對應的小石或道具 ID。`kind = "sitone"` 時必須存在於 `sitones.toml`；`kind = "item"` 時必須存在於 `items.toml`，否則載入時報錯。 |
| `quantity` | int | 是 | 數量，必須 > 0。 |

規則：

- 每條配方 `outputs` 至少要有一個。
- `enabled = true` 的配方 `inputs` 至少要有一個；`enabled = false` 的配方可以沒有 `inputs`（由系統發放）。

## 配方類別整理

目前檔案中的配方分為三類：

1. **進化配方**（`enabled = true`、`stage_from`/`stage_to` > 0）
   典型模式為「上一階段小石 + 1 個道具 → 下一階段小石」。每個分支有 s1→s2 與 s2→s3 兩條配方。
2. **道具合成配方**（`enabled = true`、`stage_from = stage_to = 0`）
   純道具合成，例如 `recipe_item_wooden_plank_to_star_village_signpost`：4 個木板 → 1 個星手村路標。
3. **系統發放配方**（`enabled = false`、`stage_from = stage_to = 0`、無 `inputs`）
   - `branch_id = "basic"`：五種基礎小石（探索／工程／共鳴／娛樂／靈光型）。
   - `branch_id = "event_*"`：活動獎勵小石（爐邊夜談、闖關活動等）。

## 載入與驗證

- 載入程式：`server/internal/content/fusion_recipe.go`
- 所有字串欄位載入時會 trim 空白。
- 驗證失敗（缺欄位、重複 id、引用不存在的 sitone/item 等）會導致伺服器啟動失敗。
- 相關 API：透過 content store 的 `ListFusionRecipes()` 與 `GetFusionRecipe(id)` 存取。
