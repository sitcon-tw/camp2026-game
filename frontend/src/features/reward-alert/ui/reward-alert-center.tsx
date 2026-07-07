import { SparklesIcon } from "lucide-react"
import type { ReactNode } from "react"
import { useEffect, useState } from "react"
import { z } from "zod"

import {
  itemTypeClass,
  itemTypeLabel,
  sitoneMeta,
} from "@/shared/lib/game-labels"
import { Button } from "@/shared/ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/shared/ui/dialog"
import { GameIcon } from "@/shared/ui/game-icon"
import { SitoneIcon } from "@/shared/ui/sitone-icon"
import { cn } from "@/shared/utils"

const PlayerRewardEventSchema = z.object({
  rewardId: z.string().optional(),
  kind: z.enum(["item", "sitone", "open_power"]),
  refId: z.string().optional(),
  name: z.string(),
  quantity: z.number().optional(),
  amount: z.number().optional(),
  itemType: z.string().optional(),
  sitoneType: z.string().optional(),
  iconPath: z.string().optional(),
  source: z.string().optional(),
  staffPlayerId: z.string().optional(),
  staffNickname: z.string().optional(),
  occurredAt: z.string(),
  delayed: z.boolean().optional(),
})

type PlayerRewardEvent = z.infer<typeof PlayerRewardEventSchema>

const InventoryTrimmedEventSchema = z.object({
  trimId: z.string().optional(),
  message: z.string(),
  sitoneCount: z.number().optional(),
  openPower: z.number().optional(),
  occurredAt: z.string(),
  delayed: z.boolean().optional(),
})

type InventoryTrimmedEvent = z.infer<typeof InventoryTrimmedEventSchema>

type Notice =
  | { kind: "reward"; event: PlayerRewardEvent }
  | { kind: "inventory_trimmed"; event: InventoryTrimmedEvent }

const runawaySitoneImagePath = "/game-icons/alerts/sitone-runaway-memory.png"
const legacyInventoryTrimMessages = [
  "你的小石跟開源力太多了，小石跟開源力覺得太臃腫自己離家出走了",
  "營運平衡已更新你的小石與開源力數量。",
]

type GainCardProps = {
  badge: string
  title: string
  detail: string
  accentClassName: string
  icon: ReactNode
}

function rewardDetail(event: PlayerRewardEvent, fallback: string) {
  if (event.source !== "staff_reward") return fallback
  if (event.delayed) {
    if (event.staffNickname) {
      return `${event.staffNickname} 在你離線期間發送給你`
    }
    return "工作人員在你離線期間發送給你"
  }
  if (event.staffNickname) return `${event.staffNickname} 發送給你`
  return "工作人員發送給你"
}

function GainCard({
  badge,
  title,
  detail,
  accentClassName,
  icon,
}: GainCardProps) {
  return (
    <div className="w-full rounded-[24px] border-2 border-[var(--border)] bg-[var(--normal-bg)] p-3 text-[var(--normal-text)] shadow-[0_12px_28px_rgba(23,35,58,0.2)]">
      <div className="grid grid-cols-[56px_minmax(0,1fr)] items-center gap-3">
        <div
          className={cn(
            "border-ink grid size-14 place-items-center overflow-hidden rounded-[18px] border-2",
            accentClassName,
          )}
        >
          {icon}
        </div>
        <div className="min-w-0">
          <p className="text-muted-foreground text-[11px] font-black tracking-[0.08em] uppercase">
            {badge}
          </p>
          <p className="text-[17px] leading-tight font-black break-words">
            {title}
          </p>
          <p className="text-muted-foreground mt-1 text-sm leading-snug font-bold">
            {detail}
          </p>
        </div>
      </div>
    </div>
  )
}

function RewardNoticeCard({ event }: { event: PlayerRewardEvent }) {
  if (event.kind === "open_power") {
    const amount = event.amount ?? 0
    return (
      <GainCard
        badge="獲得開源力"
        title={`+${amount} OP`}
        detail={rewardDetail(event, "開源力已入帳")}
        accentClassName="bg-pebble-spark"
        icon={<SparklesIcon className="size-7" />}
      />
    )
  }

  if (event.kind === "item") {
    const typeLabel = itemTypeLabel(event.itemType ?? "item")
    return (
      <GainCard
        badge="獲得道具"
        title={event.name}
        detail={rewardDetail(event, `+${event.quantity ?? 0} ${typeLabel}`)}
        accentClassName={itemTypeClass(event.itemType ?? "")}
        icon={
          <GameIcon
            iconPath={event.iconPath}
            alt={event.name}
            imageClassName="p-2"
            fallback={
              <span className="text-[11px] font-black">
                {typeLabel.slice(0, 2)}
              </span>
            }
          />
        }
      />
    )
  }

  const meta = sitoneMeta(event.sitoneType ?? "")
  return (
    <GainCard
      badge="獲得小石"
      title={event.name}
      detail={rewardDetail(event, `+${event.quantity ?? 0} ${meta.label}`)}
      accentClassName={meta.bgClassName}
      icon={
        <SitoneIcon
          type={event.sitoneType ?? ""}
          iconPath={event.iconPath}
          alt={event.name}
          className="size-14 rounded-[18px] border-0 text-sm"
          imageClassName="p-1.5"
        />
      }
    />
  )
}

function InventoryTrimmedNoticeCard({
  event,
}: {
  event: InventoryTrimmedEvent
}) {
  const sitoneCount = event.sitoneCount ?? 0
  const openPower = event.openPower ?? 0
  const defaultMessage = `小石看著自己的 AI server ，感覺記憶體不太夠，於是帶著 ${openPower} 開源力去排隊購買記憶體了...應該很快就會回來`
  const message = legacyInventoryTrimMessages.includes(event.message)
    ? defaultMessage
    : event.message || defaultMessage
  const parts = [
    sitoneCount > 0 ? `${sitoneCount} 顆小石` : null,
    openPower > 0 ? `${openPower} OP` : null,
  ].filter(Boolean)

  return (
    <section
      className="bg-card border-ink relative grid gap-4 overflow-hidden rounded-[28px] border-2 p-3 text-center"
      style={{ boxShadow: "6px 6px 0 var(--border)" }}
    >
      <div
        className="border-ink bg-secondary text-secondary-foreground absolute top-3 right-3 z-10 rounded-full border-2 px-3 py-1 text-xs font-black"
        aria-hidden
      >
        補記憶體
      </div>

      <div className="border-ink bg-muted relative aspect-square overflow-hidden rounded-[22px] border-2">
        <img
          src={runawaySitoneImagePath}
          alt="小石開心帶著開源力去排隊購買記憶體"
          className="size-full object-cover"
          loading="lazy"
          draggable={false}
        />
      </div>

      <div className="grid gap-2 px-1 pb-1">
        <p className="text-muted-foreground text-[11px] font-black tracking-[0.08em] uppercase">
          Asset Update
        </p>
        <h2 className="text-[24px] leading-none font-black tracking-normal">
          小石離家出走
        </h2>
        <p className="text-muted-foreground text-sm leading-relaxed font-bold">
          {message}
        </p>
      </div>

      {parts.length > 0 ? (
        <div
          className="bg-surface-raised border-border rounded-[18px] border-2 px-3 py-2 text-sm font-black"
          aria-label="離家出走資產"
        >
          {parts.join("、")}
        </div>
      ) : null}
    </section>
  )
}

export function RewardAlertCenter() {
  const [noticeQueue, setNoticeQueue] = useState<Notice[]>([])
  const activeNotice = noticeQueue[0] ?? null

  function dismissActiveNotice() {
    setNoticeQueue((current) => current.slice(1))
  }

  useEffect(() => {
    if (typeof window === "undefined") return

    let source: EventSource | null = null
    let reconnectTimeout: number | null = null
    let reconnectAttempts = 0
    let disposed = false

    const clearReconnectTimeout = () => {
      if (reconnectTimeout != null) {
        window.clearTimeout(reconnectTimeout)
        reconnectTimeout = null
      }
    }

    const closeSource = () => {
      if (source == null) return
      source.close()
      source = null
    }

    const handleRewardGranted = (message: MessageEvent<string>) => {
      try {
        const event = PlayerRewardEventSchema.parse(JSON.parse(message.data))
        setNoticeQueue((current) => [...current, { kind: "reward", event }])
      } catch {
        // Ignore malformed events and keep the stream alive.
      }
    }

    const handleInventoryTrimmed = (message: MessageEvent<string>) => {
      try {
        const event = InventoryTrimmedEventSchema.parse(
          JSON.parse(message.data),
        )
        setNoticeQueue((current) => [
          ...current,
          { kind: "inventory_trimmed", event },
        ])
      } catch {
        // Ignore malformed events and keep the stream alive.
      }
    }

    const scheduleReconnect = () => {
      if (disposed || reconnectTimeout != null) return

      const delay = Math.min(5000, 1000 * 2 ** reconnectAttempts)
      reconnectAttempts += 1
      reconnectTimeout = window.setTimeout(() => {
        reconnectTimeout = null
        connect()
      }, delay)
    }

    const handleError = () => {
      if (disposed) return
      closeSource()
      scheduleReconnect()
    }

    const connect = () => {
      if (disposed) return

      clearReconnectTimeout()
      closeSource()

      source = new EventSource("/api/me/events", {
        withCredentials: true,
      })
      source.onopen = () => {
        reconnectAttempts = 0
      }
      source.onerror = handleError
      source.addEventListener("reward_granted", handleRewardGranted)
      source.addEventListener("inventory_trimmed", handleInventoryTrimmed)
    }

    connect()

    return () => {
      disposed = true
      clearReconnectTimeout()
      closeSource()
    }
  }, [])

  return (
    <Dialog
      open={activeNotice != null}
      onOpenChange={(open) => {
        if (!open) dismissActiveNotice()
      }}
    >
      <DialogContent
        className="gap-5 p-5 sm:max-w-[390px]"
        onEscapeKeyDown={(event) => event.preventDefault()}
        onPointerDownOutside={(event) => event.preventDefault()}
      >
        <DialogHeader>
          <DialogTitle>
            {activeNotice?.kind === "inventory_trimmed"
              ? "小石離家出走"
              : "獲得獎勵"}
          </DialogTitle>
          <DialogDescription>
            {activeNotice?.kind === "inventory_trimmed"
              ? "小石暫時離開基地補充設備。"
              : "新的獎勵已加入你的帳號。"}
          </DialogDescription>
        </DialogHeader>
        {activeNotice?.kind === "reward" ? (
          <RewardNoticeCard event={activeNotice.event} />
        ) : null}
        {activeNotice?.kind === "inventory_trimmed" ? (
          <InventoryTrimmedNoticeCard event={activeNotice.event} />
        ) : null}
        <DialogFooter>
          <Button
            type="button"
            className="w-full"
            onClick={dismissActiveNotice}
          >
            我知道了
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
