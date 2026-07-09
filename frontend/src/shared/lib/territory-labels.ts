import type {
  AttackMissionStatus,
  BossStatus,
  CaptiveStatus,
  RiskLevel,
  TerritoryTier,
} from "@/shared/api/territory"

type TierMeta = {
  label: string
  description: string
  fillClassName: string
  badgeClassName: string
}

const tierMetaMap: Record<TerritoryTier, TierMeta> = {
  leader: {
    label: "領先隊",
    description: "戰局領先，只能防守、無法主動出擊",
    fillClassName: "fill-pebble-spark",
    badgeClassName: "bg-pebble-spark text-pebble-spark-foreground",
  },
  contested: {
    label: "交戰區",
    description: "可攻可守，戰況最激烈的一層",
    fillClassName: "fill-pebble-resonate",
    badgeClassName: "bg-pebble-resonate text-pebble-resonate-foreground",
  },
  challenger: {
    label: "追趕隊",
    description: "可以主動出擊，不會被其他小隊攻擊",
    fillClassName: "fill-pebble-engineer",
    badgeClassName: "bg-pebble-engineer text-pebble-engineer-foreground",
  },
}

const riskMetaMap: Record<
  RiskLevel,
  { label: string; className: string; dotClassName: string }
> = {
  low: {
    label: "低風險",
    className: "bg-pebble-engineer-muted text-pebble-engineer-foreground",
    dotClassName: "bg-status-success",
  },
  medium: {
    label: "中風險",
    className: "bg-pebble-spark-muted text-pebble-spark-foreground",
    dotClassName: "bg-status-warning",
  },
  high: {
    label: "高風險",
    className: "bg-pebble-resonate-muted text-pebble-resonate-foreground",
    dotClassName: "bg-destructive",
  },
}

const missionStatusLabels: Record<AttackMissionStatus, string> = {
  voting: "投票中",
  deployed: "出兵中",
  resolved: "已結算",
  cancelled: "已取消",
}

const bossStatusMetaMap: Record<
  BossStatus,
  { label: string; description: string; badgeClassName: string }
> = {
  closed: {
    label: "尚未開啟",
    description: "研三舍 Boss 戰尚未開放，等待工作人員宣告。",
    badgeClassName: "bg-muted text-muted-foreground",
  },
  open: {
    label: "已開啟",
    description: "Boss 戰已開放！集結全部 9 個小隊即可共同進攻研三舍。",
    badgeClassName: "bg-status-warning text-status-warning-foreground",
  },
  under_attack: {
    label: "激戰中",
    description:
      "研三舍正遭受攻擊，所有研三服務暫停；進行中的轉正／贖回視為失敗，已支付的開源力不予退還。",
    badgeClassName: "bg-destructive text-white",
  },
  finished: {
    label: "戰役結束",
    description: "Boss 戰已結束，研三舍服務恢復正常。",
    badgeClassName: "bg-status-success text-status-success-foreground",
  },
}

const rescueStatusLabels: Record<string, string> = {
  pending: "救援準備中",
  deployed: "救援進行中",
  succeeded: "救援成功",
  failed: "救援失敗",
  missing: "再次失聯",
  dead: "不幸逝世",
}

export const RESCUE_TERMINAL_STATUSES = new Set([
  "succeeded",
  "failed",
  "missing",
  "dead",
  "cancelled",
])

const captiveStatusLabels: Record<CaptiveStatus, string> = {
  captured: "被俘中",
  captive_cooldown: "冷卻中",
  convert_pending: "轉正處理中",
}

export function tierMeta(tier: TerritoryTier): TierMeta {
  return tierMetaMap[tier]
}

export function riskMeta(risk: RiskLevel) {
  return riskMetaMap[risk]
}

export function missionStatusLabel(status: AttackMissionStatus) {
  return missionStatusLabels[status]
}

export function bossStatusMeta(status: BossStatus) {
  return bossStatusMetaMap[status]
}

export function captiveStatusLabel(status: CaptiveStatus) {
  return captiveStatusLabels[status]
}

export function rescueStatusLabel(status: string) {
  return rescueStatusLabels[status] ?? status
}

export function formatCostRange(cost?: { min: number; max: number }) {
  if (!cost) return "估算中"
  return `${cost.min.toLocaleString("zh-TW")} ~ ${cost.max.toLocaleString("zh-TW")} 開源力`
}

export function formatCountdown(deadline: string | undefined, now: number) {
  if (!deadline) return null
  const remaining = new Date(deadline).getTime() - now
  if (Number.isNaN(remaining) || remaining <= 0) return "00:00"
  const totalSeconds = Math.floor(remaining / 1000)
  const hours = Math.floor(totalSeconds / 3600)
  const minutes = Math.floor((totalSeconds % 3600) / 60)
  const seconds = totalSeconds % 60
  const pad = (value: number) => String(value).padStart(2, "0")
  return hours > 0
    ? `${pad(hours)}:${pad(minutes)}:${pad(seconds)}`
    : `${pad(minutes)}:${pad(seconds)}`
}
