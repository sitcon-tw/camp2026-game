import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import {
  Clock,
  Flag,
  Gem,
  MapPinned,
  RefreshCw,
  Shield,
  Zap,
} from "lucide-react"
import { useMemo, useState, type ComponentType } from "react"
import { toast } from "sonner"

import type {
  FrontCommandKind,
  FrontCommandOption,
  FrontSessionSummary,
  FrontSitone,
  FrontSnapshot,
} from "@/shared/api/game"
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
import { StatusBadge } from "@/shared/ui/status-badge"
import { cn } from "@/shared/utils"

import {
  frontCommandMutationOptions,
  frontCurrentQueryOptions,
  frontSnapshotQueryKey,
  frontSnapshotQueryOptions,
} from "../api/front.query"
import { FrontLeaderboard } from "./front-leaderboard"
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
  const visibleSitones = useMemo(
    () => (snapshot ? getVisibleSitones(snapshot) : []),
    [snapshot],
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
      toast.success(commandSuccessMessage(result.command.kind))
    },
    onError: async (error) => {
      setSelectedCommand(null)
      await snapshotQuery.refetch()
      toast.error(error instanceof Error ? error.message : "命令送出失敗")
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
        toast.error(option?.reason ?? "這個區域目前不能使用此命令")
        return
      }

      setSelectedCommand(kind)
      commandMutation.mutate({
        clientCommandId: newClientCommandID(),
        kind,
        targetX: selectedTarget.x,
        targetY: selectedTarget.y,
        expectedRevision: snapshot.revision,
        sitoneId: selectedSitone?.sitoneId,
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
      toast.error(option?.reason ?? "這個節點目前不能使用這個命令")
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
  const frontOpenPower =
    front.currentTeamFrontOpenPower ?? myTeam?.frontOpenPower

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

      <div className="grid grid-cols-2 gap-2">
        <SummaryMetric
          icon={Clock}
          label="結束"
          value={formatTime(front.endsAt)}
        />
        <SummaryMetric
          icon={Zap}
          label="前線開源力"
          value={formatNumber(frontOpenPower)}
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
}: {
  sitones: FrontSitone[]
  selectedSitoneId: string | null
  onSelectSitone: (sitoneID: string) => void
}) {
  return (
    <Card className="gap-3">
      <CardContent className="grid gap-3 px-3">
        <div className="flex items-center gap-2">
          <Gem className="text-primary size-4" aria-hidden />
          <h2 className="text-base font-black">前線小石</h2>
        </div>
        {sitones.length > 0 ? (
          <div className="grid grid-cols-3 gap-2">
            {sitones.slice(0, 5).map((sitone) => {
              const selected = sitone.sitoneId === selectedSitoneId
              const cooldown =
                sitone.remainingCooldownTicks ??
                (sitone.cooldownUntilTick ? 1 : undefined)

              return (
                <Button
                  key={sitone.sitoneId}
                  type="button"
                  variant={selected ? "default" : "outline"}
                  className="h-[4.75rem] min-w-0 flex-col rounded-[1rem] px-2 py-2"
                  disabled={!sitone.available}
                  aria-pressed={selected}
                  onClick={() => onSelectSitone(sitone.sitoneId)}
                >
                  <span className="max-w-full truncate text-sm">
                    {sitone.name}
                  </span>
                  <span className="text-xs font-bold opacity-80">
                    {cooldown ? `${cooldown} tick` : (sitone.type ?? "ready")}
                  </span>
                </Button>
              )
            })}
          </div>
        ) : (
          <div className="text-muted-foreground text-sm font-bold">
            尚無可派遣小石
          </div>
        )}
      </CardContent>
    </Card>
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

function getVisibleSitones(snapshot: FrontSnapshot) {
  const sitoneMap = new Map<string, FrontSitone>()

  snapshot.selectedSitones.forEach((sitone) =>
    sitoneMap.set(sitone.sitoneId, sitone),
  )
  snapshot.availableSitones.forEach((sitone) =>
    sitoneMap.set(sitone.sitoneId, sitone),
  )

  return [...sitoneMap.values()].slice(0, 5)
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

function commandSuccessMessage(kind: FrontCommandKind) {
  const labels: Record<FrontCommandKind, string> = {
    expand: "擴張已送出",
    attack: "攻擊已送出",
    reinforce: "防守已送出",
    repair: "修復已完成",
    scout: "偵查已送出",
    rescue: "救援已完成",
    support: "支援已送出",
    answer_challenge: "挑戰已送出",
  }

  return labels[kind]
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
