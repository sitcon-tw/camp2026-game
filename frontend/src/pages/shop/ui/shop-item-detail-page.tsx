import { useQuery } from "@tanstack/react-query"
import { Link } from "@tanstack/react-router"
import { Check } from "lucide-react"

import { ShopPurchaseConfirmButton } from "@/features/shop-index/ui/shop-purchase-confirm-button"
import { gameApi } from "@/shared/api/game"
import {
  itemTypeClass,
  itemTypeLabel,
  itemSourceLabel,
  rarityLabel,
} from "@/shared/lib/game-labels"
import { Badge } from "@/shared/ui/badge"
import { Button } from "@/shared/ui/button"
import { Card, CardContent } from "@/shared/ui/card"
import { GameFeatureIcon } from "@/shared/ui/game-feature-icon"
import { GamePageShell } from "@/shared/ui/game-page-shell"
import { GameIcon } from "@/shared/ui/game-icon"
import { PageHeader } from "@/shared/ui/page-header"

type ShopItemDetailPageProps = {
  itemID: string
}

export function ShopItemDetailPage({ itemID }: ShopItemDetailPageProps) {
  const itemQuery = useQuery({
    queryKey: ["shop", "item", itemID],
    queryFn: () => gameApi.shopItem(itemID),
  })
  const statusQuery = useQuery({
    queryKey: ["me", "status"],
    queryFn: gameApi.status,
  })
  const item = itemQuery.data

  return (
    <GamePageShell contentClassName="grid content-start gap-y-2">
      <PageHeader title="道具資訊" headline="Item Detail" backTo="/shop" />

      {itemQuery.isPending || !item ? (
        <Card>
          <CardContent>
            <span className="text-muted-foreground font-bold">
              正在同步道具資訊
            </span>
          </CardContent>
        </Card>
      ) : (
        <>
          <div
            className={[
              "border-foreground mx-auto mb-2 grid size-36 -rotate-3 place-items-center rounded-lg border-2",
              itemTypeClass(item.type),
            ].join(" ")}
            aria-hidden
          >
            <GameIcon
              iconPath={item.iconPath}
              imageClassName="p-3"
              fallback={<GameFeatureIcon name="shop" className="size-14" />}
            />
          </div>

          <Card>
            <CardContent className="grid gap-y-2">
              <div className="flex gap-2">
                {[
                  itemTypeLabel(item.type),
                  rarityLabel(item.rarity),
                  itemSourceLabel(item.source),
                ]
                  .filter(Boolean)
                  .map((tag) => (
                    <Badge key={tag} variant="outline">
                      {tag}
                    </Badge>
                  ))}
              </div>
              <span className="text-2xl font-bold">{item.name}</span>
              <div className="grid gap-y-1">
                <span className="text-accent-foreground text-lg font-bold">
                  道具簡介
                </span>
                <span>{item.description}</span>
              </div>
              <div className="grid gap-1">
                <span className="text-accent-foreground text-lg font-bold">
                  狀態
                </span>
                <span>{item.redeemed ? "已擁有" : "可使用開源力兌換"}</span>
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardContent className="flex items-center justify-between">
              <div className="grid">
                <span className="text-muted-foreground text-sm font-bold">
                  花費開源力
                </span>
                <div className="flex items-center">
                  <span className="text-2xl font-bold">
                    {item.priceOpenPower}
                  </span>
                  <span className="whitespace-pre">
                    {" "}
                    / {statusQuery.data?.openPower ?? 0} 開源力
                  </span>
                </div>
              </div>
              {item.redeemed ? (
                <Button size="lg" disabled variant="outline">
                  <Check />
                  已擁有
                </Button>
              ) : (
                <ShopPurchaseConfirmButton
                  item={item}
                  currentOpenPower={statusQuery.data?.openPower}
                  size="lg"
                  label="確認購買"
                />
              )}
            </CardContent>
          </Card>

          <Button asChild variant="secondary">
            <Link to="/shop">返回商店</Link>
          </Button>
        </>
      )}
    </GamePageShell>
  )
}
