import { ArrowDown, ArrowRight, ArrowUp, Trophy } from "lucide-react"

import type {
  FrontLeaderboardEntry,
  FrontMapMode,
  FrontTeamState,
} from "@/shared/api/game"
import { Badge } from "@/shared/ui/badge"
import { Card, CardContent, CardHeader, CardTitle } from "@/shared/ui/card"
import { cn } from "@/shared/utils"

import {
  getTeamName,
  getTeamTone,
  getTerritoryTeamColor,
} from "./front-map-style"

type FrontLeaderboardProps = {
  entries: FrontLeaderboardEntry[]
  teams: FrontTeamState[]
  myTeamId?: string
  mapMode?: FrontMapMode
}

export function FrontLeaderboard({
  entries,
  teams,
  myTeamId,
  mapMode = "node",
}: FrontLeaderboardProps) {
  const topEntries = [...entries]
    .sort((first, second) => first.rank - second.rank)
    .slice(0, 5)

  return (
    <Card className="gap-3">
      <CardHeader className="px-4">
        <CardTitle className="flex items-center gap-2 text-base font-black">
          <Trophy className="text-primary size-4" aria-hidden />
          小隊戰況
        </CardTitle>
      </CardHeader>
      <CardContent className="grid gap-2 px-3 pb-4">
        {topEntries.length > 0 ? (
          topEntries.map((entry) => (
            <LeaderboardRow
              key={entry.teamId}
              entry={entry}
              teams={teams}
              current={entry.current || entry.teamId === myTeamId}
              mapMode={mapMode}
            />
          ))
        ) : (
          <div className="text-muted-foreground px-1 py-2 text-sm font-bold">
            尚無排名資料
          </div>
        )}
      </CardContent>
    </Card>
  )
}

function LeaderboardRow({
  entry,
  teams,
  current,
  mapMode,
}: {
  entry: FrontLeaderboardEntry
  teams: FrontTeamState[]
  current: boolean
  mapMode: FrontMapMode
}) {
  const tone = getTeamTone(entry.teamId, teams)
  const territoryColor = getTerritoryTeamColor(entry.teamId, teams)
  const teamColor = teams.find((team) => team.teamId === entry.teamId)?.color
  const rankDelta = getRankDelta(entry.rank, entry.previousRank)

  return (
    <div
      className={cn(
        "border-border bg-card grid grid-cols-[34px_1fr_auto] items-center gap-2 rounded-[1rem] border px-3 py-2",
        current && "border-ink bg-surface-raised",
      )}
    >
      <div className="text-sm font-black">#{entry.rank}</div>
      <div className="min-w-0">
        <div className="flex min-w-0 items-center gap-2">
          <span
            className={cn(
              "size-2.5 shrink-0 rounded-full",
              mapMode === "territory_grid"
                ? (territoryColor?.dot ?? "bg-muted-foreground")
                : tone.dot,
            )}
            style={
              mapMode === "territory_grid" && teamColor
                ? { backgroundColor: teamColor }
                : undefined
            }
            aria-hidden
          />
          <span className="truncate text-sm font-black">
            {entry.teamName ?? getTeamName(entry.teamId, teams)}
          </span>
          {current ? <Badge variant="secondary">你的隊伍</Badge> : null}
        </div>
        <div className="text-muted-foreground mt-1 flex flex-wrap gap-x-3 gap-y-1 text-xs font-bold">
          <span>
            {entry.controlledCells}{" "}
            {mapMode === "territory_grid" ? "格領土" : "節點"}
          </span>
          <span>{entry.repairedEvents} 修復</span>
          <span>{entry.rescuedSitones} 救援</span>
        </div>
      </div>
      <div className="flex flex-col items-end gap-1">
        <span className="text-sm font-black">{entry.score}</span>
        <RankDeltaBadge delta={rankDelta} />
      </div>
    </div>
  )
}

function getRankDelta(rank: number, previousRank: number | undefined) {
  if (!previousRank) return 0

  return previousRank - rank
}

function RankDeltaBadge({ delta }: { delta: number }) {
  if (delta > 0) {
    return (
      <Badge variant="secondary" className="gap-1 px-2">
        <ArrowUp className="size-3" aria-hidden />
        {delta}
      </Badge>
    )
  }

  if (delta < 0) {
    return (
      <Badge variant="outline" className="gap-1 px-2">
        <ArrowDown className="size-3" aria-hidden />
        {Math.abs(delta)}
      </Badge>
    )
  }

  return (
    <Badge variant="outline" className="gap-1 px-2">
      <ArrowRight className="size-3" aria-hidden />0
    </Badge>
  )
}
