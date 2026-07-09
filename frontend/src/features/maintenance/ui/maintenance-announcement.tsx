import { useEffect, useState } from "react"
import { useQuery } from "@tanstack/react-query"
import { useLocation } from "@tanstack/react-router"
import { gameApi, type MaintenanceStatus } from "@/shared/api/game"
import { Button } from "@/shared/ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/shared/ui/dialog"
import { toOptimizedImageSrc } from "@/shared/utils/image-src"

const hiddenPathPrefixes = ["/admin"] as const
const maintenanceImagePath = "/game-icons/alerts/sitone-maintenance.png"

function isHiddenPath(pathname: string) {
  return hiddenPathPrefixes.some(
    (path) => pathname === path || pathname.startsWith(`${path}/`),
  )
}

export function MaintenanceAnnouncement() {
  const [lastNotice, setLastNotice] = useState<MaintenanceStatus | null>(null)
  const [dismissedRevision, setDismissedRevision] = useState<string | null>(
    null,
  )
  const location = useLocation()
  const hidden = isHiddenPath(location.pathname)
  const maintenanceQuery = useQuery({
    queryKey: ["maintenance"],
    queryFn: gameApi.maintenanceStatus,
    enabled: !hidden,
    refetchInterval: 5_000,
    staleTime: 3_000,
  })

  const activeNotice = maintenanceQuery.data?.enabled
    ? maintenanceQuery.data
    : null
  const completedNotice =
    !activeNotice &&
    lastNotice &&
    dismissedRevision !== noticeRevision(lastNotice)
      ? lastNotice
      : null
  const hiddenForActiveBattle =
    activeNotice?.mode === "draining" &&
    location.pathname === "/battle/question"
  const notice = hiddenForActiveBattle
    ? null
    : (activeNotice ?? completedNotice)
  const completed = Boolean(completedNotice)

  useEffect(() => {
    if (!activeNotice) return

    const id = window.setTimeout(() => {
      setLastNotice(activeNotice)
      setDismissedRevision(null)
    }, 0)
    return () => window.clearTimeout(id)
  }, [activeNotice])

  if (hidden || !notice) return null

  const title = completed
    ? "維護已完成"
    : notice.mode === "draining"
      ? "即將進入維護"
      : "系統維護中"
  const description = completed
    ? "維護已經結束，可以繼續使用營隊遊戲。"
    : notice.message || "系統正在更新，暫時停止新的遊戲操作。"
  const buttonText = completed
    ? "關閉通知"
    : notice.mode === "draining"
      ? notice.activeMatchCount > 0
        ? `等待 ${notice.activeMatchCount} 場對戰結束`
        : "準備維護中"
      : "維護中"

  return (
    <Dialog
      open
      onOpenChange={(open) => {
        if (!open && completed) {
          setDismissedRevision(noticeRevision(notice))
        }
      }}
    >
      <DialogContent
        className="gap-5 p-5 sm:max-w-[390px]"
        showCloseButton={completed}
        onEscapeKeyDown={(event) => {
          if (!completed) event.preventDefault()
        }}
        onPointerDownOutside={(event) => {
          if (!completed) event.preventDefault()
        }}
      >
        <DialogHeader>
          <DialogTitle>{title}</DialogTitle>
          <DialogDescription>
            {completed
              ? "維護完成通知。"
              : "請先完成目前操作，等待系統重新開放。"}
          </DialogDescription>
        </DialogHeader>
        <MaintenanceNoticeCard
          notice={notice}
          completed={completed}
          description={description}
        />
        <DialogFooter>
          <Button
            type="button"
            className="w-full"
            disabled={!completed}
            onClick={() => setDismissedRevision(noticeRevision(notice))}
          >
            {buttonText}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

function MaintenanceNoticeCard({
  notice,
  completed,
  description,
}: {
  notice: MaintenanceStatus
  completed: boolean
  description: string
}) {
  const badge = completed
    ? "維護完成"
    : notice.mode === "draining"
      ? "準備維護"
      : "維護中"
  const title = completed
    ? "系統重新開放"
    : notice.mode === "draining"
      ? "系統準備更新"
      : "系統維護中"
  const stats = completed
    ? ["可以繼續使用營隊遊戲"]
    : [
        `${notice.activeMatchCount} 場進行中對戰`,
        `${notice.openMatchCount} 間等待或進行中的房間`,
      ]

  return (
    <section
      className="bg-card border-ink relative grid gap-4 overflow-hidden rounded-[28px] border-2 p-3 text-center"
      style={{ boxShadow: "6px 6px 0 var(--border)" }}
    >
      <div
        className="border-ink bg-secondary text-secondary-foreground absolute top-3 right-3 z-10 rounded-full border-2 px-3 py-1 text-xs font-black"
        aria-hidden
      >
        {badge}
      </div>

      <div className="border-ink bg-muted relative aspect-square overflow-hidden rounded-[22px] border-2">
        <img
          src={toOptimizedImageSrc(maintenanceImagePath)}
          alt="小石正在替系統伺服器維護"
          className="size-full object-cover"
          loading="lazy"
          draggable={false}
        />
      </div>

      <div className="grid gap-2 px-1 pb-1">
        <p className="text-muted-foreground text-[11px] font-black tracking-[0.08em] uppercase">
          Maintenance Update
        </p>
        <h2 className="text-[24px] leading-none font-black tracking-normal">
          {title}
        </h2>
        <p className="text-muted-foreground text-sm leading-relaxed font-bold">
          {description}
        </p>
      </div>

      <div
        className="bg-surface-raised border-border rounded-[18px] border-2 px-3 py-2 text-sm font-black"
        aria-label="維護狀態"
      >
        {stats.join("、")}
      </div>
    </section>
  )
}

function noticeRevision(notice: MaintenanceStatus) {
  return notice.startedAt ?? notice.updatedAt ?? notice.mode
}
