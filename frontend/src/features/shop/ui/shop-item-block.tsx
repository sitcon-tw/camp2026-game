import type { ShopItem } from "../model/shop.schema"
import { Button } from "@/shared/ui/button"

type ShopItemBlockProps = {
  item: ShopItem
  onSelect: (item: ShopItem) => void
}

export function ShopItemBlock({ item, onSelect }: ShopItemBlockProps) {
  return (
    <Button
      type="button"
      variant="ghost"
      className="relative h-auto w-full cursor-pointer p-0"
      onClick={() => onSelect(item)}
    >
      <div className="relative w-fit">
        <img
          src={item.pictureSrc}
          alt={item.name}
          className="relative top-0 left-0 rounded-lg"
        />
        <div className="from-foreground from-10% absolute bottom-0 h-full w-full rounded-lg bg-linear-to-t to-transparent to-50%">
          <p className="text-background absolute bottom-2 left-4 text-lg">{item.name}</p>
          <p className="text-background absolute right-4 bottom-2 text-lg">{item.price} 元</p>
        </div>
      </div>
    </Button>
  )
}
