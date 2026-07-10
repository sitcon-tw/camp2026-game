# 開源戰線 Style System

## 目標

`開源戰線` 要看起來像目前 camp2026-game 的延伸，而不是另一個產品。第一版不先依賴生圖，使用現有小石、功能 icon、厚邊框、暖色紙面、深色 ink、明確狀態色與 SVG territory map 即可完成可玩體驗。

## 視覺定位

- 類型：營隊小隊戰線地圖。
- 情緒：緊張、有排名壓力，但仍然像營隊遊戲。
- 第一眼訊號：地圖顏色、節點倒數、排行榜、前線開源力警告。
- 避免：軍武寫實、黑暗科幻、純資料儀表板、過度 RPG 技能樹。

## 現有資產使用

優先使用既有 assets：

- 小石：`frontend/public/game-icons/stones/*`
- 功能：`frontend/public/game-icons/features/leaderboard.png`
- 導覽：`frontend/public/game-icons/nav/*`
- 頁面容器：既有 `GamePageShell`
- 圖示按鈕：`lucide-react`

第一版不需要新 bitmap。若後續要生圖，才考慮：

- `開源戰線` 主 icon。
- `前線開源力訊號` 節點 icon。
- `小石受困` 狀態 icon。
- Day 4 surge 背景圖。

## 色彩規則

遵守 `frontend/AGENTS.md`：

- 不在 feature code 使用 raw Tailwind palette。
- 優先用既有 tokens：`bg-background`、`bg-card`、`bg-surface-raised`、`bg-ink`、`bg-primary`、`bg-secondary`、`text-muted-foreground`、`border-ink`。
- 小石類型使用既有 pebble tokens。

Front map team colors 第一版可由後端提供 `color` 字串，但前端不要直接任意套 hex 到 class。建議先以 CSS variable style 放在 SVG node 上：

```tsx
style={{ fill: teamColor }}
```

後續若要嚴格語意化，可把小隊顏色映射到 CSS variables。

## 地圖節點

節點是主要互動單位。

視覺狀態：

- 中立：淡紙色填滿、深色邊框。
- 我方：小隊色填滿、深色厚邊框。
- 敵方：對方色填滿、深色厚邊框。
- contested：雙層邊框或斜線 overlay。
- 倒數：節點右上角小 badge。
- 受困小石：節點上顯示小石 icon + 警告 badge。
- 前線開源力訊號：節點閃爍或高亮 ring。

第一版用 SVG polygon / rect 即可，不做 pathfinding 動畫。

## 排行榜

排行榜是核心，不是底部統計。

Front 頁應常駐：

- 前 3 或前 5 小隊。
- 自己小隊排名。
- 名次變化。
- 分數差距。

排行榜卡片要緊湊，避免 hero 尺寸。顯示 `#1`、小隊名、分數、節點數、救援數即可。

## 前線開源力

前線開源力需要明確壓力：

- `>= 40`：正常。
- `15-39`：警告。
- `< 15`：危急。
- `0`：觸發 setback。

UI 建議：

- 頂部 progress bar。
- 危急時文字使用 `前線開源力快不夠了`。
- 不顯示完整公式。

## Command Toolbar

行動按鈕要短：

- `擴張`
- `攻擊`
- `防守`
- `修復`
- `偵查`
- `救援`

每個按鈕使用 lucide icon + 文字。不要用長段說明塞在按鈕內；詳情放節點 drawer。

## 手機佈局

手機第一版：

- 上方：狀態條與前線開源力。
- 中間：地圖。
- 下方：小石列與主要行動。
- 節點詳情：bottom sheet / drawer。
- 排行榜：可收合，但前 3 名與自己名次要常駐。

## 生圖 Prompt 方向

若後續要生 bitmap，參考方向：

```text
Cute camp strategy game icon, small magical pebble companions defending a colorful territory map, warm paper texture, thick ink outlines, playful but competitive, Taiwanese student camp board-game feel, clear readable shapes, no realistic military weapons, no dark sci-fi, no text.
```

生成後要確認：

- 圖像主體是小石與地圖，不是抽象光效。
- 手機小尺寸仍可辨識。
- 不和既有小石風格衝突。
- 不使用過暗或單一紫藍漸層。
