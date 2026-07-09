import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { useEffect, useState } from "react"
import { toast } from "sonner"

import { SitoneLoadoutPicker } from "@/features/territory/ui/sitone-loadout-picker"
import { useNow } from "@/features/territory/model/use-now"
import { gameApi } from "@/shared/api/game"
import { territoryApi } from "@/shared/api/territory"
import { formatCountdown } from "@/shared/lib/territory-labels"
import { Button } from "@/shared/ui/button"
import { Card } from "@/shared/ui/card"
import { GamePageShell } from "@/shared/ui/game-page-shell"
import { PageHeader } from "@/shared/ui/page-header"

function sameIDs(a: string[], b: string[]) {
  if (a.length !== b.length) return false
  const sortedA = [...a].sort()
  const sortedB = [...b].sort()
  return sortedA.every((id, index) => id === sortedB[index])
}

export function TerritoryDefensePage() {
  const queryClient = useQueryClient()
  const now = useNow()
  const [selectedIds, setSelectedIds] = useState<string[] | null>(null)

  const defenseQuery = useQuery({
    queryKey: ["territory", "defense"],
    queryFn: territoryApi.defense,
  })
  const sitonesQuery = useQuery({
    queryKey: ["me", "sitones"],
    queryFn: gameApi.playerSitones,
  })

  const defense = defenseQuery.data
  useEffect(() => {
    if (defense && selectedIds === null) {
      setSelectedIds(defense.sitoneIds)
    }
  }, [defense, selectedIds])

  const saveMutation = useMutation({
    mutationFn: territoryApi.updateDefense,
    onSuccess: (updated) => {
      toast.success("防守配置已更新")
      queryClient.setQueryData(["territory", "defense"], updated)
      setSelectedIds(updated.sitoneIds)
    },
    onError: (error) => {
      toast.error(error instanceof Error ? error.message : "防守配置更新失敗")
    },
  })

  const lockedCountdown = formatCountdown(defense?.lockedUntil, now)
  const isLocked =
    defense?.lockedUntil != null &&
    new Date(defense.lockedUntil).getTime() > now
  const currentSelection = selectedIds ?? defense?.sitoneIds ?? []
  const dirty = defense ? !sameIDs(currentSelection, defense.sitoneIds) : false

  return (
    <GamePageShell contentClassName="grid content-start gap-y-3">
      <PageHeader title="防守配置" headline="Defense" backTo="/territory" />

      <Card className="bg-ink text-primary-foreground gap-1.5 rounded-[24px] px-5 py-4">
        <p className="text-primary-foreground/70 text-xs font-black tracking-[0.08em] uppercase">
          Garrison
        </p>
        <h2 className="text-xl font-black">留守小石陣</h2>
        <p className="text-primary-foreground/75 text-sm leading-relaxed">
          防守配置常駐生效，被攻擊時以「結算瞬間」的配置為準。每次修改後會有短暫冷卻，無法連續調整。
        </p>
      </Card>

      {isLocked ? (
        <Card className="border-status-warning gap-1 rounded-[18px] px-4 py-3">
          <p className="text-sm font-black">
            配置修改冷卻中：{lockedCountdown}
          </p>
          <p className="text-muted-foreground text-xs font-bold">
            冷卻結束後才能再次調整防守小石。
          </p>
        </Card>
      ) : null}

      {defenseQuery.isPending || sitonesQuery.isPending ? (
        <Card className="rounded-[18px] px-3 py-4">
          <span className="text-muted-foreground text-sm font-extrabold">
            正在同步防守配置
          </span>
        </Card>
      ) : defenseQuery.isError || sitonesQuery.isError ? (
        <Card className="rounded-[18px] px-3 py-4">
          <span className="text-muted-foreground text-sm font-extrabold">
            防守資料讀取失敗，請稍後再試
          </span>
        </Card>
      ) : defense ? (
        <>
          <SitoneLoadoutPicker
            sitones={sitonesQuery.data ?? []}
            selectedIds={currentSelection}
            cap={defense.cap}
            disabled={isLocked || saveMutation.isPending}
            onChange={setSelectedIds}
          />
          <Button
            className="min-h-12 rounded-2xl text-base"
            disabled={isLocked || !dirty || saveMutation.isPending}
            onClick={() => saveMutation.mutate(currentSelection)}
          >
            {saveMutation.isPending
              ? "儲存中"
              : dirty
                ? "儲存防守配置"
                : "目前配置已儲存"}
          </Button>
        </>
      ) : null}
    </GamePageShell>
  )
}
