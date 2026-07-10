import type { ComponentType } from "react"
import {
  LifeBuoy,
  MoveUpRight,
  Radar,
  ShieldPlus,
  Swords,
  Wrench,
  Zap,
} from "lucide-react"

import type {
  FrontCell,
  FrontCommandKind,
  FrontCommandOption,
  FrontMapEvent,
  FrontSessionSummary,
  FrontSitone,
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

import { getTeamName, getTeamTone } from "./front-map-style"

type FrontNodeDrawerProps = {
  front: FrontSessionSummary
  cell: FrontCell | null
  teams: FrontTeamState[]
  activeEvents: FrontMapEvent[]
  availableCommands: FrontCommandOption[]
  selectedSitone: FrontSitone | null
  selectedCommand: FrontCommandKind | null
  onOpenChange: (cellID: string | null) => void
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
  { kind: "scout", label: "偵查", icon: Radar },
  { kind: "rescue", label: "救援", icon: LifeBuoy },
]

export function FrontNodeDrawer({
  front,
  cell,
  teams,
  activeEvents,
  availableCommands,
  selectedSitone,
  selectedCommand,
  onOpenChange,
  onSelectCommand,
  commandPending,
}: FrontNodeDrawerProps) {
  return (
    <Sheet
      open={Boolean(cell)}
      onOpenChange={(open) => {
        if (!open) onOpenChange(null)
      }}
    >
      {cell ? (
        <SheetContent side="right" className="w-[88%] gap-3 overflow-y-auto">
          <NodeDrawerContent
            front={front}
            cell={cell}
            teams={teams}
            activeEvents={activeEvents}
            availableCommands={availableCommands}
            selectedSitone={selectedSitone}
            selectedCommand={selectedCommand}
            onSelectCommand={onSelectCommand}
            commandPending={commandPending}
          />
        </SheetContent>
      ) : null}
    </Sheet>
  )
}

function NodeDrawerContent({
  front,
  cell,
  teams,
  activeEvents,
  availableCommands,
  selectedSitone,
  selectedCommand,
  onSelectCommand,
  commandPending,
}: Omit<FrontNodeDrawerProps, "onOpenChange" | "cell"> & {
  cell: FrontCell
}) {
  const ownerTone = getTeamTone(cell.ownerTeamId, teams)
  const event = activeEvents.find((item) => item.cellId === cell.id)
  const pressureEntries = Object.entries(cell.pressureByTeam).filter(
    ([, value]) => value > 0,
  )
  const playable = isPlayable(front.status)
  const hasCommandOptions = availableCommands.length > 0

  return (
    <>
      <SheetHeader>
        <div className="flex flex-wrap items-center gap-2 pr-9">
          <span
            className={cn("size-3 rounded-full", ownerTone.dot)}
            aria-hidden
          />
          <Badge variant="outline">{cell.zone}</Badge>
          {event ? (
            <StatusBadge tone="warning">{event.title}</StatusBadge>
          ) : null}
        </div>
        <SheetTitle className="text-2xl">{cell.name ?? cell.id}</SheetTitle>
        <SheetDescription>
          {getTeamName(cell.ownerTeamId, teams)}
        </SheetDescription>
      </SheetHeader>

      <div className="grid gap-3 px-4">
        <NodeMeter label="控制" value={cell.control} />
        <NodeMeter label="防禦" value={cell.defense} />

        <div className="border-border grid grid-cols-2 gap-2 border-y py-3">
          <NodeStat label="資源" value={cell.resource} />
          <NodeStat label="鄰接" value={cell.neighborIds.length} />
        </div>

        <div className="grid gap-2">
          <div className="text-sm font-black">壓力</div>
          {pressureEntries.length > 0 ? (
            <div className="grid gap-2">
              {pressureEntries.map(([teamID, value]) => (
                <div
                  key={teamID}
                  className="bg-muted flex items-center justify-between rounded-[0.875rem] px-3 py-2 text-sm font-bold"
                >
                  <span>{getTeamName(teamID, teams)}</span>
                  <span>{value}</span>
                </div>
              ))}
            </div>
          ) : (
            <div className="text-muted-foreground text-sm font-bold">
              目前穩定
            </div>
          )}
        </div>

        <div className="grid gap-2">
          <div className="flex items-center justify-between gap-2">
            <div className="text-sm font-black">命令</div>
            {selectedSitone ? (
              <Badge variant="secondary" className="max-w-[9rem] truncate">
                {selectedSitone.name}
              </Badge>
            ) : null}
          </div>
          <div className="grid grid-cols-2 gap-2">
            {commandItems.map(({ kind, label, icon: Icon }) => {
              const option = findCommandOption(availableCommands, cell.id, kind)
              const enabled = Boolean(
                playable &&
                selectedSitone &&
                (hasCommandOptions ? option?.enabled : true),
              )
              const active = selectedCommand === kind
              const pending = commandPending && active

              return (
                <Button
                  key={kind}
                  type="button"
                  variant={active ? "default" : "outline"}
                  className="h-11 justify-start rounded-[0.875rem]"
                  disabled={!enabled || commandPending}
                  aria-pressed={active}
                  onClick={() => onSelectCommand(kind)}
                >
                  <Icon className="size-4" aria-hidden />
                  <span className="truncate">
                    {pending ? "送出中" : (option?.label ?? label)}
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
          {!playable ? (
            <div className="text-muted-foreground text-xs font-bold">
              戰線目前不可操作
            </div>
          ) : null}
        </div>
      </div>
    </>
  )
}

function NodeMeter({ label, value }: { label: string; value: number }) {
  const percent = clampPercent(value)

  return (
    <div className="grid gap-1.5">
      <div className="flex items-center justify-between text-sm font-bold">
        <span>{label}</span>
        <span>{percent}</span>
      </div>
      <Progress value={percent} />
    </div>
  )
}

function NodeStat({ label, value }: { label: string; value: number }) {
  return (
    <div>
      <div className="text-muted-foreground text-xs font-bold">{label}</div>
      <div className="mt-1 text-lg font-black">{value}</div>
    </div>
  )
}

function findCommandOption(
  options: FrontCommandOption[],
  cellID: string,
  kind: FrontCommandKind,
) {
  return options.find(
    (option) =>
      option.kind === kind && (!option.toCellId || option.toCellId === cellID),
  )
}

function isPlayable(status: FrontSessionSummary["status"]) {
  return (
    status === "open_play" || status === "surge" || status === "booth_window"
  )
}

function clampPercent(value: number) {
  return Math.max(0, Math.min(100, Math.round(value)))
}
