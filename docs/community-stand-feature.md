# 社群攤位 QRCode 獎勵功能

## 功能摘要

這個功能新增社群攤位 QRCode 獎勵流程：

- 後端不會在部署或啟動時自動建立預設攤位。
- 每個社群攤位建立時會由後端產生 UUID ID，例如 `/community/{standId}`。
- 社群攤位夥伴可開啟 `/community/$standId` 看板頁，顯示攤位資訊、短效 QRCode、來過學員數、已領獎數。
- 學員不直接使用 `/community/$standId` 領獎，而是在「冒險者通行證」內掃描攤位 QRCode。
- 學員掃描成功後會在 modal 顯示攤位資訊與獎勵，點擊「領取獎勵」後取得對應道具。
- 每位學員每個攤位只能領取一次。

## 開發組說明

### 攤位資料建立方式

部署與 API 啟動不會自動新增預設攤位。`make seed` 也不會建立社群攤位資料。

社群攤位必須透過 admin API 或管理介面建立，建立時後端會產生 UUID `standId`。如果資料庫沒有攤位資料，社群攤位列表會維持空白，直到管理員新增攤位。

### MongoDB Collections

#### `community_stands`

紀錄攤位基本資訊。

主要欄位：

- `_id`: 攤位 UUID，由後端建立攤位時產生
- `name`: 攤位名稱
- `description`: 攤位介紹
- `logo_url`: Logo URL
- `website_url`: 社群網站
- `qr_token`: 攤位看板頁簽發的短效 QR token
- `qr_token_expires_at`: QR token 強制失效時間
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
- `reward`: 領取當下的獎勵快照
  - `kind`: `item` / `sitone` / `open_power`
  - `ref_id`: item 或 sitone id
  - `quantity`: 數量
  - `amount`: open power 數量
- `created_at`

有 unique index：

- `(stand_id, player_id)`

這是「每位學員每個攤位只能領一次」的主要保護。

### API

#### 學員掃描後取得攤位資訊

`GET /api/community/scans/{qrToken}`

需要學員登入。

用途：

- 通行證 scanner 掃描後呼叫。
- 回傳攤位資訊、獎勵資訊、目前學員是否已領取。
- 成功讀取時會記錄 `community_stand_visits`。

#### 學員領取攤位獎勵

`POST /api/community/scans/{qrToken}/claim`

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
- QRCode 內容使用 `camp2026-community-stand|{qrToken}` 掃描 payload，不放入 `/community/{standID}` 看板網址。
- `qrToken` 只會在攤位看板頁讀取 display API 時簽發，且兩分鐘後由後端強制失效。

#### 管理員查看攤位領取紀錄

`GET /api/admin/community-stand-claims`

需要管理員登入。

用途：

- 管理員面板使用。
- 回傳最近社群攤位領取紀錄，包含領取者、領取時間、攤位 ID、攤位名稱與領取當下的獎勵快照。
- 可用 `standId` query parameter 篩選單一攤位。

## 課活組說明

### 這個 feature 怎麼運作

每個社群攤位會有一個由 admin 建立後取得的專屬網址，例如：

- `https://camp.sitcon.party/community/{standId}`

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
- 若相機權限無法開啟，學員可手動輸入 QRCode 內容。
- QRCode 對應的是兩分鐘有效的不透明 qrToken，不是攤位網址，也不是學員自己的個人 QRCode。
