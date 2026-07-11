import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import {
  Check,
  Clock,
  Flag,
  Gem,
  Handshake,
  ListFilter,
  MapPinned,
  Minus,
  Plus,
  RefreshCw,
  Search,
  Zap,
} from "lucide-react"
import { useEffect, useMemo, useState, type ComponentType } from "react"
import { toast } from "sonner"

import type {
  FrontCommandKind,
  FrontCommandOption,
  FrontCommandResponse,
  FrontSessionSummary,
  FrontSitone,
  FrontSnapshot,
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
import { Skeleton } from "@/shared/ui/skeleton"
import { StatusBadge } from "@/shared/ui/status-badge"
import { cn } from "@/shared/utils"

import {
  frontCommandMutationOptions,
  frontCurrentQueryOptions,
  frontSnapshotQueryKey,
  frontSnapshotQueryOptions,
} from "../api/front.query"
import {
  type FrontConnectionState,
  useFrontEvents,
} from "../api/use-front-events"
import { FrontHelpDialog } from "./front-help-dialog"
import { FrontLeaderboard } from "./front-leaderboard"
import { FrontTerritoryDrawer } from "./front-territory-drawer"
import {
  decodeTerritoryRows,
  TerritoryGridMap,
  territoryCellKey,
  type TerritoryCellState,
  type TerritoryTarget,
} from "./territory-grid-map"

const maxFrontSitones = 5

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
  const connectionState = useFrontEvents(frontID)
  const [selectedTarget, setSelectedTarget] = useState<TerritoryTarget | null>(
    null,
  )
  const [selectedSitoneOverride, setSelectedSitoneOverride] = useState<
    string[] | null
  >(() => readStoredSitoneIDs(frontID))
  const [selectedCommand, setSelectedCommand] =
    useState<FrontCommandKind | null>(null)
  const snapshot = snapshotQuery.data
  const visibleSitones = useMemo(
    () => (snapshot ? getVisibleSitones(snapshot) : []),
    [snapshot],
  )
  const selectedSitoneIds = sanitizeSitoneSelection(
    selectedSitoneOverride ??
      snapshot?.selectedSitones.map((sitone) => sitone.sitoneId) ??
      [],
    visibleSitones,
  )

  useEffect(() => {
    if (
      selectedSitoneIds.length === 0 ||
      typeof sessionStorage === "undefined"
    ) {
      return
    }
    sessionStorage.setItem(
      frontSitoneStorageKey(frontID),
      JSON.stringify(selectedSitoneIds),
    )
  }, [frontID, selectedSitoneIds])

  const selectedSitones = selectedSitoneIds.flatMap((sitoneID) => {
    const sitone = visibleSitones.find((entry) => entry.sitoneId === sitoneID)
    return sitone ? [sitone] : []
  })
  const territory = useMemo(
    () =>
      snapshot?.grid
        ? decodeTerritoryRows(
            snapshot.territoryRows,
            snapshot.grid.width,
            snapshot.grid.height,
          )
        : new Map<string, TerritoryCellState>(),
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
  const selectedGarrison =
    (selectedTarget
      ? snapshot?.garrisons.find(
          (garrison) =>
            garrison.x === selectedTarget.x && garrison.y === selectedTarget.y,
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
        ...result.front.availableSitones,
      ])
      toast.success(feedback.title, { description: feedback.description })
      void queryClient.invalidateQueries({ queryKey: ["me", "status"] })
    },
    onError: async (error) => {
      setSelectedCommand(null)
      await snapshotQuery.refetch()
      toast.error(error instanceof Error ? error.message : "命令送出失敗")
    },
  })

  function handleSelectCommand(kind: FrontCommandKind) {
    if (!snapshot || !selectedTarget || !selectedTerritoryCell) return
    if (!snapshot.canPlay) {
      toast.error("觀戰隊伍不能送出戰線命令")
      return
    }
    if (kind !== "withdraw" && selectedSitoneIds.length === 0) {
      toast.error("請先選擇至少一顆前線小石")
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
      toast.error(option?.reason ?? "這個區域目前不能使用此命令")
      return
    }
    const commandCost =
      kind === "attack" && selectedGarrison
        ? selectedGarrison.attackOpenPowerCost
        : kind === "attack"
          ? selectedTerritoryCell.attackOpenPowerCost
          : (option?.cost ?? 0)
    if (snapshot.currentPlayerOpenPower < commandCost) {
      toast.error("開源力不足")
      return
    }

    setSelectedCommand(kind)
    commandMutation.mutate({
      clientCommandId: newClientCommandID(),
      kind,
      targetX: selectedTarget.x,
      targetY: selectedTarget.y,
      expectedRevision: snapshot.revision,
      sitoneIds: kind === "withdraw" ? [] : selectedSitoneIds,
    })
  }

  if (snapshotQuery.isPending) return <FrontPanelSkeleton />
  if (snapshotQuery.isError) {
    return <FrontErrorCard onRetry={() => void snapshotQuery.refetch()} />
  }

  const currentSnapshot = snapshotQuery.data
  return (
    <div className="grid gap-3">
      <FrontSummaryCard
        front={currentSnapshot}
        isFetching={snapshotQuery.isFetching}
        connectionState={connectionState}
        onRefresh={() => void snapshotQuery.refetch()}
      />
      {currentSnapshot.grid ? (
        <TerritoryGridMap
          grid={currentSnapshot.grid}
          rows={currentSnapshot.territoryRows}
          bases={currentSnapshot.bases}
          landmarks={currentSnapshot.landmarks}
          teams={currentSnapshot.teams}
          garrisons={currentSnapshot.garrisons}
          railSegments={currentSnapshot.railSegments}
          tradeRoutes={currentSnapshot.tradeRoutes}
          selectedTarget={selectedTarget}
          onSelectTarget={setSelectedTarget}
        />
      ) : (
        <FrontErrorCard onRetry={() => void snapshotQuery.refetch()} />
      )}
      {currentSnapshot.canPlay ? (
        <FrontSitoneToolbar
          sitones={visibleSitones}
          selectedSitoneIds={selectedSitoneIds}
          onChange={setSelectedSitoneOverride}
        />
      ) : null}
      <FrontLeaderboard
        entries={currentSnapshot.leaderboard}
        teams={currentSnapshot.teams}
        myTeamId={currentSnapshot.myTeamId}
      />
      <FrontTerritoryDrawer
        front={currentSnapshot}
        canPlay={currentSnapshot.canPlay}
        myTeamId={currentSnapshot.myTeamId}
        target={selectedTarget}
        cell={selectedTerritoryCell}
        base={selectedBase}
        landmark={selectedLandmark}
        garrison={selectedGarrison}
        canAttackSelectedOwner={canAttackSelectedOwner}
        teams={currentSnapshot.teams}
        availableCommands={currentSnapshot.availableCommands}
        selectedSitones={selectedSitones}
        selectedCommand={selectedCommand}
        onOpenChange={setSelectedTarget}
        onSelectCommand={handleSelectCommand}
        commandPending={commandMutation.isPending}
      />
    </div>
  )
}

function FrontSummaryCard({
  front,
  isFetching,
  connectionState,
  onRefresh,
}: {
  front: FrontSnapshot
  isFetching: boolean
  connectionState: FrontConnectionState
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
            <StatusBadge
              tone={connectionState === "live" ? "success" : "warning"}
            >
              {connectionState === "live" ? "即時" : "重新連線"}
            </StatusBadge>
            {!front.canPlay ? (
              <StatusBadge tone="magic">唯讀觀戰</StatusBadge>
            ) : null}
          </div>
          <h2 className="text-2xl leading-tight font-black break-words">
            校園領土戰
          </h2>
        </div>
        <div className="flex shrink-0 items-center gap-2">
          <FrontHelpDialog />
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
        <SummaryMetric
          icon={MapPinned}
          label="領土"
          value={front.canPlay ? formatNumber(myTeam?.controlledCells) : "-"}
        />
      </div>
      {front.canPlay && myTeam?.nextSitoneMilestone ? (
        <div className="bg-primary-foreground/10 flex items-center gap-2 rounded-md px-3 py-2 text-xs font-bold">
          <Gem className="size-3.5" aria-hidden />
          歷史最高 {myTeam.maxControlledCells} 格 · 下一顆小石{" "}
          {myTeam.nextSitoneMilestone} 格
        </div>
      ) : null}
      {front.canPlay && myTeam ? (
        <div className="bg-primary-foreground/10 flex items-center gap-2 rounded-md px-3 py-2 text-xs font-bold">
          <Handshake className="size-3.5" aria-hidden />
          本小時交易 {formatNumber(myTeam.tradeHourlyEarned)} /{" "}
          {formatNumber(myTeam.tradeHourlyLimit)} · 進行中{" "}
          {
            front.tradeRoutes.filter(
              (route) =>
                route.status === "active" &&
                (route.sourceTeamId === myTeam.teamId ||
                  route.targetTeamId === myTeam.teamId ||
                  route.waypoints.some(
                    (waypoint) => waypoint.teamId === myTeam.teamId,
                  )),
            ).length
          }
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
    <div className="bg-primary-foreground/10 rounded-md px-3 py-2">
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
  selectedSitoneIds,
  onChange,
}: {
  sitones: FrontSitone[]
  selectedSitoneIds: string[]
  onChange: (sitoneIDs: string[]) => void
}) {
  const [pickerOpen, setPickerOpen] = useState(false)
  const [searchQuery, setSearchQuery] = useState("")
  const [draft, setDraft] = useState<string[]>(selectedSitoneIds)
  const selectedCounts = countSitoneIDs(selectedSitoneIds)
  const draftCounts = countSitoneIDs(draft)
  const normalizedSearch = searchQuery.trim().toLocaleLowerCase("zh-TW")
  const filteredSitones = sitones.filter((sitone) => {
    if (!normalizedSearch) return true
    return [sitone.name, sitone.type, sitone.sitoneId, sitone.abilityName].some(
      (value) => value?.toLocaleLowerCase("zh-TW").includes(normalizedSearch),
    )
  })

  return (
    <>
      <Card className="gap-3 py-4">
        <CardContent className="grid gap-3 px-3">
          <div className="flex items-center justify-between gap-2">
            <div className="flex min-w-0 items-center gap-2">
              <Gem className="text-primary size-4" aria-hidden />
              <h2 className="text-base font-black">前線編隊</h2>
              <Badge variant="outline">{selectedSitoneIds.length}/5</Badge>
            </div>
            <Button
              type="button"
              size="sm"
              variant="outline"
              disabled={sitones.length === 0}
              onClick={() => {
                setDraft(selectedSitoneIds)
                setPickerOpen(true)
              }}
            >
              <ListFilter className="size-4" aria-hidden />
              編隊
            </Button>
          </div>
          {selectedSitoneIds.length > 0 ? (
            <div className="border-border flex flex-wrap gap-2 border-y py-3">
              {Object.entries(selectedCounts).map(([sitoneID, quantity]) => {
                const sitone = sitones.find(
                  (entry) => entry.sitoneId === sitoneID,
                )
                if (!sitone) return null
                return (
                  <div
                    key={sitoneID}
                    className="bg-muted flex min-w-0 items-center gap-2 rounded-md px-2 py-1.5"
                  >
                    <SitoneIcon
                      type={sitone.type ?? ""}
                      iconPath={sitone.iconPath}
                      className="size-8 shrink-0"
                    />
                    <span className="max-w-32 truncate text-xs font-black">
                      {sitone.name}
                    </span>
                    {quantity > 1 ? <Badge>x{quantity}</Badge> : null}
                  </div>
                )
              })}
            </div>
          ) : (
            <div className="text-muted-foreground py-2 text-sm font-bold">
              尚無可派遣小石
            </div>
          )}
          <div className="text-muted-foreground text-xs font-bold">
            第 2–5 顆各提供 5% 編隊加成，符合命令專長時再套用能力值。
          </div>
        </CardContent>
      </Card>

      <Sheet open={pickerOpen} onOpenChange={setPickerOpen}>
        <SheetContent
          side="bottom"
          className="max-h-[82dvh] w-full gap-3 overflow-y-auto rounded-t-md md:inset-x-auto md:inset-y-0 md:top-0 md:right-0 md:bottom-0 md:left-auto md:h-full md:max-h-none md:w-[26rem] md:rounded-t-none md:border-2"
        >
          <SheetHeader className="pr-12 pb-2">
            <SheetTitle>編排前線小石</SheetTitle>
            <SheetDescription>已選 {draft.length}/5 顆</SheetDescription>
          </SheetHeader>
          <div className="grid gap-3 px-4 pb-[max(1rem,env(safe-area-inset-bottom))]">
            <InputGroup>
              <InputGroupAddon>
                <Search className="size-4" aria-hidden />
              </InputGroupAddon>
              <InputGroupInput
                value={searchQuery}
                placeholder="搜尋小石或能力"
                aria-label="搜尋已擁有的小石"
                onChange={(event) => setSearchQuery(event.target.value)}
              />
            </InputGroup>
            <div className="grid gap-2">
              {filteredSitones.map((sitone) => {
                const quantity = draftCounts[sitone.sitoneId] ?? 0
                const canAdd =
                  draft.length < maxFrontSitones &&
                  quantity < sitone.ownedQuantity
                return (
                  <div
                    key={sitone.sitoneId}
                    className="border-border grid grid-cols-[auto_1fr_auto] items-center gap-3 rounded-md border p-2"
                  >
                    <SitoneIcon
                      type={sitone.type ?? ""}
                      iconPath={sitone.iconPath}
                      className="size-10"
                    />
                    <div className="min-w-0">
                      <div className="truncate text-sm font-black">
                        {sitone.name}
                      </div>
                      <div className="text-muted-foreground mt-0.5 text-xs font-bold">
                        {sitone.type ? sitoneMeta(sitone.type).label : "小石"} ·{" "}
                        {sitone.abilityName} {sitone.abilityValue}%
                      </div>
                      <div className="text-muted-foreground mt-0.5 truncate text-[11px] font-bold">
                        專長：
                        {sitone.frontAffinityCommands
                          .map(commandLabel)
                          .join("、") || "無"}
                      </div>
                    </div>
                    <div className="flex items-center gap-1">
                      <Button
                        type="button"
                        size="icon-xs"
                        variant="outline"
                        aria-label={`減少 ${sitone.name}`}
                        disabled={quantity === 0}
                        onClick={() =>
                          setDraft(removeOneSitone(draft, sitone.sitoneId))
                        }
                      >
                        <Minus className="size-3" aria-hidden />
                      </Button>
                      <span className="w-5 text-center text-sm font-black">
                        {quantity}
                      </span>
                      <Button
                        type="button"
                        size="icon-xs"
                        variant="outline"
                        aria-label={`增加 ${sitone.name}`}
                        disabled={!canAdd}
                        onClick={() => setDraft([...draft, sitone.sitoneId])}
                      >
                        <Plus className="size-3" aria-hidden />
                      </Button>
                    </div>
                  </div>
                )
              })}
            </div>
            <Button
              type="button"
              disabled={draft.length < 1 || draft.length > maxFrontSitones}
              onClick={() => {
                onChange(draft)
                setPickerOpen(false)
              }}
            >
              <Check className="size-4" aria-hidden />
              套用編隊
            </Button>
          </div>
        </SheetContent>
      </Sheet>
    </>
  )
}

function getVisibleSitones(snapshot: FrontSnapshot) {
  const sitoneMap = new Map<string, FrontSitone>()
  for (const sitone of [
    ...snapshot.availableSitones,
    ...snapshot.selectedSitones,
  ]) {
    const existing = sitoneMap.get(sitone.sitoneId)
    sitoneMap.set(sitone.sitoneId, {
      ...existing,
      ...sitone,
      ownedQuantity: Math.max(
        existing?.ownedQuantity ?? 0,
        sitone.ownedQuantity,
      ),
    })
  }
  return [...sitoneMap.values()].sort((a, b) =>
    a.name.localeCompare(b.name, "zh-TW"),
  )
}

function sanitizeSitoneSelection(ids: string[], sitones: FrontSitone[]) {
  const available = new Map(sitones.map((sitone) => [sitone.sitoneId, sitone]))
  const used: Record<string, number> = {}
  const result: string[] = []
  for (const sitoneID of ids) {
    const sitone = available.get(sitoneID)
    if (!sitone || !sitone.available || result.length >= maxFrontSitones)
      continue
    used[sitoneID] = (used[sitoneID] ?? 0) + 1
    if (used[sitoneID] <= sitone.ownedQuantity) result.push(sitoneID)
  }
  if (result.length === 0 && sitones[0]?.available)
    result.push(sitones[0].sitoneId)
  return result
}

function countSitoneIDs(ids: string[]) {
  return ids.reduce<Record<string, number>>((counts, id) => {
    counts[id] = (counts[id] ?? 0) + 1
    return counts
  }, {})
}

function removeOneSitone(ids: string[], target: string) {
  const index = ids.lastIndexOf(target)
  return index < 0 ? ids : ids.filter((_, itemIndex) => itemIndex !== index)
}

function frontSitoneStorageKey(frontID: string) {
  return `front-sitone-selection:${frontID}`
}

function readStoredSitoneIDs(frontID: string) {
  if (typeof sessionStorage === "undefined") return null
  try {
    const value: unknown = JSON.parse(
      sessionStorage.getItem(frontSitoneStorageKey(frontID)) ?? "null",
    )
    return Array.isArray(value) &&
      value.every((item) => typeof item === "string")
      ? value
      : null
  } catch {
    return null
  }
}

function commandLabel(kind: FrontCommandKind) {
  const labels: Record<FrontCommandKind, string> = {
    expand: "擴張",
    attack: "攻擊",
    reinforce: "防守",
    repair: "修復",
    rescue: "救援",
    support: "支援",
    answer_challenge: "挑戰",
    station: "駐點",
    withdraw: "撤回",
  }
  return labels[kind]
}

function FrontPanelSkeleton() {
  return (
    <div className="grid gap-3">
      <Skeleton className="h-[164px] rounded-md" />
      <Skeleton className="h-[312px] rounded-md" />
      <Skeleton className="h-[132px] rounded-md" />
      <Skeleton className="h-[220px] rounded-md" />
    </div>
  )
}

function FrontErrorCard({ onRetry }: { onRetry: () => void }) {
  return (
    <Card>
      <CardContent className="px-4">
        <Empty className="min-h-[260px] border-2 border-dashed">
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
  return isBase ? null : "attack"
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
  sitones: FrontSitone[],
) {
  const details: string[] = []
  if (command.capturedCellCount > 0)
    details.push(`取得 ${command.capturedCellCount} 格`)
  if (command.enclosedCellCount > 0)
    details.push(`包圍 ${command.enclosedCellCount} 格`)
  if (command.sitoneEffect.affectedCellBonus > 0) {
    details.push(`小石多影響 ${command.sitoneEffect.affectedCellBonus} 格`)
  }
  if (command.sitoneEffect.defenseBonus > 0) {
    details.push(`小石多修復 ${command.sitoneEffect.defenseBonus}`)
  }
  if (command.scoreDelta > 0) details.push(`+${command.scoreDelta} 戰線分`)
  if (command.frontOpenPowerReward > 0) {
    details.push(`獲得 ${command.frontOpenPowerReward} 開源力`)
  }
  if (command.rewardSitoneId && command.rewardSitoneQuantity > 0) {
    const name = sitones.find(
      (sitone) => sitone.sitoneId === command.rewardSitoneId,
    )?.name
    details.push(`${name ?? "獲得小石"} x${command.rewardSitoneQuantity}`)
  }
  if (command.kind === "station") details.push("小石已開始駐守與自動交易")
  if (command.kind === "withdraw") details.push("駐點小石已返回庫存")
  return {
    title: `${commandLabel(command.kind)}完成`,
    description: details.length > 0 ? details.join(" · ") : undefined,
  }
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
