import { useQuery } from "@tanstack/react-query"
import { useLocation, useNavigate } from "@tanstack/react-router"
import type { ReactNode } from "react"
import { useEffect } from "react"

import { AppError } from "@/shared/api/error"
import { gameApi } from "@/shared/api/game"
import { Button } from "@/shared/ui/button"
import { Card, CardContent } from "@/shared/ui/card"
import { GamePageShell } from "@/shared/ui/game-page-shell"
import { Spinner } from "@/shared/ui/spinner"
import { RewardAlertCenter } from "@/features/reward-alert/ui/reward-alert-center"

const publicPathPrefixes = ["/login", "/codex", "/admin"] as const

function isPublicPath(pathname: string) {
  return publicPathPrefixes.some(
    (path) => pathname === path || pathname.startsWith(`${path}/`),
  )
}

export function AuthGate({ children }: { children: ReactNode }) {
  const location = useLocation()
  const navigate = useNavigate()
  const protectedRoute = !isPublicPath(location.pathname)
  const statusQuery = useQuery({
    queryKey: ["me", "status"],
    queryFn: gameApi.status,
    enabled: protectedRoute,
  })
  const unauthorized =
    statusQuery.error instanceof AppError && statusQuery.error.status === 401

  useEffect(() => {
    if (protectedRoute && unauthorized) {
      navigate({ to: "/login", replace: true })
    }
  }, [navigate, protectedRoute, unauthorized])

  if (!protectedRoute) {
    return children
  }

  if (statusQuery.isPending) {
    return (
      <GamePageShell contentClassName="justify-center">
        <Card className="border-ink w-full max-w-sm rounded-[22px] border-2">
          <CardContent className="flex items-center gap-3 p-5">
            <Spinner className="size-5" />
            <span className="font-black">正在確認登入狀態</span>
          </CardContent>
        </Card>
      </GamePageShell>
    )
  }

  if (unauthorized) {
    return null
  }

  if (statusQuery.error) {
    return (
      <GamePageShell contentClassName="justify-center">
        <Card className="border-ink w-full max-w-sm rounded-[22px] border-2">
          <CardContent className="grid gap-3 p-5">
            <h1 className="text-2xl font-black">無法連線到遊戲服務</h1>
            <p className="text-muted-foreground leading-relaxed">
              請確認後端 API 正在執行，然後重新整理頁面。
            </p>
            <Button onClick={() => statusQuery.refetch()}>重新檢查</Button>
          </CardContent>
        </Card>
      </GamePageShell>
    )
  }

  return (
    <>
      <RewardAlertCenter />
      {children}
    </>
  )
}
