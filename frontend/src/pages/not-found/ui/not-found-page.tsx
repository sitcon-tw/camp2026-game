import { Link } from "@tanstack/react-router"
import { ChevronRight, SearchX } from "lucide-react"

import { Button } from "@/shared/ui/button"
import { GameFeatureIcon } from "@/shared/ui/game-feature-icon"
import { GameIcon } from "@/shared/ui/game-icon"
import { GamePageShell } from "@/shared/ui/game-page-shell"

const wanderingStones = [
  {
    label: "探索小石",
    iconPath: "/game-icons/stones/basic_blue.png",
    className: "top-3 left-4 rotate-[-13deg] bg-pebble-explore-muted",
  },
  {
    label: "靈光小石",
    iconPath: "/game-icons/stones/basic_yellow.png",
    className: "top-8 right-5 rotate-[12deg] bg-pebble-spark-muted",
  },
  {
    label: "共鳴小石",
    iconPath: "/game-icons/stones/basic_purple.png",
    className: "right-8 bottom-8 rotate-[-8deg] bg-pebble-play-muted",
  },
] as const

export function NotFoundPage() {
  return (
    <GamePageShell ariaLabel="找不到頁面" contentClassName="justify-center">
      <section
        className="bg-card border-ink relative grid gap-5 overflow-hidden rounded-[28px] border-2 p-5 text-center"
        style={{ boxShadow: "6px 6px 0 var(--border)" }}
      >
        <div
          className="border-ink bg-surface-raised absolute top-4 left-4 grid size-11 place-items-center rounded-[14px] border-2"
          aria-hidden
        >
          <SearchX className="text-primary size-6" />
        </div>

        <div
          className="border-ink bg-secondary text-secondary-foreground absolute top-4 right-4 rounded-full border-2 px-3 py-1 text-sm font-black"
          aria-hidden
        >
          404
        </div>

        <div className="relative mx-auto mt-10 grid h-[230px] w-full max-w-[320px] place-items-center">
          <div
            className="border-ink bg-muted absolute inset-x-2 bottom-4 h-12 rounded-[50%] border-2"
            aria-hidden
          />
          <div
            className="border-ink bg-ink absolute inset-x-12 bottom-10 h-2 rounded-full border-2"
            aria-hidden
          />

          {wanderingStones.map((stone) => (
            <span
              key={stone.label}
              className={`border-ink absolute grid size-[58px] place-items-center rounded-[18px] border-2 p-1.5 shadow-[3px_3px_0_var(--border)] ${stone.className}`}
              aria-label={stone.label}
            >
              <GameIcon
                iconPath={stone.iconPath}
                alt=""
                imageClassName="drop-shadow-sm"
                fallback={<GameFeatureIcon name="stones" className="size-8" />}
              />
            </span>
          ))}

          <div
            className="border-ink bg-primary relative grid size-[148px] place-items-center rounded-[34px] border-2 p-4 shadow-[7px_7px_0_var(--border)]"
            aria-label="迷路的小石"
          >
            <GameIcon
              iconPath="/game-icons/stones/stone_2026_camp_explorer.png"
              alt=""
              fallback={<GameFeatureIcon name="home" className="size-24" />}
            />
          </div>
        </div>

        <div className="grid gap-2">
          <p className="text-muted-foreground text-xs font-black tracking-[0.08em] uppercase">
            Route Lost
          </p>
          <h1 className="text-[36px] leading-none font-black tracking-normal">
            小石迷路了
          </h1>
          <p className="text-muted-foreground mx-auto max-w-[300px] text-sm leading-relaxed font-bold">
            這條路線還沒被冒險者開圖。回到基地整隊，或先去小石圖鑑確認收藏。
          </p>
        </div>

        <div
          className="bg-surface-raised border-border grid grid-cols-3 gap-2 rounded-[18px] border-2 p-2"
          aria-label="小石搜尋狀態"
        >
          {[
            ["掃描", "失敗"],
            ["座標", "未知"],
            ["小石", "待命"],
          ].map(([label, value]) => (
            <div key={label} className="grid gap-0.5 py-1">
              <span className="text-muted-foreground text-[11px] font-black">
                {label}
              </span>
              <strong className="text-sm font-black">{value}</strong>
            </div>
          ))}
        </div>

        <div className="grid gap-2">
          <Button asChild className="min-h-12 rounded-[14px]">
            <Link to="/">
              <GameFeatureIcon name="home" className="size-7" />
              回到基地
              <ChevronRight className="size-5" aria-hidden />
            </Link>
          </Button>
          <Button
            asChild
            variant="secondary"
            className="min-h-12 rounded-[14px]"
          >
            <Link to="/stones">
              <GameFeatureIcon name="stones" className="size-7" />
              查看小石圖鑑
            </Link>
          </Button>
        </div>
      </section>
    </GamePageShell>
  )
}
