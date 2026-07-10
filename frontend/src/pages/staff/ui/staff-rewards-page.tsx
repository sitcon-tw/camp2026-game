import { StaffRewardsPanel } from "@/features/staff-rewards"
import { GamePageShell } from "@/shared/ui/game-page-shell"
import { PageHeader } from "@/shared/ui/page-header"

export function StaffRewardsPage() {
  return (
    <GamePageShell
      ariaLabel="關主發放台"
      contentClassName="grid content-start gap-y-3"
    >
      <PageHeader title="關主發放台" headline="Staff Rewards" backTo="/" />
      <StaffRewardsPanel />
    </GamePageShell>
  )
}
