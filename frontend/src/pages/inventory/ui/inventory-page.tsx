import { useQuery } from "@tanstack/react-query"
import { Link } from "@tanstack/react-router"

import { AppError } from "@/shared/api/error"
import { gameApi, type PlayerItem } from "@/shared/api/game"
import {
  itemTypeClass,
  itemTypeLabel,
  rarityLabel,
} from "@/shared/lib/game-labels"
import { Button } from "@/shared/ui/button"
import { Card } from "@/shared/ui/card"
import { GameFeatureIcon } from "@/shared/ui/game-feature-icon"
import { GamePageShell } from "@/shared/ui/game-page-shell"
import { GameIcon } from "@/shared/ui/game-icon"
import { PageHeader } from "@/shared/ui/page-header"

function ItemIcon({ item }: { item: PlayerItem }) {
  return (
    <div
      className={[
        "border-ink relative h-[72px] overflow-hidden rounded-[20px] border-2",
        itemTypeClass(item.item.type),
      ].join(" ")}
      aria-hidden
    >
      <GameIcon
        iconPath={item.item.iconPath}
        imageClassName="p-2"
        fallback={
          <span className="border-card/80 absolute inset-[14px_18px] -rotate-[12deg] border-x-0 border-y-[3px]" />
        }
      />
      <strong className="bg-card border-ink absolute right-1.5 bottom-[5px] min-w-[28px] rounded-full border-2 px-1.5 py-px text-center text-[13px]">
        {item.quantity}
      </strong>
    </div>
  )
}

function ItemCard({ item }: { item: PlayerItem }) {
  return (
    <Card className="border-ink grid grid-cols-[72px_1fr] items-start gap-3 rounded-[22px] border-2 p-3.5">
      <ItemIcon item={item} />
      <div>
        <div className="mb-1.5 flex items-start justify-between gap-2">
          <h3 className="text-[18px] leading-tight font-black tracking-[-0.02em]">
            {item.item.name}
          </h3>
          <strong className="text-primary shrink-0 text-[18px] font-black">
            x{item.quantity}
          </strong>
        </div>
        <div className="mb-1.5 flex flex-wrap gap-1.5">
          {[itemTypeLabel(item.item.type), rarityLabel(item.item.rarity)].map(
            (tag) => (
              <span
                key={tag}
                className="bg-surface-raised border-border text-muted-foreground rounded-full border-[1.5px] px-2 py-0.5 text-xs font-black"
              >
                {tag}
              </span>
            ),
          )}
        </div>
        <p className="text-muted-foreground text-sm leading-[1.62]">
          {item.item.description}
        </p>
      </div>
    </Card>
  )
}

function EmptyBag({ message }: { message: string }) {
  return (
    <Card
      className="border-ink rounded-[22px] border-2 border-dashed p-[22px]"
      aria-label="空背包狀態"
    >
      <div className="bg-surface-raised border-ink mb-3 grid size-[54px] place-items-center rounded-[18px] border-2 text-[28px] font-black">
        <GameFeatureIcon name="backpack" className="size-9" />
      </div>
      <h3 className="mb-1 text-[17px] font-black">{message}</h3>
      <p className="text-muted-foreground mb-3.5 leading-[1.65]">
        可以到商店兌換，或在現場活動完成挑戰後取得。
      </p>
      <Button asChild type="button" className="w-full">
        <Link to="/shop">前往商店</Link>
      </Button>
    </Card>
  )
}

export function InventoryPage() {
  const {
    data: items = [],
    isPending,
    error,
  } = useQuery({
    queryKey: ["me", "items"],
    queryFn: gameApi.playerItems,
  })

  const totalCount = items.reduce((sum, item) => sum + item.quantity, 0)
  const typeCounts = items.reduce<Record<string, number>>((counts, item) => {
    counts[item.item.type] = (counts[item.item.type] ?? 0) + item.quantity
    return counts
  }, {})

  const isUnauthorized = error instanceof AppError && error.status === 401

  return (
    <GamePageShell
      ariaLabel="道具背包頁"
      contentClassName="grid content-start gap-y-3"
    >
      <PageHeader
        title="道具背包"
        headline="Field Bag"
        rightSlot={
          <Button variant="secondary" size="sm" className="border-ink border-2">
            {isPending ? "同步中" : `${items.length} 種`}
          </Button>
        }
      />

      {isUnauthorized ? (
        <EmptyBag message="請先登入才能查看背包" />
      ) : (
        <>
          <Card
            className="border-ink grid gap-4 rounded-[22px] border-2 p-5"
            style={{ boxShadow: "5px 5px 0 rgba(23,35,58,.16)" }}
            aria-label="背包摘要"
          >
            <div className="min-w-0">
              <p className="text-muted-foreground mb-[3px] text-xs font-black tracking-[0.08em] uppercase">
                目前持有
              </p>
              <strong className="block text-[48px] leading-[0.95] font-black tracking-[-0.05em]">
                {totalCount}
              </strong>
              <p className="text-muted-foreground mt-2 leading-[1.65]">
                素材、外觀與活動紀念都會先收在這裡。
              </p>
            </div>
            <div className="grid grid-cols-2 gap-2">
              <div className="bg-surface-raised border-border rounded-[18px] border-2 p-3">
                <span className="text-muted-foreground block text-xs font-black">
                  已收納種類
                </span>
                <strong className="mt-0.5 block text-[22px] font-black">
                  {items.length}
                </strong>
              </div>
              <div className="bg-surface-raised border-border rounded-[18px] border-2 p-3">
                <span className="text-muted-foreground block text-xs font-black">
                  背包狀態
                </span>
                <strong className="mt-0.5 block text-[22px] font-black">
                  {isPending ? "同步" : totalCount > 0 ? "有道具" : "空"}
                </strong>
              </div>
            </div>
          </Card>

          <section
            className="grid grid-cols-3 gap-[10px]"
            aria-label="分類數量"
          >
            {["material", "cosmetic", "event"].map((type) => (
              <div
                key={type}
                className="bg-surface-raised border-border rounded-[18px] border-2 p-3"
              >
                <span className="text-muted-foreground block text-xs font-black">
                  {itemTypeLabel(type)}
                </span>
                <strong className="mt-0.5 block text-[22px] font-black">
                  {typeCounts[type] ?? 0}
                </strong>
              </div>
            ))}
          </section>

          <section className="grid gap-3" aria-label="道具列表">
            {isPending ? (
              <EmptyBag message="正在同步背包" />
            ) : items.length > 0 ? (
              items.map((item) => <ItemCard key={item.id} item={item} />)
            ) : (
              <EmptyBag message="背包目前沒有道具" />
            )}
          </section>
        </>
      )}
    </GamePageShell>
  )
}
