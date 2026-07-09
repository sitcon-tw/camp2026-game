import type { RiskLevel } from "@/shared/api/territory"
import { riskMeta } from "@/shared/lib/territory-labels"

type RiskBadgeProps = {
  label: string
  risk?: RiskLevel
}

export function RiskBadge({ label, risk }: RiskBadgeProps) {
  if (!risk) return null
  const meta = riskMeta(risk)

  return (
    <span
      className={[
        "border-border inline-flex items-center gap-1.5 rounded-full border-[1.5px] px-2 py-0.5 text-xs font-black",
        meta.className,
      ].join(" ")}
    >
      <span
        className={["size-2 rounded-full", meta.dotClassName].join(" ")}
        aria-hidden
      />
      {label} {meta.label}
    </span>
  )
}
