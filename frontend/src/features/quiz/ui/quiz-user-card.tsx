import ButtonSuqareHeader from "../assets/button_square_header_blade_rectangle_screws.svg"
import { cn } from "@/shared/utils/cn"

type QuizUserCardProps = {
  userName: string
  userPower: number
  sitoneName: string
  sitconId: number
  sitoneType: "探索" | "靈光" | "共鳴" | "工程" | "娛樂"
  className?: string
}

export function QuizUserCard({
  userName,
  userPower,
  sitoneName,
  sitconId,
  sitoneType,
  className,
}: QuizUserCardProps) {
  return (
    <div className={cn("relative aspect-3/1 h-25", className)}>
      <img
        src={ButtonSuqareHeader}
        className="absolute top-0 left-0 -z-10 aspect-3/1 h-25 -scale-x-100"
      />
      <div className="grid aspect-3/1 h-25 grid-cols-2 gap-y-4 py-4 text-center text-xl">
        <div className="flex items-center justify-center gap-2">
          <span className="font-bold">{sitoneName}</span>
          <span className="text-sm">#{sitconId}</span>
        </div>
        <span className="font-bold">{userName}</span>
        <span className="">{sitoneType}型小石</span>
        <span className="font-bold">{userPower}</span>
      </div>
    </div>
  )
}
