import { sitoneMeta } from "@/shared/lib/game-labels"
import { GameIcon } from "@/shared/ui/game-icon"
import { cn } from "@/shared/utils"

type SitoneIconProps = {
  type: string
  iconPath?: string
  alt?: string
  className?: string
  imageClassName?: string
}

export function SitoneIcon({
  type,
  iconPath,
  alt = "",
  className,
  imageClassName,
}: SitoneIconProps) {
  const meta = sitoneMeta(type)

  return (
    <span
      className={cn(
        "border-ink grid size-9 place-items-center rounded-[12px] border-2 text-[10px] font-black",
        meta.bgClassName,
        className,
      )}
    >
      <GameIcon
        iconPath={iconPath}
        alt={alt}
        imageClassName={cn(
          "p-0.5 drop-shadow-[0_1px_0_rgba(23,35,58,0.18)]",
          imageClassName,
        )}
        fallback={<span>{meta.short}</span>}
      />
    </span>
  )
}
