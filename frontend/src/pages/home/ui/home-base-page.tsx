import { useQuery } from "@tanstack/react-query"
import { Link, useNavigate } from "@tanstack/react-router"
import { useEffect, useMemo } from "react"
import { ChevronRight, Lock, LogOut } from "lucide-react"
import { AppError } from "@/shared/api/error"
import { gameApi } from "@/shared/api/game"
import { GamePageShell } from "@/shared/ui/game-page-shell"
import { Button } from "@/shared/ui/button"
import { apiClient } from "@/shared/api/client"
import { PlayerAvatar } from "@/shared/ui/player-avatar"
import {
  GameFeatureIcon,
  type GameFeatureIconName,
} from "@/shared/ui/game-feature-icon"

type CollectionEntry = {
  label: string
  count: (input: {
    sitones: number
    items: number
    openPower: number
    rank?: number
  }) => string
  icon: GameFeatureIconName
  to: string
}

const STAFF_COLLECTIONS: CollectionEntry[] = [
  {
    label: "關主發放台",
    count: () => "掃描通行證發獎",
    icon: "pass",
    to: "/rewards",
  },
  {
    label: "宿舍管理員",
    count: () => "房號與成員管理",
    icon: "dorm",
    to: "/dorms",
  },
]

const COLLECTIONS: CollectionEntry[] = [
  {
    label: "開源力轉帳",
    count: ({ openPower }) => `可用 ${openPower} OP`,
    icon: "team",
    to: "/open-power-transfer",
  },
  {
    label: "成就圖鑑",
    count: () => "收藏里程碑",
    icon: "codex",
    to: "/achievements",
  },
  {
    label: "小石圖鑑",
    count: ({ sitones }) => `已收 ${sitones} 顆`,
    icon: "stones",
    to: "/stones",
  },
  {
    label: "戰利品背包",
    count: ({ items }) => `持有 ${items} 件`,
    icon: "backpack",
    to: "/inventory",
  },
  {
    label: "鍛造工坊",
    count: () => "合成開放",
    icon: "forge",
    to: "/stones/fusion",
  },
  {
    label: "公會排行",
    count: ({ rank }) => (rank ? `第 ${rank} 名` : "未入榜"),
    icon: "leaderboard",
    to: "/leaderboard",
  },
  {
    label: "戰鬥回放",
    count: () => "最近戰局",
    icon: "history",
    to: "/battle/history",
  },
  {
    label: "全服圖鑑",
    count: () => "偵查資料",
    icon: "codex",
    to: "/codex",
  },
]

export function HomeBasePage() {
  const navigate = useNavigate()
  const { data, isPending, error } = useQuery({
    queryKey: ["me", "home"],
    queryFn: gameApi.home,
  })
  const unauthorized = error instanceof AppError && error.status === 401

  useEffect(() => {
    if (unauthorized) {
      navigate({ to: "/login", replace: true })
    }
  }, [navigate, unauthorized])

  const player = data?.player
  const summary = data?.summary
  const teamRank = data?.teamRank
  const displayName = player?.nickname ?? "載入中"
  const teamName =
    player?.team?.name ??
    (player?.role === "staff" ? "關主模式" : "讀取小隊資料")
  const openPower = summary?.openPower ?? player?.openPower ?? 0
  const sitoneCount = summary?.sitoneCount ?? 0
  const itemCount = summary?.itemCount ?? 0
  const rank = teamRank?.rank
  const teamMembers = useMemo(
    () => player?.teamMembers ?? [],
    [player?.teamMembers],
  )
  if (unauthorized) return null

  const actionEnabledByID = new Map(
    (data?.actions ?? []).map((action) => [action.id, action.enabled]),
  )
  const battleEnabled = actionEnabledByID.get("battle") ?? true
  const battleLocked = !battleEnabled
  const frontEnabled = actionEnabledByID.get("front") ?? true
  const isStaff = player?.role === "staff"
  const collections = isStaff
    ? [...STAFF_COLLECTIONS, ...COLLECTIONS]
    : COLLECTIONS
  const logoutAction = async () => {
    await apiClient.post("/api/auth/logout")
    navigate({ to: "/login", replace: true })
  }

  return (
    <GamePageShell ariaLabel="冒險基地首頁" contentClassName="gap-3">
      <section className="grid content-start gap-3">
        <header
          className="bg-card border-ink grid grid-cols-[64px_1fr_auto] items-center gap-3 rounded-[26px] border-2 p-3.5"
          style={{ boxShadow: "4px 4px 0 rgba(23,35,58,.14)" }}
          aria-label="冒險者狀態"
        >
          <PlayerAvatar
            playerId={player?.playerId}
            nickname={displayName}
            avatarUrl={player?.avatarUrl}
            size="lg"
            className="bg-pebble-spark border-ink size-16 rounded-[22px] border-2 text-[26px]"
          />
          <div>
            <p className="text-muted-foreground mb-1 text-xs font-black tracking-[0.08em] uppercase">
              Player Card
            </p>
            <h1 className="text-[29px] leading-none font-black tracking-normal">
              {displayName}
            </h1>
            <span className="text-muted-foreground font-bold">
              {isPending ? "召回冒險者資料中" : teamName}
            </span>
          </div>
          <div className="flex gap-x-3">
            <div
              className="bg-ink text-primary-foreground min-w-[86px] rounded-[18px] border-2 border-transparent px-[9px] py-2 text-center"
              aria-label={`開源力 ${openPower}`}
            >
              <span className="text-primary-foreground/70 block text-[11px] font-black">
                開源力
              </span>
              <strong className="text-[23px] font-black">{openPower}</strong>
            </div>
            <Button
              type="button"
              size="icon"
              variant="destructive"
              aria-label="離開基地"
              onClick={() => void logoutAction()}
            >
              <LogOut aria-hidden />
            </Button>
          </div>
        </header>

        <section
          className="bg-ink text-primary-foreground relative overflow-hidden rounded-[24px] border-2 border-transparent p-[18px]"
          style={{ boxShadow: "5px 5px 0 rgba(23,35,58,.16)" }}
          aria-label="主線任務"
        >
          <div
            className="absolute top-3 right-3 grid size-14 place-items-center"
            aria-hidden
          >
            <GameFeatureIcon name="leaderboard" className="size-14" />
          </div>
          <div
            className="border-primary/35 pointer-events-none absolute inset-x-4 bottom-4 h-3 border-x-2 border-b-2"
            aria-hidden
          />
          <p className="text-primary-foreground/75 mb-1 text-xs font-black tracking-[0.08em] uppercase">
            Main Quest
          </p>
          <h2 className="mb-2 max-w-[280px] text-[31px] leading-[1.08] font-black tracking-normal">
            {battleLocked ? "知識王試煉暫停" : "知識王試煉開放"}
          </h2>
          <p className="text-primary-foreground/75 mb-4 max-w-[300px] leading-[1.65]">
            {battleLocked
              ? "目前禁止開局，請等待工作人員重新開放戰鬥入口。"
              : "集結隊友、掃描戰碼，進入答題競技場奪取開源力。"}
          </p>
          {battleLocked ? (
            <span
              aria-disabled
              className="border-ink bg-card text-muted-foreground relative z-10 flex min-h-[50px] w-full cursor-not-allowed items-center justify-center gap-2 rounded-[14px] border-2 text-base font-black no-underline opacity-80"
              style={{ boxShadow: "3px 3px 0 rgba(0,0,0,.18)" }}
            >
              <Lock className="size-5" aria-hidden />
              禁止進入
            </span>
          ) : (
            <Link
              to="/battle"
              className="border-ink bg-primary text-primary-foreground focus-visible:outline-power relative z-10 flex min-h-[50px] w-full items-center justify-center gap-2 rounded-[14px] border-2 text-base font-black no-underline transition-transform focus-visible:outline-3 focus-visible:outline-offset-2 active:translate-y-px"
              style={{ boxShadow: "3px 3px 0 rgba(0,0,0,.18)" }}
            >
              <GameFeatureIcon name="battle" className="size-7" />
              進入競技場
              <ChevronRight className="size-5" aria-hidden />
            </Link>
          )}
        </section>

        <aside
          className="bg-surface-raised border-ink relative overflow-hidden rounded-[22px] border-2 p-4"
          style={{ boxShadow: "4px 4px 0 rgba(23,35,58,.14)" }}
          aria-label="開源戰線提示"
        >
          <div
            className="bg-primary/15 pointer-events-none absolute inset-y-0 left-0 w-2"
            aria-hidden
          />
          <div className="grid grid-cols-[58px_1fr_auto] items-center gap-3">
            <div
              className="grid size-[58px] -rotate-[4deg] place-items-center"
              aria-hidden
            >
              <GameFeatureIcon name="front" className="size-[58px]" />
            </div>
            <div className="min-w-0">
              <p className="text-primary mb-0.5 text-[11px] font-black tracking-[0.08em] uppercase">
                Team Mission
              </p>
              <h2 className="text-[20px] leading-tight font-black tracking-normal">
                開源戰線
              </h2>
              <p className="text-muted-foreground mt-1 text-[13px] leading-[1.45] font-bold">
                使用開源力，和小隊爭奪領土
              </p>
            </div>
            {frontEnabled ? (
              <Link
                to="/front"
                className="bg-card border-ink focus-visible:outline-power flex min-h-[44px] min-w-[84px] items-center justify-center gap-1 rounded-[14px] border-2 px-3 text-sm font-black no-underline transition-transform focus-visible:outline-3 focus-visible:outline-offset-2 active:translate-y-px"
                style={{ boxShadow: "2px 2px 0 rgba(23,35,58,.14)" }}
              >
                出發
                <ChevronRight className="size-4" aria-hidden />
              </Link>
            ) : (
              <span
                aria-disabled
                className="bg-card border-ink text-muted-foreground flex min-h-[44px] min-w-[84px] cursor-not-allowed items-center justify-center gap-1 rounded-[14px] border-2 px-3 text-sm font-black opacity-80"
                style={{ boxShadow: "2px 2px 0 rgba(23,35,58,.14)" }}
              >
                <Lock className="size-4" aria-hidden />
                禁止
              </span>
            )}
          </div>
        </aside>

        <section className="grid grid-cols-3 gap-[9px]" aria-label="戰力摘要">
          {(
            [
              {
                label: "小石隊伍",
                value: sitoneCount,
                to: "/stones",
                icon: "stones",
              },
              {
                label: "戰利品",
                value: itemCount,
                to: "/inventory",
                icon: "backpack",
              },
              {
                label: "名次",
                value: rank ? `#${rank}` : "-",
                to: "/leaderboard",
                icon: "leaderboard",
              },
            ] as const
          ).map(({ label, value, to, icon }) => (
            <Link
              key={label}
              to={to}
              aria-label={`查看${label}`}
              className="bg-surface-raised border-border focus-visible:outline-power grid min-h-[84px] justify-items-center rounded-[16px] border-2 px-2 py-3 text-center text-inherit no-underline transition-transform focus-visible:outline-3 focus-visible:outline-offset-2 active:translate-y-px"
            >
              <GameFeatureIcon name={icon} className="mb-1 size-10" />
              <span className="text-muted-foreground block text-xs font-black">
                {label}
              </span>
              <strong className="text-[24px] font-black">{value}</strong>
            </Link>
          ))}
        </section>

        <section
          className="bg-card border-ink rounded-[22px] border-2 p-[15px]"
          aria-label="小隊成員"
        >
          <div className="mb-3 flex items-start justify-between gap-3">
            <div>
              <p className="text-muted-foreground mb-1 text-xs font-black tracking-[0.08em] uppercase">
                Party
              </p>
              <h2 className="text-[22px] font-black tracking-normal">
                小隊名冊
              </h2>
            </div>
            <span className="bg-surface-raised border-border rounded-full border-2 px-2.5 py-1 text-xs font-black whitespace-nowrap">
              {isPending ? "-" : `${teamMembers.length} 人`}
            </span>
          </div>

          {isPending ? (
            <div className="grid gap-[8px]">
              {[0, 1, 2].map((item) => (
                <div
                  key={item}
                  className="bg-surface-raised border-border grid min-h-[56px] grid-cols-[40px_1fr] items-center gap-3 rounded-[17px] border-2 px-3"
                >
                  <span className="bg-muted border-border size-10 rounded-full border-2" />
                  <span className="bg-muted h-4 w-28 rounded-full" />
                </div>
              ))}
            </div>
          ) : teamMembers.length > 0 ? (
            <ul className="grid gap-[8px]">
              {teamMembers.map((member) => {
                const current = member.playerId === player?.playerId
                return (
                  <li
                    key={member.playerId}
                    className="bg-surface-raised border-border grid min-h-[56px] grid-cols-[40px_1fr_auto] items-center gap-3 rounded-[17px] border-2 px-3 py-2"
                  >
                    <PlayerAvatar
                      playerId={member.playerId}
                      nickname={member.nickname}
                      avatarUrl={member.avatarUrl}
                      size="lg"
                      className="border-ink bg-pebble-resonate border-2"
                    />
                    <div className="min-w-0">
                      <strong className="block truncate text-[16px] font-black">
                        {member.nickname}
                      </strong>
                      {member.role === "staff" ? (
                        <small className="text-muted-foreground block text-xs font-bold">
                          關主
                        </small>
                      ) : null}
                    </div>
                    {current ? (
                      <span className="bg-secondary text-secondary-foreground border-ink rounded-full border-2 px-2 py-0.5 text-xs font-black whitespace-nowrap">
                        你
                      </span>
                    ) : null}
                  </li>
                )
              })}
            </ul>
          ) : (
            <p className="text-muted-foreground bg-surface-raised border-border rounded-[17px] border-2 px-3 py-3 text-sm font-bold">
              尚未取得小隊名冊
            </p>
          )}
        </section>

        <section
          className="bg-card border-ink rounded-[22px] border-2 p-[15px]"
          aria-label="冒險手冊"
        >
          <p className="text-muted-foreground mb-1 text-xs font-black tracking-[0.08em] uppercase">
            Adventurer Log
          </p>
          <h2 className="mb-3 text-[22px] font-black tracking-normal">
            {isStaff ? "管理、圖鑑與戰績" : "圖鑑、背包、鍛造與戰績"}
          </h2>
          <div className="grid grid-cols-2 gap-[9px]">
            {collections.map((item) => {
              return (
                <Link
                  key={item.label}
                  to={item.to}
                  className="bg-surface-raised border-ink grid min-h-[72px] grid-cols-[62px_1fr] items-center gap-[8px] rounded-[15px] border-2 px-[10px] py-[10px] text-inherit no-underline transition-transform active:translate-y-px"
                >
                  <span
                    className="row-span-2 grid size-[58px] place-items-center"
                    aria-hidden
                  >
                    <GameFeatureIcon name={item.icon} className="size-[58px]" />
                  </span>
                  <strong className="block font-black">{item.label}</strong>
                  <small className="text-muted-foreground block text-xs font-bold">
                    {item.count({
                      sitones: sitoneCount,
                      items: itemCount,
                      openPower,
                      rank,
                    })}
                  </small>
                </Link>
              )
            })}
          </div>
        </section>

        <Button
          asChild
          size="lg"
          variant="secondary"
          className="min-h-[56px] rounded-[16px] text-base"
        >
          <Link to="/gift-history" aria-label="查看禮物接收歷史">
            <GameFeatureIcon name="giftHistory" className="size-7" />
            查看禮物接收歷史
            <ChevronRight className="size-5" aria-hidden />
          </Link>
        </Button>
      </section>
    </GamePageShell>
  )
}
