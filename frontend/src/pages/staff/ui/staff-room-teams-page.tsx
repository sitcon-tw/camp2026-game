import { StaffRoomTeamsPanel } from "@/features/staff-room-teams"
import { GamePageShell } from "@/shared/ui/game-page-shell"
import { PageHeader } from "@/shared/ui/page-header"

export function StaffRoomTeamsPage() {
  return (
    <GamePageShell
      ariaLabel="宿舍管理員"
      contentClassName="grid content-start gap-y-3"
    >
      <PageHeader title="宿舍管理員" headline="Dorm Manager" backTo="/" />
      <StaffRoomTeamsPanel />
    </GamePageShell>
  )
}
