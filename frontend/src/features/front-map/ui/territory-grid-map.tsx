import {
  BookOpen,
  Flag,
  Handshake,
  LifeBuoy,
  Maximize2,
  MapPin,
  Minimize2,
  PackageOpen,
  Radar,
  RotateCcw,
  Swords,
  Wrench,
  ZoomIn,
  ZoomOut,
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
import { ButtonGroup } from "@/shared/ui/button-group"
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

type MapView = {
  scale: number
  offsetX: number
  offsetY: number
}

type PointerPosition = {
  x: number
  y: number
}

type MapGesture =
  | {
      kind: "drag"
      pointerID: number
      start: PointerPosition
      startView: MapView
      moved: boolean
    }
  | {
      kind: "pinch"
      pointerIDs: [number, number]
      startDistance: number
      startScale: number
      anchorX: number
      anchorY: number
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

const minMapScale = 1
const maxMapScale = 4
const mapScaleStep = 0.5
const fullscreenRequestTimeoutMs = 900

export function TerritoryGridMap({
  grid,
  rows,
  bases,
  landmarks,
  teams,
  selectedTarget,
  onSelectTarget,
}: TerritoryGridMapProps) {
  const sectionRef = useRef<HTMLElement>(null)
  const frameRef = useRef<HTMLDivElement>(null)
  const canvasRef = useRef<HTMLCanvasElement>(null)
  const pointersRef = useRef(new Map<number, PointerPosition>())
  const gestureRef = useRef<MapGesture | null>(null)
  const [viewportSize, setViewportSize] = useState<CanvasSize>({
    width: grid.background.width,
    height: grid.background.height,
  })
  const [mapView, setMapView] = useState<MapView>({
    scale: minMapScale,
    offsetX: 0,
    offsetY: 0,
  })
  const mapViewRef = useRef(mapView)
  const [backgroundFailed, setBackgroundFailed] = useState(false)
  const [keyboardTarget, setKeyboardTarget] = useState<TerritoryTarget | null>(
    null,
  )
  const [nativeFullscreen, setNativeFullscreen] = useState(false)
  const [fallbackFullscreen, setFallbackFullscreen] = useState(false)
  const territory = useMemo(
    () => decodeTerritoryRows(rows, grid.width, grid.height),
    [grid.height, grid.width, rows],
  )
  const canvasSize = useMemo(
    () =>
      fitMapSize(viewportSize, grid.background.width / grid.background.height),
    [grid.background.height, grid.background.width, viewportSize],
  )
  const fullscreen = nativeFullscreen || fallbackFullscreen
  const minimumMapScale = useMemo(
    () => (fullscreen ? coverMapScale(canvasSize, viewportSize) : minMapScale),
    [canvasSize, fullscreen, viewportSize],
  )
  const displayedTarget = selectedTarget ?? keyboardTarget
  const keyboardCell = keyboardTarget
    ? territory.get(territoryCellKey(keyboardTarget.x, keyboardTarget.y))
    : undefined
  const resetMapView = () => {
    resetTerritoryMapView(mapViewRef, setMapView, minimumMapScale)
  }

  useEffect(() => {
    const frame = frameRef.current
    if (!frame) return

    const updateViewportSize = () => {
      const width = frame.clientWidth
      const height = frame.clientHeight
      if (width > 0 && height > 0) setViewportSize({ width, height })
    }

    updateViewportSize()
    const observer = new ResizeObserver(updateViewportSize)
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
      mapView.scale,
    )
  }, [canvasSize, displayedTarget, grid, mapView.scale, teams, territory])

  useEffect(() => {
    const handleFullscreenChange = () => {
      const active = document.fullscreenElement === sectionRef.current
      setNativeFullscreen(active)
      if (active) {
        setFallbackFullscreen(false)
      } else {
        resetTerritoryMapView(mapViewRef, setMapView, minMapScale)
      }
    }

    document.addEventListener("fullscreenchange", handleFullscreenChange)
    return () =>
      document.removeEventListener("fullscreenchange", handleFullscreenChange)
  }, [])

  useEffect(() => {
    if (!fallbackFullscreen) return

    const previousOverflow = document.body.style.overflow
    document.body.style.overflow = "hidden"
    const handleEscape = (event: globalThis.KeyboardEvent) => {
      if (event.key === "Escape") {
        setFallbackFullscreen(false)
        resetTerritoryMapView(mapViewRef, setMapView, minMapScale)
      }
    }
    window.addEventListener("keydown", handleEscape)

    return () => {
      document.body.style.overflow = previousOverflow
      window.removeEventListener("keydown", handleEscape)
    }
  }, [fallbackFullscreen])

  useEffect(() => {
    const current = mapViewRef.current
    const clamped = clampMapView(
      current,
      canvasSize,
      viewportSize,
      minimumMapScale,
    )
    if (
      clamped.scale !== current.scale ||
      clamped.offsetX !== current.offsetX ||
      clamped.offsetY !== current.offsetY
    ) {
      mapViewRef.current = clamped
      setMapView(clamped)
    }
  }, [canvasSize, minimumMapScale, viewportSize])

  function updateMapView(nextView: MapView) {
    const clamped = clampMapView(
      nextView,
      canvasSize,
      viewportSize,
      minimumMapScale,
    )
    mapViewRef.current = clamped
    setMapView(clamped)
  }

  function changeMapScale(delta: number) {
    const current = mapViewRef.current
    const scale = clampNumber(
      current.scale + delta,
      minimumMapScale,
      maxMapScale,
    )
    const ratio = scale / current.scale
    updateMapView({
      scale,
      offsetX: current.offsetX * ratio,
      offsetY: current.offsetY * ratio,
    })
  }

  async function toggleFullscreen() {
    const section = sectionRef.current
    if (!section) return

    if (document.fullscreenElement === section) {
      await document.exitFullscreen()
      return
    }
    if (fallbackFullscreen) {
      setFallbackFullscreen(false)
      resetTerritoryMapView(mapViewRef, setMapView, minMapScale)
      return
    }

    resetMapView()
    const requestFullscreen = (
      section as { requestFullscreen?: () => Promise<void> }
    ).requestFullscreen
    if (requestFullscreen) {
      const enteredNativeFullscreen = await requestNativeFullscreen(
        section,
        requestFullscreen,
      )
      if (enteredNativeFullscreen || document.fullscreenElement === section) {
        return
      }
    }
    if (document.fullscreenElement !== section) setFallbackFullscreen(true)
  }

  function commitTarget(target: TerritoryTarget) {
    const section = sectionRef.current
    if (
      section &&
      document.fullscreenElement === section &&
      document.exitFullscreen
    ) {
      void document
        .exitFullscreen()
        .then(() => onSelectTarget(target))
        .catch(() => {
          setNativeFullscreen(false)
          setFallbackFullscreen(false)
          onSelectTarget(target)
        })
      return
    }

    if (fallbackFullscreen) {
      setFallbackFullscreen(false)
      resetTerritoryMapView(mapViewRef, setMapView, minMapScale)
    }

    onSelectTarget(target)
  }

  function handlePointerDown(event: PointerEvent<HTMLButtonElement>) {
    event.preventDefault()
    event.currentTarget.setPointerCapture(event.pointerId)
    pointersRef.current.set(event.pointerId, {
      x: event.clientX,
      y: event.clientY,
    })

    const pointers = [...pointersRef.current.entries()]
    if (pointers.length === 1) {
      gestureRef.current = {
        kind: "drag",
        pointerID: event.pointerId,
        start: { x: event.clientX, y: event.clientY },
        startView: mapViewRef.current,
        moved: false,
      }
      return
    }

    const [first, second] = pointers
    const midpoint = pointerMidpoint(first[1], second[1])
    const frameBounds = frameRef.current?.getBoundingClientRect()
    const view = mapViewRef.current
    gestureRef.current = {
      kind: "pinch",
      pointerIDs: [first[0], second[0]],
      startDistance: pointerDistance(first[1], second[1]),
      startScale: view.scale,
      anchorX:
        (midpoint.x -
          (frameBounds?.left ?? 0) -
          viewportSize.width / 2 -
          view.offsetX) /
        view.scale,
      anchorY:
        (midpoint.y -
          (frameBounds?.top ?? 0) -
          viewportSize.height / 2 -
          view.offsetY) /
        view.scale,
    }
  }

  function handlePointerMove(event: PointerEvent<HTMLButtonElement>) {
    if (!pointersRef.current.has(event.pointerId)) return
    event.preventDefault()
    pointersRef.current.set(event.pointerId, {
      x: event.clientX,
      y: event.clientY,
    })

    const gesture = gestureRef.current
    if (!gesture) return

    if (gesture.kind === "pinch") {
      const first = pointersRef.current.get(gesture.pointerIDs[0])
      const second = pointersRef.current.get(gesture.pointerIDs[1])
      if (!first || !second || gesture.startDistance <= 0) return

      const midpoint = pointerMidpoint(first, second)
      const frameBounds = frameRef.current?.getBoundingClientRect()
      const scale = clampNumber(
        gesture.startScale *
          (pointerDistance(first, second) / gesture.startDistance),
        minimumMapScale,
        maxMapScale,
      )
      updateMapView({
        scale,
        offsetX:
          midpoint.x -
          (frameBounds?.left ?? 0) -
          viewportSize.width / 2 -
          gesture.anchorX * scale,
        offsetY:
          midpoint.y -
          (frameBounds?.top ?? 0) -
          viewportSize.height / 2 -
          gesture.anchorY * scale,
      })
      return
    }

    if (gesture.pointerID !== event.pointerId) return
    const offsetX = event.clientX - gesture.start.x
    const offsetY = event.clientY - gesture.start.y
    const moved = gesture.moved || Math.hypot(offsetX, offsetY) > 5
    gestureRef.current = { ...gesture, moved }
    updateMapView({
      ...gesture.startView,
      offsetX: gesture.startView.offsetX + offsetX,
      offsetY: gesture.startView.offsetY + offsetY,
    })
  }

  function finishPointer(
    event: PointerEvent<HTMLButtonElement>,
    allowSelection: boolean,
  ) {
    const gesture = gestureRef.current
    const wasTap = Boolean(
      allowSelection &&
      gesture?.kind === "drag" &&
      gesture.pointerID === event.pointerId &&
      !gesture.moved &&
      pointersRef.current.size === 1,
    )

    pointersRef.current.delete(event.pointerId)
    if (event.currentTarget.hasPointerCapture(event.pointerId)) {
      event.currentTarget.releasePointerCapture(event.pointerId)
    }

    const remaining = [...pointersRef.current.entries()]
    if (remaining.length === 1) {
      gestureRef.current = {
        kind: "drag",
        pointerID: remaining[0][0],
        start: remaining[0][1],
        startView: mapViewRef.current,
        moved: true,
      }
    } else {
      gestureRef.current = null
    }

    if (!wasTap) return
    const bounds = event.currentTarget.getBoundingClientRect()
    const x = Math.floor(
      ((event.clientX - bounds.left) / bounds.width) * grid.width,
    )
    const y = Math.floor(
      ((event.clientY - bounds.top) / bounds.height) * grid.height,
    )
    if (territory.has(territoryCellKey(x, y))) commitTarget({ x, y })
  }

  function handleKeyboardClick(event: MouseEvent<HTMLButtonElement>) {
    if (event.detail !== 0) return

    const target =
      keyboardTarget ??
      (selectedTarget &&
      territory.has(territoryCellKey(selectedTarget.x, selectedTarget.y))
        ? selectedTarget
        : nearestTerritoryCell(territory, grid.width / 2, grid.height / 2))

    if (target) commitTarget(target)
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
      ref={sectionRef}
      className={cn(
        "border-ink bg-surface-raised -mx-4 overflow-hidden border-y-2",
        fullscreen &&
          "fixed inset-0 z-[45] mx-0 flex h-[100dvh] w-full flex-col border-0",
      )}
      aria-labelledby="territory-map-title"
    >
      <TooltipProvider delayDuration={100}>
        <div
          className="bg-card border-ink flex min-h-12 shrink-0 items-center justify-between gap-2 border-b-2 px-3 py-2"
          style={
            fullscreen
              ? { paddingTop: "max(0.5rem, env(safe-area-inset-top))" }
              : undefined
          }
        >
          <h2
            id="territory-map-title"
            className="min-w-0 truncate text-lg font-black"
          >
            校園戰線
          </h2>
          <ButtonGroup className="shrink-0" aria-label="地圖檢視控制">
            <MapControlButton
              label="縮小地圖"
              icon={ZoomOut}
              disabled={mapView.scale <= minimumMapScale}
              onClick={() => changeMapScale(-mapScaleStep)}
            />
            <MapControlButton
              label="重設地圖檢視"
              icon={RotateCcw}
              disabled={
                mapView.scale === minimumMapScale &&
                mapView.offsetX === 0 &&
                mapView.offsetY === 0
              }
              onClick={resetMapView}
            />
            <MapControlButton
              label="放大地圖"
              icon={ZoomIn}
              disabled={mapView.scale >= maxMapScale}
              onClick={() => changeMapScale(mapScaleStep)}
            />
            <MapControlButton
              label={fullscreen ? "退出滿版地圖" : "滿版顯示地圖"}
              icon={fullscreen ? Minimize2 : Maximize2}
              pressed={fullscreen}
              onClick={() => void toggleFullscreen()}
            />
          </ButtonGroup>
        </div>

        <div
          ref={frameRef}
          className={cn(
            "bg-surface-raised relative w-full overflow-hidden",
            fullscreen && "min-h-0 flex-1",
          )}
          style={
            fullscreen
              ? undefined
              : {
                  aspectRatio: `${grid.background.width} / ${grid.background.height}`,
                }
          }
          role="group"
          aria-label="校園領土地圖"
        >
          <div
            className="absolute top-1/2 left-1/2"
            style={{
              width: canvasSize.width,
              height: canvasSize.height,
              transform: "translate(-50%, -50%)",
            }}
          >
            <div
              className="absolute inset-0"
              style={{
                transform: `translate3d(${mapView.offsetX}px, ${mapView.offsetY}px, 0)`,
              }}
            >
              <div
                className="absolute inset-0 origin-center"
                style={{ transform: `scale(${mapView.scale})` }}
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
                  className="focus-visible:ring-ring/60 absolute inset-0 z-10 size-full touch-none rounded-none border-0 bg-transparent shadow-none hover:bg-transparent focus-visible:ring-4 focus-visible:ring-inset active:translate-x-0 active:translate-y-0"
                  aria-label={
                    keyboardCell
                      ? `選擇${getTeamName(keyboardCell.ownerTeamId, teams)}領土方向，防禦 ${keyboardCell.defense}`
                      : "選擇領土方向"
                  }
                  onPointerDown={handlePointerDown}
                  onPointerMove={handlePointerMove}
                  onPointerUp={(event) => finishPointer(event, true)}
                  onPointerCancel={(event) => finishPointer(event, false)}
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
                          style={markerPosition(base, grid, mapView.scale)}
                          aria-label={`${teamName}基地，不可佔領`}
                          aria-pressed={selected}
                          onClick={() => commitTarget({ x: base.x, y: base.y })}
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
                          style={markerPosition(landmark, grid, mapView.scale)}
                          aria-label={landmark.label}
                          aria-pressed={selected}
                          onClick={() =>
                            commitTarget({ x: landmark.x, y: landmark.y })
                          }
                        >
                          <Icon className="size-3.5" aria-hidden />
                        </Button>
                      </TooltipTrigger>
                      <TooltipContent side="top">
                        {landmark.label}
                      </TooltipContent>
                    </Tooltip>
                  )
                })}
              </div>
            </div>
          </div>
        </div>
      </TooltipProvider>
    </section>
  )
}

function MapControlButton({
  label,
  icon: Icon,
  disabled = false,
  pressed,
  onClick,
}: {
  label: string
  icon: ComponentType<{ className?: string }>
  disabled?: boolean
  pressed?: boolean
  onClick: () => void
}) {
  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <Button
          type="button"
          size="icon-xs"
          variant="outline"
          title={label}
          aria-label={label}
          aria-pressed={pressed}
          disabled={disabled}
          onClick={onClick}
        >
          <Icon className="size-3.5" aria-hidden />
        </Button>
      </TooltipTrigger>
      <TooltipContent side="bottom">{label}</TooltipContent>
    </Tooltip>
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
  mapScale: number,
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

  drawPlayableGrid(
    context,
    territory,
    cellWidth,
    cellHeight,
    colors.ink,
    mapScale,
  )
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

function drawPlayableGrid(
  context: CanvasRenderingContext2D,
  territory: Map<string, TerritoryCellState>,
  cellWidth: number,
  cellHeight: number,
  color: string,
  mapScale: number,
) {
  context.save()
  context.beginPath()
  context.globalAlpha = mapScale >= 2 ? 0.25 : 0.12
  context.strokeStyle = color
  context.lineWidth = Math.max(0.35, 0.7 / Math.sqrt(mapScale))

  for (const cell of territory.values()) {
    context.rect(cell.x * cellWidth, cell.y * cellHeight, cellWidth, cellHeight)
  }

  context.stroke()
  context.restore()
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
  mapScale: number,
): CSSProperties {
  return {
    left: `${((target.x + 0.5) / grid.width) * 100}%`,
    top: `${((target.y + 0.5) / grid.height) * 100}%`,
    transform: `translate(-50%, -50%) scale(${1 / mapScale})`,
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

function fitMapSize(viewport: CanvasSize, aspectRatio: number): CanvasSize {
  if (viewport.width / viewport.height > aspectRatio) {
    return {
      width: viewport.height * aspectRatio,
      height: viewport.height,
    }
  }

  return {
    width: viewport.width,
    height: viewport.width / aspectRatio,
  }
}

function coverMapScale(mapSize: CanvasSize, viewportSize: CanvasSize) {
  if (
    mapSize.width <= 0 ||
    mapSize.height <= 0 ||
    viewportSize.width <= 0 ||
    viewportSize.height <= 0
  ) {
    return minMapScale
  }

  return clampNumber(
    Math.max(
      viewportSize.width / mapSize.width,
      viewportSize.height / mapSize.height,
    ),
    minMapScale,
    maxMapScale,
  )
}

async function requestNativeFullscreen(
  element: HTMLElement,
  requestFullscreen: () => Promise<void>,
) {
  let timeoutID: number | undefined

  try {
    const requestCompleted = await Promise.race([
      requestFullscreen.call(element).then(
        () => true,
        () => false,
      ),
      new Promise<boolean>((resolve) => {
        timeoutID = window.setTimeout(
          () => resolve(false),
          fullscreenRequestTimeoutMs,
        )
      }),
    ])

    return requestCompleted && document.fullscreenElement === element
  } finally {
    if (timeoutID !== undefined) window.clearTimeout(timeoutID)
  }
}

function clampMapView(
  view: MapView,
  mapSize: CanvasSize,
  viewportSize: CanvasSize,
  minimumScale = minMapScale,
): MapView {
  const scale = clampNumber(view.scale, minimumScale, maxMapScale)
  const maxOffsetX = Math.max(
    0,
    (mapSize.width * scale - viewportSize.width) / 2,
  )
  const maxOffsetY = Math.max(
    0,
    (mapSize.height * scale - viewportSize.height) / 2,
  )

  return {
    scale,
    offsetX: clampNumber(view.offsetX, -maxOffsetX, maxOffsetX),
    offsetY: clampNumber(view.offsetY, -maxOffsetY, maxOffsetY),
  }
}

function pointerDistance(first: PointerPosition, second: PointerPosition) {
  return Math.hypot(first.x - second.x, first.y - second.y)
}

function pointerMidpoint(first: PointerPosition, second: PointerPosition) {
  return {
    x: (first.x + second.x) / 2,
    y: (first.y + second.y) / 2,
  }
}

function resetTerritoryMapView(
  mapViewRef: { current: MapView },
  setMapView: (view: MapView) => void,
  scale: number,
) {
  const reset = { scale, offsetX: 0, offsetY: 0 }
  mapViewRef.current = reset
  setMapView(reset)
}

function clampNumber(value: number, minimum: number, maximum: number) {
  return Math.max(minimum, Math.min(maximum, value))
}

function clampPercent(value: number) {
  return Math.max(0, Math.min(100, Math.round(value)))
}
