import type { TerritoryStandingTeam } from "@/shared/api/territory"
import { tierMeta } from "@/shared/lib/territory-labels"

import {
  TAIWAN_MAP_VIEW_BOX,
  TAIWAN_TERRITORY_REGIONS,
  type TerritoryRegion,
} from "../model/taiwan-map"

type TerritoryMapProps = {
  teams: TerritoryStandingTeam[]
  myTeamId: string
  bossUnderAttack?: boolean
  onSelectRegion: (
    region: TerritoryRegion,
    team: TerritoryStandingTeam | undefined,
  ) => void
}

function regionFillClassName(
  region: TerritoryRegion,
  team: TerritoryStandingTeam | undefined,
) {
  if (region.isBoss) return "fill-ink"
  if (!team) return "fill-muted"
  if (team.tier === "boss") return "fill-ink"
  return tierMeta(team.tier).fillClassName
}

export function TerritoryMap({
  teams,
  myTeamId,
  bossUnderAttack = false,
  onSelectRegion,
}: TerritoryMapProps) {
  const playerTeams = teams.filter((team) => !team.isBoss)
  const teamOfRegion = (region: TerritoryRegion) => {
    if (region.isBoss) return teams.find((team) => team.isBoss)
    return playerTeams[region.slot]
  }

  return (
    <svg
      viewBox={TAIWAN_MAP_VIEW_BOX}
      role="group"
      aria-label="小隊領地地圖（以臺灣西部縣市象徵十一處宿舍領地）"
      className="w-full"
    >
      {TAIWAN_TERRITORY_REGIONS.map((region) => {
        const team = teamOfRegion(region)
        const isMine = team != null && team.teamId === myTeamId
        const fillClassName = regionFillClassName(region, team)

        return (
          <g key={region.id}>
            <path
              d={region.path}
              role="button"
              tabIndex={0}
              aria-label={
                region.isBoss
                  ? `${region.dorm}（研三舍 Boss 領地）`
                  : `${region.dorm} · ${team?.name ?? "未分配"}${
                      isMine ? "（我的小隊）" : ""
                    }`
              }
              className={[
                fillClassName,
                "stroke-ink cursor-pointer stroke-[1.5] transition-opacity hover:opacity-80 focus-visible:opacity-80",
                region.isBoss && bossUnderAttack
                  ? "fill-destructive animate-pulse"
                  : "",
                isMine ? "stroke-[3]" : "",
              ].join(" ")}
              onClick={() => onSelectRegion(region, team)}
              onKeyDown={(event) => {
                if (event.key === "Enter" || event.key === " ") {
                  event.preventDefault()
                  onSelectRegion(region, team)
                }
              }}
            />
            {isMine ? (
              <path
                d={region.path}
                className="fill-none stroke-[color:var(--power)] stroke-[5.5]"
                strokeLinejoin="round"
                opacity={0.55}
                pointerEvents="none"
              />
            ) : null}
          </g>
        )
      })}

      {TAIWAN_TERRITORY_REGIONS.map((region) => {
        const team = teamOfRegion(region)
        const isMine = team != null && team.teamId === myTeamId
        const label = region.isBoss ? region.dorm : (team?.name ?? region.dorm)
        const width = Math.max(34, label.length * 10 + 14)

        return (
          <g
            key={`${region.id}-label`}
            pointerEvents="none"
            transform={`translate(${region.labelX}, ${region.labelY})`}
          >
            <rect
              x={-width / 2}
              y={-9}
              width={width}
              height={26}
              rx={9}
              className={
                region.isBoss
                  ? "fill-ink stroke-destructive"
                  : isMine
                    ? "fill-[color:var(--power)] stroke-ink"
                    : "fill-card stroke-ink"
              }
              strokeWidth={1.5}
            />
            <text
              textAnchor="middle"
              y={4}
              className={[
                "text-[10px] font-black",
                region.isBoss
                  ? "fill-[color:var(--primary-foreground)]"
                  : "fill-ink",
              ].join(" ")}
            >
              {region.isBoss ? "👹 " : ""}
              {label}
            </text>
            <text
              textAnchor="middle"
              y={14}
              className={
                region.isBoss
                  ? "fill-[color:var(--primary-foreground)] text-[7px] font-bold opacity-70"
                  : "fill-ink text-[7px] font-bold opacity-60"
              }
            >
              {region.dorm === label ? region.county : region.dorm}
            </text>
          </g>
        )
      })}
    </svg>
  )
}
