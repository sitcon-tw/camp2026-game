# 社群攤位 QRCode 獎勵功能

## 功能摘要

這個功能新增社群攤位 QRCode 獎勵流程：

- 後端寫死社群攤位資料，啟動時同步到 MongoDB。
- 每個社群攤位有固定亂碼 ID，例如 `/community/q7m4x2v9`。
- 社群攤位夥伴可開啟 `/community/$standId` 看板頁，顯示攤位資訊、QRCode、來過學員數、已領獎數。
- 學員不直接使用 `/community/$standId` 領獎，而是在「冒險者通行證」內掃描攤位 QRCode。
- 學員掃描成功後會在 modal 顯示攤位資訊與獎勵，點擊「領取獎勵」後取得對應道具。
- 每位學員每個攤位只能領取一次。

## 開發組說明

### 固定攤位資料

攤位資料寫在：

`server/internal/communitystand/seed.go`

目前固定攤位：

| ID | 名稱 | 獎勵 |
| --- | --- | --- |
| `q7m4x2v9` | SITCON 學生計算機年會 | `item_student_community_card` x1 |
| `r2k8p6n3` | 開源路線攤位 | `item_open_source_roadmap` x1 |
| `z5h9t1c7` | 社群交流攤位 | `item_star_village_badge` x1 |

`app` 啟動與 `cmd/seed` 都會呼叫 `communitystand.EnsureDefaults`，把這些固定資料 upsert 到 MongoDB。

如果之後要改攤位名稱、介紹、Logo、網站或獎勵，直接修改 `server/internal/communitystand/seed.go`。

### MongoDB Collections

#### `community_stands`

紀錄攤位基本資訊。

主要欄位：

- `_id`: 攤位 ID，例如 `q7m4x2v9`
- `name`: 攤位名稱
- `description`: 攤位介紹
- `logo_url`: Logo URL
- `website_url`: 社群網站
- `enabled`: 是否啟用
- `reward`: 獎勵設定
  - `kind`: `item` / `sitone` / `open_power`
  - `ref_id`: item 或 sitone id
  - `quantity`: 數量
  - `amount`: open power 數量
- `created_at`
- `updated_at`

#### `community_stand_visits`

紀錄「學員來過攤位」。

學員在通行證內掃描攤位 QRCode，成功載入攤位資訊時會記錄一次 visit。若直接領獎，也會補記 visit。

主要欄位：

- `_id`
- `stand_id`
- `player_id`
- `first_visited_at`
- `last_visited_at`

有 unique index：

- `(stand_id, player_id)`

所以同一位學員重複掃同一攤，只會算一位來過學員，但會更新 `last_visited_at`。

#### `community_stand_claims`

紀錄「學員已領過某攤獎勵」。

主要欄位：

- `_id`
- `stand_id`
- `player_id`
- `reward_id`
- `created_at`

有 unique index：

- `(stand_id, player_id)`

這是「每位學員每個攤位只能領一次」的主要保護。

### API

#### 學員掃描後取得攤位資訊

`GET /api/community/{standID}`

需要學員登入。

用途：

- 通行證 scanner 掃描後呼叫。
- 回傳攤位資訊、獎勵資訊、目前學員是否已領取。
- 成功讀取時會記錄 `community_stand_visits`。

#### 學員領取攤位獎勵

`POST /api/community/{standID}/claim`

需要學員登入。

用途：

- 學員在 modal 點擊「領取獎勵」。
- 若已領過，回傳 `409`。
- 若成功，建立 `community_stand_claims`，並把道具加到玩家 inventory。

#### 攤位夥伴看板資料

`GET /api/community/{standID}/display`

不需要登入。

用途：

- `/community/$standId` 頁面使用。
- 回傳攤位資訊、來過學員數、已領獎數。
- 給社群攤位夥伴現場展示 QRCode 與查看統計。

## 課活組說明

### 這個 feature 怎麼運作

每個社群攤位會有一個專屬網址，例如：

- `https://camp.sitcon.party/community/q7m4x2v9`
- `https://camp.sitcon.party/community/r2k8p6n3`
- `https://camp.sitcon.party/community/z5h9t1c7`

社群攤位夥伴打開這個網址後，會看到：

- 攤位名稱
- 攤位介紹
- 社群網站連結
- 攤位 QRCode
- 目前有多少學員來過
- 目前有多少學員已領獎

學員不是用瀏覽器直接開這個頁面領獎，而是進遊戲的「冒險者通行證」，按「掃描社群攤位」，掃描攤位頁上的 QRCode。

掃描成功後，學員會看到一個 modal，裡面有：

- 攤位名稱
- 攤位介紹
- Logo
- 社群網站
- 可領取的道具
- 「領取獎勵」按鈕

每位學員每個攤位只能領一次。

### 如何教社群攤位夥伴操作

1. 請社群攤位夥伴打開自己的攤位網址。
2. 把頁面停在 QRCode 看板上。
3. 讓學員用遊戲內「冒險者通行證」掃描該 QRCode。
4. 看板上的「來過學員」與「已領獎勵」會自動更新。
5. 如果人數沒有即時變動，可以按「更新人數」。

### 如何教學員操作

1. 打開遊戲。
2. 進入「冒險者通行證」。
3. 點擊「掃描社群攤位」。
4. 掃描攤位上的 QRCode。
5. 看完攤位資訊後，點擊「領取獎勵」。
6. 每個攤位只能領一次，已領過會顯示已領取。

### 注意事項

- 學員必須登入遊戲才能掃描與領獎。
- 攤位看板頁不需要登入，方便社群夥伴直接展示。
- 若相機權限無法開啟，學員可手動輸入攤位 ID。
- QRCode 對應的是攤位網址，不是學員自己的個人 QRCode。
