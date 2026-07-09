import { FrontMapPanel } from "@/features/front-map/ui/front-map-panel"
import { GamePageShell } from "@/shared/ui/game-page-shell"
import { PageHeader } from "@/shared/ui/page-header"

export function FrontPage() {
  return (
    <GamePageShell
      ariaLabel="開源戰線"
      contentClassName="grid content-start gap-y-3"
    >
      <PageHeader title="開源戰線" headline="Front" />
      <FrontMapPanel />
    </GamePageShell>
  )
}
