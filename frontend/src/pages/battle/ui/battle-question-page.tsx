import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { Link, useNavigate } from "@tanstack/react-router"
import { useCallback, useEffect, useMemo, useState } from "react"
import { toast } from "sonner"

import {
  gameApi,
  type MatchChoice,
  type MatchPlayer,
  type MatchQuestionResult,
  type Sitone,
} from "@/shared/api/game"
import {
  useMatchDeadlineRefresh,
  useMatchEvents,
} from "@/features/game/use-match-events"
import { AppError } from "@/shared/api/error"
import { Button } from "@/shared/ui/button"
import { Card, CardContent } from "@/shared/ui/card"
import { GamePageShell } from "@/shared/ui/game-page-shell"
import { PlayerAvatar } from "@/shared/ui/player-avatar"
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

function secondsUntil(value: string | undefined, now: number | null) {
  if (!value || now == null) return 0
  return Math.max(0, Math.ceil((new Date(value).getTime() - now) / 1000))
}

function scoreRatio(score: number, maxScore: number) {
  if (maxScore <= 0) return 0
  return Math.max(0, Math.min(100, (score / maxScore) * 100))
}

function answerStatus(
  player: MatchPlayer | undefined,
  phase: "answering" | "revealing" | undefined,
) {
  if (!player) return "等待中"
  if (phase === "revealing") {
    return player.answeredCurrentQuestion ? "已答" : "未作答"
  }
  return player.answeredCurrentQuestion ? "已答" : "作答中"
}

function ScoreMeter({
  score,
  maxScore,
  side,
}: {
  score: number
  maxScore: number
  side: "opponent" | "self"
}) {
  const ratio = scoreRatio(score, maxScore)
  return (
    <div
      className={cn(
        "h-3 overflow-hidden rounded-full border-2",
        side === "opponent"
          ? "bg-pebble-resonate-muted border-pebble-resonate-foreground"
          : "bg-pebble-engineer-muted border-pebble-engineer-foreground",
      )}
      aria-label={`${score} / ${maxScore} 分`}
      role="meter"
      aria-valuemin={0}
      aria-valuemax={maxScore}
      aria-valuenow={score}
    >
      <div
        className={cn(
          "h-full rounded-full transition-all",
          side === "opponent" ? "bg-pebble-resonate" : "bg-pebble-engineer",
        )}
        style={{ width: `${ratio}%` }}
      />
    </div>
  )
}

function SitoneChip({ sitone }: { sitone: Sitone }) {
  return (
    <div className="grid min-w-16 justify-items-center gap-1 text-center">
      <SitoneIcon type={sitone.type} iconPath={sitone.iconPath} />
      <span className="max-w-16 truncate text-[11px] leading-tight font-black">
        {sitone.name}
      </span>
    </div>
  )
}

function SitoneLoadout({
  sitones,
  emptyLabel,
}: {
  sitones: Sitone[]
  emptyLabel: string
}) {
  if (sitones.length === 0) {
    return (
      <div className="border-border text-muted-foreground grid h-[58px] place-items-center rounded-[18px] border-2 border-dashed px-3 text-xs font-bold">
        {emptyLabel}
      </div>
    )
  }

  return (
    <div className="flex min-w-0 gap-2 overflow-x-auto pb-1">
      {sitones.map((sitone, index) => (
        <SitoneChip key={`${sitone.id}-${index}`} sitone={sitone} />
      ))}
    </div>
  )
}

function PlayerRail({
  label,
  player,
  sitones,
  side,
  phase,
}: {
  label: string
  player: MatchPlayer | undefined
  sitones: Sitone[]
  side: "opponent" | "self"
  phase: "answering" | "revealing" | undefined
}) {
  const score = player?.score ?? 0
  const maxScore = player?.maxScore ?? 0
  const meter = (
    <div className="grid gap-1">
      <div className="flex items-end justify-between gap-2">
        <span className="text-muted-foreground text-xs font-black">
          {label}
        </span>
        <span className="text-sm leading-none font-black">
          {score}
          <span className="text-muted-foreground"> / {maxScore}</span>
        </span>
      </div>
      <ScoreMeter score={score} maxScore={maxScore} side={side} />
    </div>
  )

  return (
    <section
      className={cn(
        "border-ink bg-card grid gap-2 rounded-[22px] border-2 p-3 shadow-[4px_4px_0_var(--border)]",
        side === "self" && "sticky bottom-2 z-10",
      )}
      aria-label={`${label}資訊`}
    >
      {side === "opponent" ? meter : null}
      <div className="flex items-center justify-between gap-3">
        <div className="flex min-w-0 items-center gap-2.5">
          <PlayerAvatar
            playerId={player?.playerId}
            nickname={player?.nickname ?? label}
            kind={player?.kind}
            className={cn(
              "border-ink size-11 rounded-[16px] border-2",
              side === "opponent" ? "bg-pebble-resonate" : "bg-pebble-engineer",
            )}
          />
          <div className="min-w-0">
            <div className="truncate text-lg font-black">
              {player?.nickname ?? "等待對手"}
            </div>
            <div className="text-muted-foreground text-xs font-bold">
              {answerStatus(player, phase)}
            </div>
          </div>
        </div>
        <div className="text-muted-foreground text-xs font-black whitespace-nowrap">
          {sitones.length} 顆小石
        </div>
      </div>
      <SitoneLoadout
        sitones={sitones}
        emptyLabel={side === "opponent" ? "等待對手小石" : "正在同步小石"}
      />
      {side === "self" ? meter : null}
    </section>
  )
}

function ChoiceAnswerBadges({
  result,
  choice,
}: {
  result: MatchQuestionResult | undefined
  choice: MatchChoice
}) {
  const answers =
    result?.answers.filter((answer) => answer.choice === choice) ?? []

  if (answers.length === 0) return null

  return (
    <div className="flex max-w-[132px] flex-wrap justify-end gap-1">
      {answers.map((answer) => (
        <span
          key={answer.playerId}
          className={cn(
            "border-ink grid max-w-[132px] grid-cols-[auto_minmax(0,1fr)_auto] items-center gap-1 rounded-full border-2 px-2 py-1 text-[11px] leading-none font-black",
            answer.correct ? "bg-card text-ink" : "bg-status-warning text-ink",
          )}
        >
          <PlayerAvatar
            playerId={answer.playerId}
            nickname={answer.nickname}
            className="size-4 rounded-full"
          />
          <span className="truncate">{answer.nickname}</span>
          <span>+{answer.score}</span>
        </span>
      ))}
    </div>
  )
}

export function BattleQuestionPage() {
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const [matchID] = useState(getStoredMatchID)
  const [now, setNow] = useState(() => Date.now())
  const handleMatchDeleted = useCallback(() => {
    clearStoredMatchID()
    navigate({ to: "/battle", replace: true })
  }, [navigate])
  const matchQuery = useQuery({
    queryKey: ["matches", matchID],
    queryFn: () => gameApi.getMatch(matchID),
    enabled: matchID.length > 0,
  })
  useMatchEvents(matchID, {
    enabled: matchID.length > 0,
    onDeleted: handleMatchDeleted,
  })
  const statusQuery = useQuery({
    queryKey: ["me", "status"],
    queryFn: gameApi.status,
  })
  const catalogSitonesQuery = useQuery({
    queryKey: ["catalog", "sitones"],
    queryFn: gameApi.catalogSitones,
  })
  const answerMutation = useMutation({
    mutationFn: ({
      questionID,
      choice,
    }: {
      questionID: string
      choice: MatchChoice
    }) => gameApi.answerMatch(matchID, questionID, choice),
    onSuccess: () => {
      toast.success("答案已送出")
      queryClient.invalidateQueries({ queryKey: ["matches", matchID] })
    },
    onError: (error) => {
      toast.error(error instanceof Error ? error.message : "送出答案失敗")
    },
  })

  useEffect(() => {
    const interval = window.setInterval(() => setNow(Date.now()), 500)
    return () => window.clearInterval(interval)
  }, [])

  const match = matchQuery.data
  useMatchDeadlineRefresh(matchID, match)
  const question = match?.currentQuestion
  const players = match?.players ?? []
  const currentPlayer = match?.players.find(
    (player) => player.playerId === statusQuery.data?.playerId,
  )
  const opponentPlayer = players.find(
    (player) => player.playerId !== statusQuery.data?.playerId,
  )
  const sitonesByID = useMemo(
    () =>
      new Map(
        (catalogSitonesQuery.data ?? []).map((sitone) => [sitone.id, sitone]),
      ),
    [catalogSitonesQuery.data],
  )
  const currentPlayerSitones = useMemo(
    () =>
      (currentPlayer?.sitoneIds ?? [])
        .map((sitoneID) => sitonesByID.get(sitoneID))
        .filter((sitone) => sitone != null),
    [currentPlayer?.sitoneIds, sitonesByID],
  )
  const opponentSitones = useMemo(
    () =>
      (opponentPlayer?.sitoneIds ?? [])
        .map((sitoneID) => sitonesByID.get(sitoneID))
        .filter((sitone) => sitone != null),
    [opponentPlayer?.sitoneIds, sitonesByID],
  )
  const choices = useMemo(
    () =>
      question
        ? ([
            ["A", question.choiceA],
            ["B", question.choiceB],
            ["C", question.choiceC],
            ["D", question.choiceD],
          ] as const)
        : [],
    [question],
  )
  const phase =
    match?.status === "active" ? (match.phase ?? "answering") : undefined
  const isRevealing = phase === "revealing"
  const currentResult = match?.currentQuestionResult
  const eliminatedChoices = useMemo(
    () => new Set(currentPlayer?.eliminatedChoices ?? []),
    [currentPlayer?.eliminatedChoices],
  )
  const eliminatedBy = currentPlayer?.eliminatedBy ?? []
  const displaySeconds = secondsUntil(
    isRevealing ? match?.revealEndsAt : match?.roundEndsAt,
    now,
  )
  const answered = currentPlayer?.answeredCurrentQuestion === true

  useEffect(() => {
    if (
      matchQuery.error instanceof AppError &&
      matchQuery.error.status === 404
    ) {
      clearStoredMatchID()
      navigate({ to: "/battle", replace: true })
      return
    }
    if (match?.status === "waiting") {
      navigate({ to: "/battle/room" })
    }
    if (match?.status === "completed") {
      navigate({ to: "/battle/result" })
    }
  }, [match?.status, matchQuery.error, navigate])

  if (!matchID) {
    return (
      <GamePageShell contentClassName="grid content-start gap-y-2">
        <Card>
          <CardContent className="grid gap-3">
            <h1 className="text-2xl font-bold">找不到目前對戰</h1>
            <Button asChild>
              <Link to="/battle">回到知識王大廳</Link>
            </Button>
          </CardContent>
        </Card>
      </GamePageShell>
    )
  }

  return (
    <GamePageShell contentClassName="grid min-h-svh grid-rows-[auto_minmax(0,1fr)_auto] gap-y-2 px-2">
      <PlayerRail
        label="對手"
        player={opponentPlayer}
        sitones={opponentSitones}
        side="opponent"
        phase={phase}
      />

      <div className="grid min-h-0 content-start gap-y-2">
        <Card>
          <CardContent className="grid gap-3">
            <div className="grid grid-cols-[minmax(0,1fr)_auto] items-start gap-3">
              <div className="grid min-w-0 gap-y-1">
                <span className="text-muted-foreground text-sm font-black">
                  第 {(match?.currentQuestionIndex ?? 0) + 1} /{" "}
                  {match?.questionCount ?? 0} 題
                </span>
                <span className="text-2xl leading-tight font-black">
                  {question?.prompt ?? "正在同步題目"}
                </span>
              </div>
              <div className="grid w-16 shrink-0 justify-items-center leading-none">
                <span className="w-[3ch] text-center text-4xl font-black tabular-nums">
                  {displaySeconds}
                </span>
                <span className="text-sm font-bold">
                  {isRevealing ? "揭曉" : "秒"}
                </span>
              </div>
            </div>
            <Separator />
            {!isRevealing && eliminatedBy.length > 0 ? (
              <div className="border-ink bg-pebble-spark-muted rounded-[16px] border-2 px-3 py-2 text-sm font-black">
                {eliminatedBy.join("、")} 已排除錯誤選項
              </div>
            ) : null}
            <div className="grid">
              {choices.map(([choice, label], index) => {
                const isCorrectChoice =
                  isRevealing && currentResult?.correctChoice === choice
                const isEliminated =
                  !isRevealing && eliminatedChoices.has(choice)

                return (
                  <div key={choice}>
                    <Button
                      variant="ghost"
                      className={cn(
                        "grid h-fit w-full grid-cols-[58px_minmax(0,1fr)_auto] items-center justify-start gap-3 rounded-none py-2 pl-0 disabled:opacity-100",
                        isCorrectChoice && "bg-pebble-engineer",
                        isEliminated && "bg-muted text-muted-foreground",
                      )}
                      disabled={
                        isRevealing ||
                        answered ||
                        isEliminated ||
                        answerMutation.isPending
                      }
                      onClick={() =>
                        question &&
                        answerMutation.mutate({
                          questionID: question.questionId,
                          choice,
                        })
                      }
                    >
                      <span
                        className={cn(
                          "border-accent-foreground bg-accent text-muted-foreground grid size-12 place-items-center rounded-lg border-2 text-lg font-black",
                          isCorrectChoice &&
                            "border-ink bg-pebble-engineer text-ink",
                          isEliminated && "border-muted-foreground bg-muted",
                        )}
                      >
                        {choice}
                      </span>
                      <span
                        className={cn(
                          "min-w-0 text-left text-lg leading-tight whitespace-normal",
                          isEliminated && "line-through",
                        )}
                      >
                        {label}
                      </span>
                      {isEliminated ? (
                        <span className="text-muted-foreground text-xs font-black whitespace-nowrap">
                          已排除
                        </span>
                      ) : null}
                      {isRevealing ? (
                        <ChoiceAnswerBadges
                          result={currentResult}
                          choice={choice}
                        />
                      ) : null}
                    </Button>
                    {index < choices.length - 1 && <Separator />}
                  </div>
                )
              })}
            </div>
          </CardContent>
        </Card>
      </div>

      <PlayerRail
        label="自己"
        player={currentPlayer}
        sitones={currentPlayerSitones}
        side="self"
        phase={phase}
      />
    </GamePageShell>
  )
}
