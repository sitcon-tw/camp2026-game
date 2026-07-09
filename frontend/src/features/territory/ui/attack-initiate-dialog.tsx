import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { useState } from "react"
import { toast } from "sonner"

import { gameApi } from "@/shared/api/game"
import { territoryApi } from "@/shared/api/territory"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/shared/ui/dialog"
import { Button } from "@/shared/ui/button"

import { storeMissionID } from "../model/mission-storage"
import { SitoneLoadoutPicker } from "./sitone-loadout-picker"

type AttackInitiateDialogProps = {
  open: boolean
  defenderTeamId: string | null
  defenderTeamName: string
  cap: number
  onOpenChange: (open: boolean) => void
  onMissionCreated: (missionID: string) => void
}

export function AttackInitiateDialog({
  open,
  defenderTeamId,
  defenderTeamName,
  cap,
  onOpenChange,
  onMissionCreated,
}: AttackInitiateDialogProps) {
  const queryClient = useQueryClient()
  const [selectedIds, setSelectedIds] = useState<string[]>([])
  const sitonesQuery = useQuery({
    queryKey: ["me", "sitones"],
    queryFn: gameApi.playerSitones,
    enabled: open,
  })

  const createMutation = useMutation({
    mutationFn: (input: { defenderTeamId: string; sitoneIds: string[] }) =>
      territoryApi.createAttack(input),
    onSuccess: (mission) => {
      toast.success("攻擊任務已建立，小石已出兵")
      storeMissionID(mission.id)
      queryClient.setQueryData(["territory", "attacks", mission.id], mission)
      queryClient.invalidateQueries({ queryKey: ["territory", "standings"] })
      setSelectedIds([])
      onOpenChange(false)
      onMissionCreated(mission.id)
    },
    onError: (error) => {
      toast.error(error instanceof Error ? error.message : "發起攻擊失敗")
    },
  })

  const handleOpenChange = (next: boolean) => {
    if (createMutation.isPending) return
    if (!next) setSelectedIds([])
    onOpenChange(next)
  }

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="grid max-h-[88dvh] grid-rows-[auto_minmax(0,1fr)_auto] gap-3 overflow-hidden p-4 sm:max-h-[760px] sm:p-6">
        <DialogHeader className="pr-12 text-left">
          <DialogTitle className="text-[28px] leading-tight sm:text-3xl">
            進攻 {defenderTeamName}
          </DialogTitle>
          <DialogDescription className="text-base leading-relaxed">
            挑選最多 {cap}{" "}
            顆出戰小石。送出後會立即出兵，系統會結算開源力消耗與風險等級。
          </DialogDescription>
        </DialogHeader>

        <div className="min-h-0 overflow-hidden">
          {sitonesQuery.isPending ? (
            <div className="border-border bg-surface-raised rounded-[16px] border-2 px-3 py-4">
              <span className="text-muted-foreground text-sm font-bold">
                正在同步你的小石
              </span>
            </div>
          ) : sitonesQuery.isError ? (
            <div className="border-border bg-surface-raised rounded-[16px] border-2 px-3 py-4">
              <span className="text-muted-foreground text-sm font-bold">
                小石資料讀取失敗，請稍後再試
              </span>
            </div>
          ) : (
            <SitoneLoadoutPicker
              sitones={sitonesQuery.data ?? []}
              selectedIds={selectedIds}
              cap={cap}
              mode="attack"
              compact
              className="h-full"
              listClassName="overflow-y-auto overscroll-contain pr-1 pb-1"
              disabled={createMutation.isPending}
              onChange={setSelectedIds}
            />
          )}
        </div>

        <div className="grid gap-3">
          <p className="text-muted-foreground text-xs leading-relaxed font-bold">
            發起後會立刻扣除開源力並鎖定出戰小石；出戰小石有被俘、失聯與逝世的可能。
          </p>
          <DialogFooter className="grid grid-cols-2 gap-2 sm:grid-cols-2">
            <Button
              variant="outline"
              className="w-full"
              disabled={createMutation.isPending}
              onClick={() => handleOpenChange(false)}
            >
              再想想
            </Button>
            <Button
              className="w-full"
              disabled={
                createMutation.isPending ||
                selectedIds.length === 0 ||
                defenderTeamId == null
              }
              onClick={() => {
                if (defenderTeamId == null) return
                createMutation.mutate({
                  defenderTeamId,
                  sitoneIds: selectedIds,
                })
              }}
            >
              {createMutation.isPending ? "發起中" : "發起攻擊"}
            </Button>
          </DialogFooter>
        </div>
      </DialogContent>
    </Dialog>
  )
}
