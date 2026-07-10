import { createFileRoute } from "@tanstack/react-router"

import { StaffRewardsPage } from "@/pages/staff/ui/staff-rewards-page"

export const Route = createFileRoute("/rewards")({
  component: StaffRewardsPage,
})
