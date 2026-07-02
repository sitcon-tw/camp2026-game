import { useQuery } from "@tanstack/react-query"
import { Link } from "@tanstack/react-router"
import { ChevronDown } from "lucide-react"
import { useEffect, useMemo, useState } from "react"

import { useMatchEvents } from "@/features/game/use-match-events"
import {
  gameApi,
  type MatchQuestionResult,
  type Sitone,
} from "@/shared/api/game"
import { Button } from "@/shared/ui/button"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/shared/ui/card"
import { GameFeatureIcon } from "@/shared/ui/game-feature-icon"
import { GamePageShell } from "@/shared/ui/game-page-shell"
import { PageHeader } from "@/shared/ui/page-header"
import { PlayerAvatar } from "@/shared/ui/player-avatar"
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@/shared/ui/collapsible"
import { Separator } from "@/shared/ui/separator"
import { SitoneIcon } from "@/shared/ui/sitone-icon"
import { cn } from "@/shared/utils"

function getStoredMatchID() {
  if (typeof window === "undefined") return ""
  return window.localStorage.getItem("camp2026.currentMatchId") ?? ""
}

function clearStoredMatchID() {
  if (typeof window !== "undefined") {
    window.localStorage.removeItem("camp2026.currentMatchId")
  }
}

function choiceText(result: MatchQuestionResult, choice?: string) {
  switch (choice) {
    case "A":
      return result.choiceA
    case "B":
      return result.choiceB
    case "C":
      return result.choiceC
    case "D":
      return result.choiceD
    default:
      return "未作答"
  }
}

function isSitone(sitone: Sitone | undefined): sitone is Sitone {
  return Boolean(sitone)
}

export function BattleResultPage() {
  const [matchID] = useState(getStoredMatchID)
  const { data: match, isPending } = useQuery({
    queryKey: ["matches", matchID],
    queryFn: () => gameApi.getMatch(matchID),
    enabled: matchID.length > 0,
  })
  const { data: catalogSitones = [] } = useQuery({
    queryKey: ["catalog", "sitones"],
    queryFn: gameApi.catalogSitones,
    enabled: matchID.length > 0,
  })
  useMatchEvents(matchID, {
    enabled: matchID.length > 0 && match?.status !== "completed",
  })
  const sitonesByID = useMemo(
    () => new Map(catalogSitones.map((sitone) => [sitone.id, sitone])),
    [catalogSitones],
  )
  const players = match?.players ?? []
  const sortedPlayers = [...players].sort(
    (a, b) => (b.score ?? 0) - (a.score ?? 0),
  )
  const hasClearWinner =
    match?.status === "completed" &&
    sortedPlayers.length >= 2 &&
    (sortedPlayers[0].score ?? 0) > (sortedPlayers[1].score ?? 0)
  const winner = hasClearWinner ? sortedPlayers[0] : undefined
  const rewardedPlayers = players.filter(
    (player) => (player.openPowerReward ?? 0) > 0,
  )
  const dropPlayers = players.filter((player) => player.materialDrop != null)

  useEffect(() => {
    if (match?.status === "completed") {
      clearStoredMatchID()
    }
  }, [match?.status])

  return (
    <GamePageShell contentClassName="grid content-start gap-y-2">
      <PageHeader
        title="對戰結果"
        headline="Battle Result"
        backTo="/"
        onBack={clearStoredMatchID}
      />

      <Card>
        <CardContent className="grid gap-y-4">
          <span className="text-center text-4xl font-bold">
            {isPending
              ? "同步結果"
              : match?.status === "completed"
                ? winner
                  ? `${winner.nickname} 勝利`
                  : players.length >= 2
                    ? "平手"
                    : "對戰結束"
                : "對戰尚未結束"}
          </span>
          <div className="flex items-center gap-x-4">
            {players.slice(0, 2).map((player) => {
              const playerSitones = player.sitoneIds
                .map((sitoneID) => sitonesByID.get(sitoneID))
                .filter(isSitone)

              return (
                <Card
                  key={player.playerId}
                  className={cn(
                    "bg-accent flex-1",
                    winner?.playerId === player.playerId
                      ? "text-status-success"
                      : "text-muted-foreground",
                  )}
                >
                  <CardContent className="grid gap-y-2">
                    <PlayerAvatar
                      playerId={player.playerId}
                      nickname={player.nickname}
                      kind={player.kind}
                      className="border-ink mx-auto size-14 rounded-[20px] border-2"
                    />
                    <span className="text-center">{player.nickname}</span>
                    <span className="text-center text-4xl font-bold">
                      {player.score ?? 0}
                    </span>
                    <span className="text-center text-xs font-bold">
                      {player.sitoneIds.length} 顆小石
                    </span>
                    {playerSitones.length > 0 ? (
                      <div
                        className="flex justify-center gap-1 overflow-hidden"
                        aria-label={`${player.nickname} 本場小石`}
                      >
                        {playerSitones.map((sitone, index) => (
                          <SitoneIcon
                            key={`${sitone.id}-${index}`}
                            type={sitone.type}
                            iconPath={sitone.iconPath}
                            className="size-8 rounded-[11px]"
                          />
                        ))}
                      </div>
                    ) : null}
                    {(player.answerScoreBonusPercent ?? 0) > 0 ? (
                      <span className="text-center text-xs font-bold">
                        答題加成 +{player.answerScoreBonusPercent}%
                      </span>
                    ) : null}
                  </CardContent>
                </Card>
              )
            })}
          </div>
        </CardContent>
      </Card>

      {rewardedPlayers.length > 0 ? (
        <Card>
          <CardHeader>
            <CardTitle>獲得獎勵</CardTitle>
            <CardDescription>本場對戰的開源力獎勵會收入帳號。</CardDescription>
          </CardHeader>
          <CardContent className="grid grid-cols-[64px_1fr] gap-x-4">
            <div className="bg-accent border-secondary-foreground rounded-lg border-2 p-2">
              <GameFeatureIcon name="backpack" className="size-10 rounded-lg" />
            </div>
            <div className="grid gap-y-2 text-lg">
              {rewardedPlayers.map((player) => (
                <div
                  key={player.playerId}
                  className="grid grid-cols-[28px_1fr_auto] items-center gap-x-2"
                >
                  <PlayerAvatar
                    playerId={player.playerId}
                    nickname={player.nickname}
                    kind={player.kind}
                    className="border-ink size-7 rounded-[10px] border"
                  />
                  <span>{player.nickname}</span>
                  <span className="font-bold">
                    +{player.openPowerReward ?? 0} 開源力
                  </span>
                  {(player.openPowerBonusPercent ?? 0) > 0 ? (
                    <span className="text-muted-foreground col-span-2 col-start-2 text-sm font-bold">
                      基礎 {player.baseOpenPowerReward ?? 0}，加成 +
                      {player.openPowerBonusPercent}%
                    </span>
                  ) : null}
                </div>
              ))}
            </div>
          </CardContent>
        </Card>
      ) : null}

      {dropPlayers.length > 0 ? (
        <Card>
          <CardHeader>
            <CardTitle>小石掉落</CardTitle>
            <CardDescription>勝敗雙方都會依掉落率結算一次。</CardDescription>
          </CardHeader>
          <CardContent className="grid gap-2">
            {dropPlayers.map((player) => {
              const drop = player.materialDrop
              if (!drop) return null
              return (
                <div
                  key={player.playerId}
                  className="border-border grid grid-cols-[28px_1fr_auto] items-center gap-2 border-b pb-2 last:border-b-0 last:pb-0"
                >
                  <PlayerAvatar
                    playerId={player.playerId}
                    nickname={player.nickname}
                    kind={player.kind}
                    className="border-ink size-7 rounded-[10px] border"
                  />
                  <span className="font-bold">{player.nickname}</span>
                  <span className="text-muted-foreground text-sm font-bold">
                    {drop.dropRate}%
                  </span>
                  <span className="col-span-2 col-start-2 text-sm font-black">
                    {drop.dropped
                      ? `獲得 ${drop.sitoneName ?? drop.sitoneId ?? drop.itemName ?? drop.itemId} x${drop.quantity ?? 1}`
                      : "沒有掉落小石"}
                  </span>
                </div>
              )
            })}
          </CardContent>
        </Card>
      ) : null}

      <Separator className="my-2" />

      <span className="text-center text-2xl font-bold">逐題解析</span>
      {(match?.results ?? []).map((result, index) => (
        <Card key={result.questionId} className="overflow-hidden py-0">
          <CardContent className="p-0">
            <Collapsible>
              <CollapsibleTrigger asChild>
                <Button
                  variant="ghost"
                  className="group grid h-auto w-full grid-cols-[64px_auto_minmax(0,1fr)_auto] items-center gap-4 rounded-[22px] px-4 py-4 text-left shadow-none"
                >
                  <div className="grid justify-items-center gap-1">
                    <span className="text-muted-foreground text-[11px] leading-none font-black tracking-[0.08em]">
                      題數
                    </span>
                    <span className="border-ink bg-accent grid size-11 place-items-center rounded-lg border-2 text-lg font-black">
                      {index + 1}
                    </span>
                  </div>
                  <Separator orientation="vertical" className="h-12" />
                  <div className="min-w-0">
                    <span className="text-muted-foreground text-[11px] leading-none font-black tracking-[0.08em]">
                      名稱
                    </span>
                    <span className="mt-1 block text-base leading-tight font-black whitespace-normal">
                      {result.prompt}
                    </span>
                  </div>
                  <ChevronDown className="justify-self-end transition group-data-[state=open]:rotate-180" />
                </Button>
              </CollapsibleTrigger>
              <CollapsibleContent className="px-4 pb-4">
                <div className="border-ink bg-card grid gap-y-3 rounded-[20px] border-2 px-4 py-4">
                  <div className="grid gap-y-1">
                    <span className="text-muted-foreground text-sm font-bold">
                      正確答案
                    </span>
                    <span className="border-ink bg-pebble-engineer text-ink block rounded-lg border-2 px-3 py-2 text-base font-black">
                      {result.correctChoice}.{" "}
                      {choiceText(result, result.correctChoice)}
                    </span>
                  </div>
                  {result.answers.map((answer) => (
                    <div
                      key={answer.playerId}
                      className="grid gap-y-1 border-t pt-2"
                    >
                      <span className="text-muted-foreground inline-flex items-center gap-2 text-sm font-bold">
                        <PlayerAvatar
                          playerId={answer.playerId}
                          nickname={answer.nickname}
                          className="border-ink size-6 rounded-[9px] border"
                        />
                        {answer.nickname}
                      </span>
                      {answer.correct && answer.bonusScore > 0 ? (
                        <span className="text-muted-foreground text-xs font-bold">
                          基礎 {answer.baseScore} + 加成 {answer.bonusScore}
                        </span>
                      ) : null}
                      <span
                        className={cn(
                          "border-ink text-ink block rounded-lg border-2 px-3 py-2 text-base font-black",
                          answer.correct
                            ? "bg-pebble-engineer"
                            : "bg-status-warning",
                        )}
                      >
                        {answer.choice
                          ? `${answer.choice}. ${choiceText(
                              result,
                              answer.choice,
                            )}`
                          : "未作答"}
                      </span>
                    </div>
                  ))}
                  <Separator />
                  <span>{result.explanation}</span>
                </div>
              </CollapsibleContent>
            </Collapsible>
          </CardContent>
        </Card>
      ))}

      {match?.results.length === 0 ? (
        <Card>
          <CardContent>
            <span className="text-muted-foreground font-bold">
              結果會在對戰完成後顯示。
            </span>
          </CardContent>
        </Card>
      ) : null}

      <Separator className="my-2" />
      <Button asChild onClick={clearStoredMatchID}>
        <Link to="/">
          <GameFeatureIcon name="home" className="size-4" /> 返回首頁
        </Link>
      </Button>
    </GamePageShell>
  )
}
