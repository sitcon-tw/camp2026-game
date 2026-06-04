import { cn } from "@/shared/utils/cn"
import MinimapArrowA from '../assets/minimap_arrow_a.svg'

type SitoneAttribute = {
  id: string
  name: string
  type: "探索" | "靈光" | "共鳴" | "工程" | "娛樂"
  pictureSrc: string
}

type QuizTeamListType = {
  team: SitoneAttribute[]
  highlight: 0 | 1 | 2 | 3 | 4
  reverse: boolean
  className?: string
}

export function QuizTeamList({ team, highlight, reverse = false, className }: QuizTeamListType) {
  return (
    <div className={cn("flex gap-2", className)}>
      {team.map((sitone, index) => {
        return (
          <div className={cn("flex gap-2", reverse ? "flex-col-reverse" : "flex-col")}>
            <img src={sitone.pictureSrc} />
            {(highlight === index) && <img src={MinimapArrowA} className={cn("w-full h-8", reverse && "-scale-y-100")} /> }
          </div>
        )
      })}
    </div>
  )
}
