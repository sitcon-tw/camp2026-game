import { useQuery } from "@tanstack/react-query"
import { Link } from "@tanstack/react-router"
import { Search } from "lucide-react"
import { useMemo, useState } from "react"

import { ShopItemCard } from "@/features/shop-index/ui/shop-item-card"
import { AppError } from "@/shared/api/error"
import { gameApi, type ShopItem } from "@/shared/api/game"
import { itemTypeLabel, rarityLabel } from "@/shared/lib/game-labels"
import { Badge } from "@/shared/ui/badge"
import { Button } from "@/shared/ui/button"
import { Card, CardContent } from "@/shared/ui/card"
import { GamePageShell } from "@/shared/ui/game-page-shell"
import { Input } from "@/shared/ui/input"
import { PageHeader } from "@/shared/ui/page-header"

type ShopConcept = "growth" | "tactical"

const shopConcepts: Array<{
  id: ShopConcept
  label: string
  description: string
  types: string[]
}> = [
  {
    id: "growth",
    label: "小石養成",
    description: "合成素材、御守與收藏品",
    types: ["material", "charm", "cosmetic", "event"],
  },
  {
    id: "tactical",
    label: "攻防戰術",
    description: "攻擊與防守道具",
    types: ["attack", "defense"],
  },
]

function matchesConcept(item: ShopItem, concept: ShopConcept) {
  const selected = shopConcepts.find((entry) => entry.id === concept)
  return selected?.types.includes(item.type) ?? true
}

function matchesSearch(item: ShopItem, query: string) {
  const normalized = query.trim().toLocaleLowerCase()
  if (normalized === "") return true
  return [
    item.id,
    item.name,
    item.description,
    item.type,
    itemTypeLabel(item.type),
    rarityLabel(item.rarity),
    item.source ?? "",
  ].some((value) => value.toLocaleLowerCase().includes(normalized))
}

export function ShopPage() {
  const [concept, setConcept] = useState<ShopConcept>("growth")
  const [search, setSearch] = useState("")
  const statusQuery = useQuery({
    queryKey: ["me", "status"],
    queryFn: gameApi.status,
  })
  const itemsQuery = useQuery({
    queryKey: ["shop", "items"],
    queryFn: gameApi.shopItems,
  })
  const isUnauthorized =
    (statusQuery.error instanceof AppError &&
      statusQuery.error.status === 401) ||
    (itemsQuery.error instanceof AppError && itemsQuery.error.status === 401)
  const items = useMemo(() => itemsQuery.data ?? [], [itemsQuery.data])
  const visibleItems = useMemo(
    () =>
      items.filter(
        (item) => matchesConcept(item, concept) && matchesSearch(item, search),
      ),
    [concept, items, search],
  )

  return (
    <GamePageShell contentClassName="grid content-start gap-y-2">
      <PageHeader
        title="商店"
        headline="Item Shop"
        rightSlot={
          <div className="flex flex-col items-end">
            <span className="text-muted-foreground text-sm font-bold">
              你現在持有開源力
            </span>
            <Badge className="h-fit">
              開源力 {statusQuery.data?.openPower ?? 0}
            </Badge>
          </div>
        }
      />

      {isUnauthorized ? (
        <Card>
          <CardContent className="grid gap-3">
            <h2 className="text-2xl font-bold">請先登入</h2>
            <p className="text-muted-foreground">
              登入後才能查看可兌換商品與目前開源力。
            </p>
            <Button asChild>
              <Link to="/login">前往登入</Link>
            </Button>
          </CardContent>
        </Card>
      ) : itemsQuery.isPending ? (
        <Card>
          <CardContent>
            <span className="text-muted-foreground font-bold">
              正在同步商品
            </span>
          </CardContent>
        </Card>
      ) : (
        <div className="grid gap-y-3">
          <section className="grid gap-2" aria-label="商店概念">
            <div className="grid grid-cols-2 gap-2">
              {shopConcepts.map((entry) => {
                const count = items.filter((item) =>
                  matchesConcept(item, entry.id),
                ).length
                const active = concept === entry.id

                return (
                  <Button
                    key={entry.id}
                    type="button"
                    variant={active ? "default" : "outline"}
                    className="h-auto min-h-16 flex-col items-start gap-1 rounded-[18px] px-3 py-3 text-left"
                    onClick={() => setConcept(entry.id)}
                  >
                    <span className="text-sm font-black">
                      {entry.label}
                      <Badge variant={active ? "secondary" : "outline"}>
                        {count}
                      </Badge>
                    </span>
                    <span className="text-xs leading-snug font-bold opacity-75">
                      {entry.description}
                    </span>
                  </Button>
                )
              })}
            </div>
            <div className="relative">
              <Search
                className="text-muted-foreground pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2"
                aria-hidden
              />
              <Input
                value={search}
                onChange={(event) => setSearch(event.target.value)}
                placeholder="搜尋商品名稱、類型或 ID"
                className="pl-9"
              />
            </div>
          </section>

          {visibleItems.length > 0 ? (
            <div className="grid gap-y-2">
              {visibleItems.map((item) => (
                <ShopItemCard
                  key={item.id}
                  item={item}
                  currentOpenPower={statusQuery.data?.openPower}
                />
              ))}
            </div>
          ) : (
            <Card>
              <CardContent>
                <span className="text-muted-foreground font-bold">
                  找不到符合條件的商品
                </span>
              </CardContent>
            </Card>
          )}
        </div>
      )}
    </GamePageShell>
  )
}
