# 道具系統

## 總覽

道具分為兩大類：**合成道具**（用於小石融合）與**功能道具**（裝飾、被動加成）。

---

## 一、合成道具（40 個）

全部來自合成樹的需求，每個配方消耗 1 個道具作為素材。

### 取得方式比例

| 比例 | 取得方式 | 數量 |
|------|----------|------|
| 100% | 僅商店購買 | 40 個 |
| 0%   | 僅對戰掉落 | 0 個 |
| 0%   | 兩者皆有   | 0 個 |

### 僅商店購買（40 個）

這些合成素材皆可從商店以 150 開源力購入。

| ID | 名稱 | 對應合成 |
|----|------|----------|
| `item_adventure_backpack` | 冒險背包 | 營地背包小石 |
| `item_black_box_sticker` | 黑箱貼紙 | 2019 黑箱小石 |
| `item_booth_sticker` | 攤位貼紙 | 攤位小石 |
| `item_canvas_code` | 畫布程式碼 | p5.js 小石 |
| `item_charter_draft` | 章程草稿 | 共識草稿小石 |
| `item_container_sticker` | 容器貼紙 | Docker 小石 |
| `item_contribution_sticker` | 貢獻貼紙 | 貢獻小石 |
| `item_espresso_cup` | Espresso 杯 | Espresso 小石 |
| `item_finite_label` | 有限標籤 | 2024 有限小石 |
| `item_maze_map` | 迷宮地圖 | 2022 迷宮小石 |
| `item_microphone` | 麥克風 | 麥克風小石 |
| `item_prompt_card` | Prompt 卡 | Prompt 小石 |
| `item_public_key_tag` | 公鑰吊牌 | GPG 小石 |
| `item_ribbon` | 彩帶 | 2024 彩帶小石 |
| `item_sticky_note` | 便利貼 | 便利貼小石 |
| `item_student_community_card` | 學生社群卡 | 社群握手小石 |
| `item_terminal_cursor` | 終端機游標 | 終端機小石 |
| `item_test_sticker` | 測試貼紙 | 測試小石 |
| `item_tour_flag` | 導遊旗 | 2020 導遊旗小石 |
| `item_wooden_abacus` | 木製算珠 | 2021 算盤小石 |
| `item_wooden_plank` | 木板 | 4 個可合成星手村路標 |

### 原 Stage 2 合成素材（16 個）

這些是 Stage 2 合成素材，現在同樣透過商店購買取得；知識王對戰不再掉落素材道具，改為機率掉落基礎小石。

| ID | 名稱 | 對應合成 |
|----|------|----------|
| `item_cat_paw_print` | 貓掌印 | 2022 破牆貓小石 |
| `item_clean_spec` | 整潔規格書 | Clean Code 小石 |
| `item_cluster_core` | 叢集核心 | Kubernetes 小石 |
| `item_essence_timer` | 精華計時器 | 十分鐘精華小石 |
| `item_human_label` | 人類標籤 | Human After All 小石 |
| `item_infinite_star_map` | 無限星圖 | 2024 無限靈感小石 |
| `item_lightning_talk_script` | 閃電講稿 | Lightning Talk 小石 |
| `item_mission_map` | 任務地圖 | 2026 營地探險小石 |
| `item_open_source_roadmap` | 開源路線圖 | Open Source 路線小石 |
| `item_pixel_paint` | 像素顏料 | 科技藝術小石 |
| `item_polaroid_film` | 拍立得底片 | 2024 最後一晚拍立得小石 |
| `item_predecessor_notes` | 前人筆記 | 2021 算盤後裔小石 |
| `item_signature_inkpad` | 簽章印泥 | 加密守門員小石 |
| `item_star_village_signpost` | 星手村路標 | 星手村嚮導小石 |
| `item_system_docs` | 系統文件 | 系統實習小石 |
| `item_toolbox_key` | 工具箱鑰匙 | 2019 開箱演算法小石 |

### 工程型進化道具（可重複購買）

工程型小石的進化道具開放商店重複購買，價格依階段調整：一階 500 開源力、二階 1500 開源力。

| ID | 名稱 | 階段 | 價格 | 對應合成 |
|----|------|------|------|----------|
| `item_container_sticker` | 容器貼紙 | 一階 | 500 | Docker 小石 |
| `item_public_key_tag` | 公鑰吊牌 | 一階 | 500 | GPG 小石 |
| `item_test_sticker` | 測試貼紙 | 一階 | 500 | 測試小石 |
| `item_terminal_cursor` | 終端機游標 | 一階 | 500 | 終端機小石 |
| `item_cluster_core` | 叢集核心 | 二階 | 1500 | Kubernetes 小石 |
| `item_signature_inkpad` | 簽章印泥 | 二階 | 1500 | 加密守門員小石 |
| `item_clean_spec` | 整潔規格書 | 二階 | 1500 | Clean Code 小石 |
| `item_system_docs` | 系統文件 | 二階 | 1500 | 系統實習小石 |

### 原雙來源素材（4 個）

這些素材現在僅從商店購買取得。

| ID | 名稱 | 對應合成 |
|----|------|----------|
| `item_shared_notes_link` | 共筆連結 | 課程共筆小石 |
| `item_star_village_badge` | 星手村徽章 | 星手村交流小石 |
| `item_transparent_proposal` | 透明提案 | 學生自治小石 |
| `item_venue_route` | 會場路線 | 2020 SITCON 導遊團小石 |

---

## 二、功能道具

商店販售之一次性購買道具，不可用於合成。

### 御守（5 個）

永久生效的被動加成道具，購買後自動綁定該類型小石，效果持續整場營隊。可透過特定方式升級（+x%）。

| ID | 名稱 | 價格 | 對應小石類型 | 功能 |
|----|------|------|-------------|------|
| `item_charm_connection` | 連線成功 御守 | 800 | 探索型 | 對戰小石掉落率 +15%（總加成上限 +15%） |
| `item_charm_debug` | 順利除蟲 御守 | 800 | 工程型 | 對戰得分 +10%（每級 +5%，上限 +25%） |
| `item_charm_all_nighter` | 熬夜有成 御守 | 800 | 靈光型 | 答題時 20% 機率刪除一個錯誤選項（每級 +5%，上限 35%） |
| `item_charm_success` | 馬到成功 御守 | 800 | 娛樂型 | 對戰獲得開源力 +20%（每級 +10%，上限 +50%） |
| `item_charm_harmony` | 金玉良緣 御守 | 800 | 共鳴型 | 活動獲得開源力 +20%（每級 +10%，上限 +50%） |

## 三、無功能道具
### 明信片（多款）

純裝飾道具，無遊戲數值效果。

| ID | 名稱 | 價格 | 說明 |
|----|------|------|------|
| `item_postcard_sitcon2024` | SITCON 2024 明信片 | 300 ||
| `item_postcard_sitcon2026` | SITCON 2026 明信片 | 300 ||
| `item_postcard_star_village` | 開源星手村明信片 | 300 ||
| `item_tshirt_2026` | 2026 年會紀念 T | 1200 ||

---

## 四、物品類型定義

所有道具在 `server/content/items.toml` 中定義，欄位如下：

| 欄位 | 類型 | 說明 |
|------|------|------|
| `id` | string | 唯一識別碼，格式 `item_<name>` |
| `name` | string | 顯示名稱 |
| `type` | string | `material` / `cosmetic` / `event` |
| `rarity` | string | `base` / `common` / `rare` / `limited` |
| `description` | string | 描述文字 |
| `purchasable` | bool | 是否可在商店購買 |
| `enabled` | bool | 是否啟用 |
| `price_open_power` | int | 商店價格（開源力） |
| `repeatable` | bool | 是否可重複購買（預設 false，一次性購買） |

### 類型說明

- **material**：合成素材，用於小石融合。
- **cosmetic**：裝飾道具（明信片、T 恤），無遊戲功能。
- **event**：活動限定道具，透過特殊活動取得。
