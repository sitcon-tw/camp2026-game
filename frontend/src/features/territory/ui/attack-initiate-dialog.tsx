import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { Minus, Plus } from "lucide-react"
import { useState } from "react"
import { toast } from "sonner"

import { gameApi, type PlayerItem } from "@/shared/api/game"
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
import { GameIcon } from "@/shared/ui/game-icon"
import { cn } from "@/shared/utils"

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
  const [selectedItemIds, setSelectedItemIds] = useState<string[]>([])
  const sitonesQuery = useQuery({
    queryKey: ["me", "sitones"],
    queryFn: gameApi.playerSitones,
    enabled: open,
  })
  const itemsQuery = useQuery({
    queryKey: ["me", "items"],
    queryFn: gameApi.playerItems,
    enabled: open,
  })

  const createMutation = useMutation({
    mutationFn: (input: {
      defenderTeamId: string
      sitoneIds: string[]
      itemIds: string[]
    }) => territoryApi.createAttack(input),
    onSuccess: (mission) => {
      toast.success("攻擊任務已建立，小石已出兵")
      storeMissionID(mission.id)
      queryClient.setQueryData(["territory", "attacks", mission.id], mission)
      queryClient.invalidateQueries({ queryKey: ["territory", "standings"] })
      setSelectedIds([])
      setSelectedItemIds([])
      onOpenChange(false)
      onMissionCreated(mission.id)
    },
    onError: (error) => {
      toast.error(error instanceof Error ? error.message : "發起攻擊失敗")
    },
  })

  const handleOpenChange = (next: boolean) => {
    if (createMutation.isPending) return
    if (!next) {
      setSelectedIds([])
      setSelectedItemIds([])
    }
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

        <div className="min-h-0 overflow-y-auto pr-1">
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

          <AttackItemPicker
            items={itemsQuery.data ?? []}
            selectedIds={selectedItemIds}
            disabled={createMutation.isPending}
            className="mt-3"
            onChange={setSelectedItemIds}
          />
        </div>

        <div className="grid gap-3">
          <p className="text-muted-foreground text-xs leading-relaxed font-bold">
            發起後會立刻扣除開源力並鎖定出戰小石；攻擊道具會隨任務記錄。出戰小石可能戰損、失聯或逝世。
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
                  itemIds: selectedItemIds,
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

function countByID(ids: string[]) {
  const counts = new Map<string, number>()
  for (const id of ids) counts.set(id, (counts.get(id) ?? 0) + 1)
  return counts
}

function AttackItemPicker({
  items,
  selectedIds,
  disabled,
  className,
  onChange,
}: {
  items: PlayerItem[]
  selectedIds: string[]
  disabled: boolean
  className?: string
  onChange: (itemIds: string[]) => void
}) {
  const attackItems = items.filter(
    (entry) => entry.item.type === "attack" && entry.quantity > 0,
  )
  const counts = countByID(selectedIds)
  const total = selectedIds.length

  const adjust = (itemID: string, delta: number, ownedQuantity: number) => {
    const current = counts.get(itemID) ?? 0
    const next = Math.max(0, Math.min(ownedQuantity, current + delta))
    if (next === current) return

    const remaining = selectedIds.filter((id) => id !== itemID)
    onChange([...remaining, ...Array.from({ length: next }, () => itemID)])
  }

  return (
    <section className={cn("grid gap-1.5", className)} aria-label="攻擊道具">
      <div className="flex items-center justify-between px-1">
        <span className="text-muted-foreground text-xs font-black">
          攻擊道具
        </span>
        <span className="text-muted-foreground text-xs font-black">
          已選 {total} 個
        </span>
      </div>
      {attackItems.length === 0 ? (
        <div className="border-border bg-surface-raised rounded-[14px] border-2 px-3 py-3">
          <span className="text-muted-foreground text-xs font-bold">
            目前沒有可放入任務的攻擊道具
          </span>
        </div>
      ) : (
        <div className="grid gap-1.5">
          {attackItems.map((entry) => {
            const selectedCount = counts.get(entry.itemId) ?? 0

            return (
              <div
                key={entry.itemId}
                className={cn(
                  "border-border bg-card grid grid-cols-[38px_1fr_auto] items-center gap-2 rounded-[14px] border-2 px-2.5 py-2",
                  selectedCount > 0 && "border-ink bg-surface-raised",
                )}
              >
                <div
                  className="border-ink bg-background/70 grid size-[38px] place-items-center overflow-hidden rounded-[12px] border-2"
                  aria-hidden
                >
                  <GameIcon
                    iconPath={entry.item.iconPath}
                    imageClassName="p-1"
                    fallback={
                      <span className="text-[10px] font-black">道具</span>
                    }
                  />
                </div>
                <div className="min-w-0">
                  <p className="truncate text-[13px] font-black">
                    {entry.item.name}
                  </p>
                  <p className="text-muted-foreground truncate text-xs font-bold">
                    持有 {entry.quantity}
                  </p>
                </div>
                <div className="flex items-center gap-1.5">
                  <Button
                    type="button"
                    variant="outline"
                    size="icon-sm"
                    aria-label={`減少 ${entry.item.name}`}
                    disabled={disabled || selectedCount === 0}
                    onClick={() => adjust(entry.itemId, -1, entry.quantity)}
                  >
                    <Minus aria-hidden />
                  </Button>
                  <span className="min-w-5 text-center text-sm font-black">
                    {selectedCount}
                  </span>
                  <Button
                    type="button"
                    variant="outline"
                    size="icon-sm"
                    aria-label={`增加 ${entry.item.name}`}
                    disabled={disabled || selectedCount >= entry.quantity}
                    onClick={() => adjust(entry.itemId, 1, entry.quantity)}
                  >
                    <Plus aria-hidden />
                  </Button>
                </div>
              </div>
            )
          })}
        </div>
      )}
    </section>
  )
}
