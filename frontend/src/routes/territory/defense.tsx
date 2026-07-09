import { createFileRoute } from "@tanstack/react-router"

import { TerritoryDefensePage } from "@/pages/territory/ui/territory-defense-page"

export const Route = createFileRoute("/territory/defense")({
  component: TerritoryDefensePage,
})
