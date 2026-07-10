import type { FrontTeamState } from "@/shared/api/game"
import type { PebbleTone } from "@/shared/config/color-palette"

type TeamToneClasses = {
  key: PebbleTone
  dot: string
  node: string
  nodeMuted: string
  text: string
}

type TerritoryTeamColor = {
  dot: string
  cssVariable: string
}

const teamTones: TeamToneClasses[] = [
  {
    key: "explore",
    dot: "bg-pebble-explore",
    node: "fill-pebble-explore",
    nodeMuted: "fill-pebble-explore-muted",
    text: "text-pebble-explore-foreground",
  },
  {
    key: "spark",
    dot: "bg-pebble-spark",
    node: "fill-pebble-spark",
    nodeMuted: "fill-pebble-spark-muted",
    text: "text-pebble-spark-foreground",
  },
  {
    key: "resonate",
    dot: "bg-pebble-resonate",
    node: "fill-pebble-resonate",
    nodeMuted: "fill-pebble-resonate-muted",
    text: "text-pebble-resonate-foreground",
  },
  {
    key: "engineer",
    dot: "bg-pebble-engineer",
    node: "fill-pebble-engineer",
    nodeMuted: "fill-pebble-engineer-muted",
    text: "text-pebble-engineer-foreground",
  },
  {
    key: "play",
    dot: "bg-pebble-play",
    node: "fill-pebble-play",
    nodeMuted: "fill-pebble-play-muted",
    text: "text-pebble-play-foreground",
  },
]

const territoryTeamColors: TerritoryTeamColor[] = [
  { dot: "bg-pebble-explore", cssVariable: "--pebble-explore" },
  { dot: "bg-pebble-spark", cssVariable: "--pebble-spark" },
  { dot: "bg-pebble-resonate", cssVariable: "--pebble-resonate" },
  { dot: "bg-pebble-engineer", cssVariable: "--pebble-engineer" },
  { dot: "bg-pebble-play", cssVariable: "--pebble-play" },
  { dot: "bg-primary", cssVariable: "--primary" },
  { dot: "bg-moss", cssVariable: "--moss" },
  { dot: "bg-destructive", cssVariable: "--destructive" },
  { dot: "bg-ink", cssVariable: "--ink" },
]

export const neutralTeamTone: TeamToneClasses = {
  key: "spark",
  dot: "bg-muted-foreground",
  node: "fill-muted",
  nodeMuted: "fill-muted",
  text: "text-muted-foreground",
}

export function getTeamTone(
  teamID: string | undefined,
  teams: FrontTeamState[],
) {
  if (!teamID) return neutralTeamTone

  const index = teams.findIndex((team) => team.teamId === teamID)

  if (index < 0) return neutralTeamTone

  return teamTones[index % teamTones.length]
}

export function getTeamName(
  teamID: string | undefined,
  teams: FrontTeamState[],
) {
  if (!teamID) return "中立"

  return (
    teams.find((team) => team.teamId === teamID)?.name ??
    `小隊 ${teamID.slice(0, 4)}`
  )
}

export function getTerritoryTeamColor(
  teamID: string | undefined,
  teams: FrontTeamState[],
) {
  if (!teamID) return null

  const index = teams.findIndex((team) => team.teamId === teamID)
  if (index < 0) return null

  return territoryTeamColors[index % territoryTeamColors.length]
}
