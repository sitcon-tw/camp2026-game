import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import {
  Activity,
  CheckCircle2,
  Clock,
  ImageIcon,
  LogOut,
  Pencil,
  Percent,
  RefreshCw,
  Save,
  Settings,
  ShieldCheck,
  X,
} from "lucide-react"
import { type FormEvent, type ReactNode, useState } from "react"
import {
  CartesianGrid,
  Line,
  LineChart as RechartsLineChart,
  XAxis,
  YAxis,
} from "recharts"
import { toast } from "sonner"

import { AppError } from "@/shared/api/error"
import {
  gameApi,
  type AdminDashboard,
  type AdminDashboardHistory,
  type AdminDashboardInventoryEntry,
  type AdminDashboardPlayer,
  type AdminDashboardPlayerRank,
  type AdminDashboardTeam,
  type AdminSettings,
} from "@/shared/api/game"
import { Avatar, AvatarFallback, AvatarImage } from "@/shared/ui/avatar"
import { Badge } from "@/shared/ui/badge"
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
  ChartContainer,
  ChartTooltip,
  ChartTooltipContent,
  type ChartConfig,
} from "@/shared/ui/chart"
import { Field } from "@/shared/ui/field"
import { GameFeatureIcon } from "@/shared/ui/game-feature-icon"
import { GamePageShell } from "@/shared/ui/game-page-shell"
import { Input } from "@/shared/ui/input"
import { Label } from "@/shared/ui/label"
import { PageHeader } from "@/shared/ui/page-header"
import { PlayerAvatar } from "@/shared/ui/player-avatar"
import { Progress } from "@/shared/ui/progress"
import { Spinner } from "@/shared/ui/spinner"
import { Switch } from "@/shared/ui/switch"
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

const numberFormatter = new Intl.NumberFormat("zh-TW")
const compactNumberFormatter = new Intl.NumberFormat("zh-TW", {
  notation: "compact",
  maximumFractionDigits: 1,
})
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
      contentClassName="grid max-w-[1440px] content-start gap-y-4 px-5 pb-8 lg:px-8"
    >
      <PageHeader
        title="Admin"
        headline="Game Operations"
        rightSlot={
          <div className="flex items-center gap-2">
            <Button
              type="button"
              variant="outline"
              size="icon"
              aria-label="重新整理 dashboard"
              disabled={dashboardQuery.isFetching || historyQuery.isFetching}
              onClick={() => {
                void dashboardQuery.refetch()
                void historyQuery.refetch()
              }}
            >
              <RefreshCw
                className={cn(
                  "size-4",
                  (dashboardQuery.isFetching || historyQuery.isFetching) &&
                    "animate-spin",
                )}
              />
            </Button>
            <Button
              type="button"
              variant="outline"
              disabled={logoutMutation.isPending}
              onClick={() => logoutMutation.mutate()}
            >
              <LogOut />
              登出
            </Button>
          </div>
        }
      />

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
          historyError={historyQuery.error}
          historyPending={historyQuery.isPending}
          onRetryHistory={() => historyQuery.refetch()}
        />
      ) : null}

      <AdminSettingsPanel
        settings={settings}
        isPending={updateMutation.isPending}
        onSubmit={handleUpdate}
        onUpdate={updateDraft}
      />
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

function AdminDashboardView({
  dashboard,
  history,
  historyError,
  historyPending,
  onRetryHistory,
}: {
  dashboard: AdminDashboard
  history?: AdminDashboardHistory
  historyError: unknown
  historyPending: boolean
  onRetryHistory: () => void
}) {
  const { summary, matches } = dashboard

  return (
    <div className="grid gap-4">
      <section className="grid gap-3 xl:grid-cols-[minmax(0,1fr)_360px]">
        <div className="grid grid-cols-2 gap-3 md:grid-cols-4 xl:grid-cols-6">
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
            icon={<CheckCircle2 className="size-4" />}
            label="答題正確率"
            value={formatPercent(summary.answerAccuracy)}
            detail={`${summary.correctAnswerCount}/${summary.answerCount} 題`}
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

      <ResourceHistoryPanel
        history={history}
        isPending={historyPending}
        error={historyError}
        onRetry={onRetryHistory}
      />

      <MostOwnedPanel inventory={dashboard.inventory} />

      <section className="grid gap-4 xl:grid-cols-[minmax(0,1.15fr)_minmax(420px,0.85fr)]">
        <TopPlayersPanel topPlayers={dashboard.topPlayers} />
        <TeamsPanel teams={dashboard.teams} />
      </section>

      <Tabs defaultValue="players" className="gap-3">
        <div className="flex flex-wrap items-center justify-between gap-3">
          <div>
            <h2 className="text-xl font-black">完整統計</h2>
            <p className="text-muted-foreground text-sm font-semibold">
              玩家、庫存與對戰活動的營運檢視。
            </p>
          </div>
          <TabsList className="grid w-full grid-cols-4 md:w-fit">
            <TabsTrigger value="players">玩家</TabsTrigger>
            <TabsTrigger value="inventory">庫存</TabsTrigger>
            <TabsTrigger value="matches">對戰</TabsTrigger>
            <TabsTrigger value="teams">隊伍</TabsTrigger>
          </TabsList>
        </div>
        <TabsContent value="players">
          <PlayersTable players={dashboard.players} />
        </TabsContent>
        <TabsContent value="inventory">
          <InventoryPanel inventory={dashboard.inventory} />
        </TabsContent>
        <TabsContent value="matches">
          <MatchesPanel matches={dashboard.matches} />
        </TabsContent>
        <TabsContent value="teams">
          <TeamsDetailTable teams={dashboard.teams} />
        </TabsContent>
      </Tabs>
    </div>
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
  return (
    <section className="grid gap-3 lg:grid-cols-2">
      <MostOwnedListCard
        icon={<GameFeatureIcon name="stones" className="size-5" />}
        title="最多拿到的小石"
        emptyLabel="目前沒有小石持有資料"
        entries={inventory.sitones.slice(0, 6)}
        unit="顆"
      />
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
}: {
  topPlayers: AdminDashboard["topPlayers"]
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
        <CardTitle className="flex items-center gap-2 text-lg font-black">
          <GameFeatureIcon name="leaderboard" className="size-5" />
          玩家排行
        </CardTitle>
        <CardDescription>
          從不同維度看目前誰拿最多、誰開源力最高、誰答題表現最好。
        </CardDescription>
      </CardHeader>
      <CardContent className="px-5">
        <Tabs defaultValue="sitones">
          <TabsList className="grid w-full grid-cols-5">
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

function PlayersTable({ players }: { players: AdminDashboardPlayer[] }) {
  return (
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
          </TableRow>
        ))}
      </TableBody>
    </Table>
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
  return (
    <Avatar className={cn("bg-surface-raised border-ink border", className)}>
      {team.avatarUrl ? (
        <AvatarImage
          src={team.avatarUrl}
          alt=""
          aria-hidden="true"
          draggable={false}
          className="block size-full object-cover"
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
  player: Pick<AdminDashboardPlayer, "playerId" | "nickname">
}) {
  return (
    <div className="flex items-center gap-2">
      <PlayerAvatar
        playerId={player.playerId}
        nickname={player.nickname}
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
  return (
    <Card className="rounded-[18px] py-5">
      <form className="grid gap-3" onSubmit={onSubmit}>
        <CardHeader className="px-5">
          <CardTitle className="flex items-center gap-2 text-lg font-black">
            <Settings className="size-5" />
            管理設定
          </CardTitle>
          <CardDescription>控制玩家是否能建立電腦對戰。</CardDescription>
          <CardAction>
            <Badge variant="outline">Admin only</Badge>
          </CardAction>
        </CardHeader>
        <CardContent className="grid gap-4 px-5 lg:grid-cols-[320px_minmax(0,1fr)]">
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
