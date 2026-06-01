import { z } from "zod"

export const ShopItemSchema = z.object({
  id: z.number(),
  name: z.string(),
  pictureSrc: z.string(),
  price: z.number(),
  description: z.string(),
  lore: z.string(),
})

export type ShopItem = z.infer<typeof ShopItemSchema>

export const ShopItemsResponseSchema = z.object({
  items: z.array(ShopItemSchema),
})

export type ShopItemsResponse = z.infer<typeof ShopItemsResponseSchema>
