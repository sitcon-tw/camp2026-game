# SITCON Camp 2026 小石開源戰線大型競爭遊戲計畫

## 0. 文件目的

這份文件把目前 `camp2026-game` 專案既有的「小石」、「開源力」、「知識王」、「小隊」系統，整理成一個可以落地開發的營隊大型競爭戰略遊戲設計。社群攤位因為預計 Day 5 才出現，只能作為 finale 特殊加成與回顧素材，不能作為 Day 1-4 核心循環或追趕機制的依賴。

最新主方向不是單場短回合任務，而是做出類似 OpenFront 的大型持續戰線：全營小隊在同一張地圖上爭奪節點、搶救小石、收集快要散逸的開源力、處理即將爆發的事件。玩家應該感覺到「現在不做，小石就會陷入危險」、「這一波開源力再不收就會被別隊拿走或散掉」。

原本文件中的 `Operation`、短回合任務、題目事件與小石行動，保留為教學任務、救援任務、節點事件與局部遭遇戰的素材；它們不再是主玩法本身。

本文件預設讀者包含：

- 開發組：需要知道資料模型、API、前端頁面、實作順序。
- 課活組：需要知道玩法如何引導學員參與課程、任務窗口、團隊合作，以及 Day 5 社群攤位如何接入 finale。
- 遊戲設計與內容組：需要知道小石、題庫、道具、開源力怎麼被使用。
- Staff：需要知道營期現場怎麼說明、怎麼控節奏、怎麼避免刷分。

## 1. 一句話定位

> 這是一個以全營共享地圖為舞台、以小隊競爭為壓力、以小石部署與開源力爭奪為核心的持續戰略遊戲。

更短的產品描述：

> 在一張持續運作的開源戰線上派出小石，占領節點、守住資源、救回快被困住的小石，和其他小隊競爭誰能把營隊推向更好的狀態。

## 2. 現有專案狀態判讀

### 2.1 已存在且可直接利用的系統

目前專案已經具備下列基礎，不需要從零開始：

- Go server，路由集中在 `server/internal/http/router.go`。
- React frontend，路由與頁面集中在 `frontend/src/routes` 與 `frontend/src/pages`。
- MongoDB model 層，現有 collection 包含 player、team、match、sitone、item、open power、community stand 等。
- 玩家登入、玩家資料、小隊資料。
- 小石圖鑑、玩家小石庫存、預設小石編隊。
- 小石合成與素材商店。
- 知識王 match 流程。
- 題目、選項、正解、解析。
- 開源力總量計算與紀錄。
- 社群攤位 QR 掃描與獎勵。
- Staff 發獎與 admin 面板。
- SSE 或事件同步基礎。

### 2.2 小石現況

小石已經有五種類型：

- `exploration`：探索型。
- `inspiration`：靈光型。
- `resonance`：共鳴型。
- `engineering`：工程型。
- `entertainment`：娛樂型。

目前小石能力是被動詞條，種類包含：

- 小石掉落率提升。
- 答題分數提升。
- 開源力加成。
- 機率刪去錯誤選項。

目前 battle loadout 限制為 1 到 5 顆小石。這非常適合直接轉成「小隊編制」。

### 2.3 開源力現況

開源力目前是以 `open_power_records` 累積的數值，主要用途包含：

- 對戰獎勵。
- 商店購買。
- 道具與合成素材消費。
- 排行榜與 home summary 顯示。

下一步可以把開源力從「結算貨幣」提升成「任務中的有限戰術資源」，但必須加上單場花費上限，避免高分玩家滾雪球。

### 2.4 知識王現況

目前知識王是雙人或電腦對戰：

- 每回合一道四選一題。
- 答題後進入揭曉階段。
- 正解、解析、答題分數會顯示。
- 分數與速度仍有關。
- 結算後寫入開源力和小石掉落。

非即時戰略模式不應直接取代知識王，而應使用題庫作為「行動解決方式」之一。

### 2.5 社群攤位現況與限制

社群攤位已經有 QR 掃描、visit、claim、獎勵設定和 public display，但活動時程上會到 Day 5 才出現。因此它不能支撐 Day 1-4 的核心玩法。

可以使用的地方：

- Day 5 finale 加成。
- Day 5 特殊節點或短時間支援。
- Day 5 回顧、highlight、社群探索榜。
- 營後收藏與回顧。

不能依賴的地方：

- Day 1-4 追趕補助。
- Day 1-4 前線資源來源。
- Day 1-4 小石救援必要條件。
- Day 1-4 任務解鎖主線。

## 3. 設計原則

### 3.1 是 soft real-time，不是硬同步微操

本模式需要有即時壓力，但不應變成比手速、比網路延遲、或要求學員整天盯手機。建議採用 `soft real-time`：

- 地圖持續運作。
- 節點會在數分鐘內產生、衰退、被搶、被破壞。
- 小石部署後需要等待行動完成。
- 開源力若不回收會散逸、被事件吞掉、或被其他小隊搶走。
- 伺服器以 30 到 60 秒 tick 推進世界，而不是毫秒級同步。
- 玩家可以在任何時間進來做 1 到 3 個決策，然後回到課程或活動。

所有玩家仍然應該可以：

- 看清楚本回合局勢。
- 看清楚節點下一步會發生什麼。
- 和隊友討論。
- 選擇行動。
- 提交後看到結果與解析。

因此這不是傳統即時戰略的 APM 遊戲，而是營隊版持續戰線：

- 有倒數，所以會緊張。
- 有地圖，所以會競爭。
- 有部署時間，所以會取捨。
- 有保護期與追趕機制，所以不會把慢進場玩家打到出局。

### 3.2 策略來自取捨，不來自複雜公式

策略深度應來自：

- 這回合要先解哪個威脅。
- 哪顆小石最適合處理這個事件。
- 要不要花開源力保底。
- 要不要讓隊友協作。
- 要先拿獎勵，還是先保護基地。

策略深度不應來自：

- 難懂的傷害公式。
- 大量隱藏倍率。
- 複雜屬性相剋。
- 高速搶答。
- 需要查表的最佳化。

### 3.3 小石是學習風格與行動工具

小石不只是戰力。每顆小石應代表一種學員行為或營隊記憶：

- 探索：主動逛、找線索、問未知問題。
- 靈光：理解概念、抓關鍵、從錯誤中修正。
- 共鳴：合作、分享、跨隊交流。
- 工程：拆解問題、測試、部署、debug。
- 娛樂：展示、創作、讓活動變得有趣。

### 3.4 開源力是有限的公共資源

開源力可以在任務中使用，但要符合開源文化的隱喻：

- 花開源力不是買勝利，而是投入社群資源。
- 用來取得提示、協作、修正、支援。
- 個人開源力可以貢獻給小隊任務，但單場有上限。

### 3.5 財富平均是全系統規則

財富平均不應只套用在 Front。原本的知識王、商店、合成、Staff 發獎、Day 5 社群攤位、leaderboard 都要一起調整。

核心原則：

- 不沒收玩家已經累積的貢獻。
- 不讓高開源力玩家在玩法上永久碾壓低開源力玩家。
- 可花資源要有每日上限、單場上限與追趕補助。
- 額外收益應部分轉成小隊或全營公共資源。
- 排行榜必須保留，而且要有即時競爭壓力；只是不能只鼓勵財富累積，而要同時鼓勵佔點、救援、協作、探索、修復與回饋。

建議把開源力拆成兩個概念：

- `終身貢獻值`：永久累積，代表玩家對營隊與社群的總貢獻。用於回顧、稱號、收藏、個人成就，不作為強戰力來源。
- `可用開源力`：可以在商店、合成、任務中花費的餘額。這個部分需要每日上限、追趕補助與公共基金。

玩家可以很有成就感地看到自己貢獻很多，但遊戲不應讓「先賺很多的人」直接取得無法追上的戰術優勢。

### 3.6 營隊體驗優先，但不能把壓力拿掉

營隊體驗優先不等於低壓、無排名、無輸贏。這個遊戲需要明確競爭感，否則大型地圖會失去核心樂趣。

可以保留甚至強化：

- 即時戰況榜。
- 小隊排名。
- 節點倒數。
- 資源快散逸的壓力。
- 小石受困等待救援的壓力。
- 別隊正在逼近同一個節點的壓力。
- Day 4 主要競爭窗口。

需要避免的是：

- 永久沒收玩家收藏。
- 課程中持續扣資源，逼玩家一直看手機。
- 公開羞辱個人最低貢獻或答錯次數。
- 讓高資源玩家可以直接用錢買勝利。

所有玩法仍要符合現有 `agent.md` 的精神：

- 鼓勵互動、探索、課程專注、小隊貢獻。
- 允許玩家為了排行榜努力，但高價值競爭應集中在 Staff 開放的遊玩窗口。
- 允許分數差距存在，因為沒有差距就沒有競爭；但落後隊伍仍要有可見翻盤窗口。
- 允許壓力存在，因為「現在不做就沒了」是樂趣來源；但損失要可救援、可回復、可被 Staff 暫停。

### 3.7 多重審視後的硬性門檻

經過玩家樂趣、經濟平衡、營隊營運、工程落地四個角度檢查後，這個設計必須滿足下列門檻，否則即使功能完成也不算成功：

- 學員不用讀完整規則，也能在 1 分鐘內送出第一個有效 command。
- 正式模式必須有可見排行榜；沒有排行榜或戰況榜的版本不算完成。
- 玩家要能感覺到至少一個東西正在倒數：節點、開源力、小石訊號、受困小石或世界事件。
- 玩家要能感覺到別隊也在搶同一張地圖，而不是只是在玩單人任務。
- 每次重要決策至少有兩個合理選項，不是看事件類型就知道唯一正解。
- 至少有一次自然的小隊討論，而不是每個人各玩各的手機。
- 答錯的人仍然能提供線索、降低損害、或在下一回合提出主意。
- 每位參與者都能說出自己本場做了什麼貢獻。
- 財富平均不讓高貢獻玩家覺得被沒收，也不讓低資源玩家覺得被貼標籤。
- Staff 可以控制什麼時候開放、暫停、凍結、關閉，不需要現場救火。
- 工程上必須可漸進導入，不得一次改壞商店、排行榜、知識王和新模式。

第一版應優先驗證這些門檻，而不是先追求完整規則、完整經濟或完整多人同步。

### 3.8 排行榜是主要樂趣，不是附屬功能

排行榜要讓玩家想多玩一局、想叫隊友來幫忙、想在倒數前救回節點。它不是最後才補上的統計頁。

第一版至少要有一個小隊即時戰況榜：

- `戰線控制`：目前佔領或穩定維持的節點分數。
- `開源力確保`：成功回收並守住的前線開源力。
- `小石救援`：救回受困小石或保護小石訊號。
- `事件修復`：處理 bug、危機、課程事件、系統事件的分數。
- `協作連線`：同隊多人接力、跨區支援、基地後勤支援。

排行榜顯示規則：

- Home 和戰線頁都要能看到前 5 名、自己的小隊排名、和上一個更新週期的名次變化。
- 更新週期可以是 30 到 60 秒，不需要毫秒即時。
- Day 1 可以顯示 `暖身戰況`，不直接計入最終排名，但仍要讓學員看到小隊正在和其他隊伍比較。
- Day 2 起顯示正式小隊排名。
- Day 4 是主要競爭日，可以開啟更高壓的限時榜。
- Day 5 進入 freeze 後，排行榜轉成 finale 展示與 highlights。

排行榜可以公開：

- 小隊總排名。
- 小隊控制區域。
- 小隊本輪得分。
- 小隊救援次數。
- 小隊連勝或防守 streak。
- 代表性 highlight。

排行榜不要公開：

- 個人最低貢獻。
- 個人答錯次數。
- 個人開源力貧富。
- 誰害小隊掉點。

這樣競爭感會存在，而且很明顯；但壓力指向小隊決策與地圖局勢，不指向個人羞辱。

## 4. 參考遊戲與可借鑑點

### 4.1 OpenFront / territory.io 類型大型地圖競爭

參考重點：

- 所有玩家在同一張地圖上看到戰線推進。
- 擴張、佔領、防守、資源維持形成自然壓力。
- 地圖上有「現在不搶就沒了」的機會點。
- 排行榜與領土視覺是核心樂趣，不是附屬統計。
- 玩家不需要讀大量文字，只要看到顏色、邊界、倒數與分數變化就能理解局勢。

借鑑方式：

- 主模式改成 `Front`：全營共享戰線。
- 小隊透過小石對節點下指令：擴張、攻擊、支援、修復、救援。
- 地圖每 30 到 60 秒或每 1 秒低頻 tick 推進，前端用動畫做即時感。
- 不做自由座標單位、不做碰撞、不做毫秒同步。
- 領土、節點、分數、倒數、排名必須第一眼可見。

不直接照抄：

- 不做純吞地圖的數值競速。
- 不讓強隊靠資源雪球把弱隊完全清出遊戲。
- 不讓玩家在課程中因離線被重罰。
- 不做匿名大亂鬥；這是小隊制營隊活動。

### 4.2 Into the Breach

參考重點：

- 敵人行動會預告。
- 玩家每回合處理一個可讀的戰術 puzzle。
- 小隊能力差異明確。
- 單場短，失敗也能理解原因。

借鑑方式：

- 任務板上預告下回合的 bug、issue、魔王行動。
- 玩家不是比反應，而是解每回合的最佳取捨。
- 每回合結果要可預測，避免黑箱。

參考連結：https://subsetgames.com/itb.html

### 4.3 Tactical Breach Wizards

參考重點：

- 小隊成員能力明確。
- 玩家可在提交前預覽結果。
- 允許回溯試錯，降低挫折。

借鑑方式：

- 行動提交前顯示「預期結果」。
- 開源力可提供一次撤回或重新規劃。
- 避免單一錯誤導致整場崩盤。

參考連結：https://store.steampowered.com/app/1043810/Tactical_Breach_Wizards/

### 4.4 Battle for Wesnoth

參考重點：

- 開源回合制戰略。
- 單位、任務、地圖可資料化。
- 劇情與關卡可由內容檔維護。

借鑑方式：

- 任務定義放在 TOML/JSON。
- 小石能力走資料驅動，不硬寫在前端。
- 關卡可以逐步增加，不需要一次寫完整戰役。

參考連結：https://github.com/wesnoth/wesnoth

### 4.5 OpenXcom

參考重點：

- 小隊、任務、裝備、戰術層清楚。
- 適合研究「出戰前準備」和「任務後結算」。

不建議直接借：

- 複雜命中率。
- 長線基地經營。
- 大量裝備管理。

參考連結：https://github.com/OpenXcom/OpenXcom

### 4.6 OpenDuelyst

參考重點：

- 卡牌和棋盤的短局混合。
- 編組與手上資源決定戰術選擇。

借鑑方式：

- 小石 loadout 類似 deck / squad。
- 道具與開源力類似手牌資源。
- 5x5 或 lane-based board 足夠支撐短局策略。

參考連結：https://github.com/open-duelyst/duelyst

### 4.7 boardgame.io / rot.js

參考重點：

- boardgame.io 適合回合制狀態機思維。
- rot.js 適合格子地圖、pathfinding、roguelike prototype。

專案建議：

- 正式核心邏輯仍寫在 Go server。
- 可以借它們的概念，不建議把整個規則引擎塞進前端。

參考連結：

- https://boardgame.io/
- https://ondras.github.io/rot.js/hp/

## 5. 新模式名稱與產品概念

### 5.1 建議名稱

主模式暫定名稱：

- 中文：開源戰線。
- 英文內部代號：Fronts。
- API namespace：`fronts`。

保留但降級的副模式：

- 中文：小石任務。
- 英文內部代號：Operations。
- API namespace：`operations`。
- 定位：教學任務、節點事件、救援遭遇、題目事件，不是主玩法。

本文後續用 `Front` 指大型共享地圖，用 `Operation` 指短任務或局部遭遇。

### 5.2 主模式定位：OpenFront-like 開源戰線

Front 是一張全營共享、持續運作的戰略地圖。每個小隊都有自己的顏色、基地、前線節點與排名。

目前實作只保留 `territory_grid` 校園領土戰；早期的 node map、相鄰節點指令與獨立節點規則引擎已移除。玩家直接選擇校園地圖座標，領土邊界與地標事件是唯一的 Front 操作介面。

玩家帶 1 到 5 顆小石進入戰線，對領土或地標下達指令，也可以把小石駐點在己方領土：

- `擴張`：從己方節點往相鄰空白或中立節點推進。
- `攻擊`：和其他小隊爭奪 contested 節點。
- `防守`：提高己方節點穩定度，延緩被搶或被事件破壞。
- `修復`：處理 bug、事故、課程事件、系統事件。
- `偵查`：揭露即將出現的小石訊號、開源力風暴或敵方壓力。
- `救援`：救回受困小石或保住快消失的小石訊號。
- `協作`：同隊多人接力，或把基地後勤支援轉成地圖效果。
- `駐點`：把 1 到 5 顆小石託管在己方非基地領土，提高防禦並自動建立跨隊交易路線。
- `撤回`：領土仍屬己方時，由原部署玩家取回駐點小石。

地圖不是一場打完就消失，而是持續累積狀態：

- 節點有 owner、control、defense、pressure、resource、event。
- 小隊持有的節點會給分、產生前線資源、開啟新的相鄰路線。
- 節點會衰退、被事件攻擊、被其他隊伍施壓。
- 開源力訊號與小石訊號會倒數，沒處理就散逸或進入受困狀態。
- Staff 可以依照活動流程開啟、暫停、加速、凍結或結算戰線。

### 5.3 Operation 的新定位

Operation 不再是主模式，而是 Front 內部的短遭遇。

用法：

- Day 1 教學：用 3 到 4 步教會派小石、處理事件、看解析。
- 節點挑戰：爭奪特殊節點時跳出一個短任務或題目。
- 小石救援：小石受困時，用短任務救回。
- Day 5 社群支援：社群攤位開放後，掃攤位或完成互動可解鎖一次 finale 支援事件。
- 世界事件：大型危機節點可以用 Operation 當局部解法。

這樣可以保留原本短任務設計的價值，但玩家主要記得的是「我們在大地圖上搶節點、救小石、衝排行榜」。

### 5.4 第一版社交形態：Team Front

第一版不需要完整多人物理同步，但必須讓小隊覺得在同一張地圖上共同競爭。

建議第一版支援：

- 每位玩家都能從自己的手機送出 Front command。
- 同隊 command 合併到同一個 team state。
- 一個節點可以被同隊多人接力加壓。
- 每個玩家每段時間有 command cooldown，避免單一玩家洗指令。
- 小隊可以指定一位 `driver` 在投影或隊伍手機上看大局，但其他人仍能送支援、答題、救援。
- 競爭主要以小隊排名呈現，不以個人羞辱資料呈現。

### 5.5 核心樂趣

玩家應該感覺到：

- 我打開地圖就知道目前誰在領先、哪裡正在被搶。
- 我現在不派小石，這個小石訊號可能就消失。
- 我現在不補防，這個節點可能 10 分鐘後掉到別隊手上。
- 開源力訊號正在散逸，再不收就沒了。
- 我的小石不是收藏品而已，牠真的能在地圖上做事。
- 排行榜在動，我們小隊可以追上前一名。
- 答題不是考試，而是能讓戰線加速、救援或反打。
- 輸掉一個節點會痛，但不是帳號毀掉；下一波還能救回來。
- 和隊友分工會比自己亂點更強。

### 5.6 壓力來源設計

這個模式需要壓力，壓力來源要明確、可預告、可反制。

主要壓力：

| 壓力 | 玩家感受 | 失敗後果 | 安全邊界 |
| --- | --- | --- | --- |
| 小石訊號倒數 | 現在不處理，小石就沒了 | 訊號消失或變受困 | 不沒收既有收藏，只錯過 bonus 或進救援 |
| 開源力散逸 | 開源力快沒了 | 前線開源力減少、節點少拿分 | 不每秒扣真實可用開源力 |
| 節點衰退 | 不守就掉點 | control / defense 下降 | Staff 可暫停，全營 quiet mode 不扣 |
| 別隊施壓 | 對方快搶走了 | contested 節點 owner 改變 | 有防守與反打窗口 |
| 小石編隊與駐點 | 不同編隊影響命令、駐點防禦與交易 | 一般命令不消耗；駐點時 1 到 5 顆進入託管 | 駐點領土失守時，小石永久轉給執行占領的玩家 |
| 世界事件 | 全營地圖變難 | 某區 debuff 或事件擴散 | Staff 可發全營護盾 |

壓力要能推動玩家行動，但不能變成不可逆懲罰。

### 5.7 事件卡設計原則

每張事件卡都必須避免「看到類型就知道唯一解」。

規則：

- 每個主要事件至少提供兩個合理處理方向。
- 每個方向都要預覽後果。
- 正確選擇不一定是最高分選擇，可能是保穩定、拿線索、推進目標、保留資源。
- 小石類型提供偏好與加成，但不應只有一種小石能處理。
- 錯誤或不完整處理也要產生線索、降低傷害或生成回顧。

範例：

```text
事件：部署後服務讀不到設定檔

選項 A：看 log 揭露真因
- 效果：揭露下一張系統事件，若使用工程型或探索型小石，額外降低 1 點風險。

選項 B：先 rollback 保穩定
- 效果：本回合穩定度不下降，但主要目標不推進。

選項 C：補一個測試
- 效果：答一題調查型問題。答對推進目標，答錯也揭露一個線索。
```

### 5.8 調查型題目，而不是考試題

Operation 中的題目要像調查，不要像突然考試。

不建議文案：

```text
請回答 Docker 和 VM 的差異。
```

建議文案：

```text
服務在部署後讀不到設定檔。以下哪個觀察最能幫你縮小問題？
```

答題結果：

- 答對：事件完整處理，顯示原因與解析。
- 答錯但看解析：事件部分處理，降低傷害或揭露下一步線索。
- 使用提示：不直接給答案，只給觀察方向。
- 小隊討論：可以讓一名隊友標記為 `分享者` 或 `支援者`。

## 6. 遊戲流程

### 6.1 主循環

完整循環：

1. 玩家取得或整理小石。
2. 玩家打開 `開源戰線`，看到全營地圖、倒數事件、自己小隊排名。
3. 玩家選 1 到 5 顆小石與一個領土座標，下達 `擴張`、`攻擊`、`防守`、`修復`、`救援`、`支援` 或 `挑戰` command。
4. Server 接收 command，進入 pending 狀態，前端立即顯示箭頭、進度條或小石行動中。
5. Server 以固定 tick 推進地圖，彙整同隊與敵隊壓力。
6. 節點 owner、control、defense、pressure、resource、event、駐點或交易變化後，以 SSE 推送最新個人化 snapshot。
7. 玩家看到戰線變化、排行榜跳動、開源力訊號倒數、小石可能受困。
8. 部分節點觸發題目或短 Operation；答對可加速，答錯也給情報或降低損害。
9. 一段活動窗口結束後，系統結算本輪分數、回顧、獎勵與 highlight。
10. 地圖不消失，下一個窗口繼續從新的局勢開始。

### 6.2 活動窗口節奏

Front 可以無限遊玩，但高價值競爭應集中在 Staff 開放的窗口，避免學員整天盯手機。

建議節奏：

| 模式 | 時長 | 用途 | 壓力 |
| --- | --- | --- | --- |
| `closed` | 課程中、維護中 | 只能看摘要，不推進地圖 | 無流失 |
| `quiet` | 課程間或休息前後 | 低頻資源、低傷害事件 | 低 |
| `open_play` | 自由時間 | 正式搶點、排行榜、救援 | 中 |
| `surge` | Day 4 主要競爭窗口 | 高價值節點、快速倒數、強排名壓力 | 高 |
| `finale_freeze` | Day 5 結算前 | 凍結排名、修資料、產生展示 | 無新競爭 |

前 4 天不依賴社群攤位。Day 5 才新增 `booth_window`，用於社群攤位 QR 與 finale 加成。

### 6.3 Tick 結構

每個 tick 做固定順序，讓規則可預測：

1. 收集 pending command。
2. 驗證 command：隊伍、冷卻、相鄰節點、小石狀態、front status。
3. 套用小石類型加成。
4. 累積 pressure / defense / repair / scout。
5. 套用節點衰退與事件倒數。
6. 判斷 owner 變更、事件爆發、小石受困、開源力散逸。
7. 更新 team score 與 leaderboard。
8. 寫入 event log，推送 delta。

MVP 可以每 1 秒 tick 一次，但地圖上的重要倒數以 30 秒、60 秒、5 分鐘為單位。這樣畫面有即時感，決策不會變成手速競賽。

### 6.4 勝利與失敗

Front 不需要傳統單場勝敗，而是以活動窗口和整天結算判斷。

小隊得分來源：

- 控制節點。
- 守住高價值節點。
- 回收快散逸的前線開源力。
- 救回受困小石。
- 處理系統事故與課程事件。
- 成功答題加速。
- 同隊多人接力。
- Day 5 社群攤位 finale 加成。

小隊失分或 setback：

- 節點被其他隊伍搶走。
- 節點衰退到故障。
- 前線開源力散逸。
- 小石訊號消失。
- 小石暫時受困。

安全邊界：

- 未駐點的小石不會被搶走。
- 玩家主動駐點時會看到失守風險；領土被占領後，該駐點小石由攻擊玩家取得。
- 真實可用開源力不會被每秒扣。
- 課程或睡眠時間不自動重罰。
- Staff 可以暫停、凍結、發護盾或補救。

## 7. 地圖設計

### 7.1 MVP 使用 territory map，不使用 lane board

主模式需要玩家一眼看到小隊顏色與戰線推進，因此 MVP 應做「格子 / 省份式 territory map」，不是短任務 lane board。

地圖節點欄位：

- `id`
- `x`
- `y`
- `terrain`
- `zone`
- `ownerTeamID`
- `control`
- `defense`
- `pressureByTeam`
- `resource`
- `eventID`
- `neighborIDs`

### 7.2 地圖區域

Day 1-4 使用三種核心區域，不使用社群攤位：

| 區域 | 玩法定位 | 主要收益 | 主要風險 |
| --- | --- | --- | --- |
| 課程區 | 題目、理解、加速 | 答題加速、情報、學習分 | 答錯造成低效率，不造成羞辱 |
| 系統區 | 修復、防守、基礎收益 | 穩定分、前線開源力、防守 streak | bug、故障、節點衰退 |
| 基地區 | 後勤、救援、保底 | 小石恢復、支援 token、落後追趕 | 收益較低，難以衝榜 |

Day 5 才新增：

| 區域 | 玩法定位 | 主要收益 | 主要風險 |
| --- | --- | --- | --- |
| 社群區 | 攤位互動、finale 加成、回顧 | 社群探索榜、支援卡、highlight | QR 排隊與刷碼，需要 cooldown |

### 7.3 MVP 地圖範例

```text
       [課程 A] -- [課程 B] -- [高價值題目]
          |            |              |
[基地 1] -- [系統 A] -- [中央前線] -- [系統 B] -- [基地 2]
          |            |              |
       [修復站] -- [開源力訊號] -- [小石訊號]
```

特性：

- 每隊從自己的基地附近開始。
- 中央有 contested 高分節點。
- 外圍有小石訊號與開源力訊號，逼玩家分兵。
- 基地區提供保底與救援，不會讓落後隊伍無事可做。
- Day 5 才把社群攤位節點接到地圖外圈或 finale 區。

### 7.4 不做完整 RTS 單位移動

第一版不要做：

- 自由座標單位。
- pathfinding。
- 兵種碰撞。
- 視野迷霧細節。
- 每顆小石在地圖上逐格走路。

第一版只做：

- 節點相鄰關係。
- command 從一個節點指向相鄰節點。
- pressure 隨 tick 累積。
- 前端畫箭頭、光效、進度條，營造即時感。

## 8. 小石在 Front 中的角色

### 8.1 五型能力映射

| 小石類型 | Front 定位 | 主要行動 | 戰術價值 |
| --- | --- | --- | --- |
| 探索型 | Scout | 偵查、揭露、小石訊號追蹤 | 找到高價值節點與即將散逸的資源 |
| 靈光型 | Analyst | 提示、解題加速、預測倒數 | 降低題目失誤，讓爭奪更有效率 |
| 共鳴型 | Support | 接力、共享、防守連線 | 讓同隊多人行動合併成更強壓力 |
| 工程型 | Engineer | 修 bug、加防、穩定節點 | 守住核心節點與抵抗事件 |
| 娛樂型 | Momentum | 鼓舞、展示、士氣 streak | 提高排名追分、回顧與活動高潮 |

Day 1 玩家只看到一個動詞：

| 小石類型 | Day 1 顯示 |
| --- | --- |
| 探索型 | 看線索 |
| 靈光型 | 拿提示 |
| 共鳴型 | 一起討論 |
| 工程型 | 修問題 |
| 娛樂型 | 留回顧 |

完整能力表從 Day 2 之後再逐步顯示。

### 8.2 小石行動範例

#### 探索型

行動：

- `探路`：揭露相鄰未知節點或隱藏事件。
- `掃描`：看某節點 5 分鐘內可能發生的衰退、資源或敵方壓力。
- `追蹤訊號`：提高小石訊號捕捉率，延長訊號倒數。

被動：

- 若本輪至少成功偵查 2 次，小隊獲得探索分與小石碎片機會。

#### 靈光型

行動：

- `提示`：答題或節點挑戰前顯示關鍵詞。
- `刪選項`：刪除一個錯誤選項。
- `預測`：看某個節點若不處理會在幾個 tick 後造成什麼後果。

被動：

- 答錯後仍可閱讀解析並取得部分 pressure、情報或減少懲罰。

#### 共鳴型

行動：

- `接力`：同隊多人的 command 在同一節點加成。
- `分享`：把一題解析轉成全隊小幅前線資源或 cooldown 減免。
- `支援`：保護一個節點在短時間內不衰退。

被動：

- 若隊伍中有不同玩家在同一窗口貢獻，結算協作分提高。

#### 工程型

行動：

- `Debug`：移除 bug 或降低節點事件傷害。
- `加固`：提高節點 defense。
- `部署`：提高己方 control 或縮短佔領時間。

被動：

- 正確答題後，額外推進修復或佔領進度。

#### 娛樂型

行動：

- `展示`：把成功節點轉成 highlight 或 streak。
- `鼓舞`：讓下一次前線 command 少花前線開源力。
- `衝刺`：短時間提高小隊追分效率，但不能直接買勝利。

被動：

- 若本輪完成 highlight 條件，排行榜與 finale 顯示特殊回顧。

### 8.3 小石不要新增太多主動技能

第一版不要讓每顆具名小石都有完全不同技能。建議：

- Day 1 每種 `type` 只顯示 1 個動詞。
- Day 2 後每種 `type` 可顯示 2 到 3 個行動。
- 具名小石只改變強度或觸發文案。
- 稀有小石提供額外小 bonus。

這樣可以避免規則爆炸。

## 9. 開源力與前線資源

### 9.1 三層資源模型

為了同時做到「開源力快沒了」的刺激與「不毀掉既有經濟」的安全邊界，必須拆成三層：

| 資源 | 是否會快速流失 | 是否影響商店/合成 | 主要用途 |
| --- | --- | --- | --- |
| 可用開源力 | 否 | 是 | 商店、合成、少量活動消費 |
| 今日支援額度 | 當日過期 | 否 | 提示、加派、緊急修補、救援 |
| 前線開源力 | 會，在 Front 內快速消耗或散逸 | 否 | 擴張、防守、修復、搶點、維持節點 |

玩家感受到的「開源力就要沒了」主要來自前線開源力，不是每秒扣玩家帳戶裡的可用開源力。

### 9.2 前線開源力壓力曲線

前線開源力是 Front session 或小隊窗口內的戰術資源。

建議初始值：

- Day 1 教學：60。
- Day 2 一般窗口：100。
- Day 4 surge：120，但消耗也更快。

壓力區間：

```text
100-40：正常擴張、攻擊、防守。
39-15：警告，UI 顯示「前線開源力快不夠了」。
14-1：只能做防守、撤退、救援、低成本行動。
0：失去一個前線節點、錯過一個資源訊號，或一顆出戰小石進入受困狀態。
```

規則：

- 前線開源力不可帶出 Front session。
- 前線開源力不可轉成可用開源力。
- 前線開源力低時可以靠答題、救援、基地區、今日支援額度回補。
- 擴張越多，維持成本越高，避免無腦吞地圖。
- Staff 在 quiet / closed / freeze 模式可暫停前線開源力流失。

建議：

- 單人支援：每個活動窗口最多投入 30 今日支援額度。
- 小隊支援：每個活動窗口隊伍總上限 150。
- `重新規劃` 與 `緊急修補` 每個窗口各最多 1 次。
- 低等級或新玩家可有免費支援 token。
- 今日支援額度不可用於商店、合成、收藏品或直接轉成可用開源力。

### 9.3 前線行動花費

| 花費 | 行動 | 效果 |
| --- | --- | --- |
| 5 | 偵查 | 顯示相鄰節點事件或倒數 |
| 10 | 防守 | 節點 defense + 小量 |
| 15 | 擴張 | 對相鄰節點增加 pressure |
| 20 | 修復 | 降低 bug / 故障事件傷害 |
| 25 | 攻擊 | 對 contested 節點增加較高 pressure |
| 30 | 救援 | 推進受困小石救援進度 |
| 今日支援 20 | 提示 | 顯示題目提示 |
| 今日支援 30 | 加派 | 額外送出一個弱化 command |
| 今日支援 40 | 緊急修補 | 抵消一次節點衰退或小石受困 |

### 9.4 開源力回收

活動窗口結束後開源力獎勵不應只給第一名，但排名必須有差異，否則競爭會失去意義。

建議結算：

- 參與基礎獎勵。
- 小隊排名獎勵。
- 節點控制獎勵。
- 小石救援獎勵。
- 事件修復獎勵。
- 閱讀解析獎勵。
- 小隊多人貢獻獎勵。
- 每日上限內額外獎勵。

### 9.5 全域財富平均設計

財富平均要套用到整個遊戲，而不是只套用 Front。原因是現有商店、合成、知識王、leaderboard 都和開源力總額綁在一起，如果只有新模式做平均，玩家仍然可以透過原本玩法累積巨大優勢。

建議經濟模型：

```text
終身貢獻值 lifetime_open_power
  = 所有正向貢獻記錄總和
  = 不會因購買或補助而下降

可用開源力 spendable_open_power
  = 可消費帳戶餘額
  = 商店、合成、少量活動消費從這裡扣

每日戰術預算 daily_tactical_budget
  = 每日固定重置或補到上限
  = Front、提示、撤回、救援等戰術行動優先使用這裡

小隊公共基金 team_open_power_pool
  = 小隊共同資源
  = 由隊員額外收益回饋、任務協作、Day 5 攤位互動累積
```

四個帳戶用途：

| 帳戶 | 是否永久累積 | 是否可花 | 是否進排行榜 | 主要用途 |
| --- | --- | --- | --- | --- |
| 終身貢獻值 | 是 | 否 | 可顯示但不作唯一排序 | 回顧、稱號、成就 |
| 可用開源力 | 否 | 是 | 不建議主導排行 | 商店、合成 |
| 每日戰術預算 | 每日重置或補足 | 是，限前線支援 | 否 | 提示、撤回、救援、前線支援 |
| 前線開源力 | 活動窗口內變動 | 是，限 Front 內 | 影響戰況榜 | 擴張、防守、修復、搶點 |
| 小隊公共基金 | 營期內累積 | 是，限小隊用途 | 可作小隊合作指標 | 救援、補助、共同解鎖 |

### 9.6 追趕補助公式與防套利

追趕補助不要看最高分，也不要直接看玩家「現在」的可用開源力。直接看現在餘額會產生套利：玩家先買東西把餘額花低，再領補助。

建議第一版使用每日快照：

```text
snapshot_balance = 今日開始時的可用開源力
earned_today = 今日已取得的可用開源力獎勵
team_median_snapshot = 同隊玩家今日開始時可用開源力中位數
eligibility_balance = max(snapshot_balance, snapshot_balance + earned_today)

若 eligibility_balance < team_median_snapshot * 0.6：
  tactical_bonus = min(30, floor((team_median_snapshot * 0.6 - eligibility_balance) * 0.25))
  material_voucher = min(75, floor((team_median_snapshot * 0.6 - eligibility_balance) * 0.5))
否則：
  tactical_bonus = 0
  material_voucher = 0
```

規則：

- 補助只補到接近小隊中位數的 60%，不會讓玩家靠補助超車。
- 每人每日最多補 30 今日支援額度。
- 每人每日最多補 75 材料折價券。
- 材料折價券最多折抵符合資格的一階/common 合成素材 50%。
- 補助不直接發可用開源力。
- 已經花掉的可用開源力不會提高補助資格。
- 小隊人數太少或中位數異常時，改用全營同日同年級或同隊伍分位數。
- 補助來源可以顯示為「小隊互助」或「基地支援」，避免玩家感覺像失敗救濟。Day 5 之後才可顯示「社群支援」。

### 9.7 回饋池公式與小隊基金限制

高資源玩家不要被扣錢，但他們的「額外收益」可以部分進公共池。

建議：

```text
base_reward = 本次玩法原始開源力獎勵
lifetime_percentile = 玩家終身貢獻值在小隊或全營的分位數

pool_rate = clamp((lifetime_percentile - 60) / 40 * 0.25, 0, 0.25)
team_pool_reward = floor(base_reward * pool_rate)
personal_reward = base_reward - team_pool_reward
```

注意：

- 這不是扣既有財產，而是把部分新增收益轉成小隊公共資源。
- 使用終身貢獻分位數，不使用目前可用餘額，避免玩家靠花錢調整分位。
- 回饋比例平滑上升，不使用 80% 硬切線。
- UI 文案應寫成「你把一部分成果回饋給小隊公共基金」。
- 小隊公共基金應能產生可見好處，例如小隊提示、共同任務入場、低資源隊友合成補助。

小隊公共基金第一版限制：

- 每隊每日最多收入 300。
- 每隊每日最多支出 250。
- 每位受益者每日最多使用 75。
- 不可直接轉成個人可用開源力。
- 不可購買收藏品、御守、稀有或高階合成素材。
- 不可資助會直接擴大戰力差距的商品。
- 大額支出需顯示 audit log；team-lite 現場版可由 driver 與另一位隊員確認。

### 9.8 原本遊戲系統的調整

財富平均應套用到既有玩法。

#### 知識王

目前知識王獎勵偏向勝者與分數。建議改成：

- 勝利仍有獎勵，但不要讓輸家完全沒有開源力。
- 閱讀解析給固定學習獎勵。
- 每日知識王高額獎勵有上限。
- 超過上限後仍可玩，但只給小額參與與回顧。
- 高資源玩家的部分額外獎勵進入小隊公共基金。

範例：

```text
參與：5
每題答對：4
每題首次閱讀解析：2
勝利：15
單場個人可用開源力上限：80
每日知識王可用開源力上限：120
超過每日上限後，只寫入終身貢獻與回顧，不再大量增加可用開源力
```

#### 商店

商店應分成三類：

- 基礎成長品：合成素材、基礎輔助道具。價格低，可補助。
- 戰術消耗品：提示、保底、任務支援。每日或單場限量。
- 收藏品：明信片、造型、紀念品。可以貴，作為高資源玩家的消費出口。

建議：

- 影響玩法的商品設購買上限。
- 低資源玩家可拿折價券或小隊基金補助。
- 高價商品以收藏與紀念為主，不直接提高勝率。
- 道具內容檔需要標記 `economy_class`、`subsidy_eligible`、`power_tier`、`daily_purchase_limit`。

建議分類：

```toml
economy_class = "growth" # growth / tactical / collectible
subsidy_eligible = true
power_tier = "common" # common / rare / limited
daily_purchase_limit = 1
```

限制：

- `collectible` 不可使用補助。
- `rare` / `limited` 不可使用小隊公共基金。
- tactical 消耗品不走現有永久購買模型，需另設每日兌換或任務內 token。

#### 合成

合成不應只看誰能買最多素材。

建議：

- 一階合成素材保持可負擔。
- 二階合成保留成就感，但不要變成戰力斷層。
- 低資源玩家可透過每日任務取得定向素材補助。
- 小隊公共基金可幫隊員補一部分素材，但每日有限制。
- 材料折價券只支援 common / 一階素材，不支援 rare、高階合成、御守、收藏品。

#### 社群攤位

社群攤位到 Day 5 才出現，因此不應作為 Day 1-4 的追趕來源、前線資源來源或小石救援必要條件。

Day 5 建議：

- 首次拜訪攤位給固定 finale 獎勵。
- 攤位互動可給 `社群探索榜`、highlight、稱號進度或 finale 支援卡。
- 社群支援卡只在 Day 5 booth window 或 finale 使用。
- 高資源玩家拜訪攤位時，額外部分可進小隊公共基金，但不影響 Day 1-4 前線平衡。
- 同一攤位不可重複刷，必須有 cooldown 與 fallback code。

#### Staff 發獎

Staff 發獎需要保留人工判斷，但系統可以給提示：

- 掃描玩家時顯示該玩家是否低於小隊中位數。
- 若玩家低資源，推薦發基礎小石、合成素材或戰術預算。
- 若玩家高資源，推薦發紀念品、稱號、回顧片段，而不是大量開源力。
- Staff 仍可覆寫，系統只做建議。

#### Leaderboard

排行榜是 Front 的主要樂趣，不能延後到最後。調整重點不是弱化排行榜，而是避免單一可用開源力榜把遊戲變成財富雪球。

Front 正式榜：

- 小隊總戰線分。
- 節點控制分。
- 前線開源力確保分。
- 小石救援分。
- 事件修復分。
- 協作接力分。
- Day 5 才新增社群探索分。

可用開源力不應作為主要排名依據，因為它會被花掉、補助、回饋，並不代表完整戰線貢獻。小隊總戰線分應該是玩家第一眼看到的正式排名。

### 9.9 全域每日上限

第一版建議先用保守 cap，避免活動期間出現不可逆的財富膨脹。

| 類型 | 上限 |
| --- | --- |
| 自動玩法可用開源力總收入 | 每人每日 250 |
| 知識王可用開源力 | 每人每日 120 |
| Front / Operation 可用開源力 | 每人每日 120 |
| Day 5 社群攤位可用開源力 | 每人每日 60，只在 booth window 生效 |
| 今日支援額度 | 每人每日 100，Asia/Taipei 午夜過期，不結轉 |
| 追趕支援額度 | 每人每日 30 |
| 前線開源力 | 每個活動窗口重置或依戰況給定，不可帶出 |
| 材料折價券 | 每人每日 75，只能折 common/一階素材，最多折 50% |
| 小隊公共基金收入 | 每隊每日 300 |
| 小隊公共基金支出 | 每隊每日 250 |
| 小隊基金單一受益者支出 | 每人每日 75 |
| Staff 單次可用開源力預設上限 | 50 |
| Staff 每人每日可用開源力預設上限 | 100 |

超過上限後：

- 仍可記錄終身貢獻。
- 仍可給回顧片段、稱號、收藏型獎勵。
- 不再增加大量可用開源力。
- Staff 可覆寫，但必須留下 override reason。

### 9.10 Front / Operation 獎勵上限

Front 活動窗口建議：

```text
參與：10
小隊排名：第一名 30，第二名 20，第三名 15，其餘 8
節點控制：每個有效節點 3，最多計 10 次
小石救援：每次 8，最多計 3 次
事件修復：每次 5，最多計 5 次
協作接力：每次 4，最多計 5 次
閱讀解析：5
單窗口個人可用開源力上限：75
每日 Front / Operation 可用開源力上限：120
```

Operation 作為短遭遇時建議：

```text
參與：10
每個目標進度：5，最多計 4 次
成功：20
閱讀解析：5
每日首次完成：20
單場個人可用開源力上限：75
每日 Front / Operation 可用開源力上限共用：120
```

Front / Operation 支援花費不應讓重複刷變成穩定套利：

- 單人每個窗口最多使用 30 今日支援額度。
- 小隊每個窗口隊伍總上限 150。
- 重複完成同一任務只給低額參與與回顧。
- 首通、每日首次、解析獎勵都必須 idempotent。

### 9.11 經濟 UI 顯示原則

玩家畫面應避免讓補助看起來像羞辱。

建議顯示：

- `累積貢獻`：你至今為營隊累積的開源力。
- `可用開源力`：目前可以在商店與合成使用。
- `今日支援額度`：今天可以用於提示、前線支援、撤回、救援。
- `前線開源力`：目前這個活動窗口內可用於擴張、防守、修復。
- `小隊公共基金`：隊伍共同投入的資源。

避免顯示：

- `你是低收入玩家`。
- `你被補助`。
- `你比隊友窮`。

推薦文案：

- `今日基地支援已補足。`
- `小隊公共基金為你折抵 30 開源力。`
- `你將 12 開源力回饋到小隊公共基金。`

## 10. 題庫與知識題整合

### 10.1 題目是戰線加速器，不是每次行動都考

Front 中的答題應是「讓戰線變快、變穩、或反打」的方式，不是每次行動都固定出題。

例如：

- `Linux 權限事故` 節點需要答 Linux 題，答對降低修復時間。
- `Prompt Injection` 節點需要答 AI / security 題，答對揭露敵方壓力來源。
- `Docker 部署節點` 需要答 container 題，答對提高工程型小石效果。
- `Day 5 社群授權節點` 才需要答 open source license / community 題。

### 10.2 題目需要新增分類欄位

目前 quiz question 欄位較簡單。Front 需要額外 metadata。

建議新增：

- `topic`：AI、Linux、security、web、open_source、sitcon、community。
- `difficulty`：intro、basic、advanced。
- `day`：camp day。
- `front_tags`：可出現在哪些地圖節點或事件。
- `operation_tags`：可出現在哪些短遭遇。
- `hint`：提示文字。
- `learning_goal`：這題希望學員學到什麼。

實作方式有兩種：

1. 擴充現有 `quiz_questions.csv`。
2. 新增 `quiz_question_metadata.toml`，用 question id 對應 metadata。

建議先採用第 2 種，避免破壞現有 CSV 匯入流程。

### 10.3 答錯也要有用

Front 模式下答錯不應直接歸零，否則競爭壓力會變成挫折。

建議：

- 答對：節點修復加速、pressure 增加、defense 增加或救援進度增加。
- 答錯但看完解析：取得少量情報、降低事件傷害或延長倒數。
- 答錯後用靈光型小石修正：取得部分效果。
- 小隊協作後答錯：不公開個人錯誤，但隊伍少拿 bonus。
- 同一題不可被同隊重複刷出完整獎勵。

## 11. 任務類型

### 11.1 課程節點

用途：

- 配合當天課程主題。
- 引導學員複習概念。
- 作為 Front 地圖上的加速節點或知識爭奪點。

例：

- Day 2 軟工：部署事故、Docker、GPG、測試。
- Day 3 AI：Prompt、有限資料、模型判斷。
- Day 4 資安：社交工程、hash、權限、攻防觀念。

### 11.2 系統節點

用途：

- 產生基礎戰線分。
- 提供防守、修復、穩定度玩法。
- 讓工程型小石有高價值位置。

例：

- 部署事故。
- 權限錯誤。
- 服務異常。
- 測試缺口。
- 監控警報。

### 11.3 基地與救援節點

用途：

- 給落後小隊保底行動。
- 讓受困小石可救回。
- 讓非衝榜玩家也能有效支援。

規則：

- 基地區收益較低，但穩定。
- 救援成功給小石救援分與協作分。
- 今日支援額度可在這裡發揮最大保底效果。

### 11.4 小隊協作事件

用途：

- 鼓勵小隊討論。
- 讓比較慢熱的學員也能貢獻。
- 讓排行榜不只看單一主力玩家。

規則：

- 同一活動窗口中，如果 2 名以上隊員對同一節點接力，給協作 bonus。
- 如果 3 名以上隊員在不同節點分工，給戰線分 bonus。
- 如果所有隊員都有至少一個貢獻片段，給回顧徽章。
- 不以答題速度排名。

### 11.5 世界事件

用途：

- 全營共同進度。
- Day 4 主要競爭高潮。
- Day 5 結算展示。

規則：

- 每個 Front 窗口成功都推進世界事件進度。
- 貢獻不是單純傷害，而是分類成：
  - 理解。
  - 修復。
  - 探索。
  - 協作。
  - 展示。
- 世界事件每個階段解鎖全營劇情、獎勵或回顧。

### 11.6 Day 5 社群攤位節點

用途：

- Day 5 才鼓勵逛攤。
- 把 QR 掃描與 finale 加成串起來。
- 產生社群探索榜與回顧素材。

規則：

- Day 1-4 不出現，不作為主循環。
- 小隊中有人拜訪過某社群攤位，可在 Day 5 使用一次社群支援卡。
- 拜訪越多不同攤位，支援種類越多，但 finale 只能選 1 到 2 個。
- 社群支援卡不能回頭改變 Day 1-4 排名。

## 12. MVP 範圍

### 12.1 MVP 必做

第一版只做能驗證核心樂趣的內容：

- 新增 Front 模式；玩家顯示名稱使用 `開源戰線`。
- 一張全營共享 territory map。
- 至少 20 到 40 個節點。
- 每個小隊有基地節點與顏色。
- 節點有 owner、control、defense、pressure、resource、event。
- Server authoritative tick，建議每 1 秒推進一次，重要倒數以 30 秒以上呈現。
- SSE 或 polling delta 更新地圖。
- 玩家可以送出 command：`擴張`、`攻擊`、`防守`、`修復`、`偵查`、`救援`。
- command 需要 cooldown、idempotency、team validation、adjacency validation。
- 小石先做 type-level bonus，不做每顆具名小石獨立技能。
- 前線開源力會在 Front 內消耗與散逸，不扣真實可用開源力。
- 今日支援 token：提示、加派、緊急修補。
- 至少一種節點會觸發知識題。
- 答題可加速戰線、修復或救援；答錯仍給部分情報。
- 即時小隊戰況榜：前 5 名、自己小隊排名、名次變化。
- Day 1 暖身榜可以不計入 finale，但必須存在。
- 結算寫入 open power record 與 front reward snapshot。
- 前端至少有 home 入口、Front 地圖頁、節點詳情、command 操作、題目 drawer、排行榜。
- Staff 最小控制：start/freeze/resume/complete front、調整流失倍率、發全營護盾、reset stuck command、reward status。

### 12.2 MVP 不做

第一版明確不做：

- 自由座標 RTS 單位移動。
- pathfinding。
- 單位碰撞。
- 毫秒級多人同步。
- 完整 fog of war。
- 複雜 AI。
- 個人對個人 PvP duel。
- 長線基地建設。
- 大量裝備。
- 每顆具名小石獨立技能。
- 複雜屬性相剋。
- 移除排行榜或只做無壓力排行榜。
- 全域經濟重寫作為 camp MVP 前置條件。
- Front 內真實可用開源力每秒消費。
- 完整 team pool、追趕補助、百分位回饋公式。
- shop tactical consumables。
- Day 1-4 社群攤位支援。
- Day 1-4 QR 任務。
- Day 5 前社群探索榜。
- 多 server instance 即時同步；MVP 先假設單一 Go instance。

### 12.3 MVP 成功標準

功能上：

- 玩家可以打開一張共享地圖，看到小隊顏色、節點、倒數、排行榜。
- 第一個有意義行動可在 60 秒內開始。
- 玩家送出 command 後，地圖會在數秒內有可見 pending / progress / result。
- 同一小隊多人 command 會共同影響 team state。
- Front 狀態可中斷恢復。
- 結算獎勵 idempotent，不重複發。
- 題目解析正常顯示。
- 今日支援 token 不會扣商店用開源力。
- 前線開源力耗盡會造成可感受的 setback。
- 小石受困可救回，不永久消失。
- 開源力獎勵可追蹤。
- Staff 可以 freeze，freeze 後不再流失資源或改變排名。

體驗上：

- 玩家不用讀長規則也知道要做什麼。
- 第一次玩之前不要求理解 loadout、完整經濟、世界事件。
- 玩家第一眼能看到排行榜或戰況榜。
- 玩家能感覺「現在不處理，節點 / 小石 / 開源力就會沒了」。
- 每個節點至少有兩個合理選擇，例如搶、守、修、偵查、放棄。
- 答錯後仍覺得有學到。
- 小石類型差異能被感受到。
- 至少有一次自發討論。
- 每位參與者能說出自己本場做了什麼。
- 結果畫面被記得是「我們在地圖上搶下/守住/救回了什麼」，並且知道小隊排名。

### 12.4 五天營隊解鎖節奏

因為營隊只有 5 天，玩法必須逐日解鎖，不能第一天就把完整規則丟給學員。

| 日期 | 開放內容 | 明確避免 |
| --- | --- | --- |
| Day 1 | Guided Front 教學、共享地圖只開基地周邊、系統自動選 3 顆小石、暖身戰況榜、前線開源力教學、1 個小石訊號倒數 | 正式 finale 排名、完整經濟名詞、QR 支援、社群攤位、複雜技能 |
| Day 2 | 正式小隊戰線、小隊排名、可選 3 顆小石、課程區與系統區節點、前線開源力警告、小石短暫受困 | 真實可用開源力即時消費、稀有技能、社群支援、完整公共基金 |
| Day 3 | 救援與協作接力、基地區保底、多人 command、協作榜、題目加速 | 社群攤位、QR 刷碼、公開個人最低貢獻、自由座標 RTS |
| Day 4 | 主要競爭日、surge 窗口、高價值中央節點、世界事件、限時榜、強排名壓力 | 新手才第一次看到核心詞彙、永久損失小石、不可暫停的全域流失 |
| Day 5 | 提前凍結主要排名、開社群攤位 booth window、社群探索榜、finale 支援卡、全營回顧與 highlights | 新核心規則、新商店/合成規則、回頭改 Day 1-4 戰績、公開最低貢獻 |

### 12.5 玩家詞彙表

第一天玩家只需要看到這些詞：

- `戰線`：全營共用的大地圖。
- `小隊`：你的營隊小隊。
- `小石`：幫你在地圖上行動的夥伴。
- `節點`：地圖上的格子或據點。
- `擴張`：往旁邊節點推進。
- `防守`：守住我們的節點。
- `修復`：處理 bug 或事故。
- `偵查`：看下一步會發生什麼。
- `救援`：救回受困小石。
- `前線開源力`：這個活動窗口能用的戰術資源。
- `支援`：有限次數的提示、加派或救急。
- `排行榜`：小隊目前名次。
- `題目`：幫你加速或修復的挑戰。
- `解析`：答完後看到的原因。
- `獎勵`：活動後得到的東西。

第一天不要顯示：

- `Operation`
- `loadout`
- `lane`
- `slot`
- `planning / challenge / resolution / upkeep`
- `今日戰術預算`
- `可用開源力`
- `終身貢獻`
- `小隊公共基金`
- `追趕補助`
- `回饋比例`
- `boss shield`
- `社群支援`
- `QR`

內部文件可以使用這些詞，但 UI 需要轉成玩家詞彙。

### 12.6 三分鐘第一場 Front 教學

第一場教學不解釋完整經濟、不要求玩家理解完整小石五型。目標只教會：

- 地圖上有自己小隊的顏色。
- 排行榜會動。
- 節點會倒數或衰退。
- 小石可以被派去節點。
- 前線開源力會花掉，快沒了要小心。
- 有些節點會出題。
- 答錯也能透過解析拿到線索或部分效果。
- 支援 token 是有限次數的幫忙。

設定：

- 小石：自動選 3 顆。
  - 工程型：修問題。
  - 靈光型：拿提示。
  - 共鳴型：一起討論。
- 前線開源力：60。
- 小隊節點：2 個基地周邊節點。
- 暖身榜：顯示自己小隊與其他小隊，但標記為暖身。
- 今日支援：提示 x1、加派 x1、緊急修補 x1。
- 地圖區：基地區、系統區、課程區。

教學步驟：

1. `看地圖`
   - 顯示自己小隊顏色與附近 3 個節點。
   - 告訴玩家：`旁邊這個節點快被 bug 影響了。`

2. `派小石修復`
   - 玩家點系統節點，選工程型小石。
   - 預覽：`花 15 前線開源力，修復進度 +1。`

3. `看到排行榜變動`
   - tick 後小隊分數 +5，暖身榜名次變動。
   - 顯示：`你們守住了一個節點。`

4. `處理小石訊號`
   - 地圖出現 3 分鐘倒數：`小石訊號快消失。`
   - 玩家可選探索/共鳴/加派。
   - 成功後取得小石碎片或救援分。

5. `答一題加速`
   - 課程節點觸發 container 題。
   - 答對：節點佔領加速。
   - 答錯：看解析後延長倒數，仍能繼續。

6. `前線開源力警告`
   - 前線開源力低於 20，UI 顯示警告。
   - 教玩家可以防守、撤退、救援或用支援 token。

7. `短結算`
   - 顯示暖身戰線分、救援、修復、學到的解析。
   - 不顯示完整經濟公式。

#### 教學結果頁

成功標題：

```text
暖身戰線完成
```

主摘要：

```text
你們派出小石守住 2 個節點、救回 1 個小石訊號，暖身戰況排名上升。
```

貢獻：

- `工程型小石修復了系統節點。`
- `靈光型小石幫助小隊理解 container 的判斷線索。`
- `共鳴型小石讓小隊完成一次接力支援。`

學習一句話：

```text
遇到部署問題時，先看現象、再縮小範圍；答錯也能透過解析找到下一步。
```

教學結果頁不要顯示：

- 開源力公式。
- 正式 finale 排名。
- 補助。
- 商店價值。
- 掉落率。
- 社群攤位或 QR。

## 13. 資料模型設計

### 13.1 新增 collection: `front_sessions`

用途：

- 保存一場全營 Front 活動窗口的權威狀態快照。
- Mongo 存 snapshot；即時 tick 狀態由 Go server in-memory manager 負責。

```go
type FrontSession struct {
    ID              string
    MapID           string
    Status          string // closed, quiet, open_play, surge, booth_window, finale_freeze, completed
    Tick            int64
    Revision        int64
    StartsAt        time.Time
    EndsAt          time.Time
    FrozenAt        time.Time
    CompletedAt     time.Time
    Cells           []FrontCell
    Teams           []FrontTeamState
    ActiveEvents    []FrontMapEvent
    Leaderboard     []FrontLeaderboardEntry
    CreatedAt       time.Time
    UpdatedAt       time.Time
}
```

### 13.2 `FrontCell`

```go
type FrontCell struct {
    ID             string
    X              int
    Y              int
    Terrain        string // base, course, system, neutral, center, community_day5
    Zone           string // base, course, system, frontier, community
    OwnerTeamID    string
    Control        int
    Defense        int
    Resource       int
    PressureByTeam map[string]int
    NeighborIDs    []string
    EventID        string
    LockedUntil    time.Time
}
```

### 13.3 `FrontTeamState`

```go
type FrontTeamState struct {
    TeamID             string
    Color              string
    Score              int
    Rank               int
    PreviousRank       int
    FrontOpenPower     int
    ControlledCells    int
    RescuedSitones     int
    RepairedEvents     int
    CollaborationScore int
    LastCommandAt      time.Time
}
```

### 13.4 新增 collection: `front_commands`

用途：

- 保存玩家意圖。
- 支援 idempotency、重播、除錯與 anti-spam。

```go
type FrontCommand struct {
    ID              string
    SessionID       string
    ClientCommandID string
    PlayerID        string
    TeamID          string
    Kind            string // expand, attack, reinforce, repair, scout, rescue, support, answer_challenge
    FromCellID      string
    ToCellID        string
    SitoneID        string
    Status          string // accepted, resolved, rejected
    RejectReason    string
    Cost            int
    ResolveTick     int64
    CreatedAt       time.Time
    ResolvedAt      time.Time
}
```

Idempotency：

- `(session_id, player_id, client_command_id)` unique。
- 同一 player command cooldown 由 server 驗證。
- 同一 sitone 在 command resolve 前不能重複派出。

### 13.5 新增 collection: `front_event_logs`

用途：

- SSE delta。
- 活動回顧。
- Staff 除錯。
- Finale highlight。

```go
type FrontEventLog struct {
    ID          string
    SessionID   string
    Tick        int64
    TeamID      string
    PlayerID    string
    CellID      string
    Kind        string // cell_captured, cell_lost, repaired, rescued, quiz_answered, resource_faded
    Message     string
    Payload     map[string]any
    CreatedAt   time.Time
}
```

### 13.6 新增 collection: `front_rewards`

用途：

- 保存活動窗口結算快照。
- 防止重複發獎。

```go
type FrontReward struct {
    ID              string
    SessionID       string
    PlayerID        string
    TeamID          string
    Rank            int
    ScoreSnapshot   int
    OpenPowerAmount int
    SitoneDrops     []FrontSitoneDrop
    ReviewSnippets  []string
    CreatedAt       time.Time
}
```

Reward ID 建議：

```text
front_reward_{sessionID}_{playerID}
```

### 13.7 `FrontMapTemplate`

內容檔而非 DB collection。

建議路徑：

- `server/content/front_maps.toml`

範例：

```toml
[[front_maps]]
id = "camp_day1_training"
name = "開源戰線暖身"
enabled = true
tick_seconds = 1

[[front_maps.cells]]
id = "base_team_1"
x = 0
y = 2
terrain = "base"
zone = "base"
neighbors = ["system_a", "course_a"]

[[front_maps.cells]]
id = "system_a"
x = 1
y = 2
terrain = "system"
zone = "system"
neighbors = ["base_team_1", "center_a"]

[[front_maps.cells]]
id = "course_a"
x = 1
y = 1
terrain = "course"
zone = "course"
neighbors = ["base_team_1", "center_a"]

[[front_maps.events]]
id = "evt_config_bug"
cell_id = "system_a"
kind = "repair"
title = "設定檔讀不到"
starts_at_tick = 30
expires_after_ticks = 300
severity = 1
```

Day 5 社群節點應放在獨立 map variant 或 event wave，不要出現在 Day 1-4 map。

### 13.8 Operation 輔助模型

Operation 仍可存在，但只作為 Front 的短遭遇或教學卡，不是第一主線。

保留建議：

- `operations`
- `operation_actions`
- `operation_rewards`
- `operations.toml`

使用情境：

- Day 1 guided popup。
- 特殊節點短挑戰。
- 小石救援遭遇。
- Day 5 社群攤位短事件。

Operation 不應擁有主要排行榜或主要 Front state。

### 13.9 全域開源力帳本調整

目前專案已經用 `open_power_records` 作為 ledger，商店購買也會寫入負數紀錄。這個方向可以保留，但需要把「這筆開源力屬於哪個帳戶」記清楚，否則無法同時支援終身貢獻、可用餘額、每日戰術預算與小隊公共基金。

建議擴充 `OpenPowerRecord`：

```go
type OpenPowerRecord struct {
    ID        string
    PlayerID  string
    TeamID    string
    Amount    int
    Reason    string
    Source    string
    Account   string // lifetime, spendable, tactical_budget, frontline, team_pool
    Direction string // earn, spend, transfer, subsidy
    ExpiresAt time.Time
    CreatedAt time.Time
}
```

帳戶規則：

- `lifetime`：只寫正向貢獻，不被 shop purchase 扣除。
- `spendable`：商店與合成消費使用，可正可負。
- `tactical_budget`：每日提示、撤回、Front / Operation 支援使用，可設 `expires_at`。
- `frontline`：Front session 內快照或結算使用，不可直接轉入 spendable。
- `team_pool`：小隊公共基金，通常使用 `team_id`，必要時也可記錄貢獻者 `player_id`。

短期相容方案：

- 舊資料沒有 `account` 時，視為 `spendable` 與 `lifetime` 的歷史來源。
- 新增 helper 時先提供 `TotalSpendableForPlayer`、`LifetimeForPlayer`、`TacticalBudgetForPlayer`、`TeamPoolForTeam`。
- Shop、fusion、Front / Operation 消費只查 `spendable + tactical_budget` 中允許的帳戶。
- Leaderboard 改查 `lifetime` 或多面向統計，不直接查 spendable。

### 13.10 新增 collection: `open_power_daily_limits`

用途：

- 記錄每日高額獎勵、每日戰術預算、每日商店補助使用量。

建議欄位：

```go
type OpenPowerDailyLimit struct {
    ID                   string
    PlayerID             string
    TeamID               string
    Date                 string // Asia/Taipei YYYY-MM-DD
    QuizRewardEarned     int
    FrontRewardEarned     int
    OperationRewardEarned int
    TacticalBudgetGranted int
    TacticalBudgetSpent   int
    ShopSubsidyUsed       int
    CatchUpBonusGranted   int
    CreatedAt             time.Time
    UpdatedAt             time.Time
}
```

用途範例：

- 知識王每日前三場完整獎勵。
- Front / Operation 每日高額獎勵。
- 每日提示/撤回預算。
- 小隊公共基金每日補助上限。

### 13.11 新增 collection: `team_open_power_pools`

用途：

- 儲存小隊公共基金快取與設定。
- ledger 仍以 `open_power_records` 為準，這個 collection 可作查詢快取。

建議欄位：

```go
type TeamOpenPowerPool struct {
    TeamID            string
    Balance           int
    LifetimeContributed int
    LifetimeSpent     int
    UpdatedAt         time.Time
}
```

小隊基金用途：

- Front / Operation 共同提示。
- 幫低資源隊友折抵基礎合成素材。
- 小隊任務入場或共同解鎖。
- Day 5 小隊回顧展示。

### 13.12 經濟 helper package

建議新增：

```text
server/internal/economy/
```

負責：

- 計算 lifetime / spendable / tactical / team pool。
- 寫入獎勵與扣款 ledger。
- 套用每日上限。
- 套用追趕補助。
- 套用公共基金回饋。
- 提供商店、知識王、Operation、Staff reward 共用 API。

不要讓各 handler 自己算補助與餘額，否則規則會分裂。

## 14. 後端 API 設計

### 14.1 路由總覽

新增 namespace：

```text
GET    /api/fronts/current
GET    /api/fronts/{frontID}
GET    /api/fronts/{frontID}/events
GET    /api/fronts/{frontID}/leaderboard
POST   /api/fronts/{frontID}/commands
POST   /api/fronts/{frontID}/answers
GET    /api/fronts/{frontID}/logs

POST   /api/admin/fronts
POST   /api/admin/fronts/{frontID}/freeze
POST   /api/admin/fronts/{frontID}/resume
POST   /api/admin/fronts/{frontID}/complete
POST   /api/admin/fronts/{frontID}/shield
POST   /api/admin/fronts/{frontID}/flow-rate
POST   /api/admin/fronts/{frontID}/reset-stuck
```

SSE 是即時感的核心：

- `GET /api/fronts/{frontID}/events` 會在領土、排行榜、駐點與交易狀態變化時推送完整個人化 snapshot，並每 20 秒送 keepalive。
- 前端不做固定輪詢；SSE 斷線時先抓一次 snapshot，再以 exponential backoff 重連，仍保留手動重新整理。

### 14.1.1 駐點與自動交易

- 每格最多一個駐點，每個駐點包含 1 到 5 顆小石；永久基地不可駐點。
- 駐點時小石離開一般庫存，撤回時歸還；失守時由攻擊玩家取得，相關進行中交易立即取消。
- 不同小隊的駐點會自動建立交易路線，依 Manhattan distance 在 10 到 60 秒抵達；距離與駐點小石加成提高收益。
- 抵達時來源與目的玩家都取得開源力，雙方小隊同時取得等量交易戰線分。
- 每隊每個 UTC 整點時窗最多取得 300 點交易開源力與 300 點交易戰線分；達上限後暫停新路線，下個整點恢復。

### 14.2 `GET /api/fronts/current`

用途：

- 取得目前 active 或最近一個 Front session。
- Home 入口判斷要顯示「開源戰線進行中」、「已凍結」、「尚未開放」。

Response：

```json
{
  "front": {
    "id": "front_day2_window_1",
    "mapId": "camp_day2_main",
    "status": "open_play",
    "tick": 128,
    "startsAt": "2026-07-09T10:00:00+08:00",
    "endsAt": "2026-07-09T10:30:00+08:00"
  }
}
```

### 14.3 `GET /api/fronts/{frontID}`

用途：

- 取得完整 Front snapshot。

Response 需要包含：

- session 基本資訊。
- server time。
- cells。
- teams。
- active events。
- leaderboard。
- player cooldowns。
- selected / available sitones。
- current team front open power。
- visible support tokens。

### 14.4 `GET /api/fronts/{frontID}/events`

用途：

- SSE delta stream。
- 推送 tick、cell updates、leaderboard updates、event logs。

Event 範例：

```json
{
  "type": "cell_updated",
  "tick": 129,
  "cells": [
    {
      "id": "system_a",
      "ownerTeamId": "team_1",
      "control": 72,
      "defense": 18,
      "pressureByTeam": { "team_2": 24 }
    }
  ]
}
```

### 14.5 `GET /api/fronts/{frontID}/leaderboard`

用途：

- 取得小隊戰況榜。
- Home、Front 頁、finale display 共用。

Response：

```json
{
  "updatedAtTick": 129,
  "entries": [
    {
      "teamId": "team_1",
      "rank": 1,
      "previousRank": 2,
      "score": 320,
      "controlledCells": 8,
      "rescuedSitones": 2,
      "repairedEvents": 5,
      "collaborationScore": 40
    }
  ],
  "myTeamRank": 3
}
```

### 14.6 `POST /api/fronts/{frontID}/commands`

用途：

- 玩家送出 Front command。

Request：

```json
{
  "clientCommandId": "uuid-from-client",
  "kind": "attack",
  "fromCellId": "system_a",
  "toCellId": "center_a",
  "sitoneId": "stone_engineering_base"
}
```

Server 行為：

- 驗證 front status 可操作。
- 驗證玩家登入與 team。
- 驗證 clientCommandId idempotency。
- 驗證 from/to cell 相鄰。
- 驗證 from cell 可由該隊發起。
- 驗證小石屬於玩家且未在 cooldown。
- 驗證前線開源力或今日支援額度足夠。
- 寫入 `front_commands`。
- 回傳 accepted command 與預估 resolve tick。

### 14.7 `POST /api/fronts/{frontID}/answers`

用途：

- 回答 Front 節點 challenge。

Request：

```json
{
  "cellId": "course_a",
  "questionId": "quiz-001",
  "choice": "A"
}
```

Server 行為：

- 驗證該 cell 有 active challenge。
- 驗證玩家 team 可回答或支援該 cell。
- 計算正誤。
- 寫入 event log。
- 答對套用加速、修復、防守或救援效果。
- 答錯套用部分情報、延長倒數或降低傷害。

### 14.8 Admin API

Staff 必須能控制節奏：

- `POST /api/admin/fronts`：建立或啟動 Front session。
- `freeze`：停止 tick 造成的 owner/score/resource 變化。
- `resume`：從 freeze 恢復。
- `complete`：結算、寫 reward、產生 highlights。
- `shield`：發全營護盾，指定分鐘內不流失。
- `flow-rate`：調整前線開源力散逸、節點衰退、小石受困倍率。
- `reset-stuck`：重置卡住 command 或受困小石。

所有 admin action 都要寫 audit log。

### 14.9 API idempotency

所有 command 與 reward 寫入必須 idempotent。

建議 ID：

- Front command：`front_command_{frontID}_{playerID}_{clientCommandID}`。
- Open power reward record：`open_power_reward_front_{frontID}_{playerID}`。
- Front reward：`front_reward_{frontID}_{playerID}`。
- Sitone drop source：`front:{frontID}:player:{playerID}`。
- Operation reward：`operation_reward_{operationID}_{playerID}`，僅短遭遇使用。

### 14.10 全域經濟 API

除了 Operation API，也需要讓原本遊戲頁面能讀到新的經濟狀態。

建議短期擴充：

```text
GET /api/me/status
GET /api/me/home
```

在既有 response 裡新增：

```json
{
  "economy": {
    "lifetimeOpenPower": 1280,
    "spendableOpenPower": 420,
    "todayTacticalBudget": 80,
    "todayTacticalBudgetLimit": 100,
    "teamOpenPowerPool": 360,
    "catchUpActive": true,
    "shopSubsidyAvailable": 60
  }
}
```

若不想讓 `status` response 變大，可新增：

```text
GET /api/me/economy
```

用途：

- Home 顯示四種資源。
- Shop 判斷可用餘額與補助。
- Fusion 顯示小隊基金可折抵多少素材。
- Front / Battle result 顯示個人獎勵與回饋到小隊基金的數量。
- Staff 發獎頁顯示推薦獎勵類型。

### 14.11 既有 API 需要改的地方

#### Shop

`POST /api/shop/purchases` 應改用 economy helper：

- 查 `spendable_open_power`，不是查所有 open power 總和。
- 若有 `shopSubsidyAvailable`，可折抵基礎成長品。
- 購買收藏品不使用補助。
- 回傳購買後的 `spendableOpenPower`，不是混合總額。

#### Matches / 知識王

知識王結算應改用 economy helper：

- 寫入 lifetime 貢獻。
- 寫入 spendable 獎勵。
- 套用每日高額獎勵上限。
- 套用追趕補助。
- 高資源玩家新增收益的一部分寫入 team pool。

#### Staff Rewards

Staff 發開源力時應明確選擇帳戶：

- `spendable`：一般可用開源力。
- `tactical_budget`：今日任務支援額度。
- `lifetime_only`：只記錄貢獻或表揚，不增加可花餘額。

Staff UI 應預設推薦低資源玩家拿 `tactical_budget` 或基礎素材，高資源玩家拿紀念品或 lifetime-only 表揚。

#### Community Stands

社群攤位 claim 只在 Day 5 booth window 接入 Front / finale。claim 應可設定：

- 固定道具或小石。
- spendable open power。
- tactical budget。
- team pool contribution。
- catch-up only reward。

#### Leaderboard

Leaderboard 不應再用可花開源力當核心排序。Front leaderboard 應以小隊戰線分為主；一般 leaderboard response 可補充顯示：

- `lifetimeOpenPower`。
- `sitoneCount`。
- `communityVisitCount`，Day 5 後才有意義。
- `learningProgress`。
- `teamContribution`。

排序規則應以 Front 小隊戰線分或多指標 score 為主，而不是可花餘額。

## 15. 後端實作位置

### 15.1 新增 package

建議新增：

```text
server/internal/economy/
server/internal/http/handler/fronts/
server/internal/front/
server/internal/content/front_map.go
server/internal/mongodb/model/front.go
```

分工：

- `economy/`：開源力帳戶、每日上限、追趕補助、公共基金、可用餘額計算。
- `content/front_map.go`：讀取 `front_maps.toml`。
- `front/`：純規則計算、tick engine、command resolution，不碰 HTTP。
- `handler/fronts/`：HTTP decode、auth、DB 操作、SSE response。
- `mongodb/model/front.go`：Mongo model。
- `operation/` 與 `handler/operations/` 可延後，只有短遭遇或教學卡需要時再做。

### 15.2 規則邏輯要可單元測試

避免把所有規則寫在 handler 裡。

建議純函式：

- `ValidateFrontMap(template)`.
- `AvailableCommands(state, player, team, sitones)`.
- `ValidateCommand(state, command)`.
- `ApplyCommand(state, command)`.
- `AdvanceTick(state, commands, now)`.
- `ApplyAnswer(state, answer)`.
- `BuildFrontRewards(state)`.
- `BuildLeaderboard(state)`.

## 16. 前端設計

### 16.1 新增路由

建議路由：

```text
/front
/front/$frontId
/front/$frontId/leaderboard
/front/$frontId/result
```

或中文現有 nav 可以顯示：

- 開源戰線。
- 地圖。
- 排行榜。
- 小石。
- 戰報。

### 16.2 第一次遊玩流程

第一次遊玩最多 5 個畫面，不顯示規則頁。

1. Home 入口卡：`開源戰線`
   - 文案：`全營小隊正在搶地圖，派小石守住你們的節點。`
   - 顯示：目前小隊排名、活動剩餘時間。
   - CTA：`進入戰線`
2. Guided overlay：`這是你們小隊的顏色`
   - 指向基地節點。
   - 不開規則頁。
3. 選一個推薦節點
   - 系統自動高亮：`這個節點快出問題了。`
   - 顯示倒數、預期分數、可能 setback。
4. 派小石
   - 自動推薦 3 顆小石。
   - 玩家點 `修復` 或 `防守`。
   - 顯示前線開源力花費。
5. 看結果與排行榜
   - pending arrow / progress bar。
   - tick 後節點變色或分數增加。
   - 排行榜名次或分數變化。

如果有題目，用 bottom sheet 開啟，不離開地圖。

### 16.3 Front 地圖頁

布局：

- 上方：活動狀態、剩餘時間、前線開源力、今日支援。
- 主區：territory map。
- 右側或 bottom sheet：節點詳情。
- 底部：小石 command toolbar。
- 側欄或 drawer：排行榜。

地圖渲染：

- MVP 用 SVG 或 Canvas。
- 不用大量 React cell DOM。
- cell 顏色代表 owner team。
- cell border / glow 顯示 contested、倒數、事件、受困小石。
- command pending 顯示箭頭與進度條。
- SSE delta 更新 cell，client 做動畫 interpolation。

### 16.4 節點詳情

點節點後顯示：

- 節點名稱。
- owner。
- control。
- defense。
- 敵方 pressure。
- 倒數或事件。
- 可用 command。
- 推薦小石。
- 預估花費。
- 預期結果。

CTA：

- `擴張`
- `攻擊`
- `防守`
- `修復`
- `偵查`
- `救援`

### 16.5 小石 command toolbar

要求：

- 顯示 3 到 5 顆可用小石。
- 顯示小石類型 icon 與短動詞。
- 顯示 cooldown / 受困 / 行動中。
- Day 1 自動選 3 顆推薦小石。
- Day 2 後允許玩家調整前線小石。

不要做：

- 複雜技能樹。
- 每顆具名小石一堆主動技能。
- 第一畫面塞滿數值公式。

### 16.6 答題挑戰

可復用現有 battle question 的視覺語言，但要改成 Front 情境：

- 顯示節點名稱。
- 顯示這題如何影響戰線。
- 顯示小石提示或今日支援提示。
- 答完顯示解析與 Front 效果：加速、修復、防守、救援或情報。

### 16.7 排行榜與戰報

排行榜必須是高可見度 UI：

- Front 頁常駐顯示前 3 或前 5。
- 玩家可展開完整小隊榜。
- 顯示自己小隊排名與上一輪名次變化。
- Day 1 顯示暖身榜。
- Day 2 起顯示正式榜。
- Day 4 surge 顯示限時榜。
- Day 5 freeze 後改為 finale 戰報。

戰報要比單純分數更有感：

- 佔領了哪些節點。
- 守住了哪些節點。
- 救回哪些小石。
- 哪次答題造成反打。
- 小隊排名變化。
- 獲得小石或道具。
- 開源力收支放在 `明細`，第一場教學結果不顯示公式。

## 17. Content 設計

### 17.1 第一批三張 Front map / event wave

#### Front 1：開源戰線暖身

用途：

- Day 1 教學。
- 只開基地區、系統區、課程區少量節點。
- 暖身榜不計入 finale 正式排名，但要讓玩家看到競爭。

節點：

- 每隊基地節點。
- 1 到 2 個系統修復節點。
- 1 個課程題目節點。
- 1 個小石訊號節點。
- 1 個前線開源力訊號節點。

教學目的：

- 看懂地圖顏色。
- 派小石。
- 看排行榜變動。
- 知道前線開源力會花掉。
- 知道小石訊號會倒數。

#### Front 2：部署與系統戰線

用途：

- Day 2-3 正式小隊戰線。
- 讓系統區與課程區成為主要競爭點。
- 開始有小石短暫受困與救援。

核心事件：

- Docker / container 概念題。
- Linux 權限 bug。
- 測試失敗。
- 系統文件缺漏。
- 中央 contested 節點。

主要分數來源：

- 節點控制。
- 修復事件。
- 小石救援。
- 課程題目加速。
- 同隊接力。

#### Front 3：AI / 資安 surge 戰線

用途：

- Day 4 主要競爭日。
- 開放高價值中央節點。
- 排行榜壓力最大。

核心事件：

- Prompt Injection 題。
- AI 回答判斷題。
- 人類標籤事件。
- 錯誤提示污染。
- 權限與攻防觀念題。
- 世界事件壓力。

主要目標：

- 在 surge window 中搶下中央節點。
- 守住已佔節點不被事件破壞。
- 救援被困小石，避免戰線失速。

教學目的：

- AI 使用判斷、prompt 安全、人機協作。
- 資安與系統防守概念。

#### Day 5：社群攤位 finale wave

用途：

- Day 5 才開。
- 不回頭改 Day 1-4 排名。
- 作為 finale 支援、社群探索榜、回顧素材。

核心事件：

- 攤位拜訪。
- 開源授權問題。
- 社群介紹。
- 共筆分享。
- 帶回一個「我問到的問題」或「我學到的一件事」。

主要目標：

- 收集社群 highlight 與 finale 支援卡。

教學目的：

- 鼓勵逛攤、問問題、了解開源社群。
- 避免把社群攤位變成單純掃 QR 任務。

攤位互動規則：

- 每個攤位提供 3 個簡單提問 prompt，讓害羞學員也能開口。
- 允許兩人同行完成拜訪。
- 掃 QR 只代表完成記錄，不代表互動本身。
- 學員可用一句短反思生成 `社群支援卡`，例如「我今天問了 g0v 如何開始參與專案」。
- 同一攤位每人只能領一次，並需有 cooldown 與 fallback code。

### 17.2 Front 事件模板

事件類型：

- `bug`：若不處理會降低節點 defense 或 control。
- `resource_signal`：前線開源力訊號，倒數後散逸。
- `sitone_signal`：小石訊號，倒數後消失或變受困。
- `question`：答題後加速佔領、修復或救援。
- `world_event`：全營事件，Day 4-5 使用。
- `community_day5`：Day 5 攤位或社群支援事件。

### 17.3 文案原則

文案要短、具體、可行動。

好：

- `部署後服務無法讀取設定檔。5 分鐘後此節點防守 -10。`
- `小石訊號剩 3 分鐘。派探索型小石可延長倒數。`
- `有人把 container 和 VM 混在一起。答對可讓佔領速度 +20%。`

避免：

- 大段世界觀。
- 抽象形容。
- 需要讀很久才知道要做什麼。

## 18. 獎勵與平衡

### 18.1 結算公式建議

Front MVP：

```text
open_power_reward =
  participation_base
  + rank_bonus
  + controlled_cells_bonus
  + rescued_sitones_bonus
  + repaired_events_bonus
  + explanation_read_bonus
  + collaboration_bonus
```

建議數值：

- 參與基礎：10。
- 排名獎勵：第一名 30、第二名 20、第三名 15、其餘 8。
- 控制節點：每個 3，最多 30。
- 小石救援：每次 8，最多 24。
- 事件修復：每次 5，最多 25。
- 閱讀解析：5。
- 小隊多人接力：每次 4，最多 20。
- 每日 Front / Operation 可用開源力總上限：120。

### 18.2 小石掉落

不要讓 Front 掉落太多高階小石。

建議：

- 成功救援小石訊號：基礎小石碎片或素材。
- 探索型貢獻：提高小石訊號成功率。
- 每日首次參與 Front：保底一個基礎素材。
- 高階小石仍主要透過合成或 Staff 發放。

### 18.3 防刷機制

必要限制：

- 每日 Front / Operation 高額獎勵上限。
- 同一活動窗口只結算一次主要 reward。
- 同一題反覆出現不再給完整學習獎勵。
- 每窗口今日支援投入上限。
- 前線開源力不可帶出窗口。
- 小隊 bonus 要看不同玩家貢獻，不看同一人刷。
- command cooldown，避免單一玩家洗指令。

### 18.4 追趕機制

避免落後者追不上：

- 每日首次 Front 有固定參與獎勵。
- 前線開源力低於 20 時，基地區提供低收益保底行動。
- 低資源玩家可獲得今日支援額度。
- 小隊戰線允許低資源玩家貢獻非答題行動：防守、偵查、救援。
- 失敗仍給解析與少量進度。

## 19. 小隊協作與競爭

### 19.1 小隊協作是 MVP 核心

Front 主模式本質就是小隊競爭，因此小隊協作不能延後。

MVP 規則：

- 同隊所有成員都可以打開同一張 Front 地圖。
- 每位玩家有 command cooldown，避免單人洗指令。
- 同隊多人對同一節點接力會得到協作 bonus。
- 同隊多人分工不同節點會得到戰線 coverage bonus。
- 小隊排名公開顯示。
- 個人最低貢獻、答錯次數、害掉節點者不公開。

### 19.2 競爭規則

競爭要足夠明顯：

- 小隊排名常駐。
- 節點顏色可見。
- contested 節點可見。
- 前 5 名與自己小隊名次可見。
- Day 4 surge 可顯示限時榜與名次變化。

競爭不應變成：

- 高資源玩家直接買勝利。
- 睡覺或課程期間被扣到崩盤。
- 個人被公開羞辱。
- 小石永久消失。

### 19.3 非同步即時設計

不需要所有人同時在線，也不需要毫秒同步。

流程：

1. 玩家打開 Front 地圖。
2. 玩家送出 command。
3. Server 接受 command，顯示 pending。
4. Tick engine 彙整同隊與敵隊 command。
5. SSE 推送地圖與排行榜 delta。
6. 玩家看到戰線變化後再決定下一步。

### 19.4 小隊貢獻顯示

戰報同時顯示排名與貢獻類型：

- 小隊總排名。
- 節點控制。
- 小石救援。
- 系統修復。
- 探索情報。
- 協作接力。
- Day 5 社群探索。

排名讓遊戲有壓力；貢獻類型讓不同玩家有不同發揮空間。

## 20. 世界事件與 Day 5 Finale

### 20.1 世界事件是地圖壓力，不是單人 boss

世界事件應是全營共同面對的 Front 壓力。

Front 對世界事件的貢獻分為：

- `understanding`：答題與閱讀解析。
- `repair`：修 bug。
- `exploration`：探索訊號與節點。
- `collaboration`：同隊接力與分工。
- `rescue`：小石救援。
- `community`：Day 5 社群攤位支援。

### 20.2 世界事件階段

範例：

1. `迷霧階段`：需要探索型貢獻，揭露高價值節點。
2. `干擾階段`：需要靈光型與工程型貢獻，解題與修復並重。
3. `戰線階段`：Day 4 surge，中央節點高壓競爭。
4. `社群階段`：Day 5 booth window，社群攤位支援加入 finale。
5. `展示階段`：凍結排名，生成 highlights。

### 20.3 Day 5 結算

Day 5 可以顯示：

- 全營共同修復了多少 bug。
- 看過多少題解析。
- 多少隊伍有跨成員貢獻。
- 哪些社群攤位被拜訪。
- 每隊的代表性貢獻。

Day 5 不應顯示：

- 單純傷害榜。
- 誰最少貢獻。
- 哪隊落後最多。
- 讓低參與隊伍被公開點名的資料。

### 20.4 Camp Mode 控制

需要一個全域 camp mode，讓 Staff 控制遊戲何時能做什麼。

建議狀態：

| 模式 | 用途 | 系統行為 |
| --- | --- | --- |
| `closed` | 課程中、維護中 | 禁止 Front command、停止流失、禁止 QR claim、保留讀取 |
| `quiet` | 課程或演講中 | 允許查看地圖/背包/回顧，Front 低頻或停止流失 |
| `open_play` | 自由時間 | 允許 Front command、知識王、商店、合成 |
| `surge` | Day 4 主要競爭窗口 | 開啟高價值節點、限時榜、較高流失倍率 |
| `booth_window` | 社群攤位時間 | 允許攤位 QR claim 與社群支援卡 |
| `finale_freeze` | Day 5 結算前 | 凍結會影響成果的寫入，只允許展示與修正 |

這個控制應該由 admin 面板設定，不需要重新部署。

### 20.5 Staff Dashboard 與 Kill Switch

Staff 需要一個低負擔的現場控制面板。

必要資訊：

- 目前 camp mode。
- 目前 Front session。
- Front tick 狀態。
- SSE 連線數或 polling 狀態。
- 卡住超過 5 分鐘的 Front command。
- 受困小石數。
- reward write status。
- QR claim counts by booth，Day 5 才需要。
- team contribution coverage。
- 小隊排名與異常分數變化。
- 前線開源力流失倍率。
- 每日 cap hit 數量。
- economy ledger write error。

必要 kill switch：

- 暫停 Front tick。
- 凍結排行榜。
- 關閉單一 Front map area。
- 關閉單一 Front event。
- 關閉單一 Operation template，僅短遭遇需要。
- 關閉單一題目。
- 關閉單一 reward type。
- 關閉單一社群攤位 QR claim，Day 5 才需要。
- 暫停 shop / fusion / battle opening。

### 20.6 QR 與社群攤位現場規則

QR 不應變成排隊刷碼。

規則：

- 每位學員每攤只能 claim 一次。
- QR token 有短效期限。
- 攤位頁顯示 `available` / `busy` / `closed`。
- 每攤提供 fallback code，給相機失敗或網路不穩時使用。
- 高峰時可以由攤位夥伴或 Staff 開啟 cooldown。
- 社群支援卡要來自一次問題、回答或短反思，不只掃 QR。

### 20.7 Day 5 Freeze 與 Finale

Day 5 應先凍結，再展示。

建議流程：

1. 結算前 1 到 2 小時進入 `finale_freeze`。
2. Staff preview finale screen。
3. Staff 可隱藏錯誤資料、補救 reward 寫入失敗、挑選代表性 highlights。
4. Finale 顯示全營分類貢獻：理解、修復、探索、協作、社群、展示。
5. 每隊至少有一個代表性片段，不用總排名決定曝光。
6. 匯出靜態 fallback summary，避免現場網路或投影出問題。

## 21. 實作里程碑

里程碑分成兩條軌：

- Camp MVP track：優先讓 5 天營隊能快速上手、好玩、可控。
- Economy refactor track：逐步導入財富平均，但不阻塞第一場小石任務。

若時間不足，先完成 Front Camp MVP track。全域經濟改造可以先做 read-only model 或延後，不能讓第一張 playable shared map 被帳戶拆分拖住。

### Phase 0：規格確認

目標：

- 確認 Front MVP 範圍。
- 確認第一張地圖與節點數。
- 確認 Day 1 暖身榜與 Day 2 正式榜規則。
- 確認小石五型能力。
- 確認前線開源力消費與結算數值。
- 確認全域財富平均規則。
- 確認終身貢獻、可用開源力、每日戰術預算、小隊公共基金的 UI 命名。
- 確認社群攤位只在 Day 5 接入。

產出：

- `Plan.md`。
- `front_maps.toml` 草案。
- Front API response 草案。
- 經濟帳戶 migration 草案。

### Phase 1A：Front 內容與規則核心

任務：

- 新增 `server/internal/content/front_map.go`。
- 新增 `server/content/front_maps.toml`。
- 新增 front map template validation。
- 新增 `server/internal/front/` 純規則 package。
- 做第一張 map：`camp_day1_training`。
- 做 Front state、cell、team state、event wave。
- 寫單元測試。

完成標準：

- server 啟動時可讀 front map templates。
- 無效 map 會報錯。
- 純函式可以建立初始 Front state。
- 純函式可以處理 tick、節點衰退、前線開源力散逸。
- 答錯能產生 partial pressure / clue / reduced damage。
- 今日支援 token 不依賴真實 open power。

### Phase 1B：Front 持久化、tick 與 API

任務：

- 新增 `FrontSession` model。
- 新增 `FrontCommand` model。
- 新增 `FrontEventLog` model。
- 新增 `FrontReward` model。
- 新增 FrontSessionManager。
- 新增 server tick loop。
- 新增 handler/fronts。
- 註冊 routes。
- 新增 current/get/events/leaderboard/commands/answers。
- 新增 admin start/freeze/resume/complete/shield/flow-rate/reset-stuck。
- reward 使用 deterministic ID，確保重試不重複發獎。

完成標準：

- API 可取得 current Front。
- 地圖會隨 tick 更新。
- 可提交 command。
- 可答題。
- SSE 或 polling 可看到 cell 與 leaderboard 變化。
- refresh 後可恢復。
- command idempotent。
- reward 重複呼叫不重複發。
- freeze 後 tick 不再改變排名或資源。

### Phase 1C：Front 前端

任務：

- 新增 fronts API client schema。
- 新增 routes。
- 新增 Home 入口卡。
- 新增 Front map page。
- 新增 territory map render。
- 新增 node detail drawer。
- 新增 command toolbar。
- 新增 leaderboard panel。
- 新增 challenge bottom sheet。
- 新增 result / battle report UI。
- 新增 staff 最小控制：start/freeze/resume/complete、flow rate、shield、reward status。

完成標準：

- 玩家 60 秒內能開始第一個有意義行動。
- 手機寬度可用。
- 第一場不顯示完整經濟術語。
- 地圖、倒數、排行榜第一眼可見。
- 玩家 command 後能看到 pending 與結果。
- 戰報主軸是搶下/守住/救回了什麼，以及排名變化。

### Phase 1D：Economy read model

這條軌可以並行，但不阻塞 Camp MVP。

Phase 1A-0：只讀模型，無行為變更。

- 新增 `server/internal/economy/`。
- 新增 `economy.Balances(playerID)`，先從現有 ledger 推導：
  - `currentNet`：相容既有 openPower。
  - `lifetimePositive`：只算正向貢獻，不算 shop/fusion 負數。
  - `spendable`：第一版等同現有 net，後續再改。
  - `tacticalBudget`：可先回傳 0 或固定測試值。
- `/api/me/home` 或 `/api/me/economy` 新增 additive 欄位。
- 既有 `openPower` response 不移除、不改名，避免前端 Zod schema 與既有頁面一次壞掉。
- 加 legacy ledger 測試，確保舊的負數消費不會被算進 lifetime。

Phase 1D-1：帳戶欄位與索引。

- `OpenPowerRecord` 新增 optional `account`、`direction`、`team_id`、`expires_at`。
- 舊資料沒有 `account` 時視為 legacy。
- 新增 index：
  - `(player_id, account)`
  - `(team_id, account)`
  - `(account, reason, source, player_id)`
  - daily limit 相關 unique index。
- 不要立刻 dual-write lifetime/spendable，避免舊 reader raw sum 雙算。

Phase 1D-2：逐個 writer 遷移。

- 新增可用餘額、終身貢獻、每日戰術預算、小隊公共基金查詢 helper。
- Shop purchase 改查 spendable balance。
- Match reward 改走 economy helper。
- Staff reward 支援不同開源力帳戶。
- Community stand reward 支援指定帳戶，但只在 Day 5 booth window 接入。
- Front leaderboard 使用戰線分；既有 leaderboard 先保留排序，另顯示 lifetime/spendable；排序改版延後到資料穩定後。

完成標準：

- 不改行為的 economy read model 已上線。
- 玩家 home 能同時顯示累積貢獻與可用開源力。
- 商店扣款不會降低終身貢獻值。
- 知識王獎勵會套用每日上限與公共基金回饋。
- 低資源玩家能取得每日戰術預算或補助。
- 現有測試更新後通過。

### Phase 2：內容擴充與 Day 2-4 玩法

任務：

- 補 2 到 3 張 Front map variant 或 event wave。
- 補題目 metadata overlay。
- Day 2 解鎖正式小隊戰線與正式榜。
- Day 3 解鎖救援與協作接力。
- Day 4 解鎖 surge、中央高價值節點、世界事件。

完成標準：

- 每張 map 至少 20 到 40 個節點。
- 每個 event wave 至少 4 個事件。
- 每個主要主題至少 1 到 2 題可用 quiz。
- 不同小石類型都有用。
- 排行榜可產生明顯名次變化。

### Phase 3：營隊現場控制

任務：

- Camp mode。
- Staff runbook。
- freeze/resume Front。
- flow-rate control。
- all-camp shield。
- stuck-command reset。
- QR fallback / cooldown，Day 5 才需要。
- Day 5 freeze / static summary export。

### Phase 4：更完整的小隊協作

MVP 已有多人 command；這階段補更完整的協作體驗。

任務：

- 更細的多人貢獻統計。
- team strategy ping / 標記目標。
- 隊長或 driver 指派建議。
- 小隊貢獻結算。
- SSE 或 polling 更新。

完成標準：

- 同隊多人可共同推進 Front。
- 結算顯示不同隊員貢獻。
- 不要求所有人同時在線。

### Phase 5：世界事件 / Finale

任務：

- 全營 world event progress collection。
- Front reward 寫入 world event contribution。
- world event progress API。
- finale display page。
- Day 5 社群攤位接入。
- Day 5 結算展示。

完成標準：

- 每個 Front window 會推進全營進度。
- 管理員可查看各分類貢獻。
- 前端可顯示全營共同進度。

## 22. 測試計畫

### 22.1 後端單元測試

需要測：

- front map validation。
- initial front state build。
- command validation：team、adjacency、cooldown、sitone ownership。
- command idempotency。
- tick advance。
- cell ownership change。
- pressure / defense / control calculation。
- front open power spend / warning / depletion。
- sitone signal countdown / rescue / trapped state。
- answer correct / incorrect。
- leaderboard calculation。
- reward idempotency。
- freeze mode：tick 不改排名、不扣資源。
- lifetime / spendable / tactical budget 分開計算。
- shop purchase 只扣 spendable，不扣 lifetime。
- daily tactical budget 補足與過期。
- catch-up bonus 不會讓玩家超過補助門檻。
- high percentile reward split 會寫入 team pool。
- daily reward cap。
- economy helper idempotency。

### 22.2 後端 integration test

需要測：

- get current front。
- get front snapshot。
- SSE / polling delta。
- submit command。
- answer challenge。
- leaderboard update。
- complete front reward。
- repeated reward call 不重複寫 open power。
- 玩家不能操作別隊 front command。
- 玩家不能使用未擁有小石。
- admin freeze / resume / complete。
- shop purchase 使用新 spendable balance。
- match reward 寫入 lifetime、spendable、team pool。
- staff reward 可指定 open power account。
- community stand reward 只在 Day 5 booth window 接入。
- general leaderboard 不使用 spendable balance 當主要排序。

### 22.3 前端測試

至少手動 QA：

- 桌面版 Front 地圖。
- 手機版 Front 地圖。
- 地圖節點點擊與 drawer。
- command toolbar。
- pending command 動畫。
- SSE 斷線重連與 snapshot fallback。
- 排行榜更新。
- 小石編隊超過 5 顆時禁止。
- 沒有足夠小石時顯示 empty state。
- Front 中刷新後可恢復。
- 答題 challenge 正常顯示解析。
- 戰報與獎勵顯示正確。
- freeze 後 UI 顯示凍結狀態。

### 22.4 營隊現場測試

需要找 3 到 5 人試玩：

- 一人不知道規則時能不能在 1 分鐘內開始。
- 是否第一眼知道自己小隊顏色與排名。
- 是否知道下一個要搶或守的節點。
- 小石差異是否被理解。
- 答錯是否仍覺得有學到。
- 是否會想和隊友討論。
- 排行榜是否讓人想繼續玩。
- 壓力是否來自地圖局勢，而不是個人被羞辱。
- 是否有刷分或單人洗 command 誘因。

## 23. 風險與對策

### 23.1 規則過複雜

風險：

- 學員看不懂。
- Staff 難以說明。

對策：

- 第一版只做 territory map，不做自由 RTS。
- 每種小石只給 1 到 2 個常用行動。
- 每個節點最多顯示 1 個主要威脅。
- 前端顯示推薦行動。

### 23.2 開源力變成 pay-to-win

風險：

- 高開源力玩家可以輾壓。

對策：

- 每窗口投入上限。
- 免費提示或新手補助。
- 真實可用開源力不每秒扣。
- 前線開源力只在 Front 內流動，不可帶出。
- 開源力主要用於修正和支援，不直接買勝利。

### 23.3 答題仍變成刷題

風險：

- 玩家只想刷題拿獎勵。

對策：

- 每日高額獎勵上限。
- 題目與 Front 節點綁定。
- 閱讀解析給獎。
- 答錯也有部分進度。
- 防守、偵查、救援、協作提供非答題貢獻。
- 攤位事件只在 Day 5 出現。

### 23.4 開發時間不足

風險：

- 新模式做一半不能上線。

對策：

- 先做 Day 1 guided Front playable prototype。
- 全域經濟改造先做 read-only，不阻塞玩法。
- Front current/get/command/leaderboard 可用後就能內部測。
- 前端先做地圖、節點 drawer、command toolbar、排行榜。
- 世界事件與 Day 5 社群攤位延後。

### 23.5 財富平均被誤解成懲罰

風險：

- 高貢獻玩家覺得自己被扣獎勵。
- 低資源玩家覺得補助很尷尬。
- 小隊公共基金被看成稅。

對策：

- 終身貢獻值永久保留並清楚顯示。
- 不扣既有資產，只調整新增收益的分配。
- 文案使用「回饋小隊」、「基地支援」、「今日支援額度」。
- Day 5 之後才使用「社群支援」。
- 高資源玩家應有稱號、收藏、回顧、展示型獎勵。
- 補助以折抵、提示額度、任務支援呈現，不直接標記玩家貧富。

### 23.6 手機 UI 擁擠

風險：

- 戰略 board 在手機上不好操作。

對策：

- MVP 使用低節點數 territory map。
- 每個節點互動放 drawer。
- 詳情放 drawer。
- 小石列橫向 scroll。
- 避免 5x5 grid 首版上線。

## 24. 開發順序建議

如果時間有限，順序應該是：

1. 鎖定 Day 1 guided Front：`開源戰線暖身`。
2. `front_maps.toml` + content loader。
3. 純 Go front engine。
4. Front persistence/API：current/get/events/leaderboard/commands/answers。
5. Tick manager + SSE/polling fallback。
6. 今日支援 token 與前線開源力，不使用真實可用開源力即時消費。
7. 前端 Front map、node drawer、command toolbar、leaderboard。
8. Staff 最小控制：start/freeze/resume/complete、flow-rate、shield、reward status。
9. Economy read model additive API。
10. Day 2-4 map / event wave / surge 擴充。
11. 經濟 writer 逐步遷移。
12. Day 5 社群攤位 / finale display。

不要先做：

- 美術動畫。
- 大量地圖。
- 自由座標 RTS。
- 多 server 即時同步。
- 複雜世界事件。
- 全域經濟重寫作為第一場的阻塞條件。
- 移除或延後排行榜。
- 真實可用開源力即時消費。
- Day 1-4 社群攤位支援。

## 25. 第一個可開發切片

### 25.1 第一切片：Day 1 Guided Front

最小切片應先讓學員玩起來：

> 學員從 Home 點「進入戰線」，看到自己小隊顏色、附近節點、暖身排行榜；系統自動推薦 3 顆小石。學員在 60 秒內派出第一個 command，幾秒後看到節點或排行榜變化，並理解前線開源力會消耗、小石訊號會倒數。

這個切片需要：

- 一個 front map template。
- 純 front engine。
- current/get/events/leaderboard/commands/answers API。
- FrontSession / FrontCommand / FrontReward。
- command idempotency。
- tick manager。
- SSE 或 polling fallback。
- 今日支援 token。
- 前線開源力。
- 一個 guided Front frontend flow。
- 一個 battle report / result page。
- 基本單元測試與 API integration test。

這個切片不需要：

- 全域經濟重寫。
- 小隊公共基金消費。
- 知識王獎勵重算。
- 完整追趕補助。
- 既有 leaderboard 全面改版。
- 多張地圖。
- 真實可用開源力即時消費。
- 自由座標 RTS。
- Day 5 社群攤位。

完成這個切片後，可以直接做 3 到 5 人 playtest，確認是否真的好上手、好討論、好玩。

### 25.2 第二切片：Staff Safety

第二切片確保營隊現場可控：

> Staff 可以啟動、暫停、凍結 Front，調整流失倍率，重置卡住的 command，確認 reward 是否寫入，並用靜態戰報或人工方式處理 fallback。

這個切片需要：

- start/freeze/resume/complete front。
- flow-rate control。
- all-camp shield。
- reset stuck command。
- reward status query。
- staff runbook。
- static fallback map / battle report。
- scheduled play window policy。

### 25.3 第三切片：Economy Read Model

第三切片才開始全域經濟地基，但先不改行為：

> Home 或 `/api/me/economy` 可以顯示累積貢獻、可花開源力、今日支援、小隊支援，但既有 `openPower` response 保持相容。

這個切片需要：

- `economy` helper。
- legacy ledger read model。
- additive API response。
- lifetime 不計舊 shop/fusion 負數。
- 單元測試。

這個切片不需要：

- shop purchase 行為改變。
- 知識王 reward 行為改變。
- team pool 實際消費。
- leaderboard 改排序。

## 26. 後續決策點

在進入實作前，需要確認：

1. 玩家顯示名稱是否統一叫 `開源戰線`。
2. Day 1 guided Front 是否固定使用 `開源戰線暖身`。
3. 第一場是否完全自動選 3 顆小石，或允許替換但不要求。
4. 第一版 Front 是否完全不扣真實可用開源力，只用前線開源力與今日支援 token。
5. 題庫 metadata 要擴充 CSV 還是新增 TOML。
6. Day 1 暖身榜是否完全不計入 finale，或保留少量紀念分。
7. Day 2 正式榜是否開始計入 finale。
8. 終身貢獻、可用開源力、今日支援額度、小隊公共基金的正式中文名稱。
9. 高資源玩家回饋比例要用小隊分位數還是全營分位數。
10. 補助要優先發成戰術預算、商店折價，還是合成素材。
11. 社群攤位 Day 5 支援卡是否只影響 finale，或也影響最後展示分類。

## 27. 建議結論

最務實的方向是：

> 先做一張 60 秒內可開始的 guided `開源戰線` 共享地圖，讓學員看到小隊顏色、節點倒數、前線開源力快沒了、排行榜正在變動；全域開源力經濟改造保留為漸進式 read model 與後續 writer migration，不阻塞第一個可玩版本。

第一階段做出 `開源戰線暖身`，優先驗證：

- 學員能不能不用讀規則就開始。
- 小隊是否會自然討論。
- 小石是否像夥伴，而不是數值裝備。
- 答題能不能自然融入戰線加速。
- 答錯是否仍然有學習與貢獻。
- 排行榜是否真的讓人想玩。
- 「現在不做，小石/開源力/節點就沒了」是否帶來刺激。
- 手機 UI 是否可行。
- Staff 是否能控制現場節奏。

第二階段再驗證：

- 財富平均是否能避免 pay-to-win。
- 高貢獻玩家是否仍有成就感。
- 低資源玩家是否能追上基本體驗。
- 商店、知識王、Staff 發獎是否能共用同一套經濟規則。

驗證成功後，再擴充 Day 2-4 正式戰線、Day 4 surge、Day 5 社群攤位 finale。
