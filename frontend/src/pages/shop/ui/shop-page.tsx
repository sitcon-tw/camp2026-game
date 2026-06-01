import { useState } from "react"

import type { ShopItem } from "@/features/shop/model/shop.schema"
import { ShopItemBlock } from "@/features/shop/ui/shop-item-block"
import { ShopItemDialog } from "@/features/shop/ui/shop-item-dialog"

import { House } from "lucide-react"
import { Button } from "@/shared/ui/button"

const shopItems: ShopItem[] = [
  {
    id: 0,
    name: "餅乾",
    pictureSrc: "https://placehold.co/150x150/orange/svg",
    price: 100,
    description: "好吃的餅乾，體力++",
    lore: "從某個偉大組長家裡偷來的餅乾，但是讀書人的事，怎麼能說偷呢？\n有此經驗後，這個餅乾彷彿給了一個勇氣的翅膀，體力提昇！",
  },
  {
    id: 1,
    name: "水果",
    pictureSrc: "https://placehold.co/150x150/purple/svg",
    price: 50,
    description: "好吃的餅乾，體力++",
    lore: "從某個偉大組長家裡偷來的餅乾，但是讀書人的事，怎麼能說偷呢？\n有此經驗後，這個餅乾彷彿給了一個勇氣的翅膀，體力提昇！",
  },
  {
    id: 2,
    name: "餅乾",
    pictureSrc: "https://placehold.co/150x150/cyan/svg",
    price: 100,
    description: "好吃的餅乾，體力++",
    lore: "從某個偉大組長家裡偷來的餅乾，但是讀書人的事，怎麼能說偷呢？\n有此經驗後，這個餅乾彷彿給了一個勇氣的翅膀，體力提昇！",
  },
  {
    id: 3,
    name: "餅乾",
    pictureSrc: "https://placehold.co/150x150/green/svg",
    price: 100,
    description: "好吃的餅乾，體力++",
    lore: "從某個偉大組長家裡偷來的餅乾，但是讀書人的事，怎麼能說偷呢？\n有此經驗後，這個餅乾彷彿給了一個勇氣的翅膀，體力提昇！",
  },
  {
    id: 4,
    name: "餅乾",
    pictureSrc: "https://placehold.co/150x150/yellow/svg",
    price: 100,
    description: "好吃的餅乾，體力++",
    lore: "從某個偉大組長家裡偷來的餅乾，但是讀書人的事，怎麼能說偷呢？\n有此經驗後，這個餅乾彷彿給了一個勇氣的翅膀，體力提昇！",
  },
]

export function ShopPage() {
  const [selectedItem, setSelectedItem] = useState<ShopItem | null>(null)

  return (
    <div className="flex min-h-dvh flex-col gap-4 border px-8 py-4">
      <h2 className="text-center text-4xl">商店</h2>
      <div className="grid w-full grid-cols-2 gap-4">
        {shopItems.map((item) => (
          <ShopItemBlock key={item.id} item={item} onSelect={setSelectedItem} />
        ))}
      </div>
      <ShopItemDialog
        item={selectedItem}
        onClose={() => setSelectedItem(null)}
      />
      <Button className="w-fit">
        <House />
        返回首頁
      </Button>
    </div>
  )
}
