import { X, Check } from "lucide-react"
import { useEffect } from "react"

import type { ShopItem } from "../model/shop.schema"
import { Button } from "@/shared/ui/button"
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/shared/ui/card"

type ShopItemDialogProps = {
  item: ShopItem | null
  onClose: () => void
}

function buyItem(itemId: number) {
  // TODO: 串接 API
  console.log('buy')

  // TODO: 針對購買成功與失敗，呼叫不同的 callout notify
  // ;)

  return
}

export function ShopItemDialog({ item, onClose }: ShopItemDialogProps) {
  useEffect(() => {
    if (!item) return

    function handleKeyDown(event: KeyboardEvent) {
      if (event.key === "Escape") onClose()
    }

    document.addEventListener("keydown", handleKeyDown)

    return () => document.removeEventListener("keydown", handleKeyDown)
  }, [item, onClose])

  if (!item) return null

  return (
    <div
      className="bg-foreground/60 fixed inset-0 z-50 flex items-center justify-center"
      role="presentation"
      onClick={onClose}
    >
      <Card
        role="dialog"
        aria-modal="true"
        aria-labelledby="shop-item-dialog-title"
        className="w-full max-w-sm"
        onClick={(event) => event.stopPropagation()}
      >
        <CardHeader className="relative">
          <CardTitle id="shop-item-dialog-title" className="truncate text-xl">
            {item.name}
          </CardTitle>
          <CardDescription>{item.description}</CardDescription>
        </CardHeader>
        <CardContent className="grid grid-cols-1 gap-y-2">
          <span>價格：{item.price} 元</span>
          <span className="whitespace-pre-wrap">{item.lore}</span>
          <img
            src={item.pictureSrc}
            alt={item.name}
            className="mx-auto aspect-square w-1/2 rounded-lg object-cover"
          />
        </CardContent>
        <CardFooter className="flex items-center justify-end gap-x-4">
          <Button type="button" variant="secondary" onClick={onClose}>
            <X />
            取消
          </Button>
          <Button type="button" onClick={buyItem(item.id)}>
            <Check />
            購買
          </Button>
        </CardFooter>
      </Card>
    </div>
  )
}
