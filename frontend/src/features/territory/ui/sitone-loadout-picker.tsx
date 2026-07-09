import { Minus, Plus } from "lucide-react"

import type { PlayerSitone } from "@/shared/api/game"
import { rarityLabel, sitoneMeta } from "@/shared/lib/game-labels"
import { Button } from "@/shared/ui/button"
import { GameIcon } from "@/shared/ui/game-icon"

type SitoneLoadoutPickerProps = {
  sitones: PlayerSitone[]
  selectedIds: string[]
  cap: number
  disabled?: boolean
  onChange: (sitoneIds: string[]) => void
}

function countByID(ids: string[]) {
  const counts = new Map<string, number>()
  for (const id of ids) {
    counts.set(id, (counts.get(id) ?? 0) + 1)
  }
  return counts
}

export function SitoneLoadoutPicker({
  sitones,
  selectedIds,
  cap,
  disabled = false,
  onChange,
}: SitoneLoadoutPickerProps) {
  const counts = countByID(selectedIds)
  const total = selectedIds.length

  const adjust = (sitoneID: string, delta: number, ownedQuantity: number) => {
    const current = counts.get(sitoneID) ?? 0
    const next = Math.max(0, Math.min(ownedQuantity, current + delta))
    if (next === current) return
    if (delta > 0 && total + delta > cap) return

    const remaining = selectedIds.filter((id) => id !== sitoneID)
    onChange([...remaining, ...Array.from({ length: next }, () => sitoneID)])
  }

  if (sitones.length === 0) {
    return (
      <div className="border-border bg-surface-raised rounded-[16px] border-2 px-3 py-4">
        <span className="text-muted-foreground text-sm font-bold">
          目前沒有可出戰的小石
        </span>
      </div>
    )
  }

  return (
    <div className="grid gap-2">
      <div className="flex items-center justify-between px-1">
        <span className="text-muted-foreground text-xs font-black">
          已選 {total} / {cap} 顆
        </span>
        {total >= cap ? (
          <span className="text-primary text-xs font-black">已達上限</span>
        ) : null}
      </div>
      {sitones.map((entry) => {
        const meta = sitoneMeta(entry.sitone.type)
        const selectedCount = counts.get(entry.sitoneId) ?? 0

        return (
          <div
            key={entry.sitoneId}
            className={[
              "border-border bg-card grid grid-cols-[44px_1fr_auto] items-center gap-2.5 rounded-[16px] border-2 px-3 py-2.5",
              selectedCount > 0 ? "border-ink bg-surface-raised" : "",
            ].join(" ")}
          >
            <div
              className={[
                "border-ink grid size-11 place-items-center overflow-hidden rounded-[14px] border-2",
                meta.bgClassName,
              ].join(" ")}
              aria-hidden
            >
              <GameIcon
                iconPath={entry.sitone.iconPath}
                imageClassName="p-1"
                fallback={
                  <span className="text-[10px] font-black">{meta.short}</span>
                }
              />
            </div>
            <div className="min-w-0">
              <p className="truncate text-sm font-black">
                {entry.sitone.name}
              </p>
              <p className="text-muted-foreground truncate text-xs font-bold">
                {meta.label} · {rarityLabel(entry.sitone.rarity)} · 持有{" "}
                {entry.quantity}
              </p>
            </div>
            <div className="flex items-center gap-1.5">
              <Button
                type="button"
                variant="outline"
                size="icon-sm"
                aria-label={`減少 ${entry.sitone.name}`}
                disabled={disabled || selectedCount === 0}
                onClick={() => adjust(entry.sitoneId, -1, entry.quantity)}
              >
                <Minus aria-hidden />
              </Button>
              <span className="min-w-6 text-center text-sm font-black">
                {selectedCount}
              </span>
              <Button
                type="button"
                variant="outline"
                size="icon-sm"
                aria-label={`增加 ${entry.sitone.name}`}
                disabled={
                  disabled || selectedCount >= entry.quantity || total >= cap
                }
                onClick={() => adjust(entry.sitoneId, 1, entry.quantity)}
              >
                <Plus aria-hidden />
              </Button>
            </div>
          </div>
        )
      })}
    </div>
  )
}
