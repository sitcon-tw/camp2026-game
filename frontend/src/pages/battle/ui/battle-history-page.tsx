import { useQuery } from "@tanstack/react-query"
import { Link } from "@tanstack/react-router"
import { ArrowRight } from "lucide-react"

import { gameApi, type CompletedMatch } from "@/shared/api/game"
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
import { PlayerAvatar } from "@/shared/ui/player-avatar"
import { Separator } from "@/shared/ui/separator"
import { cn } from "@/shared/utils"

function storeMatchID(matchID: string) {
  if (typeof window !== "undefined") {
    window.localStorage.setItem("camp2026.currentMatchId", matchID)
  }
}

function formatDateTime(value?: string) {
  if (!value) return "未完成"

  return new Intl.DateTimeFormat("zh-TW", {
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
  }).format(new Date(value))
}

function winnerName(match: CompletedMatch) {
  const winner = rankedPlayers(match)[0]

  return winner ? `${winner.nickname} 勝利` : "對戰結束"
}

function rankedPlayers(match: CompletedMatch) {
  return [...match.players].sort((a, b) => (b.score ?? 0) - (a.score ?? 0))
}

export function BattleHistoryPage() {
  const { data: matches, isPending } = useQuery({
    queryKey: ["me", "matches"],
    queryFn: gameApi.completedMatches,
  })
  const completedMatches = matches ?? []

  return (
    <GamePageShell
      ariaLabel="對戰紀錄頁"
      contentClassName="grid content-start gap-y-3"
    >
      <PageHeader title="對戰紀錄" headline="Battle Record" />

      {isPending ? (
        <Card>
          <CardContent>
            <span className="text-muted-foreground font-bold">
              正在同步對戰紀錄
            </span>
          </CardContent>
        </Card>
      ) : completedMatches.length === 0 ? (
        <Card>
          <CardHeader>
            <CardTitle>目前沒有對戰紀錄</CardTitle>
            <CardDescription>
              完成一場知識王戰後，紀錄會顯示在這裡。
            </CardDescription>
          </CardHeader>
          <CardFooter className="grid grid-cols-2 gap-2">
            <Button asChild variant="secondary">
              <Link to="/">
                <GameFeatureIcon name="home" className="size-5" /> 首頁
              </Link>
            </Button>
            <Button asChild>
              <Link to="/battle">
                <GameFeatureIcon name="battle" className="size-5" /> 開始對戰
              </Link>
            </Button>
          </CardFooter>
        </Card>
      ) : (
        <div className="grid gap-y-3">
          <ul className="grid gap-y-3">
            {completedMatches.map((match) => {
              const players = rankedPlayers(match).slice(0, 2)

              return (
                <li key={match.matchId}>
                  <Card className="py-0">
                    <CardContent className="grid gap-y-3 p-4">
                      <div className="grid grid-cols-[1fr_auto] items-start gap-3">
                        <div className="grid gap-y-1">
                          <CardTitle className="flex items-center gap-2 text-lg">
                            <GameFeatureIcon
                              name="leaderboard"
                              className="size-5"
                            />
                            {winnerName(match)}
                          </CardTitle>
                          <CardDescription className="flex flex-wrap items-center gap-x-2 gap-y-1 text-sm">
                            <span className="flex items-center gap-1">
                              <GameFeatureIcon
                                name="history"
                                className="size-4"
                              />
                              {formatDateTime(
                                match.completedAt ??
                                  match.startedAt ??
                                  match.createdAt,
                              )}
                            </span>
                            <span>{match.questionCount} 題</span>
                          </CardDescription>
                        </div>

                        <Button asChild size="sm" variant="secondary">
                          <Link
                            to="/battle/result"
                            onClick={() => storeMatchID(match.matchId)}
                          >
                            查看 <ArrowRight />
                          </Link>
                        </Button>
                      </div>

                      <Separator />

                      <ul className="grid gap-y-1">
                        {players.map((player, index) => (
                          <li
                            key={player.playerId}
                            className={cn(
                              "grid grid-cols-[32px_1fr_auto] items-center gap-2 rounded-lg px-3 py-2",
                              index === 0 ? "bg-card" : "bg-surface-raised",
                            )}
                          >
                            <PlayerAvatar
                              playerId={player.playerId}
                              nickname={player.nickname}
                              className="border-ink size-8 rounded-[11px] border"
                            />
                            <span className="font-black">
                              {player.nickname}
                            </span>
                            <span className="text-xl font-black">
                              {player.score}
                            </span>
                          </li>
                        ))}
                      </ul>
                    </CardContent>
                  </Card>
                </li>
              )
            })}
          </ul>

          <Button asChild variant="secondary">
            <Link to="/">
              <GameFeatureIcon name="home" className="size-5" /> 返回首頁
            </Link>
          </Button>
        </div>
      )}
    </GamePageShell>
  )
}
