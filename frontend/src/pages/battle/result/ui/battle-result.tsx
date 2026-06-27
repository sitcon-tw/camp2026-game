import { Button } from "@/shared/ui/button"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/shared/ui/card"
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@/shared/ui/collapsible"
import { GameFeatureIcon } from "@/shared/ui/game-feature-icon"
import { Separator } from "@/shared/ui/separator"
import { cn } from "@/shared/utils"
import { ChevronDown } from "lucide-react"

const answerExplain = [
  {
    question: "開源授權允許什麼？",
    correctAnswer: "檢視、使用與改作",
    playerAnswer: "只能下載一次",
    correct: false,
    explain: "開源的重點是讓原始碼與授權規則可被社群理解、使用與改作。",
  },
  {
    question: "開源授權允許什麼？",
    correctAnswer: "檢視、使用與改作",
    playerAnswer: "只能下載一次",
    correct: true,
    explain: "開源的重點是讓原始碼與授權規則可被社群理解、使用與改作。",
  },
  {
    question: "開源授權允許什麼？",
    correctAnswer: "檢視、使用與改作",
    playerAnswer: "只能下載一次",
    correct: false,
    explain: "開源的重點是讓原始碼與授權規則可被社群理解、使用與改作。",
  },
  {
    question: "開源授權允許什麼？",
    correctAnswer: "檢視、使用與改作",
    playerAnswer: "只能下載一次",
    correct: false,
    explain: "開源的重點是讓原始碼與授權規則可被社群理解、使用與改作。",
  },
]

export function BattleResultPage() {
  return (
    <main className="mx-auto grid w-full max-w-sm gap-y-2 py-4">
      {/* 對戰結果 */}
      <Card>
        <CardContent className="grid gap-y-4">
          <span className="animate-bounce text-center text-4xl font-bold">
            勝利
          </span>
          <div className="flex items-center gap-x-4">
            <Card className="bg-accent text-status-success flex-1">
              <CardContent className="grid gap-y-2">
                <span className="text-center">混凝土</span>
                <span className="text-center text-4xl font-bold">640</span>
              </CardContent>
            </Card>
            <span className="text-2xl font-bold">VS</span>
            <Card className="bg-accent text-muted-foreground flex-1">
              <CardContent className="grid gap-y-2">
                <span className="text-center">義大利麵</span>
                <span className="text-center text-4xl font-bold">520</span>
              </CardContent>
            </Card>
          </div>
        </CardContent>
      </Card>
      {/* 獲得道具 */}
      <Card>
        <CardHeader>
          <CardTitle>獲得道具</CardTitle>
          <CardDescription>本場對戰的獎勵已收入背包！</CardDescription>
        </CardHeader>
        <CardContent className="flex gap-x-4">
          <div className="bg-accent border-secondary-foreground rounded-lg border-2 p-2">
            <GameFeatureIcon name="backpack" className="size-10" />
          </div>
          <div className="grid grid-cols-2 gap-x-4 text-lg">
            <div className="grid gap-y-2">
              <span>開源力</span>
              <span>螢光靈燈</span>
            </div>
            <div className="text-foreground grid gap-y-2 text-lg">
              <span className="flex items-center gap-x-2">+80</span>
              <span className="flex items-center gap-x-2">+1</span>
            </div>
          </div>
        </CardContent>
      </Card>
      {/* 題目解析 */}
      <Separator className="my-2" />

      <span className="text-center text-2xl font-bold">逐題解析</span>
      {answerExplain.map((item, index) => {
        return (
          <Card>
            <CardContent>
              <Collapsible>
                <CollapsibleTrigger asChild>
                  <Button
                    variant="ghost"
                    className="group flex h-fit w-full items-center justify-start gap-x-4"
                  >
                    <div
                      className={cn(
                        "border-foreground rounded border-2 px-4 py-2 text-lg",
                        item.correct
                          ? "bg-status-success text-status-success-foreground"
                          : "bg-status-warning text-status-warning-foreground",
                      )}
                    >
                      {index + 1}
                    </div>
                    <span className="flex-1 text-center text-lg">
                      {item.question}
                    </span>
                    <ChevronDown className="transition group-data-[state=open]:rotate-180" />
                  </Button>
                </CollapsibleTrigger>
                <CollapsibleContent>
                  <div className="mt-2 grid gap-y-2">
                    <Card>
                      <CardContent className="grid gap-y-2">
                        <div className="grid gap-y-1">
                          <span className="text-muted-foreground text-sm font-bold">
                            正確答案
                          </span>
                          <span className="text-lg font-bold">
                            {item.correctAnswer}
                          </span>
                        </div>
                        {!item.correct && (
                          <div className="grid gap-y-1">
                            <span className="text-muted-foreground text-sm font-bold">
                              錯誤答案
                            </span>
                            <span className="decoration-muted-foreground text-lg font-bold line-through decoration-2">
                              {item.playerAnswer}
                            </span>
                          </div>
                        )}
                        <Separator />
                        <span>{item.explain}</span>
                      </CardContent>
                    </Card>
                  </div>
                </CollapsibleContent>
              </Collapsible>
            </CardContent>
          </Card>
        )
      })}
      {/* 動作區 */}
      <Separator className="my-2" />
      <Button>
        <GameFeatureIcon name="home" className="size-5" /> 返回首頁
      </Button>
    </main>
  )
}
