import { ChevronUp, Swords } from "lucide-react"
import { useEffect, useState } from "react"
import { Link, useLocation } from "@tanstack/react-router"

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
    description: "題目對戰",
    to: "/battle",
    iconSrc: "/game-icons/nav/nav-battle.png",
  },
  {
    label: "開源戰線",
    description: "陣營地圖",
    to: "/front",
    iconSrc: undefined,
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
          className="bg-ink/55 animate-in fade-in-0 fixed inset-0 z-30 cursor-default backdrop-blur-[1px] duration-200 motion-reduce:animate-none"
          onClick={() => setBattleMenuPathname(null)}
          aria-label="關閉戰鬥選單"
        />
      ) : null}

      {battleMenuOpen ? (
        <div
          id="battle-menu"
          className="border-ink bg-surface-raised animate-in fade-in-0 slide-in-from-bottom-3 fixed bottom-[calc(5.65rem+env(safe-area-inset-bottom))] left-1/2 z-50 w-[min(calc(100%-3rem),18rem)] -translate-x-1/2 rounded-lg border-2 p-2 shadow-[4px_4px_0_rgba(23,35,58,0.2)] duration-200 motion-reduce:animate-none"
          aria-label="選擇戰鬥模式"
        >
          <div className="grid grid-cols-2 gap-2">
            {battleItems.map((item) => {
              const active =
                pathname === item.to || pathname.startsWith(`${item.to}/`)

              return (
                <Link
                  key={item.to}
                  to={item.to}
                  onClick={() => setBattleMenuPathname(null)}
                  className={[
                    "border-ink bg-card focus-visible:outline-power flex min-h-32 flex-col items-center justify-center gap-2 rounded-lg border-2 px-2 py-3 text-center no-underline shadow-[2px_2px_0_rgba(23,35,58,0.2)] transition-transform duration-150 hover:-translate-y-1 focus-visible:outline-3 focus-visible:outline-offset-2 active:translate-y-px motion-reduce:transition-none",
                    active ? "text-primary" : "text-ink",
                  ].join(" ")}
                >
                  {item.iconSrc ? (
                    <img
                      src={toOptimizedImageSrc(item.iconSrc)}
                      alt=""
                      className="size-11 object-contain"
                      draggable={false}
                      decoding="async"
                      aria-hidden
                    />
                  ) : (
                    <span className="bg-primary/15 grid size-11 place-items-center rounded-md">
                      <Swords className="size-6" aria-hidden />
                    </span>
                  )}
                  <span className="grid gap-1">
                    <span className="text-sm leading-none font-black">
                      {item.label}
                    </span>
                    <span className="text-muted-foreground text-[11px] leading-none font-bold">
                      {item.description}
                    </span>
                  </span>
                </Link>
              )
            })}
          </div>
        </div>
      ) : null}

      <nav
        className="bg-surface-raised border-ink fixed bottom-0 left-1/2 z-40 w-full max-w-[430px] -translate-x-1/2 border-t-2 px-3 pt-2 pb-[calc(0.65rem+env(safe-area-inset-bottom))] shadow-[0_-4px_0_rgba(23,35,58,0.12)]"
        aria-label="主要導覽"
      >
        <ul className="grid grid-cols-5 gap-1.5">
          {navItems.map((item) => {
            const active = isActivePath(pathname, item.to)

            return (
              <li key={item.to} className="min-w-0">
                <Link
                  to={item.to}
                  aria-current={active ? "page" : undefined}
                  className={[
                    "focus-visible:outline-power relative flex min-w-0 flex-col items-center justify-center gap-0.5 px-1 pt-0.5 pb-1 text-[11px] leading-none font-black no-underline transition-transform focus-visible:rounded-[14px] focus-visible:outline-3 focus-visible:outline-offset-2 active:translate-y-px",
                    active
                      ? "text-primary"
                      : "text-muted-foreground hover:text-ink",
                  ].join(" ")}
                >
                  <img
                    src={toOptimizedImageSrc(item.iconSrc)}
                    alt=""
                    className={[
                      "size-11 object-contain transition-transform",
                      active ? "scale-110" : "",
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
                      "mt-1 h-1 rounded-full transition-opacity",
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
                "focus-visible:outline-power relative flex w-full min-w-0 flex-col items-center justify-center gap-0.5 px-1 pt-0.5 pb-1 text-[11px] leading-none font-black transition-transform focus-visible:rounded-[14px] focus-visible:outline-3 focus-visible:outline-offset-2 active:translate-y-px",
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
                    "size-11 object-contain transition-transform duration-200",
                    battleActive || battleMenuOpen ? "scale-110" : "",
                  ].join(" ")}
                  draggable={false}
                  decoding="async"
                  aria-hidden
                />
                <ChevronUp
                  className={[
                    "bg-surface-raised border-ink absolute -right-1 -bottom-0.5 size-4 rounded-full border p-0.5 transition-transform duration-200",
                    battleMenuOpen ? "rotate-180" : "",
                  ].join(" ")}
                  aria-hidden
                />
              </span>
              <span className="block max-w-full truncate">戰鬥</span>
              <span
                className={[
                  "mt-1 h-1 rounded-full transition-[width,opacity] duration-200",
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
