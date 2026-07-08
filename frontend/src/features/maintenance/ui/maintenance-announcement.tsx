import { useEffect, useState } from "react"
import { useQuery } from "@tanstack/react-query"
import { useLocation } from "@tanstack/react-router"
import { Megaphone } from "lucide-react"

import { gameApi, type MaintenanceStatus } from "@/shared/api/game"
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogMedia,
  AlertDialogTitle,
} from "@/shared/ui/alert-dialog"
import { Button } from "@/shared/ui/button"

export function MaintenanceAnnouncement() {
  const [lastNotice, setLastNotice] = useState<MaintenanceStatus | null>(null)
  const [dismissedRevision, setDismissedRevision] = useState<string | null>(
    null,
  )
  const location = useLocation()
  const maintenanceQuery = useQuery({
    queryKey: ["maintenance"],
    queryFn: gameApi.maintenanceStatus,
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

  if (!notice) return null

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
    <AlertDialog open>
      <AlertDialogContent size="sm">
        <AlertDialogHeader>
          <AlertDialogMedia>
            <Megaphone className="text-primary size-8" />
          </AlertDialogMedia>
          <AlertDialogTitle>{title}</AlertDialogTitle>
          <AlertDialogDescription className="grid gap-2 text-center leading-6">
            <span>{description}</span>
            {!completed ? (
              <span>
                目前尚有 {notice.activeMatchCount} 場進行中對戰、
                {notice.openMatchCount} 間等待或進行中的房間。
              </span>
            ) : null}
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          {completed ? (
            <AlertDialogAction
              onClick={() => setDismissedRevision(noticeRevision(notice))}
            >
              {buttonText}
            </AlertDialogAction>
          ) : (
            <Button disabled>{buttonText}</Button>
          )}
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  )
}

function noticeRevision(notice: MaintenanceStatus) {
  return notice.startedAt ?? notice.updatedAt ?? notice.mode
}
