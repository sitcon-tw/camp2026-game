import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { RefreshCw, XCircle } from "lucide-react"
import { type ReactNode, useState } from "react"
import { toast } from "sonner"

import {
  adminTerritoryApi,
  type AdminAttackMission,
  type AdminCaptiveRecord,
  type AdminEventLog,
  type AdminSitoneInstance,
  type AdminYansanProcess,
} from "@/shared/api/admin-territory"
import { AppError } from "@/shared/api/error"
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "@/shared/ui/alert-dialog"
import { Badge } from "@/shared/ui/badge"
import { Button } from "@/shared/ui/button"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/shared/ui/card"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/shared/ui/select"
import { Spinner } from "@/shared/ui/spinner"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/shared/ui/table"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/shared/ui/tabs"
import { cn } from "@/shared/utils"

const territoryDateTimeFormatter = new Intl.DateTimeFormat("zh-TW", {
  month: "2-digit",
  day: "2-digit",
  hour: "2-digit",
  minute: "2-digit",
  second: "2-digit",
})

const allFilterValue = "all"

const missionStatusOptions = [
  "voting",
  "deployed",
  "resolved",
  "cancelled",
  "expired",
]

const instanceStatusOptions = [
  "normal",
  "deployed",
  "fatigued",
  "captured",
  "captive_cooldown",
  "convert_pending",
  "missing",
  "rescued",
  "dead",
  "protected_at_yansan",
]

const eventTypeOptions = [
  "attack_initiated",
  "attack_vote_cast",
  "attack_vote_expired",
  "attack_cancelled",
  "open_power_spent",
  "sitones_deployed",
  "mission_resolved",
  "steal_success",
  "sitone_captured",
  "sitone_missing",
  "rescue_started",
  "rescue_resolved",
  "sitone_died",
  "sitone_sent_yansan",
  "redeem_started",
  "redeem_succeeded",
  "redeem_failed",
  "convert_started",
  "convert_succeeded",
  "convert_failed",
  "boss_opened",
  "boss_closed",
  "boss_attacked",
  "boss_interrupted_yansan",
  "staff_adjustment",
  "defense_updated",
  "captive_recaptured",
]

const yansanProcessKindOptions = ["convert", "redeem", "calibrate", "rescue"]
const yansanProcessStatusOptions = [
  "pending",
  "succeeded",
  "failed",
  "aborted_boss",
]

function territoryErrorMessage(error: unknown, fallback: string) {
  if (error instanceof AppError) return error.message
  return fallback
}

function formatTerritoryDateTime(value?: string) {
  if (!value) return "-"
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return "-"
  return territoryDateTimeFormatter.format(date)
}

function statusBadgeVariant(status: string) {
  switch (status) {
    case "voting":
    case "deployed":
    case "pending":
    case "captive_cooldown":
    case "convert_pending":
      return "secondary" as const
    case "cancelled":
    case "expired":
    case "missing":
    case "dead":
    case "failed":
    case "aborted_boss":
      return "destructive" as const
    default:
      return "outline" as const
  }
}

function StatusFilterSelect({
  value,
  options,
  placeholder,
  onChange,
}: {
  value: string
  options: string[]
  placeholder: string
  onChange: (value: string) => void
}) {
  return (
    <Select value={value} onValueChange={onChange}>
      <SelectTrigger className="w-full min-w-0 sm:w-[220px]">
        <SelectValue placeholder={placeholder} />
      </SelectTrigger>
      <SelectContent>
        <SelectItem value={allFilterValue}>全部</SelectItem>
        {options.map((option) => (
          <SelectItem key={option} value={option}>
            {option}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  )
}

function PayloadDetails({
  payload,
  label = "payload",
}: {
  payload?: Record<string, unknown>
  label?: string
}) {
  if (!payload || Object.keys(payload).length === 0) {
    return <span className="text-muted-foreground text-xs font-bold">-</span>
  }
  return (
    <details className="min-w-0">
      <summary className="text-xs font-black cursor-pointer select-none">
        {label}
      </summary>
      <pre className="bg-surface-raised border-border mt-1 max-h-64 max-w-[420px] overflow-auto rounded-[10px] border p-2 text-[11px] leading-snug whitespace-pre-wrap break-all">
        {JSON.stringify(payload, null, 2)}
      </pre>
    </details>
  )
}

function TerritoryQueryStateCard({
  isPending,
  error,
  pendingLabel,
  onRetry,
}: {
  isPending: boolean
  error: unknown
  pendingLabel: string
  onRetry: () => void
}) {
  if (isPending) {
    return (
      <div className="flex items-center gap-3 p-4">
        <Spinner className="size-5" />
        <span className="font-black">{pendingLabel}</span>
      </div>
    )
  }
  return (
    <div className="grid gap-3 p-4">
      <span className="text-muted-foreground font-semibold">
        {territoryErrorMessage(error, "資料暫時無法讀取。")}
      </span>
      <Button
        type="button"
        variant="outline"
        className="w-fit"
        onClick={onRetry}
      >
        重新整理
      </Button>
    </div>
  )
}

function EmptyRow({ colSpan, label }: { colSpan: number; label: string }) {
  return (
    <TableRow>
      <TableCell
        colSpan={colSpan}
        className="text-muted-foreground py-6 text-center font-semibold"
      >
        {label}
      </TableCell>
    </TableRow>
  )
}

export function AdminTerritoryPanel() {
  return (
    <Card className="rounded-[18px] py-5">
      <CardHeader className="px-5">
        <CardTitle className="text-lg font-black">領地攻防監控</CardTitle>
        <CardDescription>
          攻擊任務、石頭實例、俘虜、事件與硯山流程。補償請使用既有的 staff
          發獎流程。
        </CardDescription>
      </CardHeader>
      <CardContent className="min-w-0 px-5">
        <Tabs defaultValue="missions" className="min-w-0 gap-3">
          <TabsList className="grid w-full max-w-full grid-cols-2 overflow-x-auto sm:grid-cols-[repeat(5,minmax(5rem,1fr))] md:w-fit">
            <TabsTrigger value="missions">任務</TabsTrigger>
            <TabsTrigger value="instances">石頭實例</TabsTrigger>
            <TabsTrigger value="captives">俘虜</TabsTrigger>
            <TabsTrigger value="events">事件</TabsTrigger>
            <TabsTrigger value="yansan">硯山流程</TabsTrigger>
          </TabsList>
          <TabsContent value="missions" className="min-w-0">
            <TerritoryMissionsTab />
          </TabsContent>
          <TabsContent value="instances" className="min-w-0">
            <TerritoryInstancesTab />
          </TabsContent>
          <TabsContent value="captives" className="min-w-0">
            <TerritoryCaptivesTab />
          </TabsContent>
          <TabsContent value="events" className="min-w-0">
            <TerritoryEventsTab />
          </TabsContent>
          <TabsContent value="yansan" className="min-w-0">
            <TerritoryYansanTab />
          </TabsContent>
        </Tabs>
      </CardContent>
    </Card>
  )
}

function TabToolbar({
  count,
  refreshing,
  onRefresh,
  children,
}: {
  count: number
  refreshing: boolean
  onRefresh: () => void
  children?: ReactNode
}) {
  return (
    <div className="flex flex-wrap items-center gap-2 pb-3">
      {children}
      <div className="ml-auto flex items-center gap-2">
        <Badge variant="secondary">{count} 筆</Badge>
        <Button
          type="button"
          variant="outline"
          size="icon"
          aria-label="重新整理"
          disabled={refreshing}
          onClick={onRefresh}
        >
          <RefreshCw className={cn("size-4", refreshing && "animate-spin")} />
        </Button>
      </div>
    </div>
  )
}

function TerritoryMissionsTab() {
  const queryClient = useQueryClient()
  const [status, setStatus] = useState(allFilterValue)

  const missionsQuery = useQuery({
    queryKey: ["admin", "territory", "missions", status],
    queryFn: () =>
      adminTerritoryApi.missions({
        status: status === allFilterValue ? undefined : status,
        limit: 200,
      }),
    retry: false,
    refetchInterval: 30_000,
  })

  const cancelMutation = useMutation({
    mutationFn: adminTerritoryApi.cancelMission,
    onSuccess: (result) => {
      toast.success(
        `任務已取消，釋放 ${result.releasedInstances} 顆已部署石頭。`,
      )
      void queryClient.invalidateQueries({
        queryKey: ["admin", "territory"],
      })
    },
    onError: (error) => {
      toast.error(territoryErrorMessage(error, "任務取消失敗"))
    },
  })

  if (missionsQuery.isPending || missionsQuery.error) {
    return (
      <TerritoryQueryStateCard
        isPending={missionsQuery.isPending}
        error={missionsQuery.error}
        pendingLabel="正在載入攻擊任務"
        onRetry={() => missionsQuery.refetch()}
      />
    )
  }

  const missions = missionsQuery.data ?? []

  return (
    <div className="min-w-0">
      <TabToolbar
        count={missions.length}
        refreshing={missionsQuery.isFetching}
        onRefresh={() => void missionsQuery.refetch()}
      >
        <StatusFilterSelect
          value={status}
          options={missionStatusOptions}
          placeholder="狀態"
          onChange={setStatus}
        />
      </TabToolbar>
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>時間</TableHead>
            <TableHead>任務</TableHead>
            <TableHead>攻擊 → 防守</TableHead>
            <TableHead>狀態</TableHead>
            <TableHead>石頭</TableHead>
            <TableHead className="text-right">OP 成本</TableHead>
            <TableHead>結果</TableHead>
            <TableHead>操作</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {missions.length === 0 ? (
            <EmptyRow colSpan={8} label="目前沒有符合條件的任務。" />
          ) : (
            missions.map((mission) => (
              <MissionRow
                key={mission.missionId}
                mission={mission}
                cancelling={
                  cancelMutation.isPending &&
                  cancelMutation.variables === mission.missionId
                }
                onCancel={() => cancelMutation.mutate(mission.missionId)}
              />
            ))
          )}
        </TableBody>
      </Table>
    </div>
  )
}

function MissionRow({
  mission,
  cancelling,
  onCancel,
}: {
  mission: AdminAttackMission
  cancelling: boolean
  onCancel: () => void
}) {
  const cancellable =
    mission.status === "voting" || mission.status === "deployed"
  return (
    <TableRow>
      <TableCell className="font-semibold whitespace-nowrap">
        {formatTerritoryDateTime(mission.createdAt)}
      </TableCell>
      <TableCell className="min-w-[180px] whitespace-normal">
        <div className="grid min-w-0 gap-1">
          <span className="text-xs font-semibold break-all">
            {mission.missionId}
          </span>
          {mission.randomSeedRef ? (
            <span className="text-muted-foreground text-xs break-all">
              seed: {mission.randomSeedRef}
            </span>
          ) : null}
        </div>
      </TableCell>
      <TableCell className="min-w-[160px] whitespace-normal">
        <div className="grid min-w-0 gap-1 text-xs font-semibold">
          <span className="break-all">
            {mission.attackerTeamId}
            {mission.attackerTier ? ` (T${mission.attackerTier})` : ""}
          </span>
          <span className="text-muted-foreground break-all">
            → {mission.defenderTeamId}
            {mission.defenderTier ? ` (T${mission.defenderTier})` : ""}
          </span>
          <span className="text-muted-foreground break-all">
            發起 {mission.initiatorPlayerId}
          </span>
        </div>
      </TableCell>
      <TableCell>
        <Badge variant={statusBadgeVariant(mission.status)}>
          {mission.status}
        </Badge>
      </TableCell>
      <TableCell className="min-w-[140px] whitespace-normal">
        <span className="text-xs font-semibold break-all">
          {mission.selectedSitoneIds.join(", ") || "-"}
        </span>
      </TableCell>
      <TableCell className="text-right font-black">
        {mission.costOpenPower}
      </TableCell>
      <TableCell>
        <PayloadDetails payload={mission.resultSummary} label="summary" />
      </TableCell>
      <TableCell>
        {cancellable ? (
          <AlertDialog>
            <AlertDialogTrigger asChild>
              <Button
                type="button"
                variant="destructive"
                size="sm"
                disabled={cancelling}
              >
                <XCircle className="size-4" />
                取消
              </Button>
            </AlertDialogTrigger>
            <AlertDialogContent>
              <AlertDialogHeader>
                <AlertDialogTitle>強制取消任務？</AlertDialogTitle>
                <AlertDialogDescription>
                  任務 {mission.missionId}（{mission.status}
                  ）將被標記為 cancelled。已部署的石頭會釋放回持有者的庫存；
                  已扣的開源力不會自動退還，請用 staff 發獎流程補償。
                </AlertDialogDescription>
              </AlertDialogHeader>
              <AlertDialogFooter>
                <AlertDialogCancel>返回</AlertDialogCancel>
                <AlertDialogAction onClick={onCancel}>
                  強制取消
                </AlertDialogAction>
              </AlertDialogFooter>
            </AlertDialogContent>
          </AlertDialog>
        ) : (
          <span className="text-muted-foreground text-xs font-bold">-</span>
        )}
      </TableCell>
    </TableRow>
  )
}

function TerritoryInstancesTab() {
  const [status, setStatus] = useState(allFilterValue)

  const instancesQuery = useQuery({
    queryKey: ["admin", "territory", "instances", status],
    queryFn: () =>
      adminTerritoryApi.instances({
        status: status === allFilterValue ? undefined : status,
        limit: 200,
      }),
    retry: false,
    refetchInterval: 30_000,
  })

  if (instancesQuery.isPending || instancesQuery.error) {
    return (
      <TerritoryQueryStateCard
        isPending={instancesQuery.isPending}
        error={instancesQuery.error}
        pendingLabel="正在載入石頭實例"
        onRetry={() => instancesQuery.refetch()}
      />
    )
  }

  const instances = instancesQuery.data ?? []

  return (
    <div className="min-w-0">
      <TabToolbar
        count={instances.length}
        refreshing={instancesQuery.isFetching}
        onRefresh={() => void instancesQuery.refetch()}
      >
        <StatusFilterSelect
          value={status}
          options={instanceStatusOptions}
          placeholder="狀態"
          onChange={setStatus}
        />
      </TabToolbar>
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>更新時間</TableHead>
            <TableHead>小石</TableHead>
            <TableHead>狀態</TableHead>
            <TableHead>持有 / 來源</TableHead>
            <TableHead>任務</TableHead>
            <TableHead className="text-right">疲勞</TableHead>
            <TableHead>冷卻至</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {instances.length === 0 ? (
            <EmptyRow colSpan={7} label="目前沒有符合條件的石頭實例。" />
          ) : (
            instances.map((instance) => (
              <InstanceRow key={instance.instanceId} instance={instance} />
            ))
          )}
        </TableBody>
      </Table>
    </div>
  )
}

function InstanceRow({ instance }: { instance: AdminSitoneInstance }) {
  return (
    <TableRow>
      <TableCell className="font-semibold whitespace-nowrap">
        {formatTerritoryDateTime(instance.updatedAt)}
      </TableCell>
      <TableCell className="min-w-[160px] whitespace-normal">
        <div className="grid min-w-0 gap-1">
          <span className="font-semibold break-all">{instance.sitoneId}</span>
          <span className="text-muted-foreground text-xs break-all">
            {instance.instanceId}
          </span>
        </div>
      </TableCell>
      <TableCell>
        <Badge variant={statusBadgeVariant(instance.status)}>
          {instance.status}
        </Badge>
      </TableCell>
      <TableCell className="min-w-[160px] whitespace-normal">
        <div className="grid min-w-0 gap-1 text-xs font-semibold">
          <span className="break-all">
            {instance.playerId || "-"}
            {instance.teamId ? ` / ${instance.teamId}` : ""}
          </span>
          <span className="text-muted-foreground break-all">
            原持有 {instance.originPlayerId || "-"}
            {instance.originTeamId ? ` / ${instance.originTeamId}` : ""}
          </span>
        </div>
      </TableCell>
      <TableCell className="max-w-[160px]">
        <span className="text-xs font-semibold break-all">
          {instance.missionId || "-"}
        </span>
      </TableCell>
      <TableCell className="text-right font-black">
        {instance.fatigue}
      </TableCell>
      <TableCell className="font-semibold whitespace-nowrap">
        {formatTerritoryDateTime(instance.cooldownUntil)}
      </TableCell>
    </TableRow>
  )
}

function TerritoryCaptivesTab() {
  const captivesQuery = useQuery({
    queryKey: ["admin", "territory", "captives"],
    queryFn: () => adminTerritoryApi.captives({ limit: 200 }),
    retry: false,
    refetchInterval: 30_000,
  })

  if (captivesQuery.isPending || captivesQuery.error) {
    return (
      <TerritoryQueryStateCard
        isPending={captivesQuery.isPending}
        error={captivesQuery.error}
        pendingLabel="正在載入俘虜紀錄"
        onRetry={() => captivesQuery.refetch()}
      />
    )
  }

  const captives = captivesQuery.data ?? []

  return (
    <div className="min-w-0">
      <TabToolbar
        count={captives.length}
        refreshing={captivesQuery.isFetching}
        onRefresh={() => void captivesQuery.refetch()}
      />
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>捕獲時間</TableHead>
            <TableHead>小石</TableHead>
            <TableHead>原隊 → 現隊</TableHead>
            <TableHead>狀態</TableHead>
            <TableHead>冷卻至</TableHead>
            <TableHead>轉化時間</TableHead>
            <TableHead>轉化流程</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {captives.length === 0 ? (
            <EmptyRow colSpan={7} label="目前沒有俘虜紀錄。" />
          ) : (
            captives.map((captive) => (
              <CaptiveRow key={captive.captiveId} captive={captive} />
            ))
          )}
        </TableBody>
      </Table>
    </div>
  )
}

function CaptiveRow({ captive }: { captive: AdminCaptiveRecord }) {
  return (
    <TableRow>
      <TableCell className="font-semibold whitespace-nowrap">
        {formatTerritoryDateTime(captive.capturedAt)}
      </TableCell>
      <TableCell className="min-w-[160px] whitespace-normal">
        <div className="grid min-w-0 gap-1">
          <span className="font-semibold break-all">{captive.sitoneId}</span>
          <span className="text-muted-foreground text-xs break-all">
            {captive.captiveId}
          </span>
        </div>
      </TableCell>
      <TableCell className="min-w-[140px] whitespace-normal">
        <span className="text-xs font-semibold break-all">
          {captive.originalTeamId} → {captive.currentTeamId}
        </span>
      </TableCell>
      <TableCell>
        <Badge variant={statusBadgeVariant(captive.status)}>
          {captive.status}
        </Badge>
      </TableCell>
      <TableCell className="font-semibold whitespace-nowrap">
        {formatTerritoryDateTime(captive.cooldownUntil)}
      </TableCell>
      <TableCell className="font-semibold whitespace-nowrap">
        {formatTerritoryDateTime(captive.convertedAt)}
      </TableCell>
      <TableCell className="min-w-[160px] whitespace-normal">
        {captive.convertProcesses.length === 0 ? (
          <span className="text-muted-foreground text-xs font-bold">-</span>
        ) : (
          <div className="grid min-w-0 gap-1">
            {captive.convertProcesses.map((process) => (
              <span
                key={process.processId}
                className="text-xs font-semibold break-all"
              >
                {process.status} · {formatTerritoryDateTime(process.startedAt)}{" "}
                · {process.actorPlayerId}
              </span>
            ))}
          </div>
        )}
      </TableCell>
    </TableRow>
  )
}

function TerritoryEventsTab() {
  const [eventType, setEventType] = useState(allFilterValue)

  const eventsQuery = useQuery({
    queryKey: ["admin", "territory", "events", eventType],
    queryFn: () =>
      adminTerritoryApi.events({
        type: eventType === allFilterValue ? undefined : eventType,
        limit: 200,
      }),
    retry: false,
    refetchInterval: 30_000,
  })

  if (eventsQuery.isPending || eventsQuery.error) {
    return (
      <TerritoryQueryStateCard
        isPending={eventsQuery.isPending}
        error={eventsQuery.error}
        pendingLabel="正在載入事件紀錄"
        onRetry={() => eventsQuery.refetch()}
      />
    )
  }

  const events = eventsQuery.data ?? []

  return (
    <div className="min-w-0">
      <TabToolbar
        count={events.length}
        refreshing={eventsQuery.isFetching}
        onRefresh={() => void eventsQuery.refetch()}
      >
        <StatusFilterSelect
          value={eventType}
          options={eventTypeOptions}
          placeholder="事件類型"
          onChange={setEventType}
        />
      </TabToolbar>
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>時間</TableHead>
            <TableHead>類型</TableHead>
            <TableHead>隊伍 / 玩家</TableHead>
            <TableHead>關聯</TableHead>
            <TableHead>Payload</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {events.length === 0 ? (
            <EmptyRow colSpan={5} label="目前沒有符合條件的事件。" />
          ) : (
            events.map((event) => <EventRow key={event.eventId} event={event} />)
          )}
        </TableBody>
      </Table>
    </div>
  )
}

function EventRow({ event }: { event: AdminEventLog }) {
  return (
    <TableRow>
      <TableCell className="font-semibold whitespace-nowrap">
        {formatTerritoryDateTime(event.createdAt)}
      </TableCell>
      <TableCell>
        <Badge variant="outline">{event.eventType}</Badge>
      </TableCell>
      <TableCell className="min-w-[160px] whitespace-normal">
        <div className="grid min-w-0 gap-1 text-xs font-semibold">
          <span className="break-all">{event.teamId || "-"}</span>
          <span className="text-muted-foreground break-all">
            {event.actorPlayerId || "-"}
            {event.targetPlayerId ? ` → ${event.targetPlayerId}` : ""}
          </span>
        </div>
      </TableCell>
      <TableCell className="min-w-[140px] whitespace-normal">
        <div className="grid min-w-0 gap-1 text-xs font-semibold">
          {event.relatedMissionId ? (
            <span className="break-all">{event.relatedMissionId}</span>
          ) : null}
          {event.relatedSitoneId ? (
            <span className="text-muted-foreground break-all">
              {event.relatedSitoneId}
            </span>
          ) : null}
          {!event.relatedMissionId && !event.relatedSitoneId ? (
            <span className="text-muted-foreground">-</span>
          ) : null}
        </div>
      </TableCell>
      <TableCell>
        <PayloadDetails payload={event.payload} />
      </TableCell>
    </TableRow>
  )
}

function TerritoryYansanTab() {
  const [kind, setKind] = useState(allFilterValue)
  const [status, setStatus] = useState(allFilterValue)

  const processesQuery = useQuery({
    queryKey: ["admin", "territory", "yansan-processes", kind, status],
    queryFn: () =>
      adminTerritoryApi.yansanProcesses({
        kind: kind === allFilterValue ? undefined : kind,
        status: status === allFilterValue ? undefined : status,
        limit: 200,
      }),
    retry: false,
    refetchInterval: 30_000,
  })

  if (processesQuery.isPending || processesQuery.error) {
    return (
      <TerritoryQueryStateCard
        isPending={processesQuery.isPending}
        error={processesQuery.error}
        pendingLabel="正在載入硯山流程"
        onRetry={() => processesQuery.refetch()}
      />
    )
  }

  const processes = processesQuery.data ?? []

  return (
    <div className="min-w-0">
      <TabToolbar
        count={processes.length}
        refreshing={processesQuery.isFetching}
        onRefresh={() => void processesQuery.refetch()}
      >
        <StatusFilterSelect
          value={kind}
          options={yansanProcessKindOptions}
          placeholder="類型"
          onChange={setKind}
        />
        <StatusFilterSelect
          value={status}
          options={yansanProcessStatusOptions}
          placeholder="狀態"
          onChange={setStatus}
        />
      </TabToolbar>
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>開始時間</TableHead>
            <TableHead>類型</TableHead>
            <TableHead>小石</TableHead>
            <TableHead>玩家 / 隊伍</TableHead>
            <TableHead className="text-right">OP 成本</TableHead>
            <TableHead>狀態</TableHead>
            <TableHead>結果</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {processes.length === 0 ? (
            <EmptyRow colSpan={7} label="目前沒有符合條件的流程。" />
          ) : (
            processes.map((process) => (
              <YansanProcessRow key={process.processId} process={process} />
            ))
          )}
        </TableBody>
      </Table>
    </div>
  )
}

function YansanProcessRow({ process }: { process: AdminYansanProcess }) {
  return (
    <TableRow>
      <TableCell className="font-semibold whitespace-nowrap">
        {formatTerritoryDateTime(process.startedAt)}
      </TableCell>
      <TableCell>
        <Badge variant="outline">{process.kind}</Badge>
      </TableCell>
      <TableCell className="min-w-[160px] whitespace-normal">
        <div className="grid min-w-0 gap-1">
          <span className="font-semibold break-all">{process.sitoneId}</span>
          <span className="text-muted-foreground text-xs break-all">
            {process.processId}
          </span>
        </div>
      </TableCell>
      <TableCell className="min-w-[160px] whitespace-normal">
        <div className="grid min-w-0 gap-1 text-xs font-semibold">
          <span className="break-all">{process.actorPlayerId}</span>
          <span className="text-muted-foreground break-all">
            {process.teamId}
          </span>
        </div>
      </TableCell>
      <TableCell className="text-right font-black">
        {process.costOpenPower}
      </TableCell>
      <TableCell>
        <Badge variant={statusBadgeVariant(process.status)}>
          {process.status}
        </Badge>
      </TableCell>
      <TableCell>
        <PayloadDetails payload={process.result} label="result" />
      </TableCell>
    </TableRow>
  )
}
