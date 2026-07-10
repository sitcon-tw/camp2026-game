import {
  BookOpen,
  Flag,
  Handshake,
  LifeBuoy,
  MapPin,
  PackageOpen,
  Radar,
  Swords,
  Wrench,
} from "lucide-react"
import {
  useEffect,
  useMemo,
  useRef,
  useState,
  type ComponentType,
  type CSSProperties,
  type KeyboardEvent,
  type MouseEvent,
  type PointerEvent,
} from "react"

import type {
  FrontTerritoryBase,
  FrontTerritoryGrid,
  FrontTerritoryLandmark,
  FrontTerritoryRow,
  FrontTeamState,
} from "@/shared/api/game"
import { Button } from "@/shared/ui/button"
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/shared/ui/tooltip"
import { cn } from "@/shared/utils"

import { getTeamName, getTerritoryTeamColor } from "./front-map-style"

export type TerritoryTarget = {
  x: number
  y: number
}

export type TerritoryCellState = TerritoryTarget & {
  ownerTeamId?: string
  defense: number
}

type TerritoryGridMapProps = {
  grid: FrontTerritoryGrid
  rows: FrontTerritoryRow[]
  bases: FrontTerritoryBase[]
  landmarks: FrontTerritoryLandmark[]
  teams: FrontTeamState[]
  selectedTarget: TerritoryTarget | null
  onSelectTarget: (target: TerritoryTarget) => void
}

type CanvasSize = {
  width: number
  height: number
}

type GridCanvasColors = {
  ink: string
  power: string
}

const landmarkIcons: Record<
  string,
  ComponentType<{ className?: string; "aria-hidden"?: boolean }>
> = {
  challenge: Swords,
  course: BookOpen,
  repair: Wrench,
  rescue: LifeBuoy,
  resource: PackageOpen,
  support: Handshake,
  system: Radar,
}

export function TerritoryGridMap({
  grid,
  rows,
  bases,
  landmarks,
  teams,
  selectedTarget,
  onSelectTarget,
}: TerritoryGridMapProps) {
  const frameRef = useRef<HTMLDivElement>(null)
  const canvasRef = useRef<HTMLCanvasElement>(null)
  const [canvasSize, setCanvasSize] = useState<CanvasSize>({
    width: grid.background.width,
    height: grid.background.height,
  })
  const [backgroundFailed, setBackgroundFailed] = useState(false)
  const [keyboardTarget, setKeyboardTarget] = useState<TerritoryTarget | null>(
    null,
  )
  const territory = useMemo(
    () => decodeTerritoryRows(rows, grid.width, grid.height),
    [grid.height, grid.width, rows],
  )
  const displayedTarget = selectedTarget ?? keyboardTarget
  const keyboardCell = keyboardTarget
    ? territory.get(territoryCellKey(keyboardTarget.x, keyboardTarget.y))
    : undefined

  useEffect(() => {
    const frame = frameRef.current
    if (!frame) return

    const updateCanvasSize = () => {
      const width = frame.clientWidth
      const height = frame.clientHeight
      if (width > 0 && height > 0) setCanvasSize({ width, height })
    }

    updateCanvasSize()
    const observer = new ResizeObserver(updateCanvasSize)
    observer.observe(frame)

    return () => observer.disconnect()
  }, [])

  useEffect(() => {
    const canvas = canvasRef.current
    if (!canvas) return

    drawTerritoryGrid(
      canvas,
      canvasSize,
      grid,
      territory,
      teams,
      displayedTarget,
    )
  }, [canvasSize, displayedTarget, grid, teams, territory])

  function handlePointerUp(event: PointerEvent<HTMLButtonElement>) {
    const bounds = event.currentTarget.getBoundingClientRect()
    const x = Math.floor(
      ((event.clientX - bounds.left) / bounds.width) * grid.width,
    )
    const y = Math.floor(
      ((event.clientY - bounds.top) / bounds.height) * grid.height,
    )

    if (territory.has(territoryCellKey(x, y))) onSelectTarget({ x, y })
  }

  function handleKeyboardClick(event: MouseEvent<HTMLButtonElement>) {
    if (event.detail !== 0) return

    const target =
      keyboardTarget ??
      (selectedTarget &&
      territory.has(territoryCellKey(selectedTarget.x, selectedTarget.y))
        ? selectedTarget
        : nearestTerritoryCell(territory, grid.width / 2, grid.height / 2))

    if (target) onSelectTarget(target)
  }

  function handleMapFocus() {
    if (keyboardTarget || selectedTarget) return

    setKeyboardTarget(
      nearestTerritoryCell(territory, grid.width / 2, grid.height / 2),
    )
  }

  function handleMapKeyDown(event: KeyboardEvent<HTMLButtonElement>) {
    const directionByKey: Record<
      string,
      readonly [number, number] | undefined
    > = {
      ArrowDown: [0, 1],
      ArrowLeft: [-1, 0],
      ArrowRight: [1, 0],
      ArrowUp: [0, -1],
    }
    const direction = directionByKey[event.key]
    if (!direction) return

    event.preventDefault()
    const current =
      keyboardTarget ??
      selectedTarget ??
      nearestTerritoryCell(territory, grid.width / 2, grid.height / 2)
    if (!current) return

    const next = nextTerritoryCell(
      territory,
      current,
      direction[0],
      direction[1],
      grid,
    )
    if (next) setKeyboardTarget(next)
  }

  return (
    <section
      className="border-ink bg-surface-raised -mx-4 overflow-hidden border-y-2"
      aria-labelledby="territory-map-title"
    >
      <div className="bg-card border-ink flex items-center gap-3 border-b-2 px-4 py-3">
        <h2 id="territory-map-title" className="text-lg font-black">
          校園戰線
        </h2>
      </div>

      <TooltipProvider delayDuration={100}>
        <div
          ref={frameRef}
          className="bg-surface-raised focus-visible:ring-ring/50 relative w-full touch-manipulation overflow-hidden outline-none focus-visible:ring-4 focus-visible:ring-inset"
          style={{
            aspectRatio: `${grid.background.width} / ${grid.background.height}`,
          }}
          role="group"
          aria-label="校園領土地圖"
        >
          {backgroundFailed ? (
            <div className="bg-muted absolute inset-0" aria-hidden />
          ) : (
            <img
              src={grid.background.src}
              alt="校園地圖"
              className="absolute inset-0 size-full object-fill select-none"
              draggable={false}
              onError={() => setBackgroundFailed(true)}
            />
          )}

          <canvas
            ref={canvasRef}
            className="pointer-events-none absolute inset-0 size-full"
            aria-hidden
          />

          <Button
            type="button"
            variant="ghost"
            className="focus-visible:ring-ring/60 absolute inset-0 z-10 size-full rounded-none border-0 bg-transparent shadow-none hover:bg-transparent focus-visible:ring-4 focus-visible:ring-inset"
            aria-label={
              keyboardCell
                ? `選擇${getTeamName(keyboardCell.ownerTeamId, teams)}領土方向，防禦 ${keyboardCell.defense}`
                : "選擇領土方向"
            }
            onPointerUp={handlePointerUp}
            onClick={handleKeyboardClick}
            onFocus={handleMapFocus}
            onBlur={() => setKeyboardTarget(null)}
            onKeyDown={handleMapKeyDown}
          />

          {bases.map((base) => {
            const selected = isSameTarget(displayedTarget, base)
            const color = getTerritoryTeamColor(base.teamId, teams)
            const team = teams.find((item) => item.teamId === base.teamId)
            const teamNumber = teamOrdinal(base.teamId)
            const teamName = getTeamName(base.teamId, teams)

            return (
              <Tooltip key={base.teamId}>
                <TooltipTrigger asChild>
                  <Button
                    type="button"
                    size="icon-sm"
                    variant="outline"
                    className={cn(
                      "bg-card/95 absolute z-30 -translate-x-1/2 -translate-y-1/2 rounded-full shadow-sm",
                      selected && "ring-power ring-4",
                    )}
                    style={markerPosition(base, grid)}
                    aria-label={`${teamName}基地，不可佔領`}
                    aria-pressed={selected}
                    onClick={() => onSelectTarget({ x: base.x, y: base.y })}
                  >
                    <span
                      className={cn(
                        "border-card absolute -top-1 -left-1 size-3 rounded-full border-2",
                        color?.dot ?? "bg-muted-foreground",
                      )}
                      style={markerColorStyle(team?.color)}
                      aria-hidden
                    />
                    <Flag className="size-4" aria-hidden />
                    <span className="bg-card border-ink absolute -right-1 -bottom-1 grid size-4 place-items-center rounded-full border text-[9px] leading-none font-black">
                      {teamNumber}
                    </span>
                  </Button>
                </TooltipTrigger>
                <TooltipContent side="top">
                  {teamName}基地 · 不可佔領
                </TooltipContent>
              </Tooltip>
            )
          })}

          {landmarks.map((landmark) => {
            const Icon = landmarkIcons[landmark.kind] ?? MapPin
            const selected = isSameTarget(displayedTarget, landmark)

            return (
              <Tooltip key={landmark.id}>
                <TooltipTrigger asChild>
                  <Button
                    type="button"
                    size="icon-xs"
                    variant="secondary"
                    className={cn(
                      "absolute z-20 -translate-x-1/2 -translate-y-1/2 rounded-full shadow-sm",
                      selected && "ring-ink ring-3",
                    )}
                    style={markerPosition(landmark, grid)}
                    aria-label={landmark.label}
                    aria-pressed={selected}
                    onClick={() =>
                      onSelectTarget({ x: landmark.x, y: landmark.y })
                    }
                  >
                    <Icon className="size-3.5" aria-hidden />
                  </Button>
                </TooltipTrigger>
                <TooltipContent side="top">{landmark.label}</TooltipContent>
              </Tooltip>
            )
          })}
        </div>
      </TooltipProvider>
    </section>
  )
}

export function decodeTerritoryRows(
  rows: FrontTerritoryRow[],
  width: number,
  height: number,
) {
  const territory = new Map<string, TerritoryCellState>()

  for (const row of rows) {
    if (row.y < 0 || row.y >= height) continue

    for (const run of row.runs) {
      const start = Math.max(0, run.x)
      const end = Math.min(width, run.x + run.length)

      for (let x = start; x < end; x += 1) {
        territory.set(territoryCellKey(x, row.y), {
          x,
          y: row.y,
          ownerTeamId: run.ownerTeamId,
          defense: clampPercent(run.defense),
        })
      }
    }
  }

  return territory
}

export function territoryCellKey(x: number, y: number) {
  return `${x}:${y}`
}

function drawTerritoryGrid(
  canvas: HTMLCanvasElement,
  size: CanvasSize,
  grid: FrontTerritoryGrid,
  territory: Map<string, TerritoryCellState>,
  teams: FrontTeamState[],
  selectedTarget: TerritoryTarget | null,
) {
  const context = canvas.getContext("2d")
  if (!context) return

  const dpr = window.devicePixelRatio || 1
  canvas.width = Math.round(size.width * dpr)
  canvas.height = Math.round(size.height * dpr)
  context.setTransform(dpr, 0, 0, dpr, 0, 0)
  context.clearRect(0, 0, size.width, size.height)

  const colors = territoryCanvasColors(canvas)
  const cellWidth = size.width / grid.width
  const cellHeight = size.height / grid.height

  context.save()
  context.globalAlpha = 0.56
  for (const cell of territory.values()) {
    if (!cell.ownerTeamId) continue

    context.fillStyle = territoryTeamFill(
      cell.ownerTeamId,
      teams,
      canvas,
      colors.ink,
    )
    context.fillRect(
      cell.x * cellWidth,
      cell.y * cellHeight,
      cellWidth + 0.5,
      cellHeight + 0.5,
    )
  }
  context.restore()

  drawPlayableBoundary(context, territory, cellWidth, cellHeight, colors.ink)
  drawOwnershipBoundaries(context, territory, cellWidth, cellHeight, colors.ink)

  if (selectedTarget) {
    const cell = territory.get(
      territoryCellKey(selectedTarget.x, selectedTarget.y),
    )
    if (cell) {
      context.save()
      context.globalAlpha = 0.52
      context.fillStyle = colors.power
      context.fillRect(
        cell.x * cellWidth,
        cell.y * cellHeight,
        cellWidth,
        cellHeight,
      )
      context.globalAlpha = 1
      context.strokeStyle = colors.ink
      context.lineWidth = Math.max(2, Math.min(cellWidth, cellHeight) * 0.3)
      context.strokeRect(
        cell.x * cellWidth,
        cell.y * cellHeight,
        cellWidth,
        cellHeight,
      )
      context.restore()
    }
  }
}

function drawPlayableBoundary(
  context: CanvasRenderingContext2D,
  territory: Map<string, TerritoryCellState>,
  cellWidth: number,
  cellHeight: number,
  color: string,
) {
  context.save()
  context.beginPath()
  context.globalAlpha = 0.42
  context.strokeStyle = color
  context.lineWidth = Math.max(0.8, Math.min(cellWidth, cellHeight) * 0.12)

  for (const cell of territory.values()) {
    drawMissingNeighborEdges(context, territory, cell, cellWidth, cellHeight)
  }

  context.stroke()
  context.restore()
}

function drawOwnershipBoundaries(
  context: CanvasRenderingContext2D,
  territory: Map<string, TerritoryCellState>,
  cellWidth: number,
  cellHeight: number,
  color: string,
) {
  context.save()
  context.beginPath()
  context.globalAlpha = 0.88
  context.strokeStyle = color
  context.lineWidth = Math.max(1.25, Math.min(cellWidth, cellHeight) * 0.2)

  for (const cell of territory.values()) {
    if (!cell.ownerTeamId) continue

    drawDifferentOwnerEdges(context, territory, cell, cellWidth, cellHeight)
  }

  context.stroke()
  context.restore()
}

function drawMissingNeighborEdges(
  context: CanvasRenderingContext2D,
  territory: Map<string, TerritoryCellState>,
  cell: TerritoryCellState,
  cellWidth: number,
  cellHeight: number,
) {
  const x = cell.x * cellWidth
  const y = cell.y * cellHeight

  if (!territory.has(territoryCellKey(cell.x, cell.y - 1))) {
    drawLine(context, x, y, x + cellWidth, y)
  }
  if (!territory.has(territoryCellKey(cell.x + 1, cell.y))) {
    drawLine(context, x + cellWidth, y, x + cellWidth, y + cellHeight)
  }
  if (!territory.has(territoryCellKey(cell.x, cell.y + 1))) {
    drawLine(context, x, y + cellHeight, x + cellWidth, y + cellHeight)
  }
  if (!territory.has(territoryCellKey(cell.x - 1, cell.y))) {
    drawLine(context, x, y, x, y + cellHeight)
  }
}

function drawDifferentOwnerEdges(
  context: CanvasRenderingContext2D,
  territory: Map<string, TerritoryCellState>,
  cell: TerritoryCellState,
  cellWidth: number,
  cellHeight: number,
) {
  const x = cell.x * cellWidth
  const y = cell.y * cellHeight
  const differs = (targetX: number, targetY: number) =>
    territory.get(territoryCellKey(targetX, targetY))?.ownerTeamId !==
    cell.ownerTeamId

  if (differs(cell.x, cell.y - 1)) {
    drawLine(context, x, y, x + cellWidth, y)
  }
  if (differs(cell.x + 1, cell.y)) {
    drawLine(context, x + cellWidth, y, x + cellWidth, y + cellHeight)
  }
  if (differs(cell.x, cell.y + 1)) {
    drawLine(context, x, y + cellHeight, x + cellWidth, y + cellHeight)
  }
  if (differs(cell.x - 1, cell.y)) {
    drawLine(context, x, y, x, y + cellHeight)
  }
}

function drawLine(
  context: CanvasRenderingContext2D,
  fromX: number,
  fromY: number,
  toX: number,
  toY: number,
) {
  context.moveTo(fromX, fromY)
  context.lineTo(toX, toY)
}

function territoryCanvasColors(canvas: HTMLCanvasElement): GridCanvasColors {
  const styles = getComputedStyle(canvas)

  return {
    ink: cssVariable(styles, "--ink"),
    power: cssVariable(styles, "--power"),
  }
}

function territoryTeamFill(
  teamID: string,
  teams: FrontTeamState[],
  canvas: HTMLCanvasElement,
  fallback: string,
) {
  const teamColor = teams.find((team) => team.teamId === teamID)?.color?.trim()
  if (teamColor && CSS.supports("color", teamColor)) return teamColor

  const semanticColor = getTerritoryTeamColor(teamID, teams)
  if (!semanticColor) return fallback

  return cssVariable(
    getComputedStyle(canvas),
    semanticColor.cssVariable,
    fallback,
  )
}

function cssVariable(
  styles: CSSStyleDeclaration,
  name: string,
  fallback = styles.color,
) {
  return styles.getPropertyValue(name).trim() || fallback
}

function markerPosition(
  target: TerritoryTarget,
  grid: FrontTerritoryGrid,
): CSSProperties {
  return {
    left: `${((target.x + 0.5) / grid.width) * 100}%`,
    top: `${((target.y + 0.5) / grid.height) * 100}%`,
  }
}

function markerColorStyle(
  color: string | undefined,
): CSSProperties | undefined {
  const value = color?.trim()
  return value ? { backgroundColor: value } : undefined
}

function nearestTerritoryCell(
  territory: Map<string, TerritoryCellState>,
  targetX: number,
  targetY: number,
) {
  let closest: TerritoryCellState | null = null
  let closestDistance = Number.POSITIVE_INFINITY

  for (const cell of territory.values()) {
    const distance =
      (cell.x - targetX) * (cell.x - targetX) +
      (cell.y - targetY) * (cell.y - targetY)
    if (distance < closestDistance) {
      closest = cell
      closestDistance = distance
    }
  }

  return closest ? { x: closest.x, y: closest.y } : null
}

function nextTerritoryCell(
  territory: Map<string, TerritoryCellState>,
  current: TerritoryTarget,
  offsetX: number,
  offsetY: number,
  grid: FrontTerritoryGrid,
) {
  const maxDistance = Math.max(grid.width, grid.height)

  for (let distance = 1; distance <= maxDistance; distance += 1) {
    const x = current.x + offsetX * distance
    const y = current.y + offsetY * distance
    if (x < 0 || y < 0 || x >= grid.width || y >= grid.height) return null
    if (territory.has(territoryCellKey(x, y))) return { x, y }
  }

  return null
}

function isSameTarget(
  selectedTarget: TerritoryTarget | null,
  target: TerritoryTarget,
) {
  return selectedTarget?.x === target.x && selectedTarget.y === target.y
}

function teamOrdinal(teamID: string) {
  const match = /(\d+)$/.exec(teamID)
  return match ? String(Number(match[1])) : "?"
}

function clampPercent(value: number) {
  return Math.max(0, Math.min(100, Math.round(value)))
}
