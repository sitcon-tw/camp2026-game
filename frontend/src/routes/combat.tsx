import { CombactPage } from "@/pages/combat/ui/combat-page"
import { createFileRoute } from "@tanstack/react-router"

export const Route = createFileRoute("/combat")({
  component: CombactPage,
})
