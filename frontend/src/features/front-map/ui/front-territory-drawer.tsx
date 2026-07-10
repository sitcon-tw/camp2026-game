import {
  BookOpenCheck,
  Eye,
  Handshake,
  LifeBuoy,
  LockKeyhole,
  MoveUpRight,
  ShieldCheck,
  ShieldPlus,
  Swords,
  Wrench,
  Zap,
} from "lucide-react"
import type { ComponentType, CSSProperties } from "react"

import type {
  FrontCommandKind,
  FrontCommandOption,
  FrontSessionSummary,
  FrontSitone,
  FrontSnapshot,
  FrontTerritoryBase,
  FrontTerritoryLandmark,
  FrontTeamState,
} from "@/shared/api/game"
import { Badge } from "@/shared/ui/badge"
import { Button } from "@/shared/ui/button"
import { Progress } from "@/shared/ui/progress"
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from "@/shared/ui/sheet"
import { StatusBadge } from "@/shared/ui/status-badge"
import { cn } from "@/shared/utils"

import { getTeamName, getTerritoryTeamColor } from "./front-map-style"
import type { TerritoryCellState, TerritoryTarget } from "./territory-grid-map"
import { calculateFrontSitonePreview } from "../lib/front-sitone-effect"

type FrontTerritoryDrawerProps = {
  front: FrontSnapshot
  canPlay: boolean
  myTeamId?: string
  target: TerritoryTarget | null
  cell: TerritoryCellState | null
  base: FrontTerritoryBase | null
  landmark: FrontTerritoryLandmark | null
  canAttackSelectedOwner: boolean
  teams: FrontTeamState[]
  availableCommands: FrontCommandOption[]
  selectedSitones: FrontSitone[]
  selectedCommand: FrontCommandKind | null
  onOpenChange: (target: TerritoryTarget | null) => void
  onSelectCommand: (kind: FrontCommandKind) => void
  commandPending: boolean
}

type CommandItem = {
  kind: FrontCommandKind
  label: string
  icon: ComponentType<{ className?: string }>
}

const commandItems: CommandItem[] = [
  { kind: "expand", label: "擴張", icon: MoveUpRight },
  { kind: "attack", label: "攻擊", icon: Swords },
  { kind: "reinforce", label: "防守", icon: ShieldPlus },
  { kind: "repair", label: "修復", icon: Wrench },
  { kind: "rescue", label: "救援", icon: LifeBuoy },
  { kind: "support", label: "支援", icon: Handshake },
  { kind: "answer_challenge", label: "挑戰", icon: BookOpenCheck },
]

const coreCommandKinds = new Set<FrontCommandKind>([
  "expand",
  "attack",
  "reinforce",
])

const landmarkCommandKinds: Record<string, FrontCommandKind[]> = {
  challenge: ["answer_challenge"],
  course: ["answer_challenge"],
  repair: ["repair"],
  rescue: ["rescue"],
  resource: ["support"],
  support: ["support"],
}

export function FrontTerritoryDrawer({
  front,
  canPlay,
  myTeamId,
  target,
  cell,
  base,
  landmark,
  canAttackSelectedOwner,
  teams,
  availableCommands,
  selectedSitones,
  selectedCommand,
  onOpenChange,
  onSelectCommand,
  commandPending,
}: FrontTerritoryDrawerProps) {
  return (
    <Sheet
      open={Boolean(target && cell)}
      onOpenChange={(open) => {
        if (!open) onOpenChange(null)
      }}
    >
      {target && cell ? (
        <SheetContent
          side="bottom"
          className="max-h-[78dvh] w-full gap-3 overflow-y-auto rounded-t-[1.25rem] md:inset-x-auto md:inset-y-0 md:top-0 md:right-0 md:bottom-0 md:left-auto md:h-full md:max-h-none md:w-[24rem] md:rounded-t-none md:rounded-l-[1.25rem] md:border-2"
        >
          <TerritoryDrawerContent
            front={front}
            canPlay={canPlay}
            myTeamId={myTeamId}
            target={target}
            cell={cell}
            base={base}
            landmark={landmark}
            canAttackSelectedOwner={canAttackSelectedOwner}
            teams={teams}
            availableCommands={availableCommands}
            selectedSitones={selectedSitones}
            selectedCommand={selectedCommand}
            onSelectCommand={onSelectCommand}
            commandPending={commandPending}
          />
        </SheetContent>
      ) : null}
    </Sheet>
  )
}

function TerritoryDrawerContent({
  front,
  canPlay,
  myTeamId,
  target,
  cell,
  base,
  landmark,
  canAttackSelectedOwner,
  teams,
  availableCommands,
  selectedSitones,
  selectedCommand,
  onSelectCommand,
  commandPending,
}: Omit<FrontTerritoryDrawerProps, "target" | "cell" | "onOpenChange"> & {
  target: TerritoryTarget
  cell: TerritoryCellState
}) {
  const ownerColor = getTerritoryTeamColor(cell.ownerTeamId, teams)
  const ownerTeam = teams.find((team) => team.teamId === cell.ownerTeamId)
  const ownerName = getTeamName(cell.ownerTeamId, teams)
  const playable = isPlayable(front.status) && canPlay
  const commandList = visibleCommands(
    availableCommands,
    target,
    landmark,
    cell,
    base,
    myTeamId,
  )
  const defenseFull =
    cell.ownerTeamId === myTeamId && clampPercent(cell.defense) >= 100
  const hasCommandOptions = availableCommands.length > 0
  const disabledCommandReasons = commandList
    .map(({ item, option }) => ({
      item,
      reason:
        item.kind === "attack" && !canAttackSelectedOwner
          ? "此小隊尚未與我方領土接壤"
          : item.kind === "reinforce" && defenseFull
            ? undefined
            : option && !option.enabled
              ? normalizeCommandReason(option.reason)
              : undefined,
    }))
    .filter(({ reason }) => Boolean(reason))
  const enabledCommandNotices = commandList
    .filter(({ option }) => option?.enabled && option.reason)
    .map(({ item, option }) => ({
      kind: item.kind,
      label: option?.label ?? item.label,
      reason: normalizeCommandReason(option?.reason),
    }))

  return (
    <>
      <SheetHeader className="gap-2 pr-12 pb-2">
        <div className="flex flex-wrap items-center gap-2">
          <span
            className={cn(
              "size-3 rounded-full",
              ownerColor?.dot ?? "bg-muted-foreground",
            )}
            style={teamColorStyle(ownerTeam?.color)}
            aria-hidden
          />
          {base ? (
            <StatusBadge tone="warning">
              <LockKeyhole className="size-3" aria-hidden />
              永久基地
            </StatusBadge>
          ) : null}
          {landmark ? (
            <StatusBadge tone="info">{landmark.label}</StatusBadge>
          ) : null}
          {!canPlay ? (
            <StatusBadge tone="magic">
              <Eye className="size-3" aria-hidden />
              觀戰
            </StatusBadge>
          ) : null}
        </div>
        <SheetTitle className="text-xl leading-tight">
          {territoryTitle(cell.ownerTeamId, base, landmark, teams)}
        </SheetTitle>
        <SheetDescription>{ownerName}</SheetDescription>
      </SheetHeader>

      <div className="grid gap-3 px-4 pb-[max(1rem,env(safe-area-inset-bottom))]">
        <TerritoryMeter
          label={base ? "基地防禦" : "領土防禦"}
          value={base?.coreDefense ?? cell.defense}
          icon={base ? ShieldCheck : ShieldPlus}
        />

        <div className="border-border grid grid-cols-2 gap-2 border-y py-3">
          <TerritoryStat
            label="狀態"
            value={territoryRelation(cell.ownerTeamId, myTeamId)}
          />
          <TerritoryStat
            label="地點"
            value={landmark?.label ?? (base ? "基地核心" : "一般領土")}
          />
        </div>

        {defenseFull ? (
          <div className="bg-muted border-border flex items-center gap-2 rounded-md border px-3 py-2 text-sm font-bold">
            <ShieldCheck className="text-status-success size-4" aria-hidden />
            此格防禦已滿
          </div>
        ) : null}

        {!canPlay ? (
          <div className="text-muted-foreground flex items-center gap-2 py-2 text-sm font-bold">
            <Eye className="size-4" aria-hidden />
            目前為唯讀觀戰
          </div>
        ) : commandList.length === 0 ? (
          <div className="bg-muted border-border flex items-center gap-2 rounded-md border px-3 py-2 text-sm font-bold">
            <LockKeyhole className="size-4" aria-hidden />
            基地核心不可佔領
          </div>
        ) : (
          <div className="grid gap-2">
            <div className="flex min-w-0 flex-wrap items-center justify-between gap-2">
              <div className="text-sm font-black">命令</div>
              <div className="ml-auto flex min-w-0 flex-wrap justify-end gap-1.5">
                <Badge variant="outline" className="gap-1">
                  <Zap className="size-3" aria-hidden />
                  開源力 {formatNumber(front.currentPlayerOpenPower)}
                </Badge>
                {selectedSitones.length > 0 ? (
                  <Badge variant="secondary" className="max-w-[9rem] truncate">
                    {selectedSitones.length} 顆小石
                  </Badge>
                ) : null}
              </div>
            </div>

            {enabledCommandNotices.map((notice) => (
              <div
                key={notice.kind}
                className="border-status-warning bg-status-warning/20 flex items-start gap-2 rounded-md border px-3 py-2 text-xs font-bold"
              >
                <Zap className="mt-0.5 size-3.5 shrink-0" aria-hidden />
                <span>
                  {notice.label}：{notice.reason}
                </span>
              </div>
            ))}

            <div className="grid grid-cols-2 gap-2">
              {commandList.map(({ item, option }) => {
                const localReason =
                  item.kind === "attack" && !canAttackSelectedOwner
                    ? "此小隊尚未與我方領土接壤"
                    : item.kind === "reinforce" && defenseFull
                      ? "此格防禦已滿"
                      : undefined
                const enabled = Boolean(
                  playable &&
                  selectedSitones.length > 0 &&
                  (!hasCommandOptions || option?.enabled) &&
                  !localReason,
                )
                const active = selectedCommand === item.kind
                const pending = commandPending && active

                return (
                  <Button
                    key={item.kind}
                    type="button"
                    variant={active ? "default" : "outline"}
                    className="h-11 min-w-0 justify-start rounded-md px-3"
                    disabled={!enabled || commandPending}
                    aria-pressed={active}
                    title={
                      localReason ?? normalizeCommandReason(option?.reason)
                    }
                    onClick={() => onSelectCommand(item.kind)}
                  >
                    <CommandIcon icon={item.icon} />
                    <span className="truncate">
                      {pending ? "送出中" : (option?.label ?? item.label)}
                    </span>
                    {option?.cost ? (
                      <span className="ml-auto inline-flex items-center gap-1 text-xs">
                        <Zap className="size-3" aria-hidden />
                        {option.cost}
                      </span>
                    ) : null}
                  </Button>
                )
              })}
            </div>

            {selectedSitones.length > 0 ? (
              <div className="bg-muted border-border grid gap-1.5 rounded-md border px-3 py-2">
                {commandList.map(({ item }) => (
                  <div
                    key={item.kind}
                    className="text-muted-foreground text-xs font-bold"
                  >
                    {sitonePreviewLabel(item.kind, selectedSitones)}
                  </div>
                ))}
              </div>
            ) : null}

            {disabledCommandReasons.map(({ item, reason }) => (
              <div
                key={item.kind}
                className="text-muted-foreground flex items-center gap-1.5 text-xs font-bold"
              >
                <ShieldCheck className="size-3.5" aria-hidden />
                {item.label}：{reason}
              </div>
            ))}

            {selectedSitones.length === 0 ? (
              <div className="text-muted-foreground text-xs font-bold">
                請先選擇至少一顆前線小石
              </div>
            ) : null}

            {!playable ? (
              <div className="text-muted-foreground text-xs font-bold">
                戰線目前不可操作
              </div>
            ) : null}
          </div>
        )}
      </div>
    </>
  )
}

function sitonePreviewLabel(
  kind: FrontCommandKind,
  selectedSitones: FrontSitone[],
) {
  const preview = calculateFrontSitonePreview(selectedSitones, kind)
  const result = preview.affectedCells
    ? `最多 ${preview.affectedCells} 格（+${preview.affectedCellBonus}）`
    : preview.defense
      ? `修復 ${preview.defense}、${preview.score} 分`
      : `${preview.score ?? 0} 分（+${preview.scoreBonus}）`
  const label = commandItems.find((item) => item.kind === kind)?.label ?? kind
  return `${label}：編隊 +${preview.squadBonusPercent}% · 專長 +${preview.affinityBonusPercent}% · ${result}`
}

function CommandIcon({
  icon: Icon,
}: {
  icon: ComponentType<{ className?: string }>
}) {
  return <Icon className="size-4" aria-hidden />
}

function visibleCommands(
  options: FrontCommandOption[],
  target: TerritoryTarget,
  landmark: FrontTerritoryLandmark | null,
  cell: TerritoryCellState,
  base: FrontTerritoryBase | null,
  myTeamId: string | undefined,
) {
  const targetOptions = options.filter(
    (option) =>
      (option.targetX === undefined && option.targetY === undefined) ||
      (option.targetX === target.x && option.targetY === target.y),
  )
  const allowedKinds = new Set<FrontCommandKind>()
  const coreKind = contextCommandKind(cell.ownerTeamId, myTeamId, base)

  if (options.length === 0) {
    if (coreKind) allowedKinds.add(coreKind)
    for (const kind of landmark
      ? (landmarkCommandKinds[landmark.kind] ?? [])
      : []) {
      allowedKinds.add(kind)
    }
  } else {
    for (const option of targetOptions) {
      const targetedLandmarkOption = Boolean(
        landmark &&
        !coreCommandKinds.has(option.kind) &&
        option.targetX === target.x &&
        option.targetY === target.y,
      )
      if (option.kind === coreKind || targetedLandmarkOption) {
        allowedKinds.add(option.kind)
      }
    }
  }

  return commandItems
    .filter((item) => allowedKinds.has(item.kind))
    .map((item) => ({
      item,
      option: targetOptions.find((option) => option.kind === item.kind),
    }))
}

function contextCommandKind(
  ownerTeamId: string | undefined,
  myTeamId: string | undefined,
  base: FrontTerritoryBase | null,
): FrontCommandKind | null {
  if (!ownerTeamId) return "expand"
  if (ownerTeamId === myTeamId) return "reinforce"
  if (!base) return "attack"

  return null
}

function TerritoryMeter({
  label,
  value,
  icon: Icon,
}: {
  label: string
  value: number
  icon: ComponentType<{ className?: string }>
}) {
  const percent = clampPercent(value)

  return (
    <div className="grid gap-1.5">
      <div className="flex items-center justify-between text-sm font-bold">
        <span className="flex items-center gap-1.5">
          <Icon className="size-4" aria-hidden />
          {label}
        </span>
        <span>{percent}</span>
      </div>
      <Progress value={percent} />
    </div>
  )
}

function TerritoryStat({ label, value }: { label: string; value: string }) {
  return (
    <div className="min-w-0">
      <div className="text-muted-foreground text-xs font-bold">{label}</div>
      <div className="mt-1 truncate text-sm font-black">{value}</div>
    </div>
  )
}

function territoryTitle(
  ownerTeamId: string | undefined,
  base: FrontTerritoryBase | null,
  landmark: FrontTerritoryLandmark | null,
  teams: FrontTeamState[],
) {
  if (base) return `${getTeamName(base.teamId, teams)}基地`
  if (landmark) return landmark.label
  if (ownerTeamId) return `${getTeamName(ownerTeamId, teams)}領土`

  return "中立區域"
}

function teamColorStyle(color: string | undefined): CSSProperties | undefined {
  const value = color?.trim()
  return value ? { backgroundColor: value } : undefined
}

function territoryRelation(ownerTeamId: string | undefined, myTeamId?: string) {
  if (!ownerTeamId) return "中立"
  if (ownerTeamId === myTeamId) return "我方"
  return "其他小隊"
}

function isPlayable(status: FrontSessionSummary["status"]) {
  return (
    status === "open_play" || status === "surge" || status === "booth_window"
  )
}

function clampPercent(value: number) {
  return Math.max(0, Math.min(100, Math.round(value)))
}

function normalizeCommandReason(reason: string | undefined) {
  return reason
}

function formatNumber(value: number | undefined) {
  return value == null ? "-" : value.toLocaleString("zh-TW")
}
