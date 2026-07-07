import { useQuery } from "@tanstack/react-query"
import { Link } from "@tanstack/react-router"

import { AppError } from "@/shared/api/error"
import { gameApi, type GiftHistoryEntry } from "@/shared/api/game"
import { Badge } from "@/shared/ui/badge"
import { Button } from "@/shared/ui/button"
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/shared/ui/card"
import { GameFeatureIcon } from "@/shared/ui/game-feature-icon"
import { GamePageShell } from "@/shared/ui/game-page-shell"
import { PageHeader } from "@/shared/ui/page-header"

const dateTimeFormatter = new Intl.DateTimeFormat("zh-TW", {
  month: "2-digit",
  day: "2-digit",
  hour: "2-digit",
  minute: "2-digit",
})

function formatDateTime(value: string) {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return "-"
  return dateTimeFormatter.format(date)
}

function rewardAmountLabel(entry: GiftHistoryEntry) {
  if (entry.kind === "open_power") {
    return `+${entry.amount ?? 0} OP`
  }
  return `x${entry.quantity ?? 0}`
}

function rewardKindLabel(entry: GiftHistoryEntry) {
  switch (entry.kind) {
    case "item":
      return "道具"
    case "sitone":
      return "小石"
    case "open_power":
      return "開源力"
  }
}

function EmptyState({
  title,
  description,
}: {
  title: string
  description: string
}) {
  return (
    <Card className="border-ink rounded-[22px] border-2">
      <CardHeader>
        <CardTitle>{title}</CardTitle>
        <CardDescription>{description}</CardDescription>
      </CardHeader>
    </Card>
  )
}

export function GiftHistoryPage() {
  const {
    data: entries = [],
    isPending,
    error,
  } = useQuery({
    queryKey: ["me", "gift-history"],
    queryFn: gameApi.giftHistory,
  })

  const isUnauthorized = error instanceof AppError && error.status === 401

  return (
    <GamePageShell
      ariaLabel="禮物接收歷史頁"
      contentClassName="grid content-start gap-y-3"
    >
      <PageHeader
        title="禮物接收歷史"
        headline="Received Gifts"
        rightSlot={
          <Badge
            variant="secondary"
            className="border-ink mt-1 border-2 px-3 py-1 text-sm font-black"
          >
            {isPending ? "同步中" : `${entries.length} 筆`}
          </Badge>
        }
      />

      {isUnauthorized ? (
        <EmptyState
          title="請先登入"
          description="登入後才能查看自己收到的禮物紀錄。"
        />
      ) : isPending ? (
        <EmptyState
          title="正在同步禮物接收歷史"
          description="系統正在整理你收到的獎勵。"
        />
      ) : entries.length === 0 ? (
        <Card className="border-ink rounded-[22px] border-2">
          <CardHeader>
            <CardTitle>目前還沒有禮物接收紀錄</CardTitle>
            <CardDescription>
              關主發放小石、道具或開源力後，紀錄會顯示在這裡。
            </CardDescription>
          </CardHeader>
          <CardFooter>
            <Button asChild className="w-full">
              <Link to="/">返回首頁</Link>
            </Button>
          </CardFooter>
        </Card>
      ) : (
        <section className="grid gap-3" aria-label="禮物接收歷史列表">
          {entries.map((entry) => (
            <Card
              key={entry.rewardId}
              className="border-ink rounded-[22px] border-2 p-0"
            >
              <CardContent className="grid gap-3 p-4">
                <div className="grid grid-cols-[48px_1fr_auto] items-start gap-3">
                  <div className="bg-surface-raised border-ink grid size-12 place-items-center rounded-[16px] border-2">
                    <GameFeatureIcon name="history" className="size-7" />
                  </div>
                  <div className="min-w-0">
                    <div className="mb-1 flex flex-wrap items-center gap-2">
                      <h2 className="text-lg leading-tight font-black break-words">
                        {entry.name}
                      </h2>
                      <Badge variant="secondary">
                        {rewardKindLabel(entry)}
                      </Badge>
                    </div>
                    <p className="text-muted-foreground text-sm font-semibold">
                      來自 {entry.staffNickname || entry.staffPlayerId}
                    </p>
                    <p className="text-muted-foreground text-sm">
                      {formatDateTime(entry.createdAt)}
                    </p>
                  </div>
                  <div className="text-right">
                    <strong className="text-primary block text-xl font-black">
                      {rewardAmountLabel(entry)}
                    </strong>
                  </div>
                </div>
              </CardContent>
            </Card>
          ))}
        </section>
      )}
    </GamePageShell>
  )
}
