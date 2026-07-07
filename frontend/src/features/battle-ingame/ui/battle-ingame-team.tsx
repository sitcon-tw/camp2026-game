import { Badge } from "@/shared/ui/badge"
import { toOptimizedImageSrc } from "@/shared/utils/image-src"
import { cn } from "@/shared/utils"

type Sitone = {
  name: string
  type: string
  pictureSrc: string
}

type BattleIngameTeamType = {
  team: Sitone[]
  highlight: 0 | 1 | 2 | 3 | 4
  reverse?: boolean
}

export function BattleIngameTeam({
  team,
  highlight,
  reverse = false,
}: BattleIngameTeamType) {
  const highlightedSitone: Sitone = team[highlight]
  return (
    <div
      className={cn(
        "flex flex-1 items-center justify-between gap-x-2",
        reverse ? "flex-row-reverse pl-4" : "flex-row pr-4",
      )}
    >
      <div className="grid gap-y-2">
        {team.map((item, index) => {
          if (index !== highlight)
            return (
              <div>
                <img
                  src={toOptimizedImageSrc(item.pictureSrc)}
                  className="h-10"
                />
              </div>
            )
        })}
      </div>
      <div className="grid gap-y-2">
        <img src={toOptimizedImageSrc(highlightedSitone.pictureSrc)} />
        <span className="text-center text-lg">{highlightedSitone.name}</span>
        <Badge className="mx-auto">{highlightedSitone.type}型小石</Badge>
      </div>
    </div>
  )
}
