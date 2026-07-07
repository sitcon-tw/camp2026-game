import { createAvatar } from "@dicebear/core"
import * as thumbs from "@dicebear/thumbs"
import * as React from "react"

import { Avatar, AvatarImage } from "@/shared/ui/avatar"
import { toOptimizedImageSrc } from "@/shared/utils/image-src"
import { cn } from "@/shared/utils"

const COMPUTER_AVATAR_SRC = "/game-icons/avatars/computer-opponent.png"

function avatarSeed(playerId?: string, nickname?: string) {
  const id = playerId?.trim()
  if (id) return id

  const name = nickname?.trim()
  if (name) return name

  return "camp2026-player"
}

function svgDataUrl(svg: string) {
  return `data:image/svg+xml;utf8,${encodeURIComponent(svg)}`
}

type PlayerAvatarProps = Omit<
  React.ComponentProps<typeof Avatar>,
  "children"
> & {
  playerId?: string
  nickname?: string
  avatarUrl?: string
  kind?: "human" | "computer"
  svgClassName?: string
}

export function PlayerAvatar({
  playerId,
  nickname,
  avatarUrl,
  kind,
  className,
  svgClassName,
  "aria-label": ariaLabel,
  ...props
}: PlayerAvatarProps) {
  const seed = avatarSeed(playerId, nickname)
  const isComputer = kind === "computer" || playerId === "computer"
  const avatarSrc = React.useMemo(
    () =>
      isComputer
        ? COMPUTER_AVATAR_SRC
        : avatarUrl
          ? avatarUrl
          : svgDataUrl(createAvatar(thumbs, { seed }).toString()),
    [avatarUrl, isComputer, seed],
  )

  return (
    <Avatar
      aria-hidden={ariaLabel ? undefined : true}
      aria-label={ariaLabel}
      role={ariaLabel ? "img" : undefined}
      className={cn("bg-surface-raised", className)}
      {...props}
    >
      <AvatarImage
        src={toOptimizedImageSrc(avatarSrc)}
        alt=""
        aria-hidden="true"
        draggable={false}
        className={cn("block size-full object-cover", svgClassName)}
      />
    </Avatar>
  )
}
