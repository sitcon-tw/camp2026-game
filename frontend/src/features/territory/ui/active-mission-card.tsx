import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { useEffect } from "react"
import { toast } from "sonner"

import { AppError } from "@/shared/api/error"
import { territoryApi, type AttackMission } from "@/shared/api/territory"
import {
  formatCostRange,
  formatCountdown,
  missionStatusLabel,
} from "@/shared/lib/territory-labels"
import { Badge } from "@/shared/ui/badge"
import { Button } from "@/shared/ui/button"
import { Card } from "@/shared/ui/card"

import { clearStoredMissionID } from "../model/mission-storage"
import { useNow } from "../model/use-now"
import { RescueLauncher } from "./rescue-launcher"
import { RiskBadge } from "./risk-badge"

type ActiveMissionCardProps = {
  missionID: string
  teamNameOf: (teamID: string | undefined) => string
  sitoneNameOf: (sitoneID: string) => string
  myPlayerID?: string
  allowMissingRetry?: boolean
  onMissionClosed?: () => void
  onMissionMissing?: (missionID: string) => void
}

function missionPollInterval(mission?: AttackMission) {
  if (!mission) return 5000
  if (mission.status === "voting") return 3000
  if (mission.status === "deployed") return 10000
  return false as const
}

export function ActiveMissionCard({
  missionID,
  teamNameOf,
  sitoneNameOf,
  myPlayerID,
  allowMissingRetry = false,
  onMissionClosed,
  onMissionMissing,
}: ActiveMissionCardProps) {
  const queryClient = useQueryClient()
  const now = useNow()
  const missionQuery = useQuery({
    queryKey: ["territory", "attacks", missionID],
    queryFn: () => territoryApi.attack(missionID),
    refetchInterval: (query) => missionPollInterval(query.state.data),
    retry: (failureCount, error) => {
      if (error instanceof AppError && error.status === 404) {
        return allowMissingRetry && failureCount < 5
      }
      return error instanceof AppError && error.retryable && failureCount < 2
    },
    retryDelay: (attemptIndex, error) =>
      error instanceof AppError && error.status === 404
        ? Math.min(400 * (attemptIndex + 1), 1600)
        : Math.min(1000 * 2 ** attemptIndex, 5000),
  })
  const mission = missionQuery.data
  const missionNotFound =
    missionQuery.error instanceof AppError && missionQuery.error.status === 404

  useEffect(() => {
    if (missionNotFound) {
      clearStoredMissionID()
      onMissionMissing?.(missionID)
      onMissionClosed?.()
    }
  }, [missionID, missionNotFound, onMissionClosed, onMissionMissing])

  const cancelMutation = useMutation({
    mutationFn: () => territoryApi.cancelAttack(missionID),
    onSuccess: () => {
      toast.success("已取消這次攻擊任務")
      clearStoredMissionID()
      queryClient.invalidateQueries({ queryKey: ["territory"] })
      onMissionClosed?.()
    },
    onError: (error) => {
      toast.error(error instanceof Error ? error.message : "取消失敗")
    },
  })

  if (missionQuery.isPending) {
    return (
      <Card className="gap-2 rounded-[22px] px-4 py-4">
        <span className="text-muted-foreground text-sm font-extrabold">
          正在同步攻擊任務
        </span>
      </Card>
    )
  }

  if (missionNotFound) {
    return null
  }

  if (missionQuery.isError || !mission) {
    return (
      <Card className="gap-2 rounded-[22px] px-4 py-4">
        <div className="flex items-center justify-between gap-2">
          <span className="text-muted-foreground text-sm font-extrabold">
            攻擊任務讀取失敗
          </span>
          <Button
            variant="outline"
            size="sm"
            onClick={() => {
              clearStoredMissionID()
              onMissionClosed?.()
            }}
          >
            關閉
          </Button>
        </div>
      </Card>
    )
  }

  const isInitiator =
    myPlayerID != null && mission.initiatorPlayerId === myPlayerID
  const resolveCountdown = formatCountdown(mission.resolveAt, now)

  return (
    <Card className="gap-3 rounded-[22px] px-4 py-4">
      <div className="flex items-start justify-between gap-2">
        <div className="min-w-0">
          <p className="text-muted-foreground text-xs font-black tracking-[0.08em] uppercase">
            Attack Mission
          </p>
          <h3 className="truncate text-lg font-black">
            進攻 {teamNameOf(mission.defenderTeamId)}
          </h3>
        </div>
        <Badge
          variant={
            mission.status === "voting"
              ? "secondary"
              : mission.status === "deployed"
                ? "default"
                : "outline"
          }
        >
          {missionStatusLabel(mission.status)}
        </Badge>
      </div>

      {mission.sitoneIds.length > 0 ? (
        <div className="flex flex-wrap gap-1.5">
          {mission.sitoneIds.map((sitoneID, index) => (
            <span
              key={`${sitoneID}-${index}`}
              className="bg-surface-raised border-border text-muted-foreground rounded-full border-[1.5px] px-2 py-0.5 text-xs font-black"
            >
              {sitoneNameOf(sitoneID)}
            </span>
          ))}
        </div>
      ) : null}

      <div className="flex flex-wrap gap-1.5">
        <RiskBadge label="整體" risk={mission.riskLevel} />
        <RiskBadge label="失聯" risk={mission.missingRisk} />
        <RiskBadge label="逝世" risk={mission.deathRisk} />
      </div>

      {mission.estimatedCost ? (
        <p className="text-muted-foreground text-sm font-bold">
          預估消耗：{formatCostRange(mission.estimatedCost)}
        </p>
      ) : null}

      {mission.status === "voting" ? (
        <div className="grid gap-2.5">
          <p className="text-muted-foreground text-sm font-bold">
            任務正在轉為出兵狀態，請稍候同步最新戰況。
          </p>
          {isInitiator ? (
            <Button
              variant="destructive"
              size="sm"
              disabled={cancelMutation.isPending}
              onClick={() => cancelMutation.mutate()}
            >
              {cancelMutation.isPending ? "取消中" : "取消這次攻擊"}
            </Button>
          ) : null}
        </div>
      ) : null}

      {mission.status === "deployed" ? (
        <div className="bg-ink text-primary-foreground grid grid-cols-[1fr_auto] items-center gap-2 rounded-[16px] px-4 py-3">
          <div>
            <p className="text-primary-foreground/70 text-xs font-black">
              小石遠征中，最慢 45 分鐘內回報戰果
            </p>
            <strong className="text-2xl font-black tabular-nums">
              {resolveCountdown ?? "結算中"}
            </strong>
          </div>
          <span className="text-2xl" aria-hidden>
            ⚔️
          </span>
        </div>
      ) : null}

      {mission.status === "resolved" && mission.result ? (
        <div className="grid gap-2">
          <div className="bg-surface-raised border-border rounded-[16px] border-2 px-3 py-2.5">
            <p className="text-sm leading-relaxed font-bold">
              {mission.result.summary ?? `任務結果：${mission.result.outcome}`}
            </p>
          </div>
          {mission.result.capturedSitoneIds.length > 0 ? (
            <ResultLine
              label="搶得"
              ids={mission.result.capturedSitoneIds}
              sitoneNameOf={sitoneNameOf}
            />
          ) : null}
          {mission.result.lostSitoneIds.length > 0 ? (
            <ResultLine
              label="被俘"
              ids={mission.result.lostSitoneIds}
              sitoneNameOf={sitoneNameOf}
            />
          ) : null}
          {mission.result.missingSitoneIds.length > 0 ? (
            <div className="grid gap-1.5">
              <p className="text-muted-foreground text-xs font-bold">
                失聯的小石需要救援才有機會找回；救援由執行者支付開源力，也可能失敗、再失聯或逝世。
              </p>
              {mission.result.missingSitoneIds.map((sitoneID, index) => (
                <RescueLauncher
                  key={`${sitoneID}-${index}`}
                  sitoneID={sitoneID}
                  sitoneName={sitoneNameOf(sitoneID)}
                />
              ))}
            </div>
          ) : null}
          {mission.result.deadSitoneIds.length > 0 ? (
            <ResultLine
              label="逝世"
              ids={mission.result.deadSitoneIds}
              sitoneNameOf={sitoneNameOf}
            />
          ) : null}
          <Button
            variant="outline"
            size="sm"
            onClick={() => {
              clearStoredMissionID()
              queryClient.invalidateQueries({ queryKey: ["territory"] })
              onMissionClosed?.()
            }}
          >
            收到，關閉戰報
          </Button>
        </div>
      ) : null}

      {mission.status === "cancelled" ? (
        <div className="grid gap-2">
          <p className="text-muted-foreground text-sm font-bold">
            這次攻擊任務已取消，小石與開源力都沒有扣除。
          </p>
          <Button
            variant="outline"
            size="sm"
            onClick={() => {
              clearStoredMissionID()
              onMissionClosed?.()
            }}
          >
            關閉
          </Button>
        </div>
      ) : null}
    </Card>
  )
}

function ResultLine({
  label,
  ids,
  sitoneNameOf,
}: {
  label: string
  ids: string[]
  sitoneNameOf: (sitoneID: string) => string
}) {
  return (
    <p className="text-muted-foreground text-xs font-bold">
      {label}：{ids.map(sitoneNameOf).join("、")}
    </p>
  )
}
