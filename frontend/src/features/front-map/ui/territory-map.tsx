import type { KeyboardEvent } from "react"

import type {
  FrontCell,
  FrontMapEvent,
  FrontTeamState,
} from "@/shared/api/game"
import { Card, CardContent, CardHeader, CardTitle } from "@/shared/ui/card"
import {
  Empty,
  EmptyDescription,
  EmptyHeader,
  EmptyTitle,
} from "@/shared/ui/empty"
import { cn } from "@/shared/utils"

import { getTeamName, getTeamTone } from "./front-map-style"

type TerritoryMapProps = {
  cells: FrontCell[]
  teams: FrontTeamState[]
  activeEvents: FrontMapEvent[]
  selectedCellId: string | null
  onSelectCell: (cellID: string) => void
}

const cellScale = 84
const mapPadding = 52
const nodeRadius = 22

export function TerritoryMap({
  cells,
  teams,
  activeEvents,
  selectedCellId,
  onSelectCell,
}: TerritoryMapProps) {
  if (cells.length === 0) {
    return (
      <Card>
        <CardContent className="px-4">
          <Empty className="min-h-[260px] border-2 border-dashed">
            <EmptyHeader>
              <EmptyTitle>目前沒有地圖資料</EmptyTitle>
              <EmptyDescription>Front snapshot 尚未回傳節點。</EmptyDescription>
            </EmptyHeader>
          </Empty>
        </CardContent>
      </Card>
    )
  }

  const layout = getMapLayout(cells)
  const cellByID = new Map(cells.map((cell) => [cell.id, cell]))
  const eventCellIDs = new Set(
    activeEvents
      .map((event) => event.cellId)
      .filter((cellID): cellID is string => Boolean(cellID)),
  )

  return (
    <Card className="gap-3">
      <CardHeader className="px-4">
        <CardTitle className="text-base font-black">戰線地圖</CardTitle>
      </CardHeader>
      <CardContent className="px-3 pb-4">
        <svg
          viewBox={`0 0 ${layout.width} ${layout.height}`}
          className="bg-surface-raised border-ink aspect-[4/3] w-full rounded-[1.25rem] border-2"
          role="img"
          aria-label="開源戰線地圖"
        >
          <g aria-hidden="true">
            {cells.flatMap((cell) =>
              cell.neighborIds
                .filter((neighborID) => cell.id < neighborID)
                .map((neighborID) => {
                  const neighbor = cellByID.get(neighborID)
                  if (!neighbor) return null

                  const from = getCellPoint(cell, layout)
                  const to = getCellPoint(neighbor, layout)

                  return (
                    <line
                      key={`${cell.id}-${neighborID}`}
                      x1={from.x}
                      y1={from.y}
                      x2={to.x}
                      y2={to.y}
                      className="stroke-border"
                      strokeWidth="5"
                      strokeLinecap="round"
                    />
                  )
                }),
            )}
          </g>

          {cells.map((cell) => {
            const point = getCellPoint(cell, layout)
            const selected = selectedCellId === cell.id
            const tone = getTeamTone(cell.ownerTeamId, teams)
            const pressure = totalPressure(cell)
            const hasEvent = eventCellIDs.has(cell.id)
            const label = cell.name ?? cell.id

            return (
              <g
                key={cell.id}
                role="button"
                tabIndex={0}
                aria-pressed={selected}
                aria-label={`${label}，${getTeamName(cell.ownerTeamId, teams)}，控制 ${cell.control}`}
                className="cursor-pointer outline-none"
                onClick={() => onSelectCell(cell.id)}
                onKeyDown={(event) =>
                  handleNodeKeyDown(event, () => onSelectCell(cell.id))
                }
              >
                {pressure > 0 ? (
                  <circle
                    cx={point.x}
                    cy={point.y}
                    r={nodeRadius + 8}
                    className="fill-destructive/20"
                  />
                ) : null}
                {hasEvent ? (
                  <circle
                    cx={point.x}
                    cy={point.y}
                    r={nodeRadius + 12}
                    className="fill-primary/20"
                  />
                ) : null}
                <circle
                  cx={point.x}
                  cy={point.y}
                  r={nodeRadius}
                  className={cn(
                    cell.ownerTeamId ? tone.node : tone.nodeMuted,
                    "stroke-ink transition-transform",
                    selected && "stroke-primary",
                  )}
                  strokeWidth={selected ? 5 : 3}
                />
                <circle
                  cx={point.x}
                  cy={point.y}
                  r={Math.max(
                    5,
                    (nodeRadius * clampPercent(cell.defense)) / 100,
                  )}
                  className="fill-card/70"
                />
                <text
                  x={point.x}
                  y={point.y + 4}
                  textAnchor="middle"
                  className="fill-ink text-[11px] font-black select-none"
                >
                  {clampPercent(cell.control)}
                </text>
                <text
                  x={point.x}
                  y={point.y + nodeRadius + 18}
                  textAnchor="middle"
                  className="fill-foreground text-[9px] font-black select-none"
                >
                  {compactLabel(label)}
                </text>
              </g>
            )
          })}
        </svg>
      </CardContent>
    </Card>
  )
}

function getMapLayout(cells: FrontCell[]) {
  const minX = Math.min(...cells.map((cell) => cell.x))
  const maxX = Math.max(...cells.map((cell) => cell.x))
  const minY = Math.min(...cells.map((cell) => cell.y))
  const maxY = Math.max(...cells.map((cell) => cell.y))

  return {
    minX,
    minY,
    width: (maxX - minX) * cellScale + mapPadding * 2,
    height: (maxY - minY) * cellScale + mapPadding * 2 + 16,
  }
}

function getCellPoint(
  cell: FrontCell,
  layout: ReturnType<typeof getMapLayout>,
) {
  return {
    x: (cell.x - layout.minX) * cellScale + mapPadding,
    y: (cell.y - layout.minY) * cellScale + mapPadding,
  }
}

function totalPressure(cell: FrontCell) {
  return Object.values(cell.pressureByTeam).reduce(
    (sum, value) => sum + value,
    0,
  )
}

function clampPercent(value: number) {
  return Math.max(0, Math.min(100, Math.round(value)))
}

function compactLabel(label: string) {
  return label.length > 6 ? `${label.slice(0, 6)}...` : label
}

function handleNodeKeyDown(
  event: KeyboardEvent<SVGGElement>,
  select: () => void,
) {
  if (event.key !== "Enter" && event.key !== " ") return

  event.preventDefault()
  select()
}
