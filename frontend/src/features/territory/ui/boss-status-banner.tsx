import type { BossStatus } from "@/shared/api/territory"
import { bossStatusMeta } from "@/shared/lib/territory-labels"
import { Badge } from "@/shared/ui/badge"

type BossStatusBannerProps = {
  status: BossStatus
  compact?: boolean
}

export function BossStatusBanner({
  status,
  compact = false,
}: BossStatusBannerProps) {
  const meta = bossStatusMeta(status)

  if (compact) {
    return (
      <div className="bg-ink text-primary-foreground border-ink flex items-center gap-2.5 rounded-[18px] border-2 px-3.5 py-2.5 shadow-[4px_4px_0_rgba(23,35,58,0.22)]">
        <span
          className={[
            "size-2.5 shrink-0 rounded-full",
            status === "under_attack"
              ? "bg-destructive animate-pulse"
              : status === "open"
                ? "bg-status-warning"
                : status === "finished"
                  ? "bg-status-success"
                  : "bg-muted-foreground",
          ].join(" ")}
          aria-hidden
        />
        <div className="min-w-0">
          <p className="text-primary-foreground/70 text-[10px] font-black tracking-[0.08em] uppercase">
            研三舍 Boss
          </p>
          <p className="truncate text-sm font-black">{meta.label}</p>
        </div>
      </div>
    )
  }

  return (
    <div
      className={[
        "border-ink rounded-[22px] border-2 px-4 py-4 shadow-[4px_4px_0_rgba(23,35,58,0.12)]",
        status === "under_attack"
          ? "bg-destructive text-white"
          : "bg-ink text-primary-foreground",
      ].join(" ")}
    >
      <div className="mb-1.5 flex items-center justify-between gap-2">
        <p className="text-xs font-black tracking-[0.08em] uppercase opacity-70">
          Yansan Boss
        </p>
        <Badge className={meta.badgeClassName}>{meta.label}</Badge>
      </div>
      <h2 className="mb-1 text-xl font-black">研三舍 Boss 戰況</h2>
      <p className="text-sm leading-relaxed opacity-80">{meta.description}</p>
    </div>
  )
}
