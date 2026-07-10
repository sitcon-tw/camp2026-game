import { GameIcon } from "@/shared/ui/game-icon"
import { cn } from "@/shared/utils"

export const gameFeatureIconPaths = {
  backpack: "/game-icons/features/inventory.png",
  battle: "/game-icons/nav/nav-battle.png",
  codex: "/game-icons/nav/nav-stones.png",
  dorm: "/game-icons/features/dorm.png",
  forge: "/game-icons/features/forge.png",
  front: "/game-icons/items/item_open_source_roadmap.png",
  giftHistory: "/game-icons/features/gift-history.png",
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
  className?: string
  imageClassName?: string
  alt?: string
}

export function GameFeatureIcon({
  name,
  className,
  imageClassName,
  alt = "",
}: GameFeatureIconProps) {
  return (
    <GameIcon
      iconPath={gameFeatureIconPaths[name]}
      alt={alt}
      className={cn("size-full", className)}
      imageClassName={imageClassName}
      fallback={null}
    />
  )
}
