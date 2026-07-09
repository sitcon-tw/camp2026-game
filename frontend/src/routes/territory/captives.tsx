import { createFileRoute } from "@tanstack/react-router"

import { TerritoryCaptivesPage } from "@/pages/territory/ui/territory-captives-page"

export const Route = createFileRoute("/territory/captives")({
  component: TerritoryCaptivesPage,
})
