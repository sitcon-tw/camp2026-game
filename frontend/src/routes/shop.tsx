import { createFileRoute } from "@tanstack/react-router"
import { ShopPage } from "@/pages/shop/ui/shop-page"

export const Route = createFileRoute("/shop")({
  component: ShopPage,
})
