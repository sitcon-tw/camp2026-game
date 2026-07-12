import { useQuery } from "@tanstack/react-query"
import { ChevronUp, Lock, Map } from "lucide-react"
import { useEffect, useState } from "react"
import { Link, useLocation } from "@tanstack/react-router"

import { battleOpeningQueryOptions } from "@/shared/api/battle-opening.query"
import { toOptimizedImageSrc } from "@/shared/utils/image-src"

const hiddenPathPrefixes = [
  "/login",
  "/codex",
  "/admin",
  "/battle/question",
  "/battle/ingame",
] as const

const navItems = [
  {
    label: "首頁",
    to: "/",
    iconSrc: "/game-icons/nav/nav-home.png",
  },
  {
    label: "小石",
    to: "/stones",
    iconSrc: "/game-icons/nav/nav-stones.png",
  },
  {
    label: "通行證",
    to: "/profile/qr",
    iconSrc: "/game-icons/nav/nav-profile.png",
  },
  {
    label: "商店",
    to: "/shop",
    iconSrc: "/game-icons/nav/nav-shop.png",
  },
] as const

const battleItems = [
  {
    label: "知識王",
    description: "即時答題對戰",
    to: "/battle",
    iconSrc: "/game-icons/nav/nav-battle.png",
    className: "bg-pebble-play-muted",
  },
  {
    label: "開源戰線",
    description: "小隊領地攻防",
    to: "/front",
    iconSrc: undefined,
    className: "bg-pebble-engineer-muted",
  },
] as const

function isHiddenPath(pathname: string) {
  return hiddenPathPrefixes.some(
    (path) => pathname === path || pathname.startsWith(`${path}/`),
  )
}

function isActivePath(pathname: string, to: (typeof navItems)[number]["to"]) {
  if (to === "/") return pathname === "/"

  return pathname === to || pathname.startsWith(`${to}/`)
}

function isBattleActive(pathname: string) {
  return battleItems.some(
    (item) => pathname === item.to || pathname.startsWith(`${item.to}/`),
  )
}

export function AppBottomNav() {
  const { pathname } = useLocation()
  const battleOpeningQuery = useQuery({
    ...battleOpeningQueryOptions(),
    enabled: !isHiddenPath(pathname),
  })
  const [battleMenuPathname, setBattleMenuPathname] = useState<string | null>(
    null,
  )
  const battleMenuOpen = battleMenuPathname === pathname

  useEffect(() => {
    if (!battleMenuOpen) return

    const closeOnEscape = (event: KeyboardEvent) => {
      if (event.key === "Escape") setBattleMenuPathname(null)
    }

    window.addEventListener("keydown", closeOnEscape)
    return () => window.removeEventListener("keydown", closeOnEscape)
  }, [battleMenuOpen])

  if (isHiddenPath(pathname)) return null

  const battleActive = isBattleActive(pathname)

  return (
    <>
      {battleMenuOpen ? (
        <button
          type="button"
          className="bg-ink/45 animate-in fade-in-0 fixed inset-0 z-30 cursor-default backdrop-blur-[2px] duration-200 motion-reduce:animate-none"
          onClick={() => setBattleMenuPathname(null)}
          aria-label="關閉戰鬥選單"
        />
      ) : null}

      {battleMenuOpen ? (
        <div
          id="battle-menu"
          className="border-ink bg-card animate-in fade-in-0 zoom-in-90 slide-in-from-bottom-2 fixed right-[max(0.75rem,calc((100vw-430px)/2+0.75rem))] bottom-[calc(5rem+env(safe-area-inset-bottom))] z-50 w-[min(calc(100%-2rem),17rem)] origin-bottom-right rounded-lg border-2 p-2.5 shadow-[5px_5px_0_rgba(23,35,58,0.24)] duration-200 motion-reduce:animate-none"
          aria-label="選擇戰鬥模式"
        >
          <span
            className="border-ink bg-card absolute right-6 -bottom-2 size-4 rotate-45 border-r-2 border-b-2"
            aria-hidden
          />
          <div className="mb-2 flex items-center justify-between px-0.5">
            <span className="text-ink text-xs font-black">選擇戰鬥模式</span>
            <span className="bg-ink/15 h-1 w-8 rounded-full" aria-hidden />
          </div>
          <div className="relative grid gap-2">
            {battleItems.map((item) => {
              const active =
                pathname === item.to || pathname.startsWith(`${item.to}/`)
              const locked =
                battleOpeningQuery.isPending ||
                battleOpeningQuery.data?.battleOpeningLocked === true
              const content = (
                <>
                  {item.iconSrc ? (
                    <img
                      src={toOptimizedImageSrc(item.iconSrc)}
                      alt=""
                      className="size-9 object-contain"
                      draggable={false}
                      decoding="async"
                      aria-hidden
                    />
                  ) : (
                    <span className="bg-card/70 grid size-9 place-items-center rounded-md border border-current/15">
                      <Map className="size-5" aria-hidden />
                    </span>
                  )}
                  <span className="grid min-w-0 flex-1 gap-1">
                    <span className="text-sm leading-none font-black">
                      {item.label}
                    </span>
                    <span className="text-muted-foreground text-[10px] leading-tight font-bold">
                      {locked ? "上課期間暫停" : item.description}
                    </span>
                  </span>
                  {locked ? (
                    <Lock className="size-4 shrink-0" aria-hidden />
                  ) : null}
                </>
              )

              const className = [
                "border-ink flex min-h-16 min-w-0 items-center gap-3 rounded-md border-2 px-3 py-2 text-left no-underline shadow-[2px_2px_0_rgba(23,35,58,0.18)]",
                item.className,
                locked
                  ? "text-muted-foreground cursor-not-allowed opacity-65"
                  : "focus-visible:outline-power transition-[transform,box-shadow] duration-200 hover:-translate-x-0.5 hover:shadow-[3px_3px_0_rgba(23,35,58,0.2)] focus-visible:outline-3 focus-visible:outline-offset-2 active:translate-x-px active:translate-y-px active:shadow-none motion-reduce:transition-none",
                locked ? "" : active ? "text-primary" : "text-ink",
              ].join(" ")

              if (locked) {
                return (
                  <span key={item.to} aria-disabled className={className}>
                    {content}
                  </span>
                )
              }

              return (
                <Link
                  key={item.to}
                  to={item.to}
                  onClick={() => setBattleMenuPathname(null)}
                  className={className}
                >
                  {content}
                </Link>
              )
            })}
          </div>
        </div>
      ) : null}

      <nav
        className="bg-card/95 border-ink fixed bottom-0 left-1/2 z-40 w-full max-w-[430px] -translate-x-1/2 border-t-2 px-2 pt-1.5 pb-[calc(0.4rem+env(safe-area-inset-bottom))] shadow-[0_-3px_0_rgba(23,35,58,0.1)] backdrop-blur-md"
        aria-label="主要導覽"
      >
        <ul className="grid h-[3.8rem] grid-cols-5 gap-1">
          {navItems.map((item) => {
            const active = isActivePath(pathname, item.to)

            return (
              <li key={item.to} className="min-w-0">
                <Link
                  to={item.to}
                  aria-current={active ? "page" : undefined}
                  className={[
                    "focus-visible:outline-power relative flex h-full min-w-0 flex-col items-center justify-center gap-0.5 px-1 text-[10px] leading-none font-black no-underline transition-[color,transform] duration-200 focus-visible:rounded-md focus-visible:outline-3 focus-visible:outline-offset-1 active:translate-y-px motion-reduce:transition-none",
                    active
                      ? "text-primary"
                      : "text-muted-foreground hover:text-ink",
                  ].join(" ")}
                >
                  <img
                    src={toOptimizedImageSrc(item.iconSrc)}
                    alt=""
                    className={[
                      "size-9 object-contain transition-transform duration-200",
                      active ? "-translate-y-0.5 scale-110" : "",
                    ].join(" ")}
                    draggable={false}
                    decoding="async"
                    aria-hidden
                  />
                  <span className="block max-w-full truncate">
                    {item.label}
                  </span>
                  <span
                    className={[
                      "absolute top-0 h-1 rounded-full transition-[width,opacity] duration-200",
                      active ? "bg-primary w-5 opacity-100" : "w-2 opacity-0",
                    ].join(" ")}
                    aria-hidden
                  />
                </Link>
              </li>
            )
          })}
          <li className="min-w-0">
            <button
              type="button"
              aria-expanded={battleMenuOpen}
              aria-controls="battle-menu"
              onClick={() =>
                setBattleMenuPathname(battleMenuOpen ? null : pathname)
              }
              className={[
                "focus-visible:outline-power relative flex h-full w-full min-w-0 flex-col items-center justify-center gap-0.5 px-1 text-[10px] leading-none font-black transition-[color,transform] duration-200 focus-visible:rounded-md focus-visible:outline-3 focus-visible:outline-offset-1 active:translate-y-px motion-reduce:transition-none",
                battleActive || battleMenuOpen
                  ? "text-primary"
                  : "text-muted-foreground hover:text-ink",
              ].join(" ")}
            >
              <span className="relative">
                <img
                  src={toOptimizedImageSrc("/game-icons/nav/nav-battle.png")}
                  alt=""
                  className={[
                    "size-9 object-contain transition-transform duration-200",
                    battleActive || battleMenuOpen
                      ? "-translate-y-0.5 scale-110"
                      : "",
                  ].join(" ")}
                  draggable={false}
                  decoding="async"
                  aria-hidden
                />
                <ChevronUp
                  className={[
                    "bg-card border-ink absolute -right-1 -bottom-0.5 size-3.5 rounded-full border p-0.5 transition-transform duration-200",
                    battleMenuOpen ? "rotate-180" : "",
                  ].join(" ")}
                  aria-hidden
                />
              </span>
              <span className="block max-w-full truncate">戰鬥</span>
              <span
                className={[
                  "absolute top-0 h-1 rounded-full transition-[width,opacity] duration-200",
                  battleActive || battleMenuOpen
                    ? "bg-primary w-5 opacity-100"
                    : "w-2 opacity-0",
                ].join(" ")}
                aria-hidden
              />
            </button>
          </li>
        </ul>
      </nav>
    </>
  )
}
