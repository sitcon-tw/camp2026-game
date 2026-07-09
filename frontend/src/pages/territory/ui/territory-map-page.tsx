import { useQuery } from "@tanstack/react-query"
import { Link } from "@tanstack/react-router"
import { useState } from "react"
import { toast } from "sonner"

import {
  getStoredMissionID,
  storeMissionID,
} from "@/features/territory/model/mission-storage"
import type { TerritoryRegion } from "@/features/territory/model/taiwan-map"
import { ActiveMissionCard } from "@/features/territory/ui/active-mission-card"
import { AttackInitiateDialog } from "@/features/territory/ui/attack-initiate-dialog"
import { BossStatusBanner } from "@/features/territory/ui/boss-status-banner"
import { TerritoryMap } from "@/features/territory/ui/territory-map"
import { AppError } from "@/shared/api/error"
import { gameApi } from "@/shared/api/game"
import {
  territoryApi,
  type TerritoryStandingTeam,
  type TerritoryTier,
} from "@/shared/api/territory"
import { tierMeta } from "@/shared/lib/territory-labels"
import { Badge } from "@/shared/ui/badge"
import { Button } from "@/shared/ui/button"
import { Card } from "@/shared/ui/card"
import { GameIcon } from "@/shared/ui/game-icon"
import { GamePageShell } from "@/shared/ui/game-page-shell"
import { PageHeader } from "@/shared/ui/page-header"

const TIER_LEGEND: TerritoryTier[] = ["leader", "contested", "challenger"]

const TERRITORY_ACTION_LINKS = [
  {
    label: "防守配置",
    to: "/territory/defense",
    iconPath: "/game-icons/items/item_mission_map.png",
  },
  {
    label: "研三舍",
    to: "/yansan",
    iconPath: "/game-icons/items/item_star_village_signpost.png",
  },
  {
    label: "排行榜",
    to: "/leaderboard",
    iconPath: "/game-icons/features/leaderboard.png",
  },
] as const

function territoryLoadErrorMessage(error: unknown) {
  if (error instanceof AppError) {
    if (error.status === 401) return "請先重新登入後再查看戰局。"
    if (error.status === 404) return "戰局 API 尚未就緒，請確認後端已部署最新版。"
    return error.message
  }
  return "戰局資料讀取失敗，請稍後再試。"
}

export function TerritoryMapPage() {
  const [attackTarget, setAttackTarget] =
    useState<TerritoryStandingTeam | null>(null)
  const [missionID, setMissionID] = useState(getStoredMissionID)

  const standingsQuery = useQuery({
    queryKey: ["territory", "standings"],
    queryFn: territoryApi.standings,
    refetchInterval: 30000,
  })
  const canLoadTerritoryDetails = standingsQuery.isSuccess
  const bossQuery = useQuery({
    queryKey: ["yansan", "boss", "status"],
    queryFn: territoryApi.bossStatus,
    enabled: canLoadTerritoryDetails,
    refetchInterval: (query) =>
      query.state.data?.bossStatus === "under_attack" ? 5000 : 20000,
  })
  const defenseQuery = useQuery({
    queryKey: ["territory", "defense"],
    queryFn: territoryApi.defense,
    enabled: canLoadTerritoryDetails,
  })
  const statusQuery = useQuery({
    queryKey: ["me", "status"],
    queryFn: gameApi.status,
    enabled: canLoadTerritoryDetails,
  })
  const catalogQuery = useQuery({
    queryKey: ["catalog", "sitones"],
    queryFn: gameApi.catalogSitones,
    enabled: canLoadTerritoryDetails,
  })

  const standings = standingsQuery.data
  const standingsErrorMessage = standingsQuery.isError
    ? territoryLoadErrorMessage(standingsQuery.error)
    : null
  const serverMissionID = standings?.activeMissionId
  const activeMissionID = missionID || serverMissionID || ""
  const myTeam = standings?.teams.find(
    (team) => team.teamId === standings.myTeamId,
  )
  const canAttack = myTeam?.canAttack ?? false

  const teamNameOf = (teamID: string | undefined) =>
    standings?.teams.find((team) => team.teamId === teamID)?.name ?? "神秘小隊"
  const sitoneNameOf = (sitoneID: string) =>
    catalogQuery.data?.find((sitone) => sitone.id === sitoneID)?.name ??
    sitoneID

  const handleSelectRegion = (
    region: TerritoryRegion,
    team: TerritoryStandingTeam | undefined,
  ) => {
    if (region.isBoss) {
      toast.info("研三舍是 Boss 領地，需全部小隊集結後從研三舍頁面共同進攻。")
      return
    }
    if (!team) {
      toast.info("這塊領地目前沒有小隊駐紮。")
      return
    }
    if (team.teamId === standings?.myTeamId) {
      toast.info(`${team.name} 是你的小隊，前往防守配置佈署留守小石吧。`)
      return
    }
    if (!canAttack) {
      toast.error("你的小隊目前是領先隊，只能防守、無法主動出擊。")
      return
    }
    if (!team.canBeAttacked) {
      toast.error(`${team.name} 目前是追趕隊，受到保護無法被攻擊。`)
      return
    }
    if (activeMissionID) {
      toast.error("已有進行中的攻擊任務，結束後才能再次發起。")
      return
    }
    setAttackTarget(team)
  }

  return (
    <GamePageShell contentClassName="grid content-start gap-y-3">
      <PageHeader title="領地攻防" headline="Territory" backTo="/" />

      {standings && myTeam ? (
        <Card className="bg-ink text-primary-foreground gap-1.5 rounded-[24px] px-5 py-4">
          <div className="flex items-center justify-between gap-2">
            <p className="text-primary-foreground/70 text-xs font-black tracking-[0.08em] uppercase">
              My Squad
            </p>
            <Badge className={tierMeta(standings.myTier).badgeClassName}>
              {tierMeta(standings.myTier).label}
            </Badge>
          </div>
          <h2 className="text-xl font-black">{myTeam.name}</h2>
          <p className="text-primary-foreground/75 text-sm leading-relaxed">
            {tierMeta(standings.myTier).description}
            ；分層由隱藏戰鬥排名決定，只公開分層與可執行動作。
          </p>
        </Card>
      ) : (
        <Card className="rounded-[22px] px-4 py-4">
          <span className="text-muted-foreground text-sm font-extrabold">
            {standingsErrorMessage ?? "正在同步戰局分層"}
          </span>
        </Card>
      )}

      {activeMissionID ? (
        <ActiveMissionCard
          missionID={activeMissionID}
          teamNameOf={teamNameOf}
          sitoneNameOf={sitoneNameOf}
          myPlayerID={statusQuery.data?.playerId}
          onMissionClosed={() => setMissionID("")}
        />
      ) : null}

      <Card className="gap-2 overflow-hidden rounded-[24px] p-3">
        {bossQuery.data ? (
          <Link to="/yansan" className="no-underline">
            <BossStatusBanner status={bossQuery.data.bossStatus} compact />
          </Link>
        ) : null}
        {standings ? (
          <TerritoryMap
            teams={standings.teams}
            myTeamId={standings.myTeamId}
            bossUnderAttack={bossQuery.data?.bossStatus === "under_attack"}
            onSelectRegion={handleSelectRegion}
          />
        ) : standingsQuery.isError ? (
          <div className="grid min-h-[320px] place-items-center px-4 py-8 text-center">
            <div className="grid max-w-xs gap-3">
              <p className="text-ink text-lg font-black">無法展開戰地圖</p>
              <p className="text-muted-foreground text-sm leading-relaxed font-bold">
                {standingsErrorMessage}
              </p>
              <Button
                variant="outline"
                onClick={() => {
                  void standingsQuery.refetch()
                }}
              >
                重新載入
              </Button>
            </div>
          </div>
        ) : (
          <div className="text-muted-foreground grid min-h-[320px] place-items-center text-sm font-extrabold">
            正在展開作戰地圖
          </div>
        )}
        <div className="flex flex-wrap items-center gap-2 px-1 pb-1">
          {TIER_LEGEND.map((tier) => (
            <span
              key={tier}
              className={[
                "border-border rounded-full border-[1.5px] px-2 py-0.5 text-xs font-black",
                tierMeta(tier).badgeClassName,
              ].join(" ")}
            >
              {tierMeta(tier).label}
            </span>
          ))}
          <span className="bg-ink text-primary-foreground border-border rounded-full border-[1.5px] px-2 py-0.5 text-xs font-black">
            研三舍 Boss
          </span>
        </div>
      </Card>

      <section className="grid grid-cols-2 gap-2" aria-label="攻防選單">
        {TERRITORY_ACTION_LINKS.map(({ label, to, iconPath }) => (
          <Link
            key={to}
            to={to}
            className="bg-card border-ink focus-visible:outline-power group grid min-h-[76px] cursor-pointer grid-cols-[52px_1fr] items-center gap-2.5 rounded-[18px] border-2 px-3 py-3 text-inherit no-underline shadow-[4px_4px_0_rgba(23,35,58,0.12)] transition-transform focus-visible:outline-3 focus-visible:outline-offset-2 active:translate-x-px active:translate-y-px"
          >
            <span className="bg-background/70 border-border grid size-[52px] place-items-center rounded-[16px] border shadow-[inset_0_-2px_0_rgba(23,35,58,0.08)] transition-transform group-hover:-rotate-2">
              <GameIcon
                iconPath={iconPath}
                alt=""
                className="size-10 rounded-none"
                imageClassName="drop-shadow-[0_2px_0_rgba(23,35,58,0.18)]"
                fallback={null}
              />
            </span>
            <span className="min-w-0 text-left text-[15px] leading-tight font-black">
              {label}
            </span>
          </Link>
        ))}
      </section>

      <p className="text-muted-foreground px-1 text-xs leading-relaxed font-bold">
        點選可被攻擊的領地即可發起攻擊；領先隊（第 1–3
        名）只能防守，追趕隊（第 7–9 名）受保護不會被攻擊。
      </p>

      <AttackInitiateDialog
        open={attackTarget != null}
        defenderTeamId={attackTarget?.teamId ?? null}
        defenderTeamName={attackTarget?.name ?? ""}
        cap={defenseQuery.data?.cap ?? 12}
        onOpenChange={(open) => {
          if (!open) setAttackTarget(null)
        }}
        onMissionCreated={(createdMissionID) => {
          storeMissionID(createdMissionID)
          setMissionID(createdMissionID)
        }}
      />
    </GamePageShell>
  )
}
