import { createFileRoute } from "@tanstack/react-router"

import { AchievementsPage } from "@/pages/achievements/ui/achievements-page"

export const Route = createFileRoute("/achievements")({
  component: AchievementsPage,
})
