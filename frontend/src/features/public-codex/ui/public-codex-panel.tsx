import { useState } from "react"

import { Button } from "@/shared/ui/button"
import { Card, CardContent } from "@/shared/ui/card"

const STONES = [
  {
    name: "基地路線石",
    type: "探索",
    rarity: "常見",
    toneClass: "bg-pebble-explore",
    description: "記錄會場路線與線索。",
  },
  {
    name: "營燈靈光",
    type: "靈光",
    rarity: "稀有",
    toneClass: "bg-pebble-spark",
    description: "答題連勝時取得的亮色小石。",
  },
  {
    name: "小隊電波石",
    type: "共鳴",
    rarity: "常見",
    toneClass: "bg-pebble-resonate",
    description: "完成同步挑戰的共鳴標記。",
  },
  {
    name: "焊點種子石",
    type: "工程",
    rarity: "稀有",
    toneClass: "bg-pebble-engineer",
    description: "工程任務後取得的小型模組。",
  },
  {
    name: "舞台節奏石",
    type: "娛樂",
    rarity: "稀有",
    toneClass: "bg-pebble-play",
    description: "活動遊戲中取得的徽章感小石。",
  },
] as const

const ITEMS = [
  {
    name: "營燈佈景券",
    type: "外觀",
    rarity: "稀有",
    toneClass: "bg-pebble-spark",
    description: "替基地換上營燈主題。",
  },
  {
    name: "地圖棉線",
    type: "素材",
    rarity: "常見",
    toneClass: "bg-pebble-explore",
    description: "合成探索系展示邊框。",
  },
  {
    name: "工程工作台佈景",
    type: "外觀",
    rarity: "稀有",
    toneClass: "bg-pebble-engineer",
    description: "替基地切換成工具與模組板風格。",
  },
  {
    name: "小隊電波徽章",
    type: "紀念",
    rarity: "稀有",
    toneClass: "bg-pebble-resonate",
    description: "可展示在個人頁的小隊徽章。",
  },
] as const

type Tab = "stones" | "items"

export function PublicCodexPanel() {
  // TODO(api): replace static STONES/ITEMS with GET /api/catalog/sitones and GET /api/catalog/items.
  const [tab, setTab] = useState<Tab>("stones")
  const entries = tab === "stones" ? STONES : ITEMS
  const symbol = tab === "stones" ? "◆" : "▣"

  return (
    <div className="flex flex-col">
      <Card className="border-ink rounded-[30px] border-2 py-0 shadow-[5px_5px_0_rgba(23,35,58,0.16)]">
        <CardContent className="p-[18px]">
          <span className="text-muted-foreground mb-1 block text-xs font-black tracking-[0.08em] uppercase">
            查詢用途
          </span>
          <h2 className="mb-2 text-[26px] leading-[1.22] font-black tracking-tight">
            查看遊戲中所有小石與道具定義。
          </h2>
          <p className="text-muted-foreground leading-[1.65]">
            這頁不顯示你的持有狀態，只整理名稱、類型、稀有度與用途描述。
          </p>
        </CardContent>
      </Card>

      <nav className="my-3.5 grid grid-cols-2 gap-2" aria-label="圖鑑分頁">
        <Button
          type="button"
          variant={tab === "stones" ? "default" : "outline"}
          className={`min-h-11 rounded-2xl border-2 font-black shadow-none ${
            tab === "stones" ? "border-ink bg-ink text-card" : "border-border"
          }`}
          onClick={() => setTab("stones")}
        >
          小石圖鑑
        </Button>
        <Button
          type="button"
          variant={tab === "items" ? "default" : "outline"}
          className={`min-h-11 rounded-2xl border-2 font-black shadow-none ${
            tab === "items" ? "border-ink bg-ink text-card" : "border-border"
          }`}
          onClick={() => setTab("items")}
        >
          道具圖鑑
        </Button>
      </nav>

      <section
        className="grid grid-cols-2 gap-2.5"
        aria-label={tab === "stones" ? "小石圖鑑" : "道具圖鑑"}
      >
        {entries.map((entry) => (
          <article
            key={entry.name}
            className="bg-card border-ink min-w-0 rounded-[var(--radius)] border-2 p-3"
          >
            <div
              className={`${entry.toneClass} border-ink mb-2 flex h-[86px] items-center justify-center rounded-[20px] border-2`}
            >
              <span
                className="text-card/90 text-[34px] drop-shadow-[0_2px_0_rgba(23,35,58,0.3)]"
                aria-hidden
              >
                {symbol}
              </span>
            </div>
            <div className="mt-[9px] mb-1.5 flex flex-wrap gap-1">
              <span className="border-border bg-surface-raised text-muted-foreground rounded-full border px-[7px] py-[3px] text-[11px] leading-none font-black">
                {entry.type}
              </span>
              <span className="border-border bg-surface-raised text-muted-foreground rounded-full border px-[7px] py-[3px] text-[11px] leading-none font-black">
                {entry.rarity}
              </span>
            </div>
            <h3 className="mb-1 text-[17px] font-black leading-tight tracking-tight">
              {entry.name}
            </h3>
            <p className="text-muted-foreground text-[13px] leading-relaxed">
              {entry.description}
            </p>
          </article>
        ))}
      </section>
    </div>
  )
}
