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
              <div
                key={`${item.name}-${index}`}
                className="grid size-10 place-items-center overflow-hidden"
              >
                <img
                  src={toOptimizedImageSrc(item.pictureSrc)}
                  alt=""
                  className="size-full object-contain"
                  loading="lazy"
                  draggable={false}
                />
              </div>
            )
        })}
      </div>
      <div className="grid gap-y-2">
        <div className="grid h-[180px] max-h-[30vh] w-[160px] place-items-center overflow-hidden sm:h-[220px] sm:w-[190px]">
          <img
            src={toOptimizedImageSrc(highlightedSitone.pictureSrc)}
            alt=""
            className="size-full object-contain"
            loading="lazy"
            draggable={false}
          />
        </div>
        <span className="text-center text-lg">{highlightedSitone.name}</span>
        <Badge className="mx-auto">{highlightedSitone.type}型小石</Badge>
      </div>
    </div>
  )
}
