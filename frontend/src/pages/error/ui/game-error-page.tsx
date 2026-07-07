import { Link } from "@tanstack/react-router"
import type { ErrorComponentProps } from "@tanstack/react-router"
import { RotateCcw, ShieldAlert } from "lucide-react"
import { useEffect } from "react"

import { AppError } from "@/shared/api/error"
import { Button } from "@/shared/ui/button"
import { GameFeatureIcon } from "@/shared/ui/game-feature-icon"
import { GameIcon } from "@/shared/ui/game-icon"
import { GamePageShell } from "@/shared/ui/game-page-shell"

const signalStones = [
  {
    label: "工程小石",
    iconPath: "/game-icons/stones/basic_orange.png",
    className: "top-4 left-5 rotate-[-12deg] bg-pebble-play-muted",
  },
  {
    label: "靈光小石",
    iconPath: "/game-icons/stones/basic_yellow.png",
    className: "top-9 right-6 rotate-[10deg] bg-pebble-spark-muted",
  },
  {
    label: "探索小石",
    iconPath: "/game-icons/stones/basic_blue.png",
    className: "right-10 bottom-7 rotate-[-7deg] bg-pebble-explore-muted",
  },
] as const

function errorHint(error: unknown) {
  const message = error instanceof Error ? error.message : String(error)

  if (
    message.includes("Failed to fetch dynamically imported module") ||
    message.includes("Importing a module script failed") ||
    message.includes("Loading chunk")
  ) {
    return "新的地圖碎片還沒同步完成，重新召喚一次通常就能繼續。"
  }

  return "小石基地剛剛遇到短暫亂流，隊伍資料先停在安全區。"
}

function errorDebugDetails(error: unknown) {
  if (error instanceof AppError) {
    return {
      name: error.name,
      message: error.message,
      status: error.status,
      code: error.code,
      requestId: error.requestId,
      retryable: error.retryable,
      stack: error.stack,
    }
  }

  if (error instanceof Error) {
    return {
      name: error.name,
      message: error.message,
      stack: error.stack,
    }
  }

  return { value: error }
}

export function GameErrorPage({ error, reset }: ErrorComponentProps) {
  useEffect(() => {
    if (typeof window === "undefined") return

    console.error("GameErrorPage rendered", {
      href: window.location.href,
      pathname: window.location.pathname,
      search: window.location.search,
      diagnostics: errorDebugDetails(error),
    })
  }, [error])

  function handleRetry() {
    reset()
    window.location.reload()
  }

  return (
    <GamePageShell ariaLabel="頁面載入失敗" contentClassName="justify-center">
      <section
        className="bg-card border-ink relative grid gap-5 overflow-hidden rounded-[28px] border-2 p-5 text-center"
        style={{ boxShadow: "6px 6px 0 var(--border)" }}
      >
        <div
          className="border-ink bg-surface-raised absolute top-4 left-4 grid size-11 place-items-center rounded-[14px] border-2"
          aria-hidden
        >
          <ShieldAlert className="text-primary size-6" />
        </div>

        <div
          className="border-ink bg-secondary text-secondary-foreground absolute top-4 right-4 rounded-full border-2 px-3 py-1 text-sm font-black"
          aria-hidden
        >
          亂流
        </div>

        <div className="relative mx-auto mt-10 grid h-[220px] w-full max-w-[320px] place-items-center">
          <div
            className="border-ink bg-muted absolute inset-x-3 bottom-5 h-12 rounded-[50%] border-2"
            aria-hidden
          />
          <div
            className="border-ink bg-ink absolute inset-x-14 bottom-11 h-2 rounded-full border-2"
            aria-hidden
          />

          {signalStones.map((stone) => (
            <span
              key={stone.label}
              className={`border-ink absolute grid size-[58px] place-items-center rounded-[18px] border-2 p-1.5 shadow-[3px_3px_0_var(--border)] ${stone.className}`}
              aria-label={stone.label}
            >
              <GameIcon
                iconPath={stone.iconPath}
                alt=""
                imageClassName="drop-shadow-sm"
                fallback={<GameFeatureIcon name="stones" className="size-6" />}
              />
            </span>
          ))}

          <div
            className="border-ink bg-primary relative grid size-[150px] place-items-center rounded-[36px] border-2 p-4 shadow-[7px_7px_0_var(--border)]"
            aria-label="守護基地的小石"
          >
            <GameIcon
              iconPath="/game-icons/stones/stone_2026_camp_explorer.png"
              alt=""
              fallback={<GameFeatureIcon name="stones" className="size-12" />}
            />
          </div>
        </div>

        <div className="grid gap-2">
          <p className="text-muted-foreground text-xs font-black tracking-[0.08em] uppercase">
            Signal Interrupted
          </p>
          <h1 className="text-[34px] leading-none font-black tracking-normal">
            小石撞到亂流
          </h1>
          <p className="text-muted-foreground mx-auto max-w-[300px] text-sm leading-relaxed font-bold">
            {errorHint(error)}
          </p>
        </div>

        <div
          className="bg-surface-raised border-border grid grid-cols-3 gap-2 rounded-[18px] border-2 p-2"
          aria-label="基地狀態"
        >
          {[
            ["地圖", "重新同步"],
            ["小石", "安全"],
            ["隊伍", "待命"],
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
          <Button
            type="button"
            className="min-h-12 rounded-[14px]"
            onClick={handleRetry}
          >
            <RotateCcw className="size-5" aria-hidden />
            再召喚一次
          </Button>
          <Button
            asChild
            variant="secondary"
            className="min-h-12 rounded-[14px]"
          >
            <Link to="/">
              <GameFeatureIcon name="home" className="size-5" />
              回到基地
            </Link>
          </Button>
        </div>
      </section>
    </GamePageShell>
  )
}
