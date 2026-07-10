import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import {
  Check,
  Clock,
  Flag,
  Gem,
  ListFilter,
  MapPinned,
  RefreshCw,
  Search,
  Shield,
  Zap,
} from "lucide-react"
import { useMemo, useState, type ComponentType } from "react"
import { toast } from "sonner"

import type {
  FrontCommandKind,
  FrontCommandOption,
  FrontCommandResponse,
  FrontSessionSummary,
  FrontSitone,
  FrontSnapshot,
  PlayerSitone,
} from "@/shared/api/game"
import { sitoneMeta } from "@/shared/lib/game-labels"
import { Badge } from "@/shared/ui/badge"
import { Button } from "@/shared/ui/button"
import { Card, CardContent } from "@/shared/ui/card"
import {
  Empty,
  EmptyContent,
  EmptyDescription,
  EmptyHeader,
  EmptyTitle,
} from "@/shared/ui/empty"
import { Skeleton } from "@/shared/ui/skeleton"
import {
  InputGroup,
  InputGroupAddon,
  InputGroupInput,
} from "@/shared/ui/input-group"
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from "@/shared/ui/sheet"
import { SitoneIcon } from "@/shared/ui/sitone-icon"
import { StatusBadge } from "@/shared/ui/status-badge"
import { cn } from "@/shared/utils"

import {
  frontCommandMutationOptions,
  frontCurrentQueryOptions,
  frontPlayerSitonesQueryOptions,
  frontPlayerSitonesQueryKey,
  frontSnapshotQueryKey,
  frontSnapshotQueryOptions,
} from "../api/front.query"
import { FrontLeaderboard } from "./front-leaderboard"
import { FrontHelpDialog } from "./front-help-dialog"
import { FrontNodeDrawer } from "./front-node-drawer"
import { FrontTerritoryDrawer } from "./front-territory-drawer"
import { TerritoryMap } from "./territory-map"
import {
  decodeTerritoryRows,
  territoryCellKey,
  type TerritoryCellState,
  type TerritoryTarget,
} from "./territory-grid-map"

export function FrontMapPanel() {
  const currentQuery = useQuery(frontCurrentQueryOptions())

  if (currentQuery.isPending) return <FrontPanelSkeleton />

  if (currentQuery.isError) {
    return <FrontErrorCard onRetry={() => void currentQuery.refetch()} />
  }

  if (!currentQuery.data.front) return <NoFrontCard />

  return <FrontSnapshotPanel frontID={currentQuery.data.front.id} />
}

function FrontSnapshotPanel({ frontID }: { frontID: string }) {
  const queryClient = useQueryClient()
  const snapshotQuery = useQuery(frontSnapshotQueryOptions(frontID))
  const [selectedCellId, setSelectedCellId] = useState<string | null>(null)
  const [selectedTarget, setSelectedTarget] = useState<TerritoryTarget | null>(
    null,
  )
  const [selectedSitoneId, setSelectedSitoneId] = useState<string | null>(null)
  const [selectedCommand, setSelectedCommand] =
    useState<FrontCommandKind | null>(null)
  const snapshot = snapshotQuery.data
  const playerSitonesQuery = useQuery({
    ...frontPlayerSitonesQueryOptions(),
    enabled: snapshot?.canPlay ?? false,
  })
  const visibleSitones = useMemo(
    () =>
      snapshot
        ? getVisibleSitones(snapshot, playerSitonesQuery.data ?? [])
        : [],
    [playerSitonesQuery.data, snapshot],
  )
  const selectedCell =
    snapshot?.cells.find((cell) => cell.id === selectedCellId) ?? null
  const territory = useMemo(
    () =>
      snapshot?.grid
        ? decodeTerritoryRows(
            snapshot.territoryRows,
            snapshot.grid.width,
            snapshot.grid.height,
          )
        : new Map(),
    [snapshot],
  )
  const selectedTerritoryCell = selectedTarget
    ? (territory.get(territoryCellKey(selectedTarget.x, selectedTarget.y)) ??
      null)
    : null
  const selectedBase =
    (selectedTarget
      ? snapshot?.bases.find(
          (base) => base.x === selectedTarget.x && base.y === selectedTarget.y,
        )
      : null) ?? null
  const selectedLandmark =
    (selectedTarget
      ? snapshot?.landmarks.find(
          (landmark) =>
            landmark.x === selectedTarget.x && landmark.y === selectedTarget.y,
        )
      : null) ?? null
  const canAttackSelectedOwner = Boolean(
    snapshot?.myTeamId &&
    selectedTerritoryCell?.ownerTeamId &&
    territoryOwnerTouchesTeam(
      territory,
      selectedTerritoryCell.ownerTeamId,
      snapshot.myTeamId,
    ),
  )
  const firstAvailableSitone =
    visibleSitones.find((sitone) => sitone.available) ?? visibleSitones[0]
  const selectedSitone =
    visibleSitones.find((sitone) => sitone.sitoneId === selectedSitoneId) ??
    firstAvailableSitone ??
    null
  const commandMutation = useMutation({
    ...frontCommandMutationOptions(frontID),
    onSuccess: (result) => {
      queryClient.setQueryData<FrontSnapshot>(
        frontSnapshotQueryKey(frontID),
        (current) => {
          if (
            current?.revision !== undefined &&
            result.front.revision !== undefined &&
            current.revision > result.front.revision
          ) {
            return current
          }

          return result.front
        },
      )
      setSelectedCommand(null)
      const feedback = commandSuccessFeedback(result.command, [
        ...visibleSitones,
        ...result.front.selectedSitones,
        ...result.front.availableSitones,
      ])
      toast.success(feedback.title, { description: feedback.description })
      void queryClient.invalidateQueries({ queryKey: ["me", "status"] })
      if (result.command.rewardSitoneQuantity > 0) {
        void queryClient.invalidateQueries({
          queryKey: frontPlayerSitonesQueryKey,
        })
      }
    },
    onError: async (error) => {
      setSelectedCommand(null)
      await snapshotQuery.refetch()
      toast.error(
        error instanceof Error
          ? (normalizeFrontCommandReason(error.message) ?? error.message)
          : "命令送出失敗",
      )
    },
  })

  function handleSelectCommand(kind: FrontCommandKind) {
    if (!snapshot) return

    if (snapshot.mapMode === "territory_grid") {
      if (!snapshot.canPlay) {
        toast.error("觀戰隊伍不能送出戰線命令")
        return
      }
      if (!selectedTarget || !selectedTerritoryCell) return
      if (!selectedSitone) {
        toast.error("請先選擇一顆前線小石")
        return
      }

      const contextualCommand = territoryContextCommandKind(
        selectedTerritoryCell.ownerTeamId,
        snapshot.myTeamId,
        selectedBase !== null,
      )
      if (isTerritoryCoreCommand(kind) && kind !== contextualCommand) {
        toast.error("這個方向不能使用此命令")
        return
      }
      if (kind === "attack" && !canAttackSelectedOwner) {
        toast.error("此小隊尚未與我方領土接壤")
        return
      }
      if (kind === "reinforce" && selectedTerritoryCell.defense >= 100) {
        toast.error("此格防禦已滿")
        return
      }

      const option = findAvailableTerritoryCommand(
        snapshot.availableCommands,
        selectedTarget,
        kind,
      )

      if (snapshot.availableCommands.length > 0 && !option?.enabled) {
        toast.error(
          normalizeFrontCommandReason(option?.reason) ??
            "這個區域目前不能使用此命令",
        )
        return
      }

      setSelectedCommand(kind)
      commandMutation.mutate({
        clientCommandId: newClientCommandID(),
        kind,
        targetX: selectedTarget.x,
        targetY: selectedTarget.y,
        expectedRevision: snapshot.revision,
        sitoneId: selectedSitone.sitoneId,
      })
      return
    }

    if (!selectedCell || !selectedSitone) return

    const option = findAvailableCommand(
      snapshot.availableCommands,
      selectedCell.id,
      kind,
    )
    if (!option?.enabled || !option.fromCellId || !option.toCellId) {
      toast.error(
        normalizeFrontCommandReason(option?.reason) ??
          "這個節點目前不能使用這個命令",
      )
      return
    }

    setSelectedCommand(kind)
    commandMutation.mutate({
      clientCommandId: newClientCommandID(),
      kind,
      fromCellId: option.fromCellId,
      toCellId: option.toCellId,
      sitoneId: selectedSitone.sitoneId,
    })
  }

  if (snapshotQuery.isPending) return <FrontPanelSkeleton />

  if (snapshotQuery.isError) {
    return <FrontErrorCard onRetry={() => void snapshotQuery.refetch()} />
  }

  return (
    <div className="grid gap-3">
      <FrontSummaryCard
        front={snapshotQuery.data}
        isFetching={snapshotQuery.isFetching}
        onRefresh={() => void snapshotQuery.refetch()}
      />
      <TerritoryMap
        mapMode={snapshotQuery.data.mapMode}
        cells={snapshotQuery.data.cells}
        teams={snapshotQuery.data.teams}
        activeEvents={snapshotQuery.data.activeEvents}
        grid={snapshotQuery.data.grid}
        territoryRows={snapshotQuery.data.territoryRows}
        bases={snapshotQuery.data.bases}
        landmarks={snapshotQuery.data.landmarks}
        selectedCellId={selectedCell?.id ?? null}
        selectedTarget={selectedTarget}
        onSelectCell={(cellID) => {
          setSelectedTarget(null)
          setSelectedCellId(cellID)
        }}
        onSelectTarget={(target) => {
          setSelectedCellId(null)
          setSelectedTarget(target)
        }}
      />
      {snapshotQuery.data.canPlay ? (
        <FrontSitoneToolbar
          sitones={visibleSitones}
          selectedSitoneId={selectedSitone?.sitoneId ?? null}
          onSelectSitone={setSelectedSitoneId}
          fullListLoading={playerSitonesQuery.isPending}
          fullListError={playerSitonesQuery.isError}
          onRetryFullList={() => void playerSitonesQuery.refetch()}
        />
      ) : null}
      <FrontLeaderboard
        entries={snapshotQuery.data.leaderboard}
        teams={snapshotQuery.data.teams}
        myTeamId={snapshotQuery.data.myTeamId}
        mapMode={snapshotQuery.data.mapMode}
      />
      {snapshotQuery.data.mapMode === "territory_grid" ? (
        <FrontTerritoryDrawer
          front={snapshotQuery.data}
          canPlay={snapshotQuery.data.canPlay}
          myTeamId={snapshotQuery.data.myTeamId}
          target={selectedTarget}
          cell={selectedTerritoryCell}
          base={selectedBase}
          landmark={selectedLandmark}
          canAttackSelectedOwner={canAttackSelectedOwner}
          teams={snapshotQuery.data.teams}
          availableCommands={snapshotQuery.data.availableCommands}
          selectedSitone={selectedSitone}
          selectedCommand={selectedCommand}
          onOpenChange={setSelectedTarget}
          onSelectCommand={handleSelectCommand}
          commandPending={commandMutation.isPending}
        />
      ) : (
        <FrontNodeDrawer
          front={snapshotQuery.data}
          cell={selectedCell}
          teams={snapshotQuery.data.teams}
          activeEvents={snapshotQuery.data.activeEvents}
          availableCommands={snapshotQuery.data.availableCommands}
          selectedSitone={selectedSitone}
          selectedCommand={selectedCommand}
          onOpenChange={setSelectedCellId}
          onSelectCommand={handleSelectCommand}
          commandPending={commandMutation.isPending}
        />
      )}
    </div>
  )
}

function FrontSummaryCard({
  front,
  isFetching,
  onRefresh,
}: {
  front: FrontSnapshot
  isFetching: boolean
  onRefresh: () => void
}) {
  const myTeam = front.myTeamId
    ? front.teams.find((team) => team.teamId === front.myTeamId)
    : undefined
  return (
    <Card className="bg-ink text-primary-foreground gap-4 px-4 py-4">
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <div className="mb-2 flex flex-wrap items-center gap-2">
            <StatusBadge tone={statusTone(front.status)}>
              {statusLabel(front.status)}
            </StatusBadge>
            <Badge variant="secondary">Tick {front.tick}</Badge>
            {!front.canPlay ? (
              <StatusBadge tone="magic">唯讀觀戰</StatusBadge>
            ) : null}
          </div>
          <h2 className="text-2xl leading-tight font-black break-words">
            {front.mapMode === "territory_grid" ? "校園領土戰" : "開源戰線"}
          </h2>
        </div>
        <div className="flex shrink-0 items-center gap-2">
          <FrontHelpDialog mapMode={front.mapMode} />
          <Button
            type="button"
            size="icon-sm"
            variant="secondary"
            aria-label="重新整理戰線"
            onClick={onRefresh}
            disabled={isFetching}
          >
            <RefreshCw
              className={cn("size-4", isFetching && "animate-spin")}
              aria-hidden
            />
          </Button>
        </div>
      </div>

      <div className="grid grid-cols-2 gap-2">
        <SummaryMetric
          icon={Clock}
          label="結束"
          value={formatTime(front.endsAt)}
        />
        <SummaryMetric
          icon={Zap}
          label="開源力"
          value={formatNumber(front.currentPlayerOpenPower)}
        />
        <SummaryMetric
          icon={Flag}
          label="小隊排名"
          value={front.myTeamRank ? `#${front.myTeamRank}` : "-"}
        />
        {front.mapMode === "territory_grid" ? (
          <SummaryMetric
            icon={MapPinned}
            label="領土"
            value={front.canPlay ? formatNumber(myTeam?.controlledCells) : "-"}
          />
        ) : (
          <SummaryMetric
            icon={Shield}
            label="支援"
            value={formatNumber(front.supportTokens)}
          />
        )}
      </div>
      {front.mapMode === "territory_grid" &&
      front.canPlay &&
      myTeam?.nextSitoneMilestone ? (
        <div className="bg-primary-foreground/10 flex items-center gap-2 rounded-md px-3 py-2 text-xs font-bold">
          <Gem className="size-3.5" aria-hidden />
          歷史最高 {myTeam.maxControlledCells} 格 · 下一顆小石{" "}
          {myTeam.nextSitoneMilestone} 格
        </div>
      ) : null}
    </Card>
  )
}

function SummaryMetric({
  icon: Icon,
  label,
  value,
}: {
  icon: ComponentType<{ className?: string }>
  label: string
  value: string
}) {
  return (
    <div className="bg-primary-foreground/10 rounded-[1rem] px-3 py-2">
      <div className="text-primary-foreground/70 flex items-center gap-1.5 text-xs font-bold">
        <Icon className="size-3.5" aria-hidden />
        {label}
      </div>
      <div className="mt-1 text-lg font-black">{value}</div>
    </div>
  )
}

function FrontSitoneToolbar({
  sitones,
  selectedSitoneId,
  onSelectSitone,
  fullListLoading,
  fullListError,
  onRetryFullList,
}: {
  sitones: SelectableFrontSitone[]
  selectedSitoneId: string | null
  onSelectSitone: (sitoneID: string) => void
  fullListLoading: boolean
  fullListError: boolean
  onRetryFullList: () => void
}) {
  const [pickerOpen, setPickerOpen] = useState(false)
  const [searchQuery, setSearchQuery] = useState("")
  const selectedSitone =
    sitones.find((sitone) => sitone.sitoneId === selectedSitoneId) ?? null
  const normalizedSearch = searchQuery.trim().toLocaleLowerCase("zh-TW")
  const filteredSitones = sitones.filter((sitone) => {
    if (!normalizedSearch) return true

    return [sitone.name, sitone.type, sitone.sitoneId].some((value) =>
      value?.toLocaleLowerCase("zh-TW").includes(normalizedSearch),
    )
  })

  return (
    <>
      <Card className="gap-3 py-4">
        <CardContent className="grid gap-3 px-3">
          <div className="flex items-center justify-between gap-2">
            <div className="flex min-w-0 items-center gap-2">
              <Gem className="text-primary size-4" aria-hidden />
              <h2 className="text-base font-black">前線小石</h2>
              <Badge variant="outline">{sitones.length}</Badge>
            </div>
            <Button
              type="button"
              size="sm"
              variant="outline"
              disabled={sitones.length === 0}
              onClick={() => setPickerOpen(true)}
            >
              <ListFilter className="size-4" aria-hidden />
              選擇
            </Button>
          </div>

          {selectedSitone ? (
            <div className="border-border flex min-w-0 items-center gap-3 border-y py-3">
              <SitoneIcon
                type={selectedSitone.type ?? ""}
                iconPath={selectedSitone.iconPath}
                className="size-11 shrink-0"
              />
              <div className="min-w-0 flex-1">
                <div className="truncate text-sm font-black">
                  {selectedSitone.name}
                </div>
                <div className="text-muted-foreground mt-1 truncate text-xs font-bold">
                  {sitoneStatusLabel(selectedSitone)}
                </div>
              </div>
              {(selectedSitone.ownedQuantity ?? 0) > 1 ? (
                <Badge variant="secondary">
                  x{selectedSitone.ownedQuantity}
                </Badge>
              ) : null}
            </div>
          ) : (
            <div className="text-muted-foreground py-2 text-sm font-bold">
              {fullListLoading ? "讀取小石中" : "尚無可派遣小石"}
            </div>
          )}

          {fullListError ? (
            <div className="bg-muted border-border flex items-center justify-between gap-2 rounded-md border px-3 py-2">
              <span className="text-muted-foreground text-xs font-bold">
                完整清單讀取失敗，目前顯示前線預設小石
              </span>
              <Button
                type="button"
                size="icon-xs"
                variant="ghost"
                aria-label="重新讀取完整小石清單"
                onClick={onRetryFullList}
              >
                <RefreshCw className="size-3" aria-hidden />
              </Button>
            </div>
          ) : null}
        </CardContent>
      </Card>

      <Sheet open={pickerOpen} onOpenChange={setPickerOpen}>
        <SheetContent
          side="bottom"
          className="max-h-[82dvh] w-full gap-3 overflow-y-auto rounded-t-[1.25rem] md:inset-x-auto md:inset-y-0 md:top-0 md:right-0 md:bottom-0 md:left-auto md:h-full md:max-h-none md:w-[24rem] md:rounded-t-none md:rounded-l-[1.25rem] md:border-2"
        >
          <SheetHeader className="pr-12 pb-2">
            <SheetTitle>選擇前線小石</SheetTitle>
            <SheetDescription>已擁有 {sitones.length} 種小石</SheetDescription>
          </SheetHeader>

          <div className="grid gap-3 px-4 pb-[max(1rem,env(safe-area-inset-bottom))]">
            <InputGroup>
              <InputGroupAddon>
                <Search className="size-4" aria-hidden />
              </InputGroupAddon>
              <InputGroupInput
                value={searchQuery}
                placeholder="搜尋小石"
                aria-label="搜尋已擁有的小石"
                onChange={(event) => setSearchQuery(event.target.value)}
              />
            </InputGroup>

            {filteredSitones.length > 0 ? (
              <div className="grid grid-cols-2 gap-2">
                {filteredSitones.map((sitone) => {
                  const selected = sitone.sitoneId === selectedSitoneId

                  return (
                    <Button
                      key={sitone.sitoneId}
                      type="button"
                      variant={selected ? "default" : "outline"}
                      className="h-16 min-w-0 justify-start gap-2 px-2"
                      disabled={!sitone.available}
                      aria-pressed={selected}
                      onClick={() => {
                        onSelectSitone(sitone.sitoneId)
                        setPickerOpen(false)
                      }}
                    >
                      <SitoneIcon
                        type={sitone.type ?? ""}
                        iconPath={sitone.iconPath}
                        className="size-9 shrink-0"
                      />
                      <span className="min-w-0 flex-1 text-left">
                        <span className="block truncate text-xs font-black">
                          {sitone.name}
                        </span>
                        <span className="mt-1 block truncate text-[11px] font-bold opacity-75">
                          {sitoneStatusLabel(sitone)}
                        </span>
                      </span>
                      {selected ? (
                        <Check className="size-4 shrink-0" aria-hidden />
                      ) : null}
                    </Button>
                  )
                })}
              </div>
            ) : (
              <div className="text-muted-foreground py-8 text-center text-sm font-bold">
                找不到符合的小石
              </div>
            )}
          </div>
        </SheetContent>
      </Sheet>
    </>
  )
}

function FrontPanelSkeleton() {
  return (
    <div className="grid gap-3">
      <Skeleton className="h-[164px] rounded-[1.375rem]" />
      <Skeleton className="h-[312px] rounded-[1.375rem]" />
      <Skeleton className="h-[132px] rounded-[1.375rem]" />
      <Skeleton className="h-[220px] rounded-[1.375rem]" />
    </div>
  )
}

function FrontErrorCard({ onRetry }: { onRetry: () => void }) {
  return (
    <Card>
      <CardContent className="px-4">
        <Empty className="min-h-[320px] border-2 border-dashed">
          <EmptyHeader>
            <EmptyTitle>戰線資料讀取失敗</EmptyTitle>
            <EmptyDescription>請稍後再同步一次。</EmptyDescription>
          </EmptyHeader>
          <EmptyContent>
            <Button type="button" onClick={onRetry}>
              <RefreshCw className="size-4" aria-hidden />
              重新整理
            </Button>
          </EmptyContent>
        </Empty>
      </CardContent>
    </Card>
  )
}

function NoFrontCard() {
  return (
    <Card>
      <CardContent className="px-4">
        <Empty className="min-h-[320px] border-2 border-dashed">
          <EmptyHeader>
            <EmptyTitle>開源戰線尚未開放</EmptyTitle>
            <EmptyDescription>
              目前沒有可進入的 Front session。
            </EmptyDescription>
          </EmptyHeader>
        </Empty>
      </CardContent>
    </Card>
  )
}

type SelectableFrontSitone = FrontSitone & {
  ownedQuantity?: number
}

function getVisibleSitones(
  snapshot: FrontSnapshot,
  ownedSitones: PlayerSitone[],
) {
  const sitoneMap = new Map<string, SelectableFrontSitone>()
  const frontStates = [
    ...snapshot.selectedSitones,
    ...snapshot.availableSitones,
  ]
  const frontStateByID = new Map(
    frontStates.map((sitone) => [sitone.sitoneId, sitone] as const),
  )

  ownedSitones
    .filter((record) => record.quantity > 0)
    .forEach((record) => {
      const frontState = frontStateByID.get(record.sitoneId)
      sitoneMap.set(record.sitoneId, {
        sitoneId: record.sitoneId,
        name: record.sitone.name,
        type: record.sitone.type,
        iconPath: record.sitone.iconPath,
        available: frontState?.available ?? true,
        cooldownUntilTick: frontState?.cooldownUntilTick,
        remainingCooldownTicks: frontState?.remainingCooldownTicks,
        assignedCellId: frontState?.assignedCellId,
        ownedQuantity: record.quantity,
      })
    })

  frontStates.forEach((sitone) => {
    const existing = sitoneMap.get(sitone.sitoneId)
    sitoneMap.set(sitone.sitoneId, {
      ...existing,
      ...sitone,
      ownedQuantity: existing?.ownedQuantity,
    })
  })

  return [...sitoneMap.values()].sort((first, second) => {
    if (first.available !== second.available) return first.available ? -1 : 1
    return first.name.localeCompare(second.name, "zh-TW")
  })
}

function sitoneStatusLabel(sitone: FrontSitone) {
  const cooldown =
    sitone.remainingCooldownTicks ?? (sitone.cooldownUntilTick ? 1 : undefined)

  if (cooldown) return `冷卻 ${cooldown} tick`
  if (!sitone.available) return "目前不可派遣"
  return sitone.type ? sitoneMeta(sitone.type).label : "可派遣"
}

function statusLabel(status: FrontSessionSummary["status"]) {
  const labels: Record<FrontSessionSummary["status"], string> = {
    closed: "未開放",
    quiet: "準備中",
    open_play: "進行中",
    surge: "衝刺",
    booth_window: "攤位窗口",
    finale_freeze: "結算凍結",
    completed: "已完成",
  }

  return labels[status]
}

function statusTone(status: FrontSessionSummary["status"]) {
  if (status === "open_play" || status === "booth_window") return "success"
  if (status === "surge") return "warning"
  if (status === "finale_freeze" || status === "completed") return "info"

  return "magic"
}

function findAvailableCommand(
  options: FrontCommandOption[],
  cellID: string,
  kind: FrontCommandKind,
) {
  return options.find(
    (option) =>
      option.kind === kind &&
      option.toCellId === cellID &&
      option.fromCellId &&
      option.enabled,
  )
}

function findAvailableTerritoryCommand(
  options: FrontCommandOption[],
  target: TerritoryTarget,
  kind: FrontCommandKind,
) {
  return options.find(
    (option) =>
      option.kind === kind &&
      (option.targetX === undefined || option.targetX === target.x) &&
      (option.targetY === undefined || option.targetY === target.y),
  )
}

function territoryOwnerTouchesTeam(
  territory: Map<string, TerritoryCellState>,
  ownerTeamID: string,
  myTeamID: string,
) {
  const directions = [
    [0, -1],
    [1, 0],
    [0, 1],
    [-1, 0],
  ] as const

  for (const cell of territory.values()) {
    if (cell.ownerTeamId !== ownerTeamID) continue

    for (const [offsetX, offsetY] of directions) {
      const neighbor = territory.get(
        territoryCellKey(cell.x + offsetX, cell.y + offsetY),
      )
      if (neighbor?.ownerTeamId === myTeamID) return true
    }
  }

  return false
}

function territoryContextCommandKind(
  ownerTeamID: string | undefined,
  myTeamID: string | undefined,
  isBase: boolean,
): FrontCommandKind | null {
  if (!ownerTeamID) return "expand"
  if (ownerTeamID === myTeamID) return "reinforce"
  if (!isBase) return "attack"

  return null
}

function isTerritoryCoreCommand(kind: FrontCommandKind) {
  return kind === "expand" || kind === "attack" || kind === "reinforce"
}

function newClientCommandID() {
  if (typeof crypto !== "undefined" && "randomUUID" in crypto) {
    return crypto.randomUUID()
  }

  return `${Date.now()}-${Math.random().toString(36).slice(2)}`
}

function commandSuccessFeedback(
  command: FrontCommandResponse,
  sitones: SelectableFrontSitone[],
) {
  const labels: Record<FrontCommandKind, string> = {
    expand: "擴張完成",
    attack: "攻擊完成",
    reinforce: "防守完成",
    repair: "修復已完成",
    scout: "偵查完成",
    rescue: "救援已完成",
    support: "小石採集完成",
    answer_challenge: "挑戰完成",
  }
  const details: string[] = []

  if (command.capturedCellCount > 0) {
    details.push(`取得 ${command.capturedCellCount} 格`)
  }
  if (command.enclosedCellCount > 0) {
    details.push(`包圍 ${command.enclosedCellCount} 格`)
  }
  if (command.scoreDelta > 0) details.push(`+${command.scoreDelta} 戰線分`)
  if (command.rewardSitoneId && command.rewardSitoneQuantity > 0) {
    const sitoneName = sitones.find(
      (sitone) => sitone.sitoneId === command.rewardSitoneId,
    )?.name
    details.push(
      sitoneName
        ? `獲得 ${sitoneName} x${command.rewardSitoneQuantity}`
        : `獲得小石 x${command.rewardSitoneQuantity}`,
    )
  }

  return {
    title: labels[command.kind],
    description: details.length > 0 ? details.join(" · ") : undefined,
  }
}

function normalizeFrontCommandReason(reason: string | undefined) {
  return reason
}

function formatTime(value: string | undefined) {
  if (!value) return "-"

  const date = new Date(value)

  if (Number.isNaN(date.getTime())) return "-"

  return date.toLocaleTimeString("zh-TW", {
    hour: "2-digit",
    minute: "2-digit",
  })
}

function formatNumber(value: number | undefined) {
  return value == null ? "-" : value.toLocaleString("zh-TW")
}
