import { CombactTeamManager } from "@/features/combat/ui/combact-team-manager"
import { Button } from "@/shared/ui/button"
import { useNavigate } from "@tanstack/react-router"
import { House, Swords, Backpack } from "lucide-react"

export function CombactPage() {
  const navigate = useNavigate()

  const toHome = () => {
    navigate({ to: "/" })
  }

  const toQuiz = () => {
    navigate({ to: "/quiz" })
  }

  const toBackpack = () => {
    navigate({ to: "/backpack" })
  }

  return (
    <div className="grid w-full grid-cols-1 gap-y-4 overflow-hidden p-4">
      <h2 className="text-center text-2xl font-bold">戰鬥準備</h2>
      {/* 隊伍管理 */}
      <CombactTeamManager />
      {/* 動作按鈕  */}
      <div className="flex items-center gap-x-4">
        {/* 返回首頁 */}
        <Button variant="outline" onClick={toHome} className="w-full flex-1">
          <House />
          返回首頁
        </Button>
        {/* 準備完成 */}
        <Button className="flex-2" onClick={toQuiz}>
          <Swords />
          開始戰鬥
        </Button>
        {/* 查看背包 */}
        <Button
          variant="outline"
          onClick={toBackpack}
          className="w-full flex-1"
        >
          <Backpack />
          背包道具
        </Button>
      </div>
    </div>
  )
}

// NOTE: 好想貓貓
