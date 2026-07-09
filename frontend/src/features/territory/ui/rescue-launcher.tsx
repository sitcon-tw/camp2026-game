import { useMutation, useQuery } from "@tanstack/react-query"
import { useState } from "react"
import { toast } from "sonner"

import { territoryApi } from "@/shared/api/territory"
import {
  RESCUE_TERMINAL_STATUSES,
  formatCostRange,
  rescueStatusLabel,
} from "@/shared/lib/territory-labels"
import { Button } from "@/shared/ui/button"

import { RiskBadge } from "./risk-badge"

type RescueLauncherProps = {
  sitoneID: string
  sitoneName: string
}

/**
 * 失聯小石的救援入口：發起救援後輪詢救援結果。
 * 救援由執行者支付開源力，可代隊友執行；結果同樣只公開狀態與風險等級。
 */
export function RescueLauncher({ sitoneID, sitoneName }: RescueLauncherProps) {
  const [rescueID, setRescueID] = useState("")

  const createMutation = useMutation({
    mutationFn: () => territoryApi.createRescue(sitoneID),
    onSuccess: (rescue) => {
      toast.success(`已為 ${sitoneName} 發起救援任務`)
      setRescueID(rescue.id)
    },
    onError: (error) => {
      toast.error(error instanceof Error ? error.message : "救援發起失敗")
    },
  })

  const rescueQuery = useQuery({
    queryKey: ["territory", "rescues", rescueID],
    queryFn: () => territoryApi.rescue(rescueID),
    enabled: rescueID !== "",
    refetchInterval: (query) => {
      const status = query.state.data?.status
      if (status != null && RESCUE_TERMINAL_STATUSES.has(status)) return false
      return 10000
    },
  })
  const rescue = rescueQuery.data

  return (
    <div className="bg-surface-raised border-border flex flex-wrap items-center justify-between gap-2 rounded-[14px] border-[1.5px] px-2.5 py-2">
      <div className="min-w-0">
        <p className="truncate text-sm font-black">{sitoneName}</p>
        <p className="text-muted-foreground text-xs font-bold">
          {rescue
            ? rescueStatusLabel(rescue.status)
            : "失聯中，需要發起救援任務"}
        </p>
      </div>
      {rescueID === "" ? (
        <Button
          size="sm"
          disabled={createMutation.isPending}
          onClick={() => createMutation.mutate()}
        >
          {createMutation.isPending ? "發起中" : "發起救援"}
        </Button>
      ) : (
        <div className="flex flex-wrap items-center gap-1.5">
          <RiskBadge label="救援" risk={rescue?.riskLevel} />
          {rescue?.estimatedCost ? (
            <span className="text-muted-foreground text-xs font-bold">
              {formatCostRange(rescue.estimatedCost)}
            </span>
          ) : null}
        </div>
      )}
    </div>
  )
}
