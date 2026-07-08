import { type ReactNode } from "react"

import { toOptimizedImageSrc } from "@/shared/utils/image-src"
import { cn } from "@/shared/utils"

type GameIconProps = {
  iconPath?: string
  alt?: string
  className?: string
  imageClassName?: string
  fallback: ReactNode
}

export function GameIcon({
  iconPath,
  alt = "",
  className,
  imageClassName,
  fallback,
}: GameIconProps) {
  return (
    <span
      className={cn(
        "relative grid size-full min-h-0 min-w-0 place-items-center overflow-hidden rounded-[inherit]",
        className,
      )}
    >
      {iconPath ? (
        <img
          src={toOptimizedImageSrc(iconPath)}
          alt={alt}
          className={cn("size-full object-contain", imageClassName)}
          loading="lazy"
          draggable={false}
        />
      ) : (
        fallback
      )}
    </span>
  )
}
