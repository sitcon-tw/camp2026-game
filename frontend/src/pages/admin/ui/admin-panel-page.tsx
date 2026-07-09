import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import {
  Activity,
  ArrowUpDown,
  CheckCircle2,
  ChevronDown,
  Clock,
  ImageIcon,
  LogOut,
  Megaphone,
  Pencil,
  Percent,
  Plus,
  RefreshCw,
  Save,
  Settings,
  ShieldCheck,
  SlidersHorizontal,
  Trash2,
  X,
} from "lucide-react"
import { type FormEvent, type ReactNode, useMemo, useState } from "react"
import {
  CartesianGrid,
  Cell,
  Line,
  LineChart as RechartsLineChart,
  Pie,
  PieChart,
  XAxis,
  YAxis,
} from "recharts"
import { toast } from "sonner"

import { AdminTerritoryPanel } from "@/pages/admin/ui/admin-territory-panel"
import { AppError } from "@/shared/api/error"
import {
  gameApi,
  type AdminCommunityStand,
  type AdminCommunityStandClaim,
  type AdminCommunityStandCreateInput,
  type AdminCommunityStandUpdateInput,
  type AdminDashboard,
  type AdminDashboardHistory,
  type AdminDashboardInventoryEntry,
  type AdminDashboardPlayer,
  type AdminDashboardPlayerRank,
  type AdminDashboardTeam,
  type AdminStudentChangeEntry,
  type AdminSettings,
  type GiftHistoryEntry,
  type StaffRewardKind,
} from "@/shared/api/game"
import { Avatar, AvatarFallback, AvatarImage } from "@/shared/ui/avatar"
import { Badge } from "@/shared/ui/badge"
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
import { Button } from "@/shared/ui/button"
import {
  Card,
  CardAction,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/shared/ui/card"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/shared/ui/dialog"
import {
  ChartContainer,
  ChartTooltip,
  ChartTooltipContent,
  type ChartConfig,
} from "@/shared/ui/chart"
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@/shared/ui/collapsible"
import { Field } from "@/shared/ui/field"
import { GameFeatureIcon } from "@/shared/ui/game-feature-icon"
import { GameIcon } from "@/shared/ui/game-icon"
import { GamePageShell } from "@/shared/ui/game-page-shell"
import { Input } from "@/shared/ui/input"
import { Label } from "@/shared/ui/label"
import { PageHeader } from "@/shared/ui/page-header"
import { PlayerAvatar } from "@/shared/ui/player-avatar"
import { Progress } from "@/shared/ui/progress"
import { Spinner } from "@/shared/ui/spinner"
import { Switch } from "@/shared/ui/switch"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/shared/ui/select"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/shared/ui/table"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/shared/ui/tabs"
import { Textarea } from "@/shared/ui/textarea"
import { cn } from "@/shared/utils"
import { imageSrcCandidates } from "@/shared/utils/image-src"

const numberFormatter = new Intl.NumberFormat("zh-TW")
const compactNumberFormatter = new Intl.NumberFormat("zh-TW", {
  notation: "compact",
  maximumFractionDigits: 1,
})
const maxClassTimeBattleLockSessions = 12
const dateTimeFormatter = new Intl.DateTimeFormat("zh-TW", {
  month: "2-digit",
  day: "2-digit",
  hour: "2-digit",
  minute: "2-digit",
})
const historyHourFormatter = new Intl.DateTimeFormat("zh-TW", {
  month: "2-digit",
  day: "2-digit",
  hour: "2-digit",
})
const historyDayFormatter = new Intl.DateTimeFormat("zh-TW", {
  month: "2-digit",
  day: "2-digit",
})
const historyChartConfig = {
  sitoneCount: {
    label: "小石",
    color: "var(--pebble-engineer)",
  },
  openPower: {
    label: "開源力",
    color: "var(--primary)",
  },
} satisfies ChartConfig

type InventorySortDirection = "asc" | "desc"
const sitoneOwnershipColors = [
  "var(--pebble-engineer)",
  "var(--pebble-inspiration)",
  "var(--pebble-harmony)",
  "var(--pebble-entertainment)",
  "var(--pebble-explorer)",
  "var(--primary)",
  "var(--accent)",
  "var(--chart-3)",
  "var(--chart-4)",
  "var(--chart-5)",
]

function errorMessage(error: unknown, fallback: string) {
  if (error instanceof AppError) return error.message
  return fallback
}

function clampPercent(value: number) {
  if (!Number.isFinite(value)) return 0
  return Math.max(0, Math.min(100, Math.floor(value)))
}

function formatNumber(value: number) {
  return numberFormatter.format(value)
}

function formatCompact(value: number) {
  return compactNumberFormatter.format(value)
}

function formatPercent(value: number) {
  return `${value}%`
}

function formatDateTime(value?: string) {
  if (!value) return "-"
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return "-"
  return dateTimeFormatter.format(date)
}

function giftRewardAmountLabel(entry: GiftHistoryEntry) {
  if (entry.kind === "open_power") {
    return `+${entry.amount ?? 0} OP`
  }
  return `x${entry.quantity ?? 0}`
}

function giftRewardKindLabel(entry: GiftHistoryEntry) {
  return resourceKindLabel(entry.kind)
}

function resourceKindLabel(kind: StaffRewardKind) {
  switch (kind) {
    case "item":
      return "道具"
    case "sitone":
      return "小石"
    case "open_power":
      return "開源力"
  }
}

function GiftRewardIcon({ entry }: { entry: GiftHistoryEntry }) {
  const fallback = (() => {
    switch (entry.kind) {
      case "item":
        return <GameFeatureIcon name="backpack" className="size-5" />
      case "sitone":
        return <GameFeatureIcon name="stones" className="size-5" />
      case "open_power":
        return <GameFeatureIcon name="giftHistory" className="size-5" />
    }
  })()

  return (
    <GameIcon
      iconPath={entry.iconPath}
      imageClassName="p-0.5"
      fallback={fallback}
    />
  )
}

function StudentChangeIcon({ entry }: { entry: AdminStudentChangeEntry }) {
  const fallback = (() => {
    switch (entry.kind) {
      case "item":
        return <GameFeatureIcon name="backpack" className="size-5" />
      case "sitone":
        return <GameFeatureIcon name="stones" className="size-5" />
      case "open_power":
        return <GameFeatureIcon name="shop" className="size-5" />
    }
  })()

  return (
    <GameIcon
      iconPath={entry.iconPath}
      imageClassName="p-0.5"
      fallback={fallback}
    />
  )
}

function studentChangeDeltaLabel(entry: AdminStudentChangeEntry) {
  const sign = entry.delta > 0 ? "+" : ""
  const unit =
    entry.kind === "open_power"
      ? " OP"
      : entry.kind === "sitone"
        ? " 顆"
        : " 個"
  return `${sign}${formatNumber(entry.delta)}${unit}`
}

function formatHistoryTimestamp(value: string, bucket: "hour" | "day") {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return "-"
  return bucket === "day"
    ? historyDayFormatter.format(date)
    : historyHourFormatter.format(date)
}

function formatSeconds(milliseconds: number) {
  if (milliseconds <= 0) return "-"
  return `${(milliseconds / 1000).toFixed(1)}s`
}

type TeamAvatarModel = {
  teamId: string
  name: string
  avatarUrl?: string
}

function teamLabel(player: Pick<AdminDashboardPlayer, "team">) {
  return player.team?.name ?? "未分組"
}

function teamAvatarFallback(team: TeamAvatarModel) {
  const label = team.name.trim() || team.teamId.trim() || "Team"
  return label.slice(0, 2).toUpperCase()
}

function catalogLabel(entry: AdminDashboardInventoryEntry) {
  const parts = [entry.type, entry.rarity].filter(Boolean)
  return parts.length > 0 ? parts.join(" / ") : "未分類"
}

export function AdminPanelPage() {
  const queryClient = useQueryClient()
  const [password, setPassword] = useState("")
  const [draft, setDraft] = useState<AdminSettings | null>(null)
  const [creatingCommunityStand, setCreatingCommunityStand] = useState(false)

  const settingsQuery = useQuery({
    queryKey: ["admin", "settings"],
    queryFn: gameApi.adminSettings,
    retry: false,
  })
  const unauthorized =
    settingsQuery.error instanceof AppError &&
    settingsQuery.error.status === 401
  const settings = draft ?? settingsQuery.data ?? null

  const dashboardQuery = useQuery({
    queryKey: ["admin", "dashboard"],
    queryFn: gameApi.adminDashboard,
    enabled: Boolean(settingsQuery.data),
    retry: false,
    refetchInterval: 30_000,
  })
  const historyQuery = useQuery({
    queryKey: ["admin", "history", "hour"],
    queryFn: () => gameApi.adminHistory("hour"),
    enabled: Boolean(settingsQuery.data),
    retry: false,
    refetchInterval: 30_000,
  })
  const giftHistoryQuery = useQuery({
    queryKey: ["admin", "gift-history"],
    queryFn: gameApi.adminGiftHistory,
    enabled: Boolean(settingsQuery.data),
    retry: false,
    refetchInterval: 30_000,
  })
  const studentChangesQuery = useQuery({
    queryKey: ["admin", "student-changes"],
    queryFn: () => gameApi.adminStudentChanges(500),
    enabled: Boolean(settingsQuery.data),
    retry: false,
    refetchInterval: 30_000,
  })
  const communityStandsQuery = useQuery({
    queryKey: ["admin", "community-stands"],
    queryFn: gameApi.adminCommunityStands,
    enabled: Boolean(settingsQuery.data),
    retry: false,
    refetchInterval: 30_000,
  })
  const communityStandClaimsQuery = useQuery({
    queryKey: ["admin", "community-stand-claims"],
    queryFn: () => gameApi.adminCommunityStandClaims(),
    enabled: Boolean(settingsQuery.data),
    retry: false,
    refetchInterval: 30_000,
  })

  const loginMutation = useMutation({
    mutationFn: gameApi.adminLogin,
    onSuccess: () => {
      setPassword("")
      void queryClient.invalidateQueries({ queryKey: ["admin"] })
    },
    onError: (error) => {
      toast.error(errorMessage(error, "登入失敗"))
    },
  })

  const logoutMutation = useMutation({
    mutationFn: gameApi.adminLogout,
    onSuccess: () => {
      setDraft(null)
      void queryClient.invalidateQueries({ queryKey: ["admin"] })
    },
  })

  const updateMutation = useMutation({
    mutationFn: gameApi.updateAdminSettings,
    onSuccess: (settings) => {
      setDraft(settings)
      queryClient.setQueryData(["admin", "settings"], settings)
      void queryClient.invalidateQueries({
        queryKey: ["matches", "computer", "settings"],
      })
      toast.success("設定已更新")
    },
    onError: (error) => {
      toast.error(errorMessage(error, "更新失敗"))
    },
  })

  const createCommunityStandMutation = useMutation({
    mutationFn: gameApi.createAdminCommunityStand,
    onSuccess: (stand) => {
      setCreatingCommunityStand(false)
      queryClient.setQueryData<AdminCommunityStand[]>(
        ["admin", "community-stands"],
        (current) => sortCommunityStands([...(current ?? []), stand]),
      )
      void queryClient.invalidateQueries({
        queryKey: ["community", "stand", stand.standId],
      })
      void queryClient.invalidateQueries({
        queryKey: ["community", "stand", "display", stand.standId],
      })
      toast.success("攤位已新增")
    },
    onError: (error) => {
      toast.error(errorMessage(error, "攤位新增失敗"))
    },
  })

  const deleteCommunityStandMutation = useMutation({
    mutationFn: gameApi.deleteAdminCommunityStand,
    onSuccess: (_result, standID) => {
      queryClient.setQueryData<AdminCommunityStand[]>(
        ["admin", "community-stands"],
        (current) =>
          (current ?? []).filter((stand) => stand.standId !== standID),
      )
      queryClient.removeQueries({ queryKey: ["community", "stand", standID] })
      queryClient.removeQueries({
        queryKey: ["community", "stand", "display", standID],
      })
      void queryClient.invalidateQueries({
        queryKey: ["admin", "community-stand-claims"],
      })
      toast.success("攤位已刪除")
    },
    onError: (error) => {
      toast.error(errorMessage(error, "攤位刪除失敗"))
    },
  })

  const updateCommunityStandMutation = useMutation({
    mutationFn: ({
      standID,
      input,
    }: {
      standID: string
      input: AdminCommunityStandUpdateInput
    }) => gameApi.updateAdminCommunityStand(standID, input),
    onSuccess: (stand) => {
      queryClient.setQueryData<AdminCommunityStand[]>(
        ["admin", "community-stands"],
        (current) => {
          const stands = current ?? []
          if (stands.some((entry) => entry.standId === stand.standId)) {
            return stands.map((entry) =>
              entry.standId === stand.standId ? stand : entry,
            )
          }
          return [...stands, stand]
        },
      )
      void queryClient.invalidateQueries({
        queryKey: ["community", "stand", stand.standId],
      })
      void queryClient.invalidateQueries({
        queryKey: ["community", "stand", "display", stand.standId],
      })
      void queryClient.invalidateQueries({
        queryKey: ["admin", "community-stand-claims"],
      })
      toast.success("攤位設定已更新")
    },
    onError: (error) => {
      toast.error(errorMessage(error, "攤位設定更新失敗"))
    },
  })

  function handleLogin(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    loginMutation.mutate(password)
  }

  function handleUpdate(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (!settings) return
    updateMutation.mutate(settings)
  }

  function updateDraft(patch: Partial<AdminSettings>) {
    setDraft((current) => {
      const base = current ?? settingsQuery.data
      return base ? { ...base, ...patch } : current
    })
  }

  const operationsRefreshing =
    dashboardQuery.isFetching ||
    historyQuery.isFetching ||
    giftHistoryQuery.isFetching ||
    studentChangesQuery.isFetching ||
    communityStandsQuery.isFetching

  if (settingsQuery.isPending) {
    return (
      <GamePageShell contentClassName="justify-center">
        <Card className="border-ink w-full max-w-sm rounded-[22px] border-2">
          <CardContent className="flex items-center gap-3 p-5">
            <Spinner className="size-5" />
            <span className="font-black">正在確認 admin 狀態</span>
          </CardContent>
        </Card>
      </GamePageShell>
    )
  }

  if (unauthorized) {
    return (
      <GamePageShell contentClassName="grid content-start gap-y-3">
        <PageHeader title="Admin" headline="Control Panel" />
        <Card className="border-ink rounded-[22px] border-2">
          <form onSubmit={handleLogin}>
            <CardHeader>
              <CardTitle className="flex items-center gap-2 text-xl font-black">
                <ShieldCheck className="size-5" />
                Admin 登入
              </CardTitle>
              <CardDescription>使用伺服器環境變數中的密碼。</CardDescription>
            </CardHeader>
            <CardContent>
              <Field>
                <Label htmlFor="admin-password">Password</Label>
                <Input
                  id="admin-password"
                  type="password"
                  value={password}
                  onChange={(event) => setPassword(event.target.value)}
                  autoComplete="current-password"
                />
              </Field>
            </CardContent>
            <CardFooter>
              <Button
                type="submit"
                className="w-full"
                disabled={!password.trim() || loginMutation.isPending}
              >
                {loginMutation.isPending ? "登入中" : "登入"}
              </Button>
            </CardFooter>
          </form>
        </Card>
      </GamePageShell>
    )
  }

  if (settingsQuery.error || !settings) {
    return (
      <GamePageShell contentClassName="grid content-start gap-y-3">
        <PageHeader title="Admin" headline="Control Panel" />
        <Card className="border-ink rounded-[22px] border-2">
          <CardHeader>
            <CardTitle>Admin 無法使用</CardTitle>
            <CardDescription>
              {errorMessage(
                settingsQuery.error,
                "請確認 ADMIN_PASSWORD 與後端服務。",
              )}
            </CardDescription>
          </CardHeader>
          <CardFooter>
            <Button onClick={() => settingsQuery.refetch()}>重新檢查</Button>
          </CardFooter>
        </Card>
      </GamePageShell>
    )
  }

  return (
    <GamePageShell
      ariaLabel="Admin dashboard"
      contentClassName="grid min-w-0 max-w-[1440px] content-start gap-y-4 overflow-x-hidden px-3 pb-8 sm:px-5 lg:px-8 [&_[data-slot=card-content]]:min-w-0 [&_[data-slot=card-header]]:min-w-0 [&_[data-slot=card]]:min-w-0 [&_[data-slot=table-container]]:max-w-full"
    >
      <PageHeader
        title="Admin"
        headline="Game Operations"
        rightSlot={
          <div className="flex max-w-full flex-wrap items-center justify-end gap-2 sm:flex-nowrap">
            <Button
              type="button"
              variant="outline"
              size="icon"
              aria-label="重新整理 dashboard"
              disabled={operationsRefreshing}
              onClick={() => {
                void dashboardQuery.refetch()
                void historyQuery.refetch()
                void giftHistoryQuery.refetch()
                void studentChangesQuery.refetch()
                void communityStandsQuery.refetch()
              }}
            >
              <RefreshCw
                className={cn("size-4", operationsRefreshing && "animate-spin")}
              />
            </Button>
            <Button
              type="button"
              variant="outline"
              size="icon"
              aria-label="登出"
              disabled={logoutMutation.isPending}
              onClick={() => logoutMutation.mutate()}
              className="sm:h-9 sm:w-auto sm:px-4"
            >
              <LogOut />
              <span className="hidden sm:inline">登出</span>
            </Button>
          </div>
        }
      />

      <AdminCollapsibleSection
        title="營運總覽"
        description="玩家、資源、紀錄與排行榜。"
        badge={dashboardQuery.data ? "Live" : undefined}
        defaultOpen
      >
        {dashboardQuery.isPending ? (
          <DashboardLoadingCard />
        ) : dashboardQuery.error ? (
          <Card className="rounded-[18px] py-5">
            <CardHeader>
              <CardTitle>Dashboard 無法讀取</CardTitle>
              <CardDescription>
                {errorMessage(
                  dashboardQuery.error,
                  "請確認後端服務與資料庫狀態。",
                )}
              </CardDescription>
            </CardHeader>
            <CardFooter>
              <Button onClick={() => dashboardQuery.refetch()}>重新整理</Button>
            </CardFooter>
          </Card>
        ) : dashboardQuery.data ? (
          <AdminDashboardView
            dashboard={dashboardQuery.data}
            history={historyQuery.data}
            giftHistory={giftHistoryQuery.data}
            studentChanges={studentChangesQuery.data}
            giftHistoryError={giftHistoryQuery.error}
            giftHistoryPending={giftHistoryQuery.isPending}
            studentChangesError={studentChangesQuery.error}
            studentChangesPending={studentChangesQuery.isPending}
            historyError={historyQuery.error}
            historyPending={historyQuery.isPending}
            onRetryHistory={() => historyQuery.refetch()}
            onRetryGiftHistory={() => giftHistoryQuery.refetch()}
            onRetryStudentChanges={() => studentChangesQuery.refetch()}
          />
        ) : null}
      </AdminCollapsibleSection>

      <AdminCollapsibleSection
        title="上課時間開局限制"
        description="控制課堂期間是否可以建立與開始對戰。"
        badge={settings.battleOpeningLocked ? "LOCKED" : "OPEN"}
        defaultOpen={settings.battleOpeningLocked}
      >
        <AdminClassTimeBattleLockPanel
          settings={settings}
          isPending={updateMutation.isPending}
          onSubmit={handleUpdate}
          onUpdate={updateDraft}
        />
      </AdminCollapsibleSection>

      <AdminCollapsibleSection
        title="管理設定"
        description="電腦對戰、同隊對戰與電腦答對率。"
        badge="Admin only"
      >
        <AdminSettingsPanel
          settings={settings}
          isPending={updateMutation.isPending}
          onSubmit={handleUpdate}
          onUpdate={updateDraft}
        />
      </AdminCollapsibleSection>

      <AdminCollapsibleSection
        title="攤位設定"
        description="社群攤位顯示、啟用狀態與掃描獎勵。"
        badge={`${formatNumber(communityStandsQuery.data?.length ?? 0)} booths`}
      >
        <AdminCommunityStandsPanel
          stands={communityStandsQuery.data ?? []}
          claims={communityStandClaimsQuery.data ?? []}
          claimsError={communityStandClaimsQuery.error}
          claimsPending={communityStandClaimsQuery.isPending}
          error={communityStandsQuery.error}
          isPending={communityStandsQuery.isPending}
          isCreateFormOpen={creatingCommunityStand}
          isCreating={createCommunityStandMutation.isPending}
          deletingStandID={
            deleteCommunityStandMutation.isPending
              ? deleteCommunityStandMutation.variables
              : undefined
          }
          pendingStandID={
            updateCommunityStandMutation.isPending
              ? updateCommunityStandMutation.variables?.standID
              : undefined
          }
          onRetryClaims={() => communityStandClaimsQuery.refetch()}
          onRetry={() => communityStandsQuery.refetch()}
          onToggleCreate={() =>
            setCreatingCommunityStand((current) => !current)
          }
          onCreate={(input) => createCommunityStandMutation.mutate(input)}
          onDelete={(standID) => deleteCommunityStandMutation.mutate(standID)}
          onSubmit={(standID, input) =>
            updateCommunityStandMutation.mutate({ standID, input })
          }
        />
      </AdminCollapsibleSection>

      <AdminCollapsibleSection
        title="領地攻防"
        description="攻擊任務、石頭實例、俘虜、事件與硯山流程監控。"
        badge="Staff tools"
      >
        <AdminTerritoryPanel />
      </AdminCollapsibleSection>
    </GamePageShell>
  )
}

function DashboardLoadingCard() {
  return (
    <Card className="rounded-[18px] py-5">
      <CardContent className="flex items-center gap-3 p-5">
        <Spinner className="size-5" />
        <span className="font-black">正在載入 dashboard 統計</span>
      </CardContent>
    </Card>
  )
}

function AdminCollapsibleSection({
  title,
  description,
  badge,
  defaultOpen = false,
  open,
  onOpenChange,
  children,
}: {
  title: string
  description?: string
  badge?: string
  defaultOpen?: boolean
  open?: boolean
  onOpenChange?: (open: boolean) => void
  children: ReactNode
}) {
  const [uncontrolledOpen, setUncontrolledOpen] = useState(defaultOpen)
  const isOpen = open ?? uncontrolledOpen
  const handleOpenChange = onOpenChange ?? setUncontrolledOpen

  return (
    <Collapsible
      open={isOpen}
      onOpenChange={handleOpenChange}
      className="grid min-w-0 gap-3"
    >
      <CollapsibleTrigger asChild>
        <button
          type="button"
          className="border-ink bg-card text-foreground focus-visible:outline-power grid w-full min-w-0 grid-cols-[minmax(0,1fr)_auto] items-center gap-3 rounded-[18px] border-2 p-4 text-left shadow-[3px_3px_0_rgba(23,35,58,0.12)] transition-transform focus-visible:outline-3 focus-visible:outline-offset-2 active:translate-y-px"
        >
          <span className="grid min-w-0 gap-1">
            <span className="truncate text-lg leading-tight font-black">
              {title}
            </span>
            {description ? (
              <span className="text-muted-foreground text-sm font-semibold break-words">
                {description}
              </span>
            ) : null}
          </span>
          <span className="flex min-w-0 items-center justify-end gap-2">
            {badge ? (
              <Badge variant="outline" className="max-w-[9rem] truncate">
                {badge}
              </Badge>
            ) : null}
            <ChevronDown
              className={cn(
                "size-5 shrink-0 transition-transform duration-200",
                isOpen && "rotate-180",
              )}
            />
          </span>
        </button>
      </CollapsibleTrigger>
      <CollapsibleContent className="data-[state=closed]:animate-accordion-up data-[state=open]:animate-accordion-down grid min-w-0 gap-3 overflow-hidden">
        <div className="grid min-w-0 gap-3">{children}</div>
      </CollapsibleContent>
    </Collapsible>
  )
}

function AdminDashboardView({
  dashboard,
  history,
  giftHistory,
  studentChanges,
  giftHistoryError,
  giftHistoryPending,
  studentChangesError,
  studentChangesPending,
  historyError,
  historyPending,
  onRetryHistory,
  onRetryGiftHistory,
  onRetryStudentChanges,
}: {
  dashboard: AdminDashboard
  history?: AdminDashboardHistory
  giftHistory?: GiftHistoryEntry[]
  studentChanges?: AdminStudentChangeEntry[]
  giftHistoryError: unknown
  giftHistoryPending: boolean
  studentChangesError: unknown
  studentChangesPending: boolean
  historyError: unknown
  historyPending: boolean
  onRetryHistory: () => void
  onRetryGiftHistory: () => void
  onRetryStudentChanges: () => void
}) {
  const { summary, matches } = dashboard
  const [statsTab, setStatsTab] = useState("players")
  const [statsOpen, setStatsOpen] = useState(false)

  function openPlayersStats() {
    setStatsTab("players")
    setStatsOpen(true)
  }

  return (
    <div className="grid min-w-0 gap-4">
      <section className="grid min-w-0 gap-3 xl:grid-cols-[minmax(0,1fr)_360px]">
        <div className="grid min-w-0 grid-cols-1 gap-3 sm:grid-cols-2 md:grid-cols-4 xl:grid-cols-6">
          <MetricTile
            icon={<GameFeatureIcon name="team" className="size-4" />}
            label="玩家"
            value={formatNumber(summary.playerCount)}
            detail={`${summary.teamCount} 隊 / ${summary.ungroupedPlayerCount} 未分組`}
          />
          <MetricTile
            icon={<GameFeatureIcon name="stones" className="size-4" />}
            label="小石總量"
            value={formatNumber(summary.totalSitones)}
            detail={`平均 ${average(summary.totalSitones, summary.playerCount)} 顆`}
          />
          <MetricTile
            icon={<GameFeatureIcon name="shop" className="size-4" />}
            label="開源力"
            value={formatCompact(summary.totalOpenPower)}
            detail={`平均 ${average(summary.totalOpenPower, summary.playerCount)} OP`}
          />
          <MetricTile
            icon={<GameFeatureIcon name="backpack" className="size-4" />}
            label="道具"
            value={formatNumber(summary.totalItems)}
            detail={`掉落 ${summary.droppedItemCount}/${summary.itemDropCount}`}
          />
          <MetricTile
            icon={<GameFeatureIcon name="battle" className="size-4" />}
            label="對戰"
            value={formatNumber(summary.totalMatches)}
            detail={`${summary.activeMatches} 進行中 / ${summary.waitingMatches} 等待`}
          />
          <MetricTile
            icon={<Activity className="size-4" />}
            label="開源力中位數"
            value={formatNumber(summary.medianOpenPower)}
            detail="全體玩家中位數"
          />
          <MetricTile
            icon={<GameFeatureIcon name="shop" className="size-4" />}
            label="商店購買"
            value={formatNumber(summary.shopPurchaseCount)}
            detail="購買紀錄"
          />
          <MetricTile
            icon={<GameFeatureIcon name="forge" className="size-4" />}
            label="合成"
            value={formatNumber(summary.fusionCount)}
            detail="合成紀錄"
          />
          <MetricTile
            icon={<GameFeatureIcon name="shop" className="size-4" />}
            label="Staff 發獎"
            value={formatNumber(summary.staffRewardCount)}
            detail={`${summary.staffCount} staff 帳號`}
          />
          <MetricTile
            icon={<Percent className="size-4" />}
            label="掉落率"
            value={formatPercent(matches.dropRate)}
            detail={`${matches.dropSuccesses}/${matches.dropAttempts} 次`}
          />
          <MetricTile
            icon={<GameFeatureIcon name="leaderboard" className="size-4" />}
            label="平均得分"
            value={
              matches.averageScore > 0 ? matches.averageScore.toFixed(1) : "-"
            }
            detail="每次答題"
          />
          <MetricTile
            icon={<Clock className="size-4" />}
            label="平均作答"
            value={formatSeconds(matches.averageElapsedMillis)}
            detail="每次答題"
          />
        </div>

        <Card className="rounded-[18px] py-5">
          <CardHeader className="px-5">
            <CardTitle className="flex items-center gap-2 text-lg font-black">
              <Activity className="size-5" />
              即時狀態
            </CardTitle>
            <CardDescription>
              更新時間 {formatDateTime(dashboard.generatedAt)}
            </CardDescription>
          </CardHeader>
          <CardContent className="grid gap-4 px-5">
            <StatusBar
              label="完成對戰"
              value={matches.completed}
              total={Math.max(matches.total, 1)}
            />
            <StatusBar
              label="PVP / 電腦"
              value={matches.pvp}
              total={Math.max(matches.total, 1)}
              suffix={`${matches.pvp} / ${matches.computer}`}
            />
            <StatusBar
              label="答題正確"
              value={matches.correctAnswerCount}
              total={Math.max(matches.answerCount, 1)}
              suffix={formatPercent(matches.answerAccuracy)}
            />
            <div className="border-border grid grid-cols-3 gap-2 border-t-2 pt-3 text-center">
              <MiniStat label="等待" value={matches.waiting} />
              <MiniStat label="進行" value={matches.active} />
              <MiniStat label="完成" value={matches.completed} />
            </div>
          </CardContent>
        </Card>
      </section>

      <AdminCollapsibleSection
        title="紀錄與曲線"
        description="資源曲線、staff 發獎與學員資源變動。"
        badge={`${formatNumber((giftHistory?.length ?? 0) + (studentChanges?.length ?? 0))} records`}
      >
        <ResourceHistoryPanel
          history={history}
          isPending={historyPending}
          error={historyError}
          onRetry={onRetryHistory}
        />

        <AdminGiftHistoryPanel
          entries={giftHistory}
          isPending={giftHistoryPending}
          error={giftHistoryError}
          onRetry={onRetryGiftHistory}
        />

        <StudentChangesPanel
          entries={studentChanges}
          isPending={studentChangesPending}
          error={studentChangesError}
          onRetry={onRetryStudentChanges}
        />
      </AdminCollapsibleSection>

      <AdminCollapsibleSection
        title="庫存與排行榜"
        description="小石取得統計、持有率、玩家排行與團隊狀態。"
      >
        <MostOwnedPanel inventory={dashboard.inventory} />

        <section className="grid min-w-0 gap-4 xl:grid-cols-[minmax(0,1.15fr)_minmax(360px,0.85fr)] 2xl:grid-cols-[minmax(0,1.15fr)_minmax(420px,0.85fr)]">
          <TopPlayersPanel
            topPlayers={dashboard.topPlayers}
            onOpenPlayers={openPlayersStats}
          />
          <TeamsPanel teams={dashboard.teams} />
        </section>
      </AdminCollapsibleSection>

      <AdminCollapsibleSection
        title="完整統計"
        description="玩家、庫存、對戰活動與隊伍明細。"
        badge={`${formatNumber(dashboard.players.length)} players`}
        open={statsOpen}
        onOpenChange={setStatsOpen}
      >
        <Tabs
          value={statsTab}
          onValueChange={setStatsTab}
          className="min-w-0 gap-3"
        >
          <div className="flex min-w-0 flex-wrap items-center justify-between gap-3">
            <div className="min-w-0">
              <h2 className="text-xl font-black">完整統計</h2>
              <p className="text-muted-foreground text-sm font-semibold">
                玩家、庫存與對戰活動的營運檢視。
              </p>
            </div>
            <TabsList className="grid w-full max-w-full grid-cols-2 overflow-x-auto sm:grid-cols-[repeat(4,minmax(5rem,1fr))] md:w-fit">
              <TabsTrigger value="players">玩家</TabsTrigger>
              <TabsTrigger value="inventory">庫存</TabsTrigger>
              <TabsTrigger value="matches">對戰</TabsTrigger>
              <TabsTrigger value="teams">隊伍</TabsTrigger>
            </TabsList>
          </div>
          <TabsContent value="players" className="min-w-0">
            <PlayersTable players={dashboard.players} />
          </TabsContent>
          <TabsContent value="inventory" className="min-w-0">
            <InventoryPanel inventory={dashboard.inventory} />
          </TabsContent>
          <TabsContent value="matches" className="min-w-0">
            <MatchesPanel matches={dashboard.matches} />
          </TabsContent>
          <TabsContent value="teams" className="min-w-0">
            <TeamsDetailTable teams={dashboard.teams} />
          </TabsContent>
        </Tabs>
      </AdminCollapsibleSection>
    </div>
  )
}

function AdminGiftHistoryPanel({
  entries,
  isPending,
  error,
  onRetry,
}: {
  entries?: GiftHistoryEntry[]
  isPending: boolean
  error: unknown
  onRetry: () => void
}) {
  if (isPending) {
    return (
      <Card className="rounded-[18px] py-5">
        <CardContent className="flex items-center gap-3 px-5">
          <Spinner className="size-5" />
          <span className="font-black">正在載入贈禮紀錄</span>
        </CardContent>
      </Card>
    )
  }

  if (error) {
    return (
      <Card className="rounded-[18px] py-5">
        <CardHeader className="px-5">
          <CardTitle className="flex items-center gap-2 text-lg font-black">
            <GameFeatureIcon name="history" className="size-5" />
            全玩家贈禮紀錄
          </CardTitle>
          <CardDescription>
            {errorMessage(error, "贈禮紀錄暫時無法讀取。")}
          </CardDescription>
        </CardHeader>
        <CardFooter className="px-5">
          <Button type="button" variant="outline" onClick={onRetry}>
            重新整理
          </Button>
        </CardFooter>
      </Card>
    )
  }

  if (!entries || entries.length === 0) {
    return (
      <Card className="rounded-[18px] py-5">
        <CardHeader className="px-5">
          <CardTitle className="flex items-center gap-2 text-lg font-black">
            <GameFeatureIcon name="history" className="size-5" />
            全玩家贈禮紀錄
          </CardTitle>
          <CardDescription>目前還沒有 staff 發獎紀錄。</CardDescription>
        </CardHeader>
      </Card>
    )
  }

  return (
    <Card className="rounded-[18px] py-5">
      <CardHeader className="gap-3 px-5">
        <div className="flex flex-wrap items-start justify-between gap-3">
          <div>
            <CardTitle className="flex items-center gap-2 text-lg font-black">
              <GameFeatureIcon name="history" className="size-5" />
              全玩家贈禮紀錄
            </CardTitle>
            <CardDescription>
              顯示最近 {entries.length} 筆工作人員發送紀錄。
            </CardDescription>
          </div>
          <Badge variant="secondary">{entries.length} 筆</Badge>
        </div>
      </CardHeader>
      <CardContent className="px-5">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>時間</TableHead>
              <TableHead>收件玩家</TableHead>
              <TableHead>發送者</TableHead>
              <TableHead>類型</TableHead>
              <TableHead>內容</TableHead>
              <TableHead className="text-right">數量</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {entries.slice(0, 20).map((entry) => (
              <TableRow key={entry.rewardId}>
                <TableCell className="font-semibold">
                  {formatDateTime(entry.createdAt)}
                </TableCell>
                <TableCell className="min-w-[180px] whitespace-normal">
                  <div className="grid min-w-0 gap-1">
                    <span className="font-semibold break-words">
                      {entry.recipientNickname || entry.recipientPlayerId}
                    </span>
                    <span className="text-muted-foreground text-xs break-all">
                      {entry.recipientPlayerId}
                    </span>
                  </div>
                </TableCell>
                <TableCell className="min-w-[180px] whitespace-normal">
                  <div className="grid min-w-0 gap-1">
                    <span className="font-semibold break-words">
                      {entry.staffNickname || entry.staffPlayerId}
                    </span>
                    <span className="text-muted-foreground text-xs break-all">
                      {entry.staffPlayerId}
                    </span>
                  </div>
                </TableCell>
                <TableCell>
                  <Badge variant="outline">{giftRewardKindLabel(entry)}</Badge>
                </TableCell>
                <TableCell className="min-w-[160px] whitespace-normal">
                  <div className="flex min-w-0 items-center gap-2">
                    <span
                      className="bg-surface-raised border-border grid size-8 shrink-0 place-items-center rounded-[10px] border"
                      aria-hidden
                    >
                      <GiftRewardIcon entry={entry} />
                    </span>
                    <span className="font-semibold break-words">
                      {entry.name}
                    </span>
                  </div>
                </TableCell>
                <TableCell className="text-right font-black">
                  {giftRewardAmountLabel(entry)}
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </CardContent>
    </Card>
  )
}

function StudentChangesPanel({
  entries,
  isPending,
  error,
  onRetry,
}: {
  entries?: AdminStudentChangeEntry[]
  isPending: boolean
  error: unknown
  onRetry: () => void
}) {
  if (isPending) {
    return (
      <Card className="rounded-[18px] py-5">
        <CardContent className="flex items-center gap-3 px-5">
          <Spinner className="size-5" />
          <span className="font-black">正在載入學員變動紀錄</span>
        </CardContent>
      </Card>
    )
  }

  if (error) {
    return (
      <Card className="rounded-[18px] py-5">
        <CardHeader className="px-5">
          <CardTitle className="flex items-center gap-2 text-lg font-black">
            <GameFeatureIcon name="history" className="size-5" />
            學員變動紀錄
          </CardTitle>
          <CardDescription>
            {errorMessage(error, "學員變動紀錄暫時無法讀取。")}
          </CardDescription>
        </CardHeader>
        <CardFooter className="px-5">
          <Button type="button" variant="outline" onClick={onRetry}>
            重新整理
          </Button>
        </CardFooter>
      </Card>
    )
  }

  if (!entries || entries.length === 0) {
    return (
      <Card className="rounded-[18px] py-5">
        <CardHeader className="px-5">
          <CardTitle className="flex items-center gap-2 text-lg font-black">
            <GameFeatureIcon name="history" className="size-5" />
            學員變動紀錄
          </CardTitle>
          <CardDescription>目前還沒有學員資源變動紀錄。</CardDescription>
        </CardHeader>
      </Card>
    )
  }

  return (
    <Card className="rounded-[18px] py-5">
      <CardHeader className="gap-3 px-5">
        <div className="flex flex-wrap items-start justify-between gap-3">
          <div>
            <CardTitle className="flex items-center gap-2 text-lg font-black">
              <GameFeatureIcon name="history" className="size-5" />
              學員變動紀錄
            </CardTitle>
            <CardDescription>
              顯示最近 {entries.length} 筆非 staff 學員資源變動。
            </CardDescription>
          </div>
          <Badge variant="secondary">{entries.length} 筆</Badge>
        </div>
      </CardHeader>
      <CardContent className="px-5">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>時間</TableHead>
              <TableHead>學員</TableHead>
              <TableHead>來源</TableHead>
              <TableHead>類型</TableHead>
              <TableHead>內容</TableHead>
              <TableHead className="text-right">變動</TableHead>
              <TableHead>備註</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {entries.map((entry) => (
              <TableRow key={entry.changeId}>
                <TableCell className="font-semibold">
                  {formatDateTime(entry.createdAt)}
                </TableCell>
                <TableCell className="min-w-[180px] whitespace-normal">
                  <div className="grid min-w-0 gap-1">
                    <span className="font-semibold break-words">
                      {entry.playerNickname || entry.playerId}
                    </span>
                    <span className="text-muted-foreground text-xs break-all">
                      {entry.playerId}
                      {entry.teamId ? ` / ${entry.teamId}` : ""}
                    </span>
                  </div>
                </TableCell>
                <TableCell>
                  <Badge variant="outline">{entry.sourceLabel}</Badge>
                </TableCell>
                <TableCell>
                  <Badge variant="outline">
                    {resourceKindLabel(entry.kind)}
                  </Badge>
                </TableCell>
                <TableCell className="min-w-[180px] whitespace-normal">
                  <div className="flex min-w-0 items-center gap-2">
                    <span
                      className="bg-surface-raised border-border grid size-8 shrink-0 place-items-center rounded-[10px] border"
                      aria-hidden
                    >
                      <StudentChangeIcon entry={entry} />
                    </span>
                    <span className="grid min-w-0">
                      <strong className="break-words">{entry.name}</strong>
                      {entry.refId ? (
                        <span className="text-muted-foreground text-xs break-all">
                          {entry.refId}
                        </span>
                      ) : null}
                    </span>
                  </div>
                </TableCell>
                <TableCell className="text-right">
                  <Badge
                    variant={entry.delta < 0 ? "destructive" : "secondary"}
                  >
                    {studentChangeDeltaLabel(entry)}
                  </Badge>
                </TableCell>
                <TableCell className="min-w-[160px] whitespace-normal">
                  <span className="text-muted-foreground text-xs font-bold break-words">
                    {entry.note || "-"}
                  </span>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </CardContent>
    </Card>
  )
}

function ResourceHistoryPanel({
  history,
  isPending,
  error,
  onRetry,
}: {
  history?: AdminDashboardHistory
  isPending: boolean
  error: unknown
  onRetry: () => void
}) {
  if (isPending) {
    return (
      <Card className="rounded-[18px] py-5">
        <CardContent className="flex items-center gap-3 px-5">
          <Spinner className="size-5" />
          <span className="font-black">正在載入歷史曲線</span>
        </CardContent>
      </Card>
    )
  }

  if (error) {
    return (
      <Card className="rounded-[18px] py-5">
        <CardHeader className="px-5">
          <CardTitle className="flex items-center gap-2 text-lg font-black">
            <Activity className="size-5" />
            資源曲線
          </CardTitle>
          <CardDescription>
            {errorMessage(error, "歷史資料暫時無法讀取。")}
          </CardDescription>
        </CardHeader>
        <CardFooter className="px-5">
          <Button type="button" variant="outline" onClick={onRetry}>
            重新整理
          </Button>
        </CardFooter>
      </Card>
    )
  }

  if (!history || history.points.length === 0) {
    return (
      <Card className="rounded-[18px] py-5">
        <CardHeader className="px-5">
          <CardTitle className="flex items-center gap-2 text-lg font-black">
            <Activity className="size-5" />
            資源曲線
          </CardTitle>
          <CardDescription>目前還沒有可顯示的歷史資料。</CardDescription>
        </CardHeader>
      </Card>
    )
  }

  const chartData = history.points.map((point) => ({
    ...point,
    label: formatHistoryTimestamp(point.timestamp, history.bucket),
  }))
  const latest = history.points[history.points.length - 1]

  return (
    <Card className="rounded-[18px] py-5">
      <CardHeader className="gap-3 px-5">
        <div className="flex flex-wrap items-start justify-between gap-3">
          <div>
            <CardTitle className="flex items-center gap-2 text-lg font-black">
              <Activity className="size-5" />
              資源曲線
            </CardTitle>
            <CardDescription>
              每小時累積小石與開源力；初始與匯入小石併入起始基準{" "}
              {formatNumber(history.sitoneBaseline)} 顆。
            </CardDescription>
          </div>
          <div className="flex flex-wrap gap-2">
            <Badge variant="secondary">
              小石 {formatNumber(latest.sitoneCount)}
            </Badge>
            <Badge variant="outline">
              開源力 {formatNumber(latest.openPower)} OP
            </Badge>
          </div>
        </div>
      </CardHeader>
      <CardContent className="px-3 sm:px-5">
        <ChartContainer
          config={historyChartConfig}
          className="aspect-auto h-[280px] w-full"
          initialDimension={{ width: 640, height: 280 }}
        >
          <RechartsLineChart
            data={chartData}
            margin={{ top: 12, right: 8, bottom: 4, left: 0 }}
            accessibilityLayer
          >
            <CartesianGrid vertical={false} />
            <XAxis
              dataKey="label"
              tickLine={false}
              axisLine={false}
              tickMargin={8}
              minTickGap={24}
            />
            <YAxis
              yAxisId="sitones"
              tickLine={false}
              axisLine={false}
              tickMargin={8}
              width={44}
              tickFormatter={(value) => formatCompact(Number(value))}
            />
            <YAxis
              yAxisId="openPower"
              orientation="right"
              tickLine={false}
              axisLine={false}
              tickMargin={8}
              width={54}
              tickFormatter={(value) => formatCompact(Number(value))}
            />
            <ChartTooltip
              cursor={false}
              content={<ChartTooltipContent indicator="line" />}
            />
            <Line
              yAxisId="sitones"
              dataKey="sitoneCount"
              type="monotone"
              stroke="var(--color-sitoneCount)"
              strokeWidth={3}
              dot={false}
              activeDot={{ r: 5 }}
            />
            <Line
              yAxisId="openPower"
              dataKey="openPower"
              type="monotone"
              stroke="var(--color-openPower)"
              strokeWidth={3}
              dot={false}
              activeDot={{ r: 5 }}
            />
          </RechartsLineChart>
        </ChartContainer>
      </CardContent>
    </Card>
  )
}

function MostOwnedPanel({
  inventory,
}: {
  inventory: AdminDashboard["inventory"]
}) {
  const [sitoneSortDirection, setSitoneSortDirection] =
    useState<InventorySortDirection>("desc")
  const sortedSitones = useMemo(
    () => sortInventoryEntries(inventory.sitones, sitoneSortDirection),
    [inventory.sitones, sitoneSortDirection],
  )

  return (
    <section className="grid gap-3">
      <div className="grid min-w-0 gap-3 xl:grid-cols-[minmax(0,1.35fr)_minmax(320px,0.65fr)]">
        <SitoneInventoryTableCard
          entries={sortedSitones}
          sortDirection={sitoneSortDirection}
          onSortDirectionChange={setSitoneSortDirection}
        />
        <SitoneOwnershipPieCard entries={inventory.sitones} />
      </div>
      <MostOwnedListCard
        icon={<GameFeatureIcon name="backpack" className="size-5" />}
        title="最多拿到的道具"
        emptyLabel="目前沒有道具持有資料"
        entries={inventory.items.slice(0, 6)}
        unit="個"
      />
    </section>
  )
}

function sortInventoryEntries(
  entries: AdminDashboardInventoryEntry[],
  direction: InventorySortDirection,
) {
  return [...entries].sort((a, b) => {
    const multiplier = direction === "desc" ? -1 : 1
    if (a.quantity !== b.quantity) return (a.quantity - b.quantity) * multiplier
    if (a.ownerCount !== b.ownerCount) {
      return (a.ownerCount - b.ownerCount) * multiplier
    }
    if (a.name !== b.name) return a.name.localeCompare(b.name, "zh-TW")
    return a.id.localeCompare(b.id, "zh-TW")
  })
}

function SitoneInventoryTableCard({
  entries,
  sortDirection,
  onSortDirectionChange,
}: {
  entries: AdminDashboardInventoryEntry[]
  sortDirection: InventorySortDirection
  onSortDirectionChange: (direction: InventorySortDirection) => void
}) {
  return (
    <Card className="min-w-0 rounded-[18px] py-5">
      <CardHeader className="gap-3 px-5">
        <div className="flex flex-wrap items-start justify-between gap-3">
          <div>
            <CardTitle className="flex items-center gap-2 text-lg font-black">
              <GameFeatureIcon name="stones" className="size-5" />
              小石取得統計
            </CardTitle>
            <CardDescription>
              合併最多與最少拿到的小石，完整顯示所有小石 catalog。
            </CardDescription>
          </div>
          <div className="flex max-w-full flex-wrap gap-2">
            <Button
              type="button"
              variant={sortDirection === "desc" ? "default" : "outline"}
              size="sm"
              onClick={() => onSortDirectionChange("desc")}
            >
              <ArrowUpDown className="size-4" />
              由多至少
            </Button>
            <Button
              type="button"
              variant={sortDirection === "asc" ? "default" : "outline"}
              size="sm"
              onClick={() => onSortDirectionChange("asc")}
            >
              <ArrowUpDown className="size-4" />
              由少至多
            </Button>
          </div>
        </div>
      </CardHeader>
      <CardContent className="px-5">
        {entries.length > 0 ? (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="w-16">#</TableHead>
                <TableHead>小石</TableHead>
                <TableHead>分類</TableHead>
                <TableHead className="text-right">總持有</TableHead>
                <TableHead className="text-right">持有人</TableHead>
                <TableHead className="text-right">持有率</TableHead>
                <TableHead>Catalog</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {entries.map((entry, index) => (
                <TableRow key={entry.id}>
                  <TableCell className="font-black">#{index + 1}</TableCell>
                  <TableCell className="min-w-[220px] whitespace-normal">
                    <div className="grid min-w-0 gap-1">
                      <strong className="break-words">{entry.name}</strong>
                      <span className="text-muted-foreground text-xs break-all">
                        {entry.id}
                      </span>
                    </div>
                  </TableCell>
                  <TableCell>{catalogLabel(entry)}</TableCell>
                  <TableCell className="text-right font-black">
                    {formatNumber(entry.quantity)} 顆
                  </TableCell>
                  <TableCell className="text-right">
                    {formatNumber(entry.ownerCount)} 人
                  </TableCell>
                  <TableCell className="text-right font-black">
                    {formatPercent(entry.ownerPercent)}
                  </TableCell>
                  <TableCell>
                    {entry.catalogMissing ? (
                      <Badge variant="destructive">Missing</Badge>
                    ) : (
                      <Badge variant="secondary">OK</Badge>
                    )}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        ) : (
          <EmptyBlock label="目前沒有小石 catalog" />
        )}
      </CardContent>
    </Card>
  )
}

function SitoneOwnershipPieCard({
  entries,
}: {
  entries: AdminDashboardInventoryEntry[]
}) {
  const chartEntries = entries.filter((entry) => entry.ownerCount > 0)

  return (
    <Card className="rounded-[18px] py-5">
      <CardHeader className="px-5">
        <CardTitle className="flex items-center gap-2 text-lg font-black">
          <Percent className="size-5" />
          小石持有率
        </CardTitle>
        <CardDescription>
          圓餅大小依持有人數，標籤顯示玩家持有率。
        </CardDescription>
      </CardHeader>
      <CardContent className="grid gap-3 px-5">
        {chartEntries.length > 0 ? (
          <>
            <ChartContainer
              config={{ owners: { label: "持有人", color: "var(--primary)" } }}
              className="mx-auto aspect-square h-[260px] w-full"
              initialDimension={{ width: 260, height: 260 }}
            >
              <PieChart>
                <ChartTooltip
                  cursor={false}
                  content={
                    <ChartTooltipContent
                      hideLabel
                      formatter={(_value, _name, item) => {
                        const entry =
                          item.payload as AdminDashboardInventoryEntry
                        return (
                          <div className="grid gap-0.5">
                            <span className="font-black">{entry.name}</span>
                            <span className="text-muted-foreground font-bold">
                              {formatNumber(entry.ownerCount)} 人持有 /{" "}
                              {formatPercent(entry.ownerPercent)}
                            </span>
                          </div>
                        )
                      }}
                    />
                  }
                />
                <Pie
                  data={chartEntries}
                  dataKey="ownerCount"
                  nameKey="name"
                  innerRadius={58}
                  outerRadius={104}
                  paddingAngle={1}
                  strokeWidth={2}
                >
                  {chartEntries.map((entry, index) => (
                    <Cell
                      key={entry.id}
                      fill={
                        sitoneOwnershipColors[
                          index % sitoneOwnershipColors.length
                        ]
                      }
                    />
                  ))}
                </Pie>
              </PieChart>
            </ChartContainer>
            <div className="grid max-h-[180px] gap-2 overflow-auto pr-1">
              {entries.slice(0, 12).map((entry, index) => (
                <div
                  key={entry.id}
                  className="grid grid-cols-[12px_minmax(0,1fr)_auto] items-center gap-2 text-xs font-bold"
                >
                  <span
                    className="size-3 rounded-full border"
                    style={{
                      background:
                        sitoneOwnershipColors[
                          index % sitoneOwnershipColors.length
                        ],
                    }}
                  />
                  <span className="truncate">{entry.name}</span>
                  <span>{formatPercent(entry.ownerPercent)}</span>
                </div>
              ))}
            </div>
          </>
        ) : (
          <EmptyBlock label="目前沒有任何小石持有人" />
        )}
      </CardContent>
    </Card>
  )
}

function MostOwnedListCard({
  icon,
  title,
  emptyLabel,
  entries,
  unit,
}: {
  icon: ReactNode
  title: string
  emptyLabel: string
  entries: AdminDashboardInventoryEntry[]
  unit: string
}) {
  return (
    <Card className="rounded-[18px] py-5">
      <CardHeader className="px-5">
        <CardTitle className="flex items-center gap-2 text-lg font-black">
          {icon}
          {title}
        </CardTitle>
        <CardDescription>依全體非 staff 玩家持有數量統計。</CardDescription>
      </CardHeader>
      <CardContent className="grid gap-2 px-5">
        {entries.length > 0 ? (
          entries.map((entry, index) => (
            <MostOwnedListRow
              key={entry.id}
              entry={entry}
              rank={index + 1}
              unit={unit}
            />
          ))
        ) : (
          <EmptyBlock label={emptyLabel} />
        )}
      </CardContent>
    </Card>
  )
}

function MostOwnedListRow({
  entry,
  rank,
  unit,
}: {
  entry: AdminDashboardInventoryEntry
  rank: number
  unit: string
}) {
  return (
    <div className="border-border bg-surface-raised grid gap-3 rounded-[16px] border-2 p-3 md:grid-cols-[48px_minmax(0,1fr)_auto] md:items-center">
      <div className="border-ink bg-card grid size-10 place-items-center rounded-full border-2 text-sm font-black">
        #{rank}
      </div>
      <div className="grid min-w-0 gap-1">
        <div className="flex min-w-0 flex-wrap items-center gap-2">
          <strong className="text-base leading-tight font-black break-words">
            {entry.name}
          </strong>
          {entry.catalogMissing ? (
            <Badge variant="destructive">Missing</Badge>
          ) : (
            <Badge variant="secondary">{catalogLabel(entry)}</Badge>
          )}
        </div>
        <span className="text-muted-foreground text-xs font-bold break-all">
          {entry.id}
        </span>
      </div>
      <div className="grid grid-cols-2 gap-4 md:min-w-[190px] md:text-right">
        <div>
          <span className="text-muted-foreground text-xs font-bold">
            總持有
          </span>
          <div className="text-xl font-black">
            {formatNumber(entry.quantity)}
            <span className="ml-1 text-sm">{unit}</span>
          </div>
        </div>
        <div>
          <span className="text-muted-foreground text-xs font-bold">
            持有人
          </span>
          <div className="text-xl font-black">
            {formatNumber(entry.ownerCount)}
            <span className="ml-1 text-sm">人</span>
          </div>
          <span className="text-muted-foreground text-xs font-bold">
            {formatPercent(entry.ownerPercent)}
          </span>
        </div>
      </div>
    </div>
  )
}

function MetricTile({
  icon,
  label,
  value,
  detail,
}: {
  icon: ReactNode
  label: string
  value: string
  detail: string
}) {
  return (
    <div className="border-ink bg-card grid min-h-[104px] gap-2 rounded-[18px] border-2 p-3 shadow-[2px_2px_0_rgba(23,35,58,0.1)]">
      <div className="text-muted-foreground flex items-center gap-2 text-xs font-black uppercase">
        {icon}
        <span>{label}</span>
      </div>
      <strong className="text-2xl leading-none font-black">{value}</strong>
      <span className="text-muted-foreground text-xs font-bold">{detail}</span>
    </div>
  )
}

function StatusBar({
  label,
  value,
  total,
  suffix,
}: {
  label: string
  value: number
  total: number
  suffix?: string
}) {
  const percent = total > 0 ? Math.round((value / total) * 100) : 0

  return (
    <div className="grid gap-2">
      <div className="flex items-center justify-between gap-3 text-sm font-black">
        <span>{label}</span>
        <span>{suffix ?? `${value}/${total}`}</span>
      </div>
      <Progress value={percent} />
    </div>
  )
}

function MiniStat({ label, value }: { label: string; value: number }) {
  return (
    <div className="grid gap-1">
      <span className="text-muted-foreground text-xs font-bold">{label}</span>
      <strong className="text-lg font-black">{formatNumber(value)}</strong>
    </div>
  )
}

function TopPlayersPanel({
  topPlayers,
  onOpenPlayers,
}: {
  topPlayers: AdminDashboard["topPlayers"]
  onOpenPlayers: () => void
}) {
  const groups: Array<{
    value: string
    label: string
    players: AdminDashboardPlayerRank[]
    metric: (player: AdminDashboardPlayerRank) => string
  }> = [
    {
      value: "sitones",
      label: "小石",
      players: topPlayers.bySitones,
      metric: (player) => `${formatNumber(player.sitoneCount)} 顆`,
    },
    {
      value: "openPower",
      label: "開源力",
      players: topPlayers.byOpenPower,
      metric: (player) => `${formatNumber(player.openPower)} OP`,
    },
    {
      value: "items",
      label: "道具",
      players: topPlayers.byItems,
      metric: (player) => `${formatNumber(player.itemCount)} 個`,
    },
    {
      value: "score",
      label: "分數",
      players: topPlayers.byScore,
      metric: (player) => `${formatNumber(player.score)} 分`,
    },
    {
      value: "accuracy",
      label: "正確率",
      players: topPlayers.byAccuracy,
      metric: (player) =>
        `${formatPercent(player.answerAccuracy)} / ${player.answerCount} 題`,
    },
  ]

  return (
    <Card className="rounded-[18px] py-5">
      <CardHeader className="px-5">
        <div className="flex flex-wrap items-start justify-between gap-3">
          <div>
            <CardTitle className="flex items-center gap-2 text-lg font-black">
              <GameFeatureIcon name="leaderboard" className="size-5" />
              玩家排行
            </CardTitle>
            <CardDescription>
              從不同維度看目前誰拿最多、誰開源力最高、誰答題表現最好。
            </CardDescription>
          </div>
          <Button type="button" variant="outline" onClick={onOpenPlayers}>
            <SlidersHorizontal className="size-4" />
            前往平衡
          </Button>
        </div>
      </CardHeader>
      <CardContent className="px-5">
        <Tabs defaultValue="sitones" className="min-w-0">
          <TabsList className="grid w-full max-w-full grid-cols-[repeat(2,minmax(0,1fr))] overflow-x-auto sm:grid-cols-[repeat(5,minmax(5.75rem,1fr))]">
            {groups.map((group) => (
              <TabsTrigger key={group.value} value={group.value}>
                {group.label}
              </TabsTrigger>
            ))}
          </TabsList>
          {groups.map((group) => (
            <TabsContent key={group.value} value={group.value}>
              <RankTable players={group.players} metric={group.metric} />
            </TabsContent>
          ))}
        </Tabs>
      </CardContent>
    </Card>
  )
}

function RankTable({
  players,
  metric,
}: {
  players: AdminDashboardPlayerRank[]
  metric: (player: AdminDashboardPlayerRank) => string
}) {
  if (players.length === 0) {
    return <EmptyBlock label="目前沒有可排行的玩家資料" />
  }

  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead className="w-16">#</TableHead>
          <TableHead>玩家</TableHead>
          <TableHead>隊伍</TableHead>
          <TableHead className="text-right">指標</TableHead>
          <TableHead className="text-right">OP</TableHead>
          <TableHead className="text-right">正確率</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {players.map((player) => (
          <TableRow key={player.playerId}>
            <TableCell className="font-black">#{player.rank}</TableCell>
            <TableCell>
              <PlayerName player={player} />
            </TableCell>
            <TableCell>{teamLabel(player)}</TableCell>
            <TableCell className="text-right font-black">
              {metric(player)}
            </TableCell>
            <TableCell className="text-right">
              {formatNumber(player.openPower)}
            </TableCell>
            <TableCell className="text-right">
              {formatPercent(player.answerAccuracy)}
            </TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  )
}

function TeamsPanel({ teams }: { teams: AdminDashboardTeam[] }) {
  return (
    <Card className="rounded-[18px] py-5">
      <CardHeader className="px-5">
        <CardTitle className="flex items-center gap-2 text-lg font-black">
          <GameFeatureIcon name="team" className="size-5" />
          團隊狀態
        </CardTitle>
        <CardDescription>
          依小石總量排序，並顯示隊內目前貢獻最高的玩家。
        </CardDescription>
      </CardHeader>
      <CardContent className="px-5">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead className="w-16">#</TableHead>
              <TableHead>隊伍</TableHead>
              <TableHead className="text-right">人數</TableHead>
              <TableHead className="text-right">小石</TableHead>
              <TableHead className="text-right">開源力</TableHead>
              <TableHead>Top</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {teams.map((team) => (
              <TableRow key={team.teamId}>
                <TableCell className="font-black">#{team.rank}</TableCell>
                <TableCell>
                  <div className="flex min-w-0 items-center gap-2">
                    <TeamAvatar team={team} className="size-9" />
                    <div className="grid min-w-0">
                      <strong className="break-words">{team.name}</strong>
                      <span className="text-muted-foreground text-xs break-all">
                        {team.teamId}
                      </span>
                    </div>
                  </div>
                </TableCell>
                <TableCell className="text-right">
                  {formatNumber(team.playerCount)}
                </TableCell>
                <TableCell className="text-right font-black">
                  {formatNumber(team.sitoneCount)}
                </TableCell>
                <TableCell className="text-right">
                  {formatNumber(team.openPower)}
                </TableCell>
                <TableCell>
                  {team.topPlayer ? (
                    <PlayerName player={team.topPlayer} />
                  ) : (
                    "-"
                  )}
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </CardContent>
    </Card>
  )
}

type PlayerBalanceDraft = {
  player: AdminDashboardPlayer
  sitoneCount: string
  openPower: string
}

function PlayersTable({ players }: { players: AdminDashboardPlayer[] }) {
  const queryClient = useQueryClient()
  const [draft, setDraft] = useState<PlayerBalanceDraft | null>(null)

  const balanceMutation = useMutation({
    mutationFn: (input: {
      playerId: string
      sitoneCount: number
      openPower: number
    }) =>
      gameApi.updateAdminPlayerBalance(input.playerId, {
        sitoneCount: input.sitoneCount,
        openPower: input.openPower,
      }),
    onSuccess: (result) => {
      setDraft(null)
      void queryClient.invalidateQueries({ queryKey: ["admin", "dashboard"] })
      void queryClient.invalidateQueries({ queryKey: ["admin", "history"] })
      void queryClient.invalidateQueries({ queryKey: ["leaderboards"] })
      void queryClient.invalidateQueries({ queryKey: ["me"] })
      toast.success(`${result.nickname} 的平衡數值已更新`)
    },
    onError: (error) => {
      toast.error(errorMessage(error, "玩家平衡更新失敗"))
    },
  })

  function startBalance(player: AdminDashboardPlayer) {
    setDraft({
      player,
      sitoneCount: String(player.sitoneCount),
      openPower: String(player.openPower),
    })
  }

  function updateDraft(patch: Partial<Omit<PlayerBalanceDraft, "player">>) {
    setDraft((current) => (current ? { ...current, ...patch } : current))
  }

  function submitBalance(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (!draft || balanceMutation.isPending) return
    const sitoneCount = Math.max(0, integerOrZero(draft.sitoneCount))
    const openPower = Math.max(0, integerOrZero(draft.openPower))
    balanceMutation.mutate({
      playerId: draft.player.playerId,
      sitoneCount,
      openPower,
    })
  }

  const activePlayerID = draft?.player.playerId

  return (
    <>
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>#</TableHead>
            <TableHead>玩家</TableHead>
            <TableHead>隊伍</TableHead>
            <TableHead className="text-right">小石</TableHead>
            <TableHead className="text-right">OP</TableHead>
            <TableHead className="text-right">道具</TableHead>
            <TableHead className="text-right">對戰</TableHead>
            <TableHead className="text-right">答題</TableHead>
            <TableHead className="text-right">正確率</TableHead>
            <TableHead className="text-right">分數</TableHead>
            <TableHead>最近活動</TableHead>
            <TableHead className="w-[112px] text-right">操作</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {players.map((player) => {
            const isPending =
              balanceMutation.isPending && activePlayerID === player.playerId

            return (
              <TableRow key={player.playerId}>
                <TableCell className="font-black">#{player.rank}</TableCell>
                <TableCell>
                  <PlayerName player={player} />
                </TableCell>
                <TableCell>{teamLabel(player)}</TableCell>
                <TableCell className="text-right font-black">
                  {formatNumber(player.sitoneCount)}
                </TableCell>
                <TableCell className="text-right">
                  {formatNumber(player.openPower)}
                </TableCell>
                <TableCell className="text-right">
                  {formatNumber(player.itemCount)}
                </TableCell>
                <TableCell className="text-right">
                  {formatNumber(player.completedMatchCount)}/
                  {formatNumber(player.matchCount)}
                </TableCell>
                <TableCell className="text-right">
                  {formatNumber(player.correctAnswerCount)}/
                  {formatNumber(player.answerCount)}
                </TableCell>
                <TableCell className="text-right">
                  {formatPercent(player.answerAccuracy)}
                </TableCell>
                <TableCell className="text-right">
                  {formatNumber(player.score)}
                </TableCell>
                <TableCell>{formatDateTime(player.lastActivityAt)}</TableCell>
                <TableCell className="text-right">
                  <Button
                    type="button"
                    variant="outline"
                    size="sm"
                    disabled={balanceMutation.isPending}
                    onClick={() => startBalance(player)}
                  >
                    {isPending ? (
                      <Spinner className="size-4" />
                    ) : (
                      <SlidersHorizontal className="size-4" />
                    )}
                    平衡
                  </Button>
                </TableCell>
              </TableRow>
            )
          })}
        </TableBody>
      </Table>

      <Dialog
        open={draft != null}
        onOpenChange={(open) => !open && setDraft(null)}
      >
        <DialogContent className="sm:max-w-[430px]">
          <form className="grid gap-5" onSubmit={submitBalance}>
            <DialogHeader>
              <DialogTitle>調整玩家平衡</DialogTitle>
              <DialogDescription>
                設定更新後的總小石數與總開源力。若數值降低，玩家會收到離家出走通知。
              </DialogDescription>
            </DialogHeader>

            {draft ? (
              <div className="grid gap-4">
                <div className="bg-surface-raised border-border grid grid-cols-[auto_minmax(0,1fr)] items-center gap-3 rounded-[16px] border-2 p-3">
                  <PlayerAvatar
                    playerId={draft.player.playerId}
                    nickname={draft.player.nickname}
                    avatarUrl={draft.player.avatarUrl}
                    className="size-12"
                  />
                  <div className="min-w-0">
                    <strong className="block truncate">
                      {draft.player.nickname}
                    </strong>
                    <span className="text-muted-foreground text-xs font-bold">
                      {draft.player.playerId} · {teamLabel(draft.player)}
                    </span>
                  </div>
                </div>

                <div className="grid gap-2">
                  <Label htmlFor="balance-sitone-count">小石總數</Label>
                  <Input
                    id="balance-sitone-count"
                    type="number"
                    min={0}
                    step={1}
                    value={draft.sitoneCount}
                    onChange={(event) =>
                      updateDraft({ sitoneCount: event.target.value })
                    }
                  />
                  <span className="text-muted-foreground text-xs font-bold">
                    目前 {formatNumber(draft.player.sitoneCount)} 顆
                  </span>
                </div>

                <div className="grid gap-2">
                  <Label htmlFor="balance-open-power">開源力總數</Label>
                  <Input
                    id="balance-open-power"
                    type="number"
                    min={0}
                    step={1}
                    value={draft.openPower}
                    onChange={(event) =>
                      updateDraft({ openPower: event.target.value })
                    }
                  />
                  <span className="text-muted-foreground text-xs font-bold">
                    目前 {formatNumber(draft.player.openPower)} OP
                  </span>
                </div>
              </div>
            ) : null}

            <DialogFooter>
              <Button
                type="button"
                variant="outline"
                disabled={balanceMutation.isPending}
                onClick={() => setDraft(null)}
              >
                取消
              </Button>
              <Button
                type="submit"
                disabled={!draft || balanceMutation.isPending}
              >
                {balanceMutation.isPending ? (
                  <Spinner className="size-4" />
                ) : (
                  <Save className="size-4" />
                )}
                Update
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>
    </>
  )
}

function InventoryPanel({
  inventory,
}: {
  inventory: AdminDashboard["inventory"]
}) {
  return (
    <Tabs defaultValue="sitones">
      <TabsList>
        <TabsTrigger value="sitones">
          小石 {formatNumber(inventory.sitones.length)}
        </TabsTrigger>
        <TabsTrigger value="items">
          道具 {formatNumber(inventory.items.length)}
        </TabsTrigger>
      </TabsList>
      <TabsContent value="sitones">
        <InventoryTable entries={inventory.sitones} kind="小石" />
      </TabsContent>
      <TabsContent value="items">
        <InventoryTable entries={inventory.items} kind="道具" />
      </TabsContent>
    </Tabs>
  )
}

function InventoryTable({
  entries,
  kind,
}: {
  entries: AdminDashboardInventoryEntry[]
  kind: string
}) {
  if (entries.length === 0) {
    return <EmptyBlock label={`目前沒有${kind}持有資料`} />
  }

  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>{kind}</TableHead>
          <TableHead>分類</TableHead>
          <TableHead className="text-right">總量</TableHead>
          <TableHead className="text-right">持有人</TableHead>
          <TableHead className="text-right">持有率</TableHead>
          <TableHead>Catalog</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {entries.map((entry) => (
          <TableRow key={entry.id}>
            <TableCell>
              <div className="grid">
                <strong>{entry.name}</strong>
                <span className="text-muted-foreground text-xs">
                  {entry.id}
                </span>
              </div>
            </TableCell>
            <TableCell>{catalogLabel(entry)}</TableCell>
            <TableCell className="text-right font-black">
              {formatNumber(entry.quantity)}
            </TableCell>
            <TableCell className="text-right">
              {formatNumber(entry.ownerCount)}
            </TableCell>
            <TableCell className="text-right font-black">
              {formatPercent(entry.ownerPercent)}
            </TableCell>
            <TableCell>
              {entry.catalogMissing ? (
                <Badge variant="destructive">Missing</Badge>
              ) : (
                <Badge variant="secondary">OK</Badge>
              )}
            </TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  )
}

function MatchesPanel({ matches }: { matches: AdminDashboard["matches"] }) {
  return (
    <div className="grid gap-4 xl:grid-cols-[360px_minmax(0,1fr)]">
      <Card className="rounded-[18px] py-5">
        <CardHeader className="px-5">
          <CardTitle className="flex items-center gap-2 text-lg font-black">
            <GameFeatureIcon name="battle" className="size-5" />
            對戰摘要
          </CardTitle>
        </CardHeader>
        <CardContent className="grid gap-3 px-5">
          <MiniMetric label="總對戰" value={matches.total} />
          <MiniMetric label="PVP" value={matches.pvp} />
          <MiniMetric label="電腦" value={matches.computer} />
          <MiniMetric label="答題" value={matches.answerCount} />
          <MiniMetric label="平均得分" value={matches.averageScore} />
          <MiniMetric
            label="平均作答秒數"
            value={matches.averageElapsedMillis / 1000}
          />
        </CardContent>
      </Card>

      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Match</TableHead>
            <TableHead>模式</TableHead>
            <TableHead className="text-right">玩家</TableHead>
            <TableHead>勝者</TableHead>
            <TableHead className="text-right">最高分</TableHead>
            <TableHead>完成時間</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {matches.recent.map((match) => (
            <TableRow key={match.matchId}>
              <TableCell>
                <div className="grid">
                  <strong>{match.code || match.matchId}</strong>
                  <span className="text-muted-foreground text-xs">
                    {match.matchId}
                  </span>
                </div>
              </TableCell>
              <TableCell>
                <Badge
                  variant={match.mode === "computer" ? "outline" : "secondary"}
                >
                  {match.mode === "computer" ? "電腦" : "PVP"}
                </Badge>
              </TableCell>
              <TableCell className="text-right">
                {formatNumber(match.playerCount)}
              </TableCell>
              <TableCell>{match.winnerNickname || "-"}</TableCell>
              <TableCell className="text-right">
                {formatNumber(match.topScore)}
              </TableCell>
              <TableCell>{formatDateTime(match.completedAt)}</TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  )
}

function MiniMetric({ label, value }: { label: string; value: number }) {
  return (
    <div className="border-border flex items-center justify-between gap-3 border-b pb-2 last:border-b-0 last:pb-0">
      <span className="text-muted-foreground text-sm font-bold">{label}</span>
      <strong className="font-black">
        {Number.isInteger(value) ? formatNumber(value) : value.toFixed(1)}
      </strong>
    </div>
  )
}

type TeamEditDraft = {
  teamId: string
  name: string
  avatarUrl: string
}

function TeamsDetailTable({ teams }: { teams: AdminDashboardTeam[] }) {
  const queryClient = useQueryClient()
  const [draft, setDraft] = useState<TeamEditDraft | null>(null)

  const updateTeamMutation = useMutation({
    mutationFn: (input: TeamEditDraft) =>
      gameApi.updateAdminTeam(input.teamId, {
        name: input.name.trim(),
        avatarUrl: input.avatarUrl.trim(),
      }),
    onSuccess: (team) => {
      setDraft(null)
      void queryClient.invalidateQueries({ queryKey: ["admin", "dashboard"] })
      void queryClient.invalidateQueries({ queryKey: ["leaderboards"] })
      void queryClient.invalidateQueries({ queryKey: ["me"] })
      void queryClient.invalidateQueries({ queryKey: ["staff", "teams"] })
      toast.success(`${team.name} 已更新`)
    },
    onError: (error) => {
      toast.error(errorMessage(error, "隊伍更新失敗"))
    },
  })

  function startEdit(team: AdminDashboardTeam) {
    setDraft({
      teamId: team.teamId,
      name: team.name,
      avatarUrl: team.avatarUrl ?? "",
    })
  }

  function updateDraft(patch: Partial<Omit<TeamEditDraft, "teamId">>) {
    setDraft((current) => (current ? { ...current, ...patch } : current))
  }

  function saveDraft() {
    if (!draft || updateTeamMutation.isPending) return
    if (!draft.name.trim()) return
    updateTeamMutation.mutate({
      ...draft,
      name: draft.name.trim(),
      avatarUrl: draft.avatarUrl.trim(),
    })
  }

  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>#</TableHead>
          <TableHead className="min-w-[320px]">隊伍</TableHead>
          <TableHead className="text-right">人數</TableHead>
          <TableHead className="text-right">小石</TableHead>
          <TableHead className="text-right">平均小石</TableHead>
          <TableHead className="text-right">OP</TableHead>
          <TableHead className="text-right">平均 OP</TableHead>
          <TableHead className="text-right">道具</TableHead>
          <TableHead className="text-right">平均道具</TableHead>
          <TableHead>Top Player</TableHead>
          <TableHead className="w-[112px] text-right">操作</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {teams.map((team) => {
          const currentDraft = draft?.teamId === team.teamId ? draft : null
          const previewTeam = currentDraft
            ? {
                ...team,
                name: currentDraft.name,
                avatarUrl: currentDraft.avatarUrl.trim() || undefined,
              }
            : team
          const isSaving =
            updateTeamMutation.isPending &&
            updateTeamMutation.variables?.teamId === team.teamId

          return (
            <TableRow key={team.teamId}>
              <TableCell className="font-black">#{team.rank}</TableCell>
              <TableCell>
                {currentDraft ? (
                  <div className="grid min-w-[280px] gap-2">
                    <div className="flex min-w-0 items-center gap-2">
                      <TeamAvatar team={previewTeam} className="size-10" />
                      <Input
                        aria-label={`${team.name} 隊伍名稱`}
                        value={currentDraft.name}
                        onChange={(event) =>
                          updateDraft({ name: event.target.value })
                        }
                        className="h-9 font-bold"
                        maxLength={64}
                      />
                    </div>
                    <div className="relative">
                      <ImageIcon className="text-muted-foreground pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2" />
                      <Input
                        aria-label={`${team.name} 隊伍頭貼 URL`}
                        value={currentDraft.avatarUrl}
                        onChange={(event) =>
                          updateDraft({ avatarUrl: event.target.value })
                        }
                        placeholder="https://... 或 /game-icons/..."
                        className="h-9 pl-9 text-xs"
                        maxLength={512}
                      />
                    </div>
                    <span className="text-muted-foreground text-xs font-bold break-all">
                      {team.teamId}
                    </span>
                  </div>
                ) : (
                  <div className="flex min-w-0 items-center gap-2">
                    <TeamAvatar team={team} className="size-10" />
                    <div className="grid min-w-0">
                      <strong className="break-words">{team.name}</strong>
                      <span className="text-muted-foreground text-xs break-all">
                        {team.teamId}
                      </span>
                    </div>
                  </div>
                )}
              </TableCell>
              <TableCell className="text-right">
                {formatNumber(team.playerCount)}
              </TableCell>
              <TableCell className="text-right font-black">
                {formatNumber(team.sitoneCount)}
              </TableCell>
              <TableCell className="text-right">
                {team.averageSitones.toFixed(1)}
              </TableCell>
              <TableCell className="text-right">
                {formatNumber(team.openPower)}
              </TableCell>
              <TableCell className="text-right">
                {team.averageOpenPower.toFixed(1)}
              </TableCell>
              <TableCell className="text-right">
                {formatNumber(team.itemCount)}
              </TableCell>
              <TableCell className="text-right">
                {team.averageItems.toFixed(1)}
              </TableCell>
              <TableCell>{team.topPlayer?.nickname ?? "-"}</TableCell>
              <TableCell>
                {currentDraft ? (
                  <div className="flex justify-end gap-2">
                    <Button
                      type="button"
                      variant="outline"
                      size="icon"
                      aria-label={`取消編輯 ${team.name}`}
                      disabled={isSaving}
                      onClick={() => setDraft(null)}
                    >
                      <X className="size-4" />
                    </Button>
                    <Button
                      type="button"
                      size="icon"
                      aria-label={`儲存 ${team.name}`}
                      disabled={!currentDraft.name.trim() || isSaving}
                      onClick={saveDraft}
                    >
                      {isSaving ? (
                        <Spinner className="size-4" />
                      ) : (
                        <Save className="size-4" />
                      )}
                    </Button>
                  </div>
                ) : (
                  <Button
                    type="button"
                    variant="outline"
                    size="icon"
                    aria-label={`編輯 ${team.name}`}
                    disabled={updateTeamMutation.isPending}
                    onClick={() => startEdit(team)}
                  >
                    <Pencil className="size-4" />
                  </Button>
                )}
              </TableCell>
            </TableRow>
          )
        })}
      </TableBody>
    </Table>
  )
}

function TeamAvatar({
  team,
  className,
}: {
  team: TeamAvatarModel
  className?: string
}) {
  const avatarSrcs = useMemo(
    () => imageSrcCandidates(team.avatarUrl),
    [team.avatarUrl],
  )
  const avatarSrcKey = useMemo(() => avatarSrcs.join("\n"), [avatarSrcs])
  const [avatarErrorState, setAvatarErrorState] = useState({
    key: "",
    index: 0,
  })
  const avatarSrcIndex =
    avatarErrorState.key === avatarSrcKey ? avatarErrorState.index : 0
  const currentAvatarSrc = avatarSrcs[avatarSrcIndex]

  return (
    <Avatar className={cn("bg-surface-raised border-ink border", className)}>
      {currentAvatarSrc ? (
        <AvatarImage
          src={currentAvatarSrc}
          alt=""
          aria-hidden="true"
          draggable={false}
          className="block size-full object-cover"
          onError={() =>
            setAvatarErrorState((state) => ({
              key: avatarSrcKey,
              index: state.key === avatarSrcKey ? state.index + 1 : 1,
            }))
          }
        />
      ) : null}
      <AvatarFallback className="text-xs font-black">
        {teamAvatarFallback(team)}
      </AvatarFallback>
    </Avatar>
  )
}

function PlayerName({
  player,
}: {
  player: Pick<AdminDashboardPlayer, "playerId" | "nickname" | "avatarUrl">
}) {
  return (
    <div className="flex items-center gap-2">
      <PlayerAvatar
        playerId={player.playerId}
        nickname={player.nickname}
        avatarUrl={player.avatarUrl}
        className="border-ink size-8 rounded-full border"
      />
      <div className="grid">
        <strong>{player.nickname}</strong>
        <span className="text-muted-foreground text-xs">{player.playerId}</span>
      </div>
    </div>
  )
}

function EmptyBlock({ label }: { label: string }) {
  return (
    <div className="border-border bg-surface-raised rounded-[18px] border-2 p-5 text-sm font-black">
      {label}
    </div>
  )
}

const communityStandRewardKindLabels: Record<StaffRewardKind, string> = {
  item: "道具",
  sitone: "小石",
  open_power: "開源力",
}

function AdminCommunityStandsPanel({
  stands,
  claims,
  claimsError,
  claimsPending,
  error,
  isPending,
  isCreateFormOpen,
  isCreating,
  deletingStandID,
  pendingStandID,
  onRetry,
  onToggleCreate,
  onCreate,
  onDelete,
  onRetryClaims,
  onSubmit,
}: {
  stands: AdminCommunityStand[]
  claims: AdminCommunityStandClaim[]
  claimsError: unknown
  claimsPending: boolean
  error: unknown
  isPending: boolean
  isCreateFormOpen: boolean
  isCreating: boolean
  deletingStandID?: string
  pendingStandID?: string
  onRetry: () => void
  onToggleCreate: () => void
  onCreate: (input: AdminCommunityStandCreateInput) => void
  onDelete: (standID: string) => void
  onRetryClaims: () => void
  onSubmit: (standID: string, input: AdminCommunityStandUpdateInput) => void
}) {
  if (isPending) {
    return (
      <Card className="rounded-[18px] py-5">
        <CardContent className="flex items-center gap-3 px-5">
          <Spinner className="size-5" />
          <span className="font-black">正在載入攤位設定</span>
        </CardContent>
      </Card>
    )
  }

  if (error) {
    return (
      <Card className="rounded-[18px] py-5">
        <CardHeader className="px-5">
          <CardTitle className="flex items-center gap-2 text-lg font-black">
            <GameFeatureIcon name="team" className="size-5" />
            攤位設定
          </CardTitle>
          <CardDescription>
            {errorMessage(error, "攤位設定暫時無法讀取。")}
          </CardDescription>
        </CardHeader>
        <CardFooter className="px-5">
          <Button type="button" variant="outline" onClick={onRetry}>
            重新整理
          </Button>
        </CardFooter>
      </Card>
    )
  }

  return (
    <section className="grid gap-3">
      <div className="flex flex-wrap items-end justify-between gap-3">
        <div>
          <h2 className="text-xl font-black">攤位設定</h2>
          <p className="text-muted-foreground text-sm font-semibold">
            管理社群攤位顯示、啟用狀態與掃描獎勵。
          </p>
        </div>
        <div className="grid w-full grid-cols-1 items-center gap-2 sm:grid-cols-[1fr_auto]">
          <Badge variant="outline" className="justify-self-start">
            {formatNumber(stands.length)} booths
          </Badge>
          <Button type="button" size="sm" onClick={onToggleCreate}>
            <Plus className="size-4" />
            新增攤位
          </Button>
        </div>
      </div>

      {isCreateFormOpen ? (
        <CommunityStandCreateCard
          isPending={isCreating}
          onCancel={onToggleCreate}
          onSubmit={onCreate}
        />
      ) : null}

      {stands.length === 0 ? (
        <EmptyBlock label="目前沒有攤位資料" />
      ) : (
        <div className="grid gap-3">
          {stands.map((stand) => (
            <CommunityStandEditorCard
              key={`${stand.standId}:${stand.updatedAt}`}
              stand={stand}
              isDeleting={deletingStandID === stand.standId}
              isPending={pendingStandID === stand.standId}
              onSubmit={onSubmit}
              onDelete={onDelete}
            />
          ))}
        </div>
      )}

      <CommunityStandClaimsPanel
        claims={claims}
        error={claimsError}
        isPending={claimsPending}
        onRetry={onRetryClaims}
      />
    </section>
  )
}

function CommunityStandClaimsPanel({
  claims,
  error,
  isPending,
  onRetry,
}: {
  claims: AdminCommunityStandClaim[]
  error: unknown
  isPending: boolean
  onRetry: () => void
}) {
  if (isPending) {
    return (
      <Card className="rounded-[18px] py-5">
        <CardContent className="flex items-center gap-3 px-5">
          <Spinner className="size-5" />
          <span className="font-black">正在載入攤位領取紀錄</span>
        </CardContent>
      </Card>
    )
  }

  if (error) {
    return (
      <Card className="rounded-[18px] py-5">
        <CardHeader className="px-5">
          <CardTitle className="flex items-center gap-2 text-lg font-black">
            <GameFeatureIcon name="history" className="size-5" />
            攤位領取紀錄
          </CardTitle>
          <CardDescription>
            {errorMessage(error, "攤位領取紀錄暫時無法讀取。")}
          </CardDescription>
        </CardHeader>
        <CardFooter className="px-5">
          <Button type="button" variant="outline" onClick={onRetry}>
            重新整理
          </Button>
        </CardFooter>
      </Card>
    )
  }

  if (claims.length === 0) {
    return <EmptyBlock label="目前沒有攤位領取紀錄" />
  }

  const visibleClaims = claims.slice(0, 50)

  return (
    <Card className="rounded-[18px] py-5">
      <CardHeader className="gap-3 px-5">
        <div className="flex flex-wrap items-start justify-between gap-3">
          <div>
            <CardTitle className="flex items-center gap-2 text-lg font-black">
              <GameFeatureIcon name="history" className="size-5" />
              攤位領取紀錄
            </CardTitle>
            <CardDescription>
              顯示最近 {visibleClaims.length} 筆社群攤位領取紀錄。
            </CardDescription>
          </div>
          <Button type="button" variant="outline" size="sm" onClick={onRetry}>
            <RefreshCw className="size-4" />
            更新
          </Button>
        </div>
      </CardHeader>
      <CardContent className="px-5">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>時間</TableHead>
              <TableHead>攤位</TableHead>
              <TableHead>領取者</TableHead>
              <TableHead>類型</TableHead>
              <TableHead>內容</TableHead>
              <TableHead className="text-right">數量</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {visibleClaims.map((claim) => (
              <TableRow key={claim.claimId}>
                <TableCell className="font-semibold">
                  {formatDateTime(claim.createdAt)}
                </TableCell>
                <TableCell className="min-w-[180px] whitespace-normal">
                  <div className="grid min-w-0 gap-1">
                    <span className="font-semibold break-words">
                      {claim.standName || claim.standId}
                    </span>
                    <span className="text-muted-foreground text-xs break-all">
                      {claim.standId}
                    </span>
                  </div>
                </TableCell>
                <TableCell className="min-w-[160px] whitespace-normal">
                  <div className="grid min-w-0 gap-1">
                    <span className="font-semibold break-words">
                      {claim.playerNickname || claim.playerId}
                    </span>
                    <span className="text-muted-foreground text-xs break-all">
                      {claim.playerId}
                    </span>
                  </div>
                </TableCell>
                <TableCell>
                  <Badge variant="outline">
                    {communityStandRewardKindLabels[claim.reward.kind]}
                  </Badge>
                </TableCell>
                <TableCell className="min-w-[160px] whitespace-normal">
                  <div className="flex min-w-0 items-center gap-2">
                    <span
                      className="bg-surface-raised border-border grid size-8 shrink-0 place-items-center overflow-hidden rounded-[10px] border"
                      aria-hidden
                    >
                      <CommunityStandClaimRewardIcon claim={claim} />
                    </span>
                    <div className="grid min-w-0 gap-1">
                      <span className="font-semibold break-words">
                        {claim.reward.name}
                      </span>
                      {claim.reward.refId ? (
                        <span className="text-muted-foreground text-xs break-all">
                          {claim.reward.refId}
                        </span>
                      ) : null}
                    </div>
                  </div>
                </TableCell>
                <TableCell className="text-right font-black">
                  {communityStandClaimAmountLabel(claim)}
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </CardContent>
    </Card>
  )
}

function CommunityStandClaimRewardIcon({
  claim,
}: {
  claim: AdminCommunityStandClaim
}) {
  if (claim.reward.iconPath) {
    return (
      <img
        src={claim.reward.iconPath}
        alt=""
        className="size-full object-cover"
      />
    )
  }
  return <ImageIcon className="text-muted-foreground size-4" />
}

function communityStandClaimAmountLabel(claim: AdminCommunityStandClaim) {
  if (claim.reward.kind === "open_power") {
    return `${formatNumber(claim.reward.amount ?? 0)} OP`
  }
  return `x${formatNumber(claim.reward.quantity ?? 1)}`
}

function CommunityStandCreateCard({
  isPending,
  onCancel,
  onSubmit,
}: {
  isPending: boolean
  onCancel: () => void
  onSubmit: (input: AdminCommunityStandCreateInput) => void
}) {
  const [draft, setDraft] = useState<AdminCommunityStandCreateInput>(() =>
    newCommunityStandDraft(),
  )
  const formID = "admin-community-stand-create"

  function patchDraft(patch: Partial<AdminCommunityStandCreateInput>) {
    setDraft((current) => ({ ...current, ...patch }))
  }

  function patchReward(
    patch: Partial<AdminCommunityStandCreateInput["reward"]>,
  ) {
    setDraft((current) => ({
      ...current,
      reward: { ...current.reward, ...patch },
    }))
  }

  function handleRewardKindChange(kind: StaffRewardKind) {
    setDraft((current) => ({
      ...current,
      reward: communityStandRewardDraftForKind(kind, current.reward),
    }))
  }

  function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    onSubmit(sanitizeCommunityStandCreateDraft(draft))
  }

  return (
    <Card className="border-ink rounded-[18px] border-2 py-5">
      <form id={formID} className="grid gap-3" onSubmit={handleSubmit}>
        <CardHeader className="gap-3 px-5">
          <div className="flex items-start justify-between gap-3">
            <div className="min-w-0">
              <CardTitle className="text-lg font-black">新增社群攤位</CardTitle>
              <CardDescription>
                建立可顯示 QR Code 的社群攤位頁。
              </CardDescription>
            </div>
            <Badge variant={draft.enabled ? "default" : "secondary"}>
              {draft.enabled ? "啟用" : "停用"}
            </Badge>
          </div>
        </CardHeader>

        <CardContent className="grid gap-4 px-5">
          <div className="bg-surface-raised border-border grid grid-cols-[minmax(0,1fr)_auto] items-center gap-3 rounded-[18px] border-2 p-3">
            <span className="text-sm font-black">建立後開放攤位看板</span>
            <Switch
              checked={draft.enabled}
              onCheckedChange={(checked) => patchDraft({ enabled: checked })}
              aria-label="新攤位啟用狀態"
            />
          </div>

          <div className="grid gap-3">
            <Field>
              <Label htmlFor={`${formID}-name`}>名稱</Label>
              <Input
                id={`${formID}-name`}
                value={draft.name}
                maxLength={96}
                onChange={(event) => patchDraft({ name: event.target.value })}
              />
            </Field>
            <Field>
              <Label htmlFor={`${formID}-description`}>介紹</Label>
              <Textarea
                id={`${formID}-description`}
                value={draft.description}
                maxLength={1000}
                onChange={(event) =>
                  patchDraft({ description: event.target.value })
                }
              />
            </Field>
            <Field>
              <Label htmlFor={`${formID}-logo`}>Logo URL</Label>
              <Input
                id={`${formID}-logo`}
                value={draft.logoUrl ?? ""}
                placeholder="/game-icons/features/team.png"
                onChange={(event) =>
                  patchDraft({ logoUrl: event.target.value })
                }
              />
            </Field>
            <Field>
              <Label htmlFor={`${formID}-website`}>網站 URL</Label>
              <Input
                id={`${formID}-website`}
                value={draft.websiteUrl ?? ""}
                placeholder="https://sitcon.org"
                onChange={(event) =>
                  patchDraft({ websiteUrl: event.target.value })
                }
              />
            </Field>
          </div>

          <div className="bg-surface-raised border-border grid gap-3 rounded-[18px] border-2 p-3">
            <div className="grid gap-3">
              <Field>
                <Label htmlFor={`${formID}-reward-kind`}>獎勵</Label>
                <Select
                  value={draft.reward.kind}
                  onValueChange={(value) =>
                    handleRewardKindChange(value as StaffRewardKind)
                  }
                >
                  <SelectTrigger
                    id={`${formID}-reward-kind`}
                    className="w-full"
                  >
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="item">道具</SelectItem>
                    <SelectItem value="sitone">小石</SelectItem>
                    <SelectItem value="open_power">開源力</SelectItem>
                  </SelectContent>
                </Select>
              </Field>

              {draft.reward.kind === "open_power" ? (
                <Field>
                  <Label htmlFor={`${formID}-reward-amount`}>開源力數量</Label>
                  <Input
                    id={`${formID}-reward-amount`}
                    type="number"
                    min={1}
                    max={100000}
                    value={draft.reward.amount ?? 50}
                    onChange={(event) =>
                      patchReward({ amount: Number(event.target.value) })
                    }
                  />
                </Field>
              ) : (
                <>
                  <Field>
                    <Label htmlFor={`${formID}-reward-ref`}>Content ID</Label>
                    <Input
                      id={`${formID}-reward-ref`}
                      value={draft.reward.refId ?? ""}
                      placeholder={
                        draft.reward.kind === "item"
                          ? "item_booth_sticker"
                          : "stone_booth"
                      }
                      onChange={(event) =>
                        patchReward({ refId: event.target.value })
                      }
                    />
                  </Field>
                  <Field>
                    <Label htmlFor={`${formID}-reward-quantity`}>數量</Label>
                    <Input
                      id={`${formID}-reward-quantity`}
                      type="number"
                      min={1}
                      max={1000}
                      value={draft.reward.quantity ?? 1}
                      onChange={(event) =>
                        patchReward({ quantity: Number(event.target.value) })
                      }
                    />
                  </Field>
                </>
              )}
            </div>
          </div>
        </CardContent>

        <CardFooter className="grid grid-cols-1 gap-2 px-5 sm:grid-cols-2">
          <Button
            type="button"
            variant="outline"
            disabled={isPending}
            onClick={onCancel}
          >
            <X />
            取消
          </Button>
          <Button type="submit" disabled={isPending}>
            <Plus />
            {isPending ? "新增中" : "新增攤位"}
          </Button>
        </CardFooter>
      </form>
    </Card>
  )
}

function CommunityStandEditorCard({
  stand,
  isPending,
  isDeleting,
  onSubmit,
  onDelete,
}: {
  stand: AdminCommunityStand
  isPending: boolean
  isDeleting: boolean
  onSubmit: (standID: string, input: AdminCommunityStandUpdateInput) => void
  onDelete: (standID: string) => void
}) {
  const [draft, setDraft] = useState<AdminCommunityStandUpdateInput>(() =>
    communityStandToDraft(stand),
  )

  const savedDraft = communityStandToDraft(stand)
  const dirty =
    JSON.stringify(sanitizeCommunityStandDraft(draft)) !==
    JSON.stringify(sanitizeCommunityStandDraft(savedDraft))
  const formID = `admin-community-stand-${stand.standId}`

  function patchDraft(patch: Partial<AdminCommunityStandUpdateInput>) {
    setDraft((current) => ({ ...current, ...patch }))
  }

  function patchReward(
    patch: Partial<AdminCommunityStandUpdateInput["reward"]>,
  ) {
    setDraft((current) => ({
      ...current,
      reward: { ...current.reward, ...patch },
    }))
  }

  function handleRewardKindChange(kind: StaffRewardKind) {
    setDraft((current) => ({
      ...current,
      reward: communityStandRewardDraftForKind(kind, current.reward),
    }))
  }

  function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    onSubmit(stand.standId, sanitizeCommunityStandDraft(draft))
  }

  return (
    <Card className="rounded-[18px] py-5">
      <form id={formID} className="grid gap-3" onSubmit={handleSubmit}>
        <CardHeader className="gap-3 px-5">
          <div className="flex items-start justify-between gap-3">
            <div className="min-w-0">
              <CardTitle className="truncate text-lg font-black">
                {draft.name || stand.name || stand.standId}
              </CardTitle>
              <CardDescription>
                ID {stand.standId} / 更新 {formatDateTime(stand.updatedAt)}
              </CardDescription>
            </div>
            <Badge variant={draft.enabled ? "default" : "secondary"}>
              {draft.enabled ? "啟用" : "停用"}
            </Badge>
          </div>
        </CardHeader>

        <CardContent className="grid gap-4 px-5">
          <div className="bg-surface-raised border-border grid grid-cols-[minmax(0,1fr)_auto] items-center gap-3 rounded-[18px] border-2 p-3">
            <div className="grid grid-cols-2 gap-3">
              <MiniStat label="拜訪" value={stand.visitCount} />
              <MiniStat label="領取" value={stand.claimCount} />
            </div>
            <Switch
              checked={draft.enabled}
              onCheckedChange={(checked) => patchDraft({ enabled: checked })}
              aria-label={`${stand.name} 啟用狀態`}
            />
          </div>

          <div className="grid gap-3">
            <Field>
              <Label htmlFor={`${formID}-name`}>名稱</Label>
              <Input
                id={`${formID}-name`}
                value={draft.name}
                maxLength={96}
                onChange={(event) => patchDraft({ name: event.target.value })}
              />
            </Field>
            <Field>
              <Label htmlFor={`${formID}-logo`}>Logo URL</Label>
              <Input
                id={`${formID}-logo`}
                value={draft.logoUrl ?? ""}
                placeholder="/game-icons/features/team.png"
                onChange={(event) =>
                  patchDraft({ logoUrl: event.target.value })
                }
              />
            </Field>
            <Field>
              <Label htmlFor={`${formID}-description`}>介紹</Label>
              <Textarea
                id={`${formID}-description`}
                value={draft.description}
                maxLength={1000}
                onChange={(event) =>
                  patchDraft({ description: event.target.value })
                }
              />
            </Field>
            <Field>
              <Label htmlFor={`${formID}-website`}>網站 URL</Label>
              <Input
                id={`${formID}-website`}
                value={draft.websiteUrl ?? ""}
                placeholder="https://sitcon.org"
                onChange={(event) =>
                  patchDraft({ websiteUrl: event.target.value })
                }
              />
            </Field>
          </div>

          <div className="bg-surface-raised border-border grid gap-3 rounded-[18px] border-2 p-3">
            <div className="grid gap-3">
              <Field>
                <Label htmlFor={`${formID}-reward-kind`}>獎勵</Label>
                <Select
                  value={draft.reward.kind}
                  onValueChange={(value) =>
                    handleRewardKindChange(value as StaffRewardKind)
                  }
                >
                  <SelectTrigger
                    id={`${formID}-reward-kind`}
                    className="w-full"
                  >
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="item">道具</SelectItem>
                    <SelectItem value="sitone">小石</SelectItem>
                    <SelectItem value="open_power">開源力</SelectItem>
                  </SelectContent>
                </Select>
              </Field>

              {draft.reward.kind === "open_power" ? (
                <Field>
                  <Label htmlFor={`${formID}-reward-amount`}>開源力數量</Label>
                  <Input
                    id={`${formID}-reward-amount`}
                    type="number"
                    min={1}
                    max={100000}
                    value={draft.reward.amount ?? 0}
                    onChange={(event) =>
                      patchReward({ amount: Number(event.target.value) })
                    }
                  />
                </Field>
              ) : (
                <>
                  <Field>
                    <Label htmlFor={`${formID}-reward-ref`}>Content ID</Label>
                    <Input
                      id={`${formID}-reward-ref`}
                      value={draft.reward.refId ?? ""}
                      placeholder={
                        draft.reward.kind === "item"
                          ? "item_booth_sticker"
                          : "stone_booth"
                      }
                      onChange={(event) =>
                        patchReward({ refId: event.target.value })
                      }
                    />
                  </Field>
                  <Field>
                    <Label htmlFor={`${formID}-reward-quantity`}>數量</Label>
                    <Input
                      id={`${formID}-reward-quantity`}
                      type="number"
                      min={1}
                      max={1000}
                      value={draft.reward.quantity ?? 1}
                      onChange={(event) =>
                        patchReward({ quantity: Number(event.target.value) })
                      }
                    />
                  </Field>
                </>
              )}
            </div>

            <div className="text-muted-foreground flex flex-wrap items-center gap-2 text-xs font-bold">
              <span>{communityStandRewardKindLabels[stand.reward.kind]}</span>
              <span>{stand.reward.name}</span>
              {stand.reward.refId ? <code>{stand.reward.refId}</code> : null}
            </div>
          </div>
        </CardContent>

        <CardFooter className="flex-wrap justify-end gap-2 px-5">
          <AlertDialog>
            <AlertDialogTrigger asChild>
              <Button
                type="button"
                variant="destructive"
                disabled={isPending || isDeleting}
              >
                <Trash2 />
                {isDeleting ? "刪除中" : "刪除"}
              </Button>
            </AlertDialogTrigger>
            <AlertDialogContent size="sm">
              <AlertDialogHeader>
                <AlertDialogTitle>刪除社群攤位</AlertDialogTitle>
                <AlertDialogDescription>
                  刪除 {stand.name} 後，該攤位網址、拜訪紀錄與領獎紀錄都會移除。
                </AlertDialogDescription>
              </AlertDialogHeader>
              <AlertDialogFooter>
                <AlertDialogCancel type="button">取消</AlertDialogCancel>
                <AlertDialogAction
                  type="button"
                  variant="destructive"
                  disabled={isDeleting}
                  onClick={() => onDelete(stand.standId)}
                >
                  刪除攤位
                </AlertDialogAction>
              </AlertDialogFooter>
            </AlertDialogContent>
          </AlertDialog>
          <Button
            type="button"
            variant="outline"
            disabled={isPending || !dirty}
            onClick={() => setDraft(savedDraft)}
          >
            <X />
            還原
          </Button>
          <Button type="submit" disabled={isPending}>
            <Save />
            {isPending ? "儲存中" : "儲存攤位"}
          </Button>
        </CardFooter>
      </form>
    </Card>
  )
}

function sortCommunityStands(stands: AdminCommunityStand[]) {
  return [...stands].sort((left, right) => {
    const byName = left.name.localeCompare(right.name, "zh-TW")
    return byName || left.standId.localeCompare(right.standId, "zh-TW")
  })
}

function newCommunityStandDraft(): AdminCommunityStandCreateInput {
  return {
    name: "",
    description: "",
    logoUrl: "",
    websiteUrl: "",
    enabled: true,
    reward: {
      kind: "item",
      refId: "",
      quantity: 1,
    },
  }
}

function sanitizeCommunityStandCreateDraft(
  draft: AdminCommunityStandCreateInput,
): AdminCommunityStandCreateInput {
  return sanitizeCommunityStandDraft(draft)
}

function communityStandToDraft(
  stand: AdminCommunityStand,
): AdminCommunityStandUpdateInput {
  return {
    name: stand.name,
    description: stand.description,
    logoUrl: stand.logoUrl ?? "",
    websiteUrl: stand.websiteUrl ?? "",
    enabled: stand.enabled,
    reward: {
      kind: stand.reward.kind,
      refId: stand.reward.refId ?? "",
      quantity: stand.reward.quantity ?? 1,
      amount: stand.reward.amount ?? 50,
    },
  }
}

function communityStandRewardDraftForKind(
  kind: StaffRewardKind,
  current: AdminCommunityStandUpdateInput["reward"],
): AdminCommunityStandUpdateInput["reward"] {
  if (kind === "open_power") {
    return {
      kind,
      amount: positiveInteger(current.amount, 50),
    }
  }
  return {
    kind,
    refId: current.kind === kind ? current.refId : "",
    quantity: positiveInteger(current.quantity, 1),
  }
}

function sanitizeCommunityStandDraft(
  draft: AdminCommunityStandUpdateInput,
): AdminCommunityStandUpdateInput {
  const reward =
    draft.reward.kind === "open_power"
      ? {
          kind: draft.reward.kind,
          amount: integerOrZero(draft.reward.amount),
        }
      : {
          kind: draft.reward.kind,
          refId: emptyToUndefined(draft.reward.refId),
          quantity: integerOrZero(draft.reward.quantity),
        }
  return {
    name: draft.name.trim(),
    description: draft.description.trim(),
    logoUrl: emptyToUndefined(draft.logoUrl),
    websiteUrl: emptyToUndefined(draft.websiteUrl),
    enabled: draft.enabled,
    reward,
  }
}

function emptyToUndefined(value?: string) {
  const trimmed = value?.trim() ?? ""
  return trimmed || undefined
}

function positiveInteger(value: unknown, fallback: number) {
  const number = Number(value)
  if (!Number.isFinite(number) || number <= 0) return fallback
  return Math.floor(number)
}

function integerOrZero(value: unknown) {
  const number = Number(value)
  if (!Number.isFinite(number)) return 0
  return Math.floor(number)
}

function battleOpeningOverrideLabel(
  value: AdminSettings["battleOpeningOverride"],
) {
  switch (value) {
    case "force_open":
      return "強制開放"
    case "force_closed":
      return "強制禁止"
    case "schedule":
      return "依排程"
  }
}

function maintenanceModeLabel(value: AdminSettings["maintenanceMode"]) {
  switch (value) {
    case "draining":
      return "等待對戰結束"
    case "active":
      return "維護中"
    case "off":
      return "關閉"
  }
}

function classTimeBattleLockSessions(settings: AdminSettings) {
  if (settings.classTimeBattleLockSessions.length > 0) {
    return settings.classTimeBattleLockSessions
  }
  return [
    {
      start: settings.classTimeBattleLockStart,
      end: settings.classTimeBattleLockEnd,
    },
  ]
}

function classTimeBattleLockSessionSummary(settings: AdminSettings) {
  const sessions = classTimeBattleLockSessions(settings)
  if (sessions.length === 0) return "未設定時段"
  if (sessions.length === 1) {
    return `${sessions[0].start} - ${sessions[0].end}`
  }
  return `${sessions[0].start} - ${sessions[0].end} 等 ${sessions.length} 個時段`
}

function classTimeBattleLockSessionPatch(
  sessions: AdminSettings["classTimeBattleLockSessions"],
): Pick<
  AdminSettings,
  | "classTimeBattleLockSessions"
  | "classTimeBattleLockStart"
  | "classTimeBattleLockEnd"
> {
  const normalized =
    sessions.length > 0 ? sessions : [{ start: "09:00", end: "17:00" }]
  return {
    classTimeBattleLockSessions: normalized,
    classTimeBattleLockStart: normalized[0].start,
    classTimeBattleLockEnd: normalized[0].end,
  }
}

function AdminClassTimeBattleLockPanel({
  settings,
  isPending,
  onSubmit,
  onUpdate,
}: {
  settings: AdminSettings
  isPending: boolean
  onSubmit: (event: FormEvent<HTMLFormElement>) => void
  onUpdate: (patch: Partial<AdminSettings>) => void
}) {
  const overrideOptions: Array<{
    value: AdminSettings["battleOpeningOverride"]
    label: string
    icon: ReactNode
  }> = [
    { value: "schedule", label: "依排程", icon: <Clock className="size-4" /> },
    {
      value: "force_open",
      label: "強制開放",
      icon: <CheckCircle2 className="size-4" />,
    },
    {
      value: "force_closed",
      label: "強制禁止",
      icon: <X className="size-4" />,
    },
  ]
  const sessions = classTimeBattleLockSessions(settings)

  function updateSession(
    index: number,
    patch: Partial<AdminSettings["classTimeBattleLockSessions"][number]>,
  ) {
    onUpdate(
      classTimeBattleLockSessionPatch(
        sessions.map((session, sessionIndex) =>
          sessionIndex === index ? { ...session, ...patch } : session,
        ),
      ),
    )
  }

  function addSession() {
    if (sessions.length >= maxClassTimeBattleLockSessions) return
    const previous = sessions[sessions.length - 1]
    onUpdate(
      classTimeBattleLockSessionPatch([
        ...sessions,
        {
          start: previous?.start ?? "09:00",
          end: previous?.end ?? "10:00",
        },
      ]),
    )
  }

  function removeSession(index: number) {
    if (sessions.length <= 1) return
    onUpdate(
      classTimeBattleLockSessionPatch(
        sessions.filter((_session, sessionIndex) => sessionIndex !== index),
      ),
    )
  }

  return (
    <Card className="rounded-[18px] py-5">
      <form className="grid gap-3" onSubmit={onSubmit}>
        <CardHeader className="px-5">
          <CardTitle className="flex items-center gap-2 text-lg font-black">
            <Clock className="size-5" />
            上課時間開局限制
          </CardTitle>
          <CardDescription>
            上課時間內禁止建立新房間，也會擋下 waiting room 進入開局。
          </CardDescription>
          <CardAction>
            <Badge
              variant={settings.battleOpeningLocked ? "destructive" : "outline"}
            >
              {settings.battleOpeningLocked ? "禁止開局中" : "可以開局"}
            </Badge>
          </CardAction>
        </CardHeader>
        <CardContent className="grid gap-4 px-5 lg:grid-cols-[minmax(0,1fr)_minmax(320px,0.8fr)]">
          <div className="grid gap-3">
            <div className="bg-surface-raised border-border flex flex-wrap items-center justify-between gap-3 rounded-[18px] border-2 p-3">
              <div className="grid gap-1">
                <Label htmlFor="class-time-battle-lock-enabled">
                  啟用上課時間禁止開局
                </Label>
                <span className="text-muted-foreground text-xs font-semibold">
                  每日以 UTC+8 零時切換；
                  {classTimeBattleLockSessionSummary(settings)}
                </span>
              </div>
              <Switch
                id="class-time-battle-lock-enabled"
                checked={settings.classTimeBattleLockEnabled}
                onCheckedChange={(checked) =>
                  onUpdate({ classTimeBattleLockEnabled: checked })
                }
              />
            </div>

            <div className="grid gap-2">
              {sessions.map((session, index) => (
                <div
                  key={index}
                  className="bg-surface-raised border-border grid gap-3 rounded-[18px] border-2 p-3 sm:grid-cols-[minmax(0,1fr)_minmax(0,1fr)_auto] sm:items-end"
                >
                  <Field>
                    <Label htmlFor={`class-time-battle-lock-${index}-start`}>
                      第 {index + 1} 堂開始
                    </Label>
                    <Input
                      id={`class-time-battle-lock-${index}-start`}
                      type="time"
                      step={60}
                      value={session.start}
                      onChange={(event) =>
                        updateSession(index, { start: event.target.value })
                      }
                    />
                  </Field>
                  <Field>
                    <Label htmlFor={`class-time-battle-lock-${index}-end`}>
                      第 {index + 1} 堂結束
                    </Label>
                    <Input
                      id={`class-time-battle-lock-${index}-end`}
                      type="time"
                      step={60}
                      value={session.end}
                      onChange={(event) =>
                        updateSession(index, { end: event.target.value })
                      }
                    />
                  </Field>
                  <Button
                    type="button"
                    variant="outline"
                    size="icon"
                    aria-label={`移除第 ${index + 1} 堂`}
                    disabled={sessions.length <= 1}
                    onClick={() => removeSession(index)}
                  >
                    <Trash2 className="size-4" />
                  </Button>
                </div>
              ))}
              <Button
                type="button"
                variant="outline"
                disabled={sessions.length >= maxClassTimeBattleLockSessions}
                onClick={addSession}
              >
                <Plus className="size-4" />
                新增時段
              </Button>
            </div>
          </div>

          <div className="bg-surface-raised border-border grid gap-3 rounded-[18px] border-2 p-3">
            <div className="flex flex-wrap items-center justify-between gap-2">
              <div>
                <h3 className="font-black">臨時覆寫</h3>
                <p className="text-muted-foreground text-xs font-semibold">
                  目前：
                  {battleOpeningOverrideLabel(settings.battleOpeningOverride)}
                </p>
              </div>
              <Badge variant="secondary">
                {settings.battleOpeningLocked ? "LOCKED" : "OPEN"}
              </Badge>
            </div>
            <div className="grid grid-cols-1 gap-2 sm:grid-cols-3 lg:grid-cols-1 xl:grid-cols-3">
              {overrideOptions.map((option) => (
                <Button
                  key={option.value}
                  type="button"
                  variant={
                    settings.battleOpeningOverride === option.value
                      ? "default"
                      : "outline"
                  }
                  className="justify-start"
                  onClick={() =>
                    onUpdate({ battleOpeningOverride: option.value })
                  }
                >
                  {option.icon}
                  {option.label}
                </Button>
              ))}
            </div>
          </div>
        </CardContent>
        <CardFooter className="justify-end px-5">
          <Button type="submit" disabled={isPending}>
            <Save />
            {isPending ? "儲存中" : "儲存限制"}
          </Button>
        </CardFooter>
      </form>
    </Card>
  )
}

function AdminSettingsPanel({
  settings,
  isPending,
  onSubmit,
  onUpdate,
}: {
  settings: AdminSettings
  isPending: boolean
  onSubmit: (event: FormEvent<HTMLFormElement>) => void
  onUpdate: (patch: Partial<AdminSettings>) => void
}) {
  const maintenanceOptions: Array<{
    value: AdminSettings["maintenanceMode"]
    label: string
    description: string
    icon: ReactNode
  }> = [
    {
      value: "off",
      label: "關閉",
      description: "恢復玩家操作",
      icon: <CheckCircle2 className="size-4" />,
    },
    {
      value: "draining",
      label: "等待對戰結束",
      description: "公告並停止新場次",
      icon: <Clock className="size-4" />,
    },
    {
      value: "active",
      label: "維護中",
      description: "阻擋玩家寫入操作",
      icon: <Megaphone className="size-4" />,
    },
  ]

  return (
    <Card className="rounded-[18px] py-5">
      <form className="grid gap-3" onSubmit={onSubmit}>
        <CardHeader className="px-5">
          <CardTitle className="flex items-center gap-2 text-lg font-black">
            <Settings className="size-5" />
            管理設定
          </CardTitle>
          <CardDescription>控制知識王房間與電腦對戰開關。</CardDescription>
          <CardAction>
            <Badge variant="outline">Admin only</Badge>
          </CardAction>
        </CardHeader>
        <CardContent className="grid gap-4 px-5 xl:grid-cols-[minmax(0,360px)_minmax(0,360px)_minmax(0,1fr)]">
          <div className="bg-surface-raised border-border flex items-center justify-between gap-3 rounded-[18px] border-2 p-3">
            <Label htmlFor="computer-battles-enabled">開放電腦對戰</Label>
            <Switch
              id="computer-battles-enabled"
              checked={settings.computerBattlesEnabled}
              onCheckedChange={(checked) =>
                onUpdate({ computerBattlesEnabled: checked })
              }
            />
          </div>

          <div className="bg-surface-raised border-border flex items-center justify-between gap-3 rounded-[18px] border-2 p-3">
            <Label htmlFor="same-team-battles-enabled">
              允許同隊知識王對戰
            </Label>
            <Switch
              id="same-team-battles-enabled"
              checked={settings.sameTeamBattlesEnabled}
              onCheckedChange={(checked) =>
                onUpdate({ sameTeamBattlesEnabled: checked })
              }
            />
          </div>

          <div className="grid gap-3 md:grid-cols-3">
            <Field>
              <Label htmlFor="computer-easy">Easy 答對率</Label>
              <Input
                id="computer-easy"
                type="number"
                min={0}
                max={100}
                value={settings.computerEasyAccuracy}
                onChange={(event) =>
                  onUpdate({
                    computerEasyAccuracy: clampPercent(
                      Number(event.target.value),
                    ),
                  })
                }
              />
            </Field>
            <Field>
              <Label htmlFor="computer-normal">Normal 答對率</Label>
              <Input
                id="computer-normal"
                type="number"
                min={0}
                max={100}
                value={settings.computerNormalAccuracy}
                onChange={(event) =>
                  onUpdate({
                    computerNormalAccuracy: clampPercent(
                      Number(event.target.value),
                    ),
                  })
                }
              />
            </Field>
            <Field>
              <Label htmlFor="computer-hard">Hard 答對率</Label>
              <Input
                id="computer-hard"
                type="number"
                min={0}
                max={100}
                value={settings.computerHardAccuracy}
                onChange={(event) =>
                  onUpdate({
                    computerHardAccuracy: clampPercent(
                      Number(event.target.value),
                    ),
                  })
                }
              />
            </Field>
          </div>
          <div className="bg-surface-raised border-border grid gap-3 rounded-[18px] border-2 p-3 xl:col-span-3">
            <div className="flex flex-wrap items-center justify-between gap-2">
              <div>
                <h3 className="font-black">維護公告</h3>
                <p className="text-muted-foreground text-xs font-semibold">
                  目前：{maintenanceModeLabel(settings.maintenanceMode)}
                </p>
              </div>
              <Badge
                variant={
                  settings.maintenanceActive ? "destructive" : "secondary"
                }
              >
                {settings.maintenanceActive ? "公告中" : "未啟用"}
              </Badge>
            </div>
            <div className="grid gap-2 md:grid-cols-3">
              {maintenanceOptions.map((option) => (
                <Button
                  key={option.value}
                  type="button"
                  variant={
                    settings.maintenanceMode === option.value
                      ? "default"
                      : "outline"
                  }
                  className="h-auto justify-start py-3 text-left"
                  onClick={() => onUpdate({ maintenanceMode: option.value })}
                >
                  {option.icon}
                  <span className="grid gap-0.5">
                    <span>{option.label}</span>
                    <span className="text-xs font-semibold opacity-75">
                      {option.description}
                    </span>
                  </span>
                </Button>
              ))}
            </div>
            <Field>
              <Label htmlFor="maintenance-message">公告內容</Label>
              <Textarea
                id="maintenance-message"
                value={settings.maintenanceMessage}
                onChange={(event) =>
                  onUpdate({ maintenanceMessage: event.target.value })
                }
                placeholder="系統即將進行更新，請完成目前對戰並暫停新的操作。"
                rows={3}
              />
            </Field>
          </div>
        </CardContent>
        <CardFooter className="justify-end px-5">
          <Button type="submit" disabled={isPending}>
            <Save />
            {isPending ? "儲存中" : "儲存設定"}
          </Button>
        </CardFooter>
      </form>
    </Card>
  )
}

function average(total: number, count: number) {
  if (count <= 0) return "0"
  return (total / count).toFixed(1)
}
