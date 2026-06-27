import { createAvatar } from "@dicebear/core"
import * as thumbs from "@dicebear/thumbs"
import * as React from "react"

import { Avatar } from "@/shared/ui/avatar"
import { cn } from "@/shared/utils"

const COMPUTER_AVATAR_SRC = "/game-icons/avatars/computer-opponent.png"

function avatarSeed(playerId?: string, nickname?: string) {
  const id = playerId?.trim()
  if (id) return id

  const name = nickname?.trim()
  if (name) return name

  return "camp2026-player"
}

type PlayerAvatarProps = Omit<
  React.ComponentProps<typeof Avatar>,
  "children"
> & {
  playerId?: string
  nickname?: string
  kind?: "human" | "computer"
  svgClassName?: string
}

export function PlayerAvatar({
  playerId,
  nickname,
  kind,
  className,
  svgClassName,
  "aria-label": ariaLabel,
  ...props
}: PlayerAvatarProps) {
  const seed = avatarSeed(playerId, nickname)
  const isComputer = kind === "computer" || playerId === "computer"
  const svg = React.useMemo(
    () => (isComputer ? "" : createAvatar(thumbs, { seed }).toString()),
    [isComputer, seed],
  )

  return (
    <Avatar
      aria-hidden={ariaLabel ? undefined : true}
      aria-label={ariaLabel}
      role={ariaLabel ? "img" : undefined}
      className={cn("bg-surface-raised", className)}
      {...props}
    >
      {isComputer ? (
        <img
          src={COMPUTER_AVATAR_SRC}
          alt=""
          draggable={false}
          className="block size-full object-cover"
        />
      ) : (
        <span
          className={cn(
            "block size-full overflow-hidden [&>svg]:block [&>svg]:size-full",
            svgClassName,
          )}
          dangerouslySetInnerHTML={{ __html: svg }}
        />
      )}
    </Avatar>
  )
}
