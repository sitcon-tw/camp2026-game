import { useEffect, useRef, useState } from "react"

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

import { getTeamName, getTeamTone } from "./front-map-style"

type TerritoryMapProps = {
  cells: FrontCell[]
  teams: FrontTeamState[]
  activeEvents: FrontMapEvent[]
  selectedCellId: string | null
  onSelectCell: (cellID: string) => void
}

type CanvasPoint = {
  x: number
  y: number
}

type CanvasLayout = {
  width: number
  height: number
  nodeRadius: number
  hitRadius: number
  labelOffset: number
  points: Map<string, CanvasPoint>
}

type CanvasColors = {
  border: string
  card: string
  destructive: string
  foreground: string
  ink: string
  muted: string
  primary: string
  surfaceRaised: string
}

const fallbackLayout: CanvasLayout = {
  width: 360,
  height: 520,
  nodeRadius: 30,
  hitRadius: 44,
  labelOffset: 50,
  points: new Map(),
}

const fontFamily =
  '"GenSenRounded TW", "Noto Sans TC", "PingFang TC", "Microsoft JhengHei", ui-sans-serif, system-ui, sans-serif'

const labelByCellID: Record<string, string> = {
  center_a: "中央",
  challenge_a: "挑戰",
  course_a: "課程",
  frontier_a: "前線",
  repair_a: "修復",
  rescue_a: "救援",
  resource_a: "資源",
  system_a: "系統",
}

export function TerritoryMap({
  cells,
  teams,
  activeEvents,
  selectedCellId,
  onSelectCell,
}: TerritoryMapProps) {
  const canvasRef = useRef<HTMLCanvasElement>(null)
  const frameRef = useRef<HTMLDivElement>(null)
  const [layout, setLayout] = useState<CanvasLayout>(fallbackLayout)

  useEffect(() => {
    const frame = frameRef.current
    if (!frame) return

    const updateLayout = () => {
      const width = frame.clientWidth
      const height = frame.clientHeight
      if (width <= 0 || height <= 0) return

      setLayout(buildCanvasLayout(cells, width, height))
    }

    updateLayout()
    const observer = new ResizeObserver(updateLayout)
    observer.observe(frame)

    return () => observer.disconnect()
  }, [cells])

  useEffect(() => {
    const canvas = canvasRef.current
    if (!canvas) return

    drawMap(canvas, layout, cells, teams, activeEvents, selectedCellId)
  }, [activeEvents, cells, layout, selectedCellId, teams])

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

  return (
    <Card className="-mx-4 w-[calc(100%+2rem)] gap-3 overflow-hidden rounded-none border-x-0 py-4 shadow-none">
      <CardHeader className="px-4">
        <CardTitle className="text-2xl font-black">戰線地圖</CardTitle>
      </CardHeader>
      <CardContent className="px-0 pb-0">
        <div
          ref={frameRef}
          className="bg-surface-raised border-ink relative h-[520px] w-full overflow-hidden border-y-2"
        >
          <canvas
            ref={canvasRef}
            className="absolute inset-0 size-full"
            role="img"
            aria-label="開源戰線地圖"
          />
          <div className="absolute inset-0" aria-label="戰線節點">
            {cells.map((cell) => {
              const point = layout.points.get(cell.id)
              if (!point) return null

              const label = cellLabel(cell)
              const hitRadius = layout.hitRadius

              return (
                <button
                  key={cell.id}
                  type="button"
                  aria-label={`${label}，${getTeamName(cell.ownerTeamId, teams)}，控制 ${clampPercent(cell.control)}`}
                  aria-pressed={selectedCellId === cell.id}
                  className="focus-visible:outline-power absolute rounded-full opacity-0 focus-visible:opacity-100 focus-visible:outline-3 focus-visible:outline-offset-2"
                  style={{
                    height: hitRadius * 2,
                    left: point.x - hitRadius,
                    top: point.y - hitRadius,
                    width: hitRadius * 2,
                  }}
                  onClick={() => onSelectCell(cell.id)}
                />
              )
            })}
          </div>
        </div>
      </CardContent>
    </Card>
  )
}

function drawMap(
  canvas: HTMLCanvasElement,
  layout: CanvasLayout,
  cells: FrontCell[],
  teams: FrontTeamState[],
  activeEvents: FrontMapEvent[],
  selectedCellID: string | null,
) {
  const context = canvas.getContext("2d")
  if (!context) return

  const dpr = window.devicePixelRatio || 1
  canvas.width = Math.round(layout.width * dpr)
  canvas.height = Math.round(layout.height * dpr)
  context.setTransform(dpr, 0, 0, dpr, 0, 0)

  const colors = canvasColors(canvas)
  context.clearRect(0, 0, layout.width, layout.height)
  context.fillStyle = colors.surfaceRaised
  context.fillRect(0, 0, layout.width, layout.height)

  const cellByID = new Map(cells.map((cell) => [cell.id, cell]))
  drawConnections(context, layout, cells, cellByID, colors)
  drawEventAuras(context, layout, cells, activeEvents, colors)
  drawCells(context, layout, cells, teams, colors, selectedCellID)
}

function drawConnections(
  context: CanvasRenderingContext2D,
  layout: CanvasLayout,
  cells: FrontCell[],
  cellByID: Map<string, FrontCell>,
  colors: CanvasColors,
) {
  context.save()
  context.strokeStyle = colors.border
  context.lineCap = "round"
  context.lineWidth = clampNumber(layout.nodeRadius * 0.26, 8, 18)

  for (const cell of cells) {
    const from = layout.points.get(cell.id)
    if (!from) continue

    for (const neighborID of cell.neighborIds) {
      if (cell.id >= neighborID) continue

      const neighbor = cellByID.get(neighborID)
      const to = neighbor ? layout.points.get(neighbor.id) : undefined
      if (!to) continue

      context.beginPath()
      context.moveTo(from.x, from.y)
      context.lineTo(to.x, to.y)
      context.stroke()
    }
  }
  context.restore()
}

function drawEventAuras(
  context: CanvasRenderingContext2D,
  layout: CanvasLayout,
  cells: FrontCell[],
  activeEvents: FrontMapEvent[],
  colors: CanvasColors,
) {
  const eventCellIDs = new Set(
    activeEvents
      .map((event) => event.cellId)
      .filter((cellID): cellID is string => Boolean(cellID)),
  )

  for (const cell of cells) {
    const point = layout.points.get(cell.id)
    if (!point) continue

    const pressure = totalPressure(cell)
    if (pressure > 0) {
      context.save()
      context.globalAlpha = 0.2
      context.fillStyle = colors.destructive
      drawCircle(context, point, layout.nodeRadius * 1.7)
      context.restore()
    }

    if (eventCellIDs.has(cell.id)) {
      context.save()
      context.globalAlpha = 0.22
      context.fillStyle = colors.primary
      drawCircle(context, point, layout.nodeRadius * 1.9)
      context.restore()
    }
  }
}

function drawCells(
  context: CanvasRenderingContext2D,
  layout: CanvasLayout,
  cells: FrontCell[],
  teams: FrontTeamState[],
  colors: CanvasColors,
  selectedCellID: string | null,
) {
  for (const cell of cells) {
    const point = layout.points.get(cell.id)
    if (!point) continue

    const selected = selectedCellID === cell.id
    const fill = teamFillColor(cell.ownerTeamId, teams, colors, context.canvas)
    const label = cellLabel(cell)
    const control = clampPercent(cell.control)
    const defenseRadius = Math.max(
      layout.nodeRadius * 0.26,
      (layout.nodeRadius * clampPercent(cell.defense)) / 130,
    )

    context.save()
    context.fillStyle = fill
    context.strokeStyle = selected ? colors.primary : colors.ink
    context.lineWidth = selected ? 7 : 4
    drawCircle(context, point, layout.nodeRadius)
    context.stroke()

    context.globalAlpha = 0.76
    context.fillStyle = colors.card
    drawCircle(context, point, defenseRadius)
    context.restore()

    context.save()
    context.fillStyle = colors.ink
    context.textAlign = "center"
    context.textBaseline = "middle"
    context.font = `900 ${Math.round(layout.nodeRadius * 0.52)}px ${fontFamily}`
    context.fillText(String(control), point.x, point.y + 1)

    context.textBaseline = "top"
    context.font = `900 ${Math.round(layout.nodeRadius * 0.36)}px ${fontFamily}`
    drawFittedText(
      context,
      label,
      point.x,
      point.y + layout.labelOffset,
      layout.nodeRadius * 3.4,
      colors.foreground,
    )
    context.restore()
  }
}

function buildCanvasLayout(
  cells: FrontCell[],
  width: number,
  height: number,
): CanvasLayout {
  if (cells.length === 0) return { ...fallbackLayout, width, height }

  const minX = Math.min(...cells.map((cell) => cell.x))
  const maxX = Math.max(...cells.map((cell) => cell.x))
  const minY = Math.min(...cells.map((cell) => cell.y))
  const maxY = Math.max(...cells.map((cell) => cell.y))
  const rangeX = Math.max(1, maxX - minX)
  const rangeY = Math.max(1, maxY - minY)
  const shortSide = Math.min(width, height)
  const nodeRadius = clampNumber(shortSide * 0.064, 30, 54)
  const hitRadius = nodeRadius * 1.58
  const labelOffset = nodeRadius + 18
  const marginX = nodeRadius + 22
  const marginTop = nodeRadius + 26
  const marginBottom = nodeRadius + 58
  const usableWidth = Math.max(1, width - marginX * 2)
  const usableHeight = Math.max(1, height - marginTop - marginBottom)
  const xScale = usableWidth / rangeX
  const yScale = usableHeight / rangeY
  const points = new Map<string, CanvasPoint>()
  const cellsByCoordinate = new Map<string, FrontCell[]>()

  for (const cell of cells) {
    const key = `${cell.x}:${cell.y}`
    cellsByCoordinate.set(key, [...(cellsByCoordinate.get(key) ?? []), cell])
  }

  for (const [coordinate, groupedCells] of cellsByCoordinate) {
    const [x, y] = coordinate.split(":").map(Number)
    const basePoint = {
      x: marginX + (x - minX) * xScale,
      y: marginTop + (y - minY) * yScale,
    }

    if (groupedCells.length === 1) {
      points.set(groupedCells[0].id, basePoint)
      continue
    }

    const offset = nodeRadius * 1.38
    groupedCells.forEach((cell, index) => {
      const angle =
        groupedCells.length === 2
          ? index === 0
            ? Math.PI
            : 0
          : -Math.PI / 2 + (index / groupedCells.length) * Math.PI * 2

      points.set(cell.id, {
        x: clampNumber(
          basePoint.x + Math.cos(angle) * offset,
          marginX,
          width - marginX,
        ),
        y: clampNumber(
          basePoint.y + Math.sin(angle) * offset,
          marginTop,
          height - marginBottom,
        ),
      })
    })
  }

  return {
    width,
    height,
    nodeRadius,
    hitRadius,
    labelOffset,
    points,
  }
}

function canvasColors(canvas: HTMLCanvasElement): CanvasColors {
  const styles = getComputedStyle(canvas)

  return {
    border: cssVar(styles, "--border"),
    card: cssVar(styles, "--card"),
    destructive: cssVar(styles, "--destructive"),
    foreground: cssVar(styles, "--foreground"),
    ink: cssVar(styles, "--ink"),
    muted: cssVar(styles, "--muted"),
    primary: cssVar(styles, "--primary"),
    surfaceRaised: cssVar(styles, "--surface-raised"),
  }
}

function teamFillColor(
  teamID: string | undefined,
  teams: FrontTeamState[],
  colors: CanvasColors,
  canvas: HTMLCanvasElement,
) {
  if (!teamID) return colors.muted

  const tone = getTeamTone(teamID, teams)
  const styles = getComputedStyle(canvas)
  return cssVar(styles, `--pebble-${tone.key}`)
}

function cssVar(styles: CSSStyleDeclaration, name: string) {
  return styles.getPropertyValue(name).trim() || styles.color
}

function drawCircle(
  context: CanvasRenderingContext2D,
  point: CanvasPoint,
  radius: number,
) {
  context.beginPath()
  context.arc(point.x, point.y, radius, 0, Math.PI * 2)
  context.fill()
}

function drawFittedText(
  context: CanvasRenderingContext2D,
  text: string,
  x: number,
  y: number,
  maxWidth: number,
  fillStyle: string,
) {
  context.fillStyle = fillStyle
  context.fillText(text, x, y, maxWidth)
}

function totalPressure(cell: FrontCell) {
  return Object.values(cell.pressureByTeam).reduce(
    (sum, value) => sum + value,
    0,
  )
}

function clampPercent(value: number) {
  return clampNumber(Math.round(value), 0, 100)
}

function clampNumber(value: number, minValue: number, maxValue: number) {
  return Math.max(minValue, Math.min(maxValue, value))
}

function cellLabel(cell: FrontCell) {
  const baseMatch = /^base_team_0*(\d+)$/.exec(cell.id)
  if (baseMatch) return `基地 ${Number(baseMatch[1])}`

  return labelByCellID[cell.id] ?? cell.name ?? cell.id
}
