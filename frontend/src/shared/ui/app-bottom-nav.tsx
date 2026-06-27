import { Link, useLocation } from "@tanstack/react-router"

const bottomNavHiddenPathPrefixes = [
  "/login",
  "/codex",
  "/admin",
  "/battle/ingame",
  "/battle/question",
] as const

const navItems = [
  {
    label: "首頁",
    to: "/",
    iconSrc: "/game-icons/nav/nav-home.png",
  },
  {
    label: "戰鬥",
    to: "/battle",
    iconSrc: "/game-icons/nav/nav-battle.png",
  },
  {
    label: "小石",
    to: "/stones",
    iconSrc: "/game-icons/nav/nav-stones.png",
  },
  {
    label: "商店",
    to: "/shop",
    iconSrc: "/game-icons/nav/nav-shop.png",
  },
  {
    label: "通行證",
    to: "/profile/qr",
    iconSrc: "/game-icons/nav/nav-profile.png",
  },
] as const

function isBottomNavHiddenPath(pathname: string) {
  return bottomNavHiddenPathPrefixes.some(
    (path) => pathname === path || pathname.startsWith(`${path}/`),
  )
}

function isActivePath(pathname: string, to: (typeof navItems)[number]["to"]) {
  if (to === "/") return pathname === "/"

  return pathname === to || pathname.startsWith(`${to}/`)
}

export function AppBottomNav() {
  const { pathname } = useLocation()

  if (isBottomNavHiddenPath(pathname)) return null

  return (
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
                  src={item.iconSrc}
                  alt=""
                  className={[
                    "size-11 object-contain transition-transform",
                    active ? "scale-110" : "",
                  ].join(" ")}
                  draggable={false}
                  decoding="async"
                  aria-hidden
                />
                <span className="block max-w-full truncate">{item.label}</span>
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
      </ul>
    </nav>
  )
}
