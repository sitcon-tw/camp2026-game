import { createFileRoute } from "@tanstack/react-router"

import { TerritoryMapPage } from "@/pages/territory/ui/territory-map-page"

export const Route = createFileRoute("/territory/")({
  component: TerritoryMapPage,
})
