import { useNavigate } from "@tanstack/react-router"
import { Check, Info } from "lucide-react"

import { type ShopItem } from "@/shared/api/game"
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
import { GameIcon } from "@/shared/ui/game-icon"

import { ShopPurchaseConfirmButton } from "./shop-purchase-confirm-button"

type ShopItemCardType = {
  item: ShopItem
  currentOpenPower?: number
}

export function ShopItemCard({ item, currentOpenPower }: ShopItemCardType) {
  const navigate = useNavigate()

  return (
    <Card className="py-4">
      <CardContent className="px-4">
        <div className="grid grid-cols-[96px_minmax(0,1fr)] gap-3">
          <div className="self-start">
            <div
              className={[
                "border-ink grid size-24 place-items-center rounded-[1.375rem] border-2",
                itemTypeClass(item.type),
              ].join(" ")}
              aria-hidden
            >
              <GameIcon
                iconPath={item.iconPath}
                imageClassName="p-2"
                fallback={<GameFeatureIcon name="shop" className="size-8" />}
              />
            </div>
          </div>
          <div className="grid min-w-0 gap-2">
            <div className="grid grid-cols-[minmax(0,1fr)_auto] items-start gap-2">
              <div className="flex min-w-0 flex-wrap gap-1.5">
                {[
                  itemTypeLabel(item.type),
                  rarityLabel(item.rarity),
                  itemSourceLabel(item.source),
                ]
                  .filter(Boolean)
                  .map((tag) => (
                    <Badge variant="outline" key={tag}>
                      {tag}
                    </Badge>
                  ))}
              </div>
              <Badge>開源力 {item.priceOpenPower}</Badge>
            </div>
            <div className="truncate text-xl leading-tight font-bold">
              {item.name}
            </div>
            <div className="text-muted-foreground line-clamp-2 min-h-[2.75rem] text-sm leading-snug">
              {item.description}
            </div>
            <div className="grid grid-cols-2 gap-2">
              <Button
                variant="secondary"
                className="px-2"
                onClick={() =>
                  navigate({
                    to: "/shop/$itemId",
                    params: { itemId: item.id },
                  })
                }
              >
                <Info />
                資訊
              </Button>
              {item.redeemed ? (
                <Button variant="outline" className="px-2" disabled>
                  <Check />
                  已擁有
                </Button>
              ) : (
                <ShopPurchaseConfirmButton
                  item={item}
                  currentOpenPower={currentOpenPower}
                  className="px-2"
                />
              )}
            </div>
          </div>
        </div>
      </CardContent>
    </Card>
  )
}
