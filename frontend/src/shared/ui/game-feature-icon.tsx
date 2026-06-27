import { cn } from "@/shared/utils"

export const gameFeatureIconPaths = {
  backpack: "/game-icons/features/inventory.png",
  battle: "/game-icons/nav/nav-battle.png",
  codex: "/game-icons/nav/nav-stones.png",
  forge: "/game-icons/features/forge.png",
  history: "/game-icons/features/history.png",
  home: "/game-icons/nav/nav-home.png",
  leaderboard: "/game-icons/features/leaderboard.png",
  pass: "/game-icons/nav/nav-profile.png",
  shop: "/game-icons/nav/nav-shop.png",
  stones: "/game-icons/nav/nav-stones.png",
  team: "/game-icons/features/team.png",
} as const

export type GameFeatureIconName = keyof typeof gameFeatureIconPaths

type GameFeatureIconProps = {
  name: GameFeatureIconName
  alt?: string
  className?: string
  imageClassName?: string
}

export function GameFeatureIcon({
  name,
  alt = "",
  className,
  imageClassName,
}: GameFeatureIconProps) {
  return (
    <span
      className={cn(
        "grid size-full min-h-0 min-w-0 place-items-center overflow-visible",
        className,
      )}
      aria-hidden={alt ? undefined : true}
    >
      <img
        src={gameFeatureIconPaths[name]}
        alt={alt}
        className={cn("size-full object-contain", imageClassName)}
        draggable={false}
        decoding="async"
      />
    </span>
  )
}
