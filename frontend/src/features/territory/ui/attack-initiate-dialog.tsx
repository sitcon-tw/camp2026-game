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
      toast.success("攻擊任務已建立，等待隊友投票")
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
      <DialogContent className="max-h-[85dvh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>進攻 {defenderTeamName}</DialogTitle>
          <DialogDescription>
            挑選最多 {cap}{" "}
            顆出戰小石。送出後由系統估算開源力消耗區間與風險等級，隊友過半同意即出兵。
          </DialogDescription>
        </DialogHeader>

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
            disabled={createMutation.isPending}
            onChange={setSelectedIds}
          />
        )}

        <p className="text-muted-foreground text-xs leading-relaxed font-bold">
          實際消耗與成敗由伺服器結算，出戰前只會看到風險等級與預估區間；出戰小石有被俘、失聯與逝世的可能，請和隊友討論後再行動。
        </p>

        <DialogFooter>
          <Button
            variant="outline"
            disabled={createMutation.isPending}
            onClick={() => handleOpenChange(false)}
          >
            再想想
          </Button>
          <Button
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
      </DialogContent>
    </Dialog>
  )
}
