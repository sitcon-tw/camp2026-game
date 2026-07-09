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

const CAMPUS_ROAD_PATHS = [
  "M44 104 C84 84 128 76 168 76 C238 76 294 90 360 132",
  "M68 350 C130 320 176 292 216 260 C252 226 284 176 356 138",
  "M92 104 L92 300 M142 86 L142 342 M230 64 L230 338 M326 76 L326 356",
  "M72 160 C108 146 158 146 204 158 C250 170 304 166 354 146",
  "M76 238 C130 224 192 226 246 238 C284 246 318 244 360 226",
]

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
      aria-label="小隊領地地圖（陽明交大光復校區校園圖）"
      className="w-full"
    >
      <rect width="420" height="420" rx="22" className="fill-[#f6efd8]" />
      <path
        d="M36 82 L92 26 L206 20 L316 28 L390 68 L392 330 L342 386 L232 402 L102 382 L36 328 Z"
        className="fill-[#cfe2a3] stroke-ink stroke-[2.5]"
      />
      {CAMPUS_ROAD_PATHS.map((path) => (
        <path
          key={path}
          d={path}
          className="fill-none stroke-[#fff4cc] stroke-[3.5] opacity-45"
          strokeLinecap="round"
          pointerEvents="none"
        />
      ))}
      <ellipse
        cx="82"
        cy="92"
        rx="34"
        ry="54"
        className="fill-[#d8e8bc] stroke-[#b65f44] stroke-[3] opacity-65"
        pointerEvents="none"
      />

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
                "stroke-ink cursor-pointer fill-opacity-90 stroke-[2.5] transition-opacity hover:fill-opacity-100 focus-visible:fill-opacity-100",
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

      <path
        d="M42 138 C66 128 100 132 118 154 C136 178 130 218 104 236 C78 254 44 244 34 214 C22 180 20 152 42 138 Z"
        className="fill-[#71bfd1] stroke-[#1e6476] stroke-[2] opacity-40"
        pointerEvents="none"
      />
      <text
        x="70"
        y="190"
        textAnchor="middle"
        className="fill-ink text-[8px] font-black opacity-50"
        pointerEvents="none"
      >
        工二湖
      </text>
      <path
        d="M312 122 C342 102 380 112 388 146 C398 188 356 210 322 196 C290 182 284 140 312 122 Z"
        className="fill-[#71bfd1] stroke-[#1e6476] stroke-[2] opacity-40"
        pointerEvents="none"
      />
      <text
        x="344"
        y="160"
        textAnchor="middle"
        className="fill-ink text-[8px] font-black opacity-50"
        pointerEvents="none"
      >
        竹湖
      </text>
      <path
        d="M36 82 L92 26 L206 20 L316 28 L390 68 L392 330 L342 386 L232 402 L102 382 L36 328 Z"
        className="fill-none stroke-ink stroke-[3]"
        pointerEvents="none"
      />

      {TAIWAN_TERRITORY_REGIONS.map((region) => {
        const team = teamOfRegion(region)
        const isMine = team != null && team.teamId === myTeamId
        const label = region.isBoss ? region.dorm : (team?.name ?? region.dorm)
        const width = Math.max(46, label.length * 10 + 18)

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
              height={28}
              rx={10}
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
              y={15}
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
