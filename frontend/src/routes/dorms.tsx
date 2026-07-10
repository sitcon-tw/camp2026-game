import { createFileRoute } from "@tanstack/react-router"

import { StaffRoomTeamsPage } from "@/pages/staff/ui/staff-room-teams-page"

export const Route = createFileRoute("/dorms")({
  component: StaffRoomTeamsPage,
})
