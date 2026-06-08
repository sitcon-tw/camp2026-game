import { ChevronLeft } from "lucide-react"

import { PublicCodexPanel } from "@/features/public-codex"
import { Button } from "@/shared/ui/button"

export function PublicCodexPage() {
  return (
    <main className="bg-background text-foreground flex min-h-svh justify-center">
      <div className="flex min-h-svh w-full max-w-[430px] flex-col px-4 pt-[18px] pb-[30px]">
        <header className="mb-3.5 flex items-center gap-3">
          <Button
            type="button"
            size="icon"
            variant="outline"
            onClick={() => window.history.back()}
            aria-label="返回營隊基地"
            className="h-11 min-h-11 w-11 rounded-2xl bg-card text-[28px] shadow-[3px_3px_0_rgba(23,35,58,0.18)]"
          >
            <ChevronLeft className="size-6 stroke-[3]" />
          </Button>
          <div>
            <p className="text-muted-foreground mb-1 text-xs font-black tracking-[0.08em] uppercase">
              PUBLIC CODEX
            </p>
            <h1 className="text-[30px] leading-[1.05] font-black tracking-tight">
              公開圖鑑
            </h1>
          </div>
        </header>
        <PublicCodexPanel />
      </div>
    </main>
  )
}
