import { queryOptions } from "@tanstack/react-query"
import { apiClient } from "@/shared/api/client"
import { ShopItemsResponseSchema } from "../model/shop.schema"

export const shopItemsQueryKey = ["shop", "item"] as const

export function shopItemsQueryOptions() {
  return queryOptions({
    queryKey: shopItemsQueryKey,
    queryFn: async () => {
      const json = await apiClient.get("/api/shop/list")
      return ShopItemsResponseSchema.parse(json)
    },
  })
}
