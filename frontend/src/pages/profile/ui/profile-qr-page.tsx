import { Link } from "@tanstack/react-router"
import { ArrowLeft } from "lucide-react"

import { ProfileQrPanel } from "@/features/profile-qr"
import { GamePageShell } from "@/shared/ui/game-page-shell"

export function ProfileQrPage() {
  return (
    <GamePageShell>
      <header className="mb-3.5 flex items-center gap-3">
        <Link
          to="/"
          aria-label="返回營隊基地"
          className="border-ink bg-card text-ink focus-visible:outline-power grid size-11 shrink-0 place-items-center rounded-2xl border-2 transition-transform focus-visible:outline-3 focus-visible:outline-offset-2 active:translate-y-px"
        >
          <ArrowLeft className="size-5" aria-hidden />
        </Link>
        <div>
          <p className="text-muted-foreground mb-1 text-xs font-black tracking-[0.08em] uppercase">
            PLAYER PASS
          </p>
          <h1 className="text-[29px] leading-[1.05] font-black tracking-tight">
            冒險者通行證
          </h1>
        </div>
      </header>
      <div className="pt-6">
        <ProfileQrPanel />
      </div>
    </GamePageShell>
  )
}
