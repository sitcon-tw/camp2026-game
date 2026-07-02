import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { useNavigate } from "@tanstack/react-router"
import { ArrowRight, ScanQrCode } from "lucide-react"
import { type ReactNode, useState } from "react"
import { toast } from "sonner"

import {
  MatchCodeScannerDialog,
  normalizeMatchCode,
} from "@/features/battle-qr"
import { AppError } from "@/shared/api/error"
import { gameApi, type MatchState } from "@/shared/api/game"
import { Button } from "@/shared/ui/button"
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/shared/ui/card"
import { GameFeatureIcon } from "@/shared/ui/game-feature-icon"
import { GamePageShell } from "@/shared/ui/game-page-shell"
import { Input } from "@/shared/ui/input"
import { PageHeader } from "@/shared/ui/page-header"

function storeMatch(match: MatchState) {
  if (typeof window !== "undefined") {
    window.localStorage.setItem("camp2026.currentMatchId", match.matchId)
  }
}

const actionButtonClassName = "h-11 w-full rounded-[14px] text-base font-black"

function LobbyActionCard({
  title,
  description,
  children,
  action,
}: {
  title: string
  description: string
  children: ReactNode
  action: ReactNode
}) {
  return (
    <Card className="gap-0 rounded-[22px] px-[15px] py-[15px]">
      <CardHeader className="gap-1 px-0">
        <CardTitle className="text-[22px] leading-tight font-black tracking-normal">
          {title}
        </CardTitle>
        <CardDescription className="text-[13px] leading-[1.45] font-black">
          {description}
        </CardDescription>
      </CardHeader>
      <CardContent className="px-0 pt-3">
        <p className="text-[15px] leading-[1.65] font-bold">{children}</p>
      </CardContent>
      <CardFooter className="px-0 pt-4">{action}</CardFooter>
    </Card>
  )
}

export function BattleLobbyPage() {
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const [code, setCode] = useState("")
  const [scannerOpen, setScannerOpen] = useState(false)
  const onMatchReady = (match: MatchState) => {
    storeMatch(match)
    navigate({ to: "/battle/room" })
  }
  const openMatchQuery = useQuery({
    queryKey: ["matches", "open"],
    queryFn: async () => {
      try {
        return await gameApi.openMatch()
      } catch (error) {
        if (error instanceof AppError && error.status === 404) return null
        throw error
      }
    },
    retry: (failureCount, error) =>
      !(error instanceof AppError && error.status < 500) && failureCount < 2,
  })
  const openMatch =
    openMatchQuery.data?.status === "waiting" ||
    openMatchQuery.data?.status === "active"
      ? openMatchQuery.data
      : null
  const createMutation = useMutation({
    mutationFn: gameApi.createMatch,
    onSuccess: onMatchReady,
    onError: (error) => {
      if (error instanceof AppError && error.status === 409) {
        queryClient.invalidateQueries({ queryKey: ["matches", "open"] })
        toast.error("已有進行中的對戰，請重新加入")
        return
      }
      toast.error(error instanceof Error ? error.message : "建立房間失敗")
    },
  })
  const computerSettingsQuery = useQuery({
    queryKey: ["matches", "computer", "settings"],
    queryFn: gameApi.computerBattleSettings,
  })
  const createComputerMutation = useMutation({
    mutationFn: gameApi.createComputerMatch,
    onSuccess: onMatchReady,
    onError: (error) => {
      toast.error(error instanceof Error ? error.message : "建立電腦對戰失敗")
    },
  })
  const joinMutation = useMutation({
    mutationFn: gameApi.joinMatch,
    onSuccess: onMatchReady,
    onError: (error) => {
      toast.error(error instanceof Error ? error.message : "加入房間失敗")
    },
  })

  function handleJoinCode(value: string) {
    const normalizedCode = normalizeMatchCode(value)
    if (!normalizedCode) {
      toast.error("請先輸入房號")
      return
    }
    joinMutation.mutate(normalizedCode)
  }

  function handleJoin() {
    handleJoinCode(code)
  }

  function handleQuickStart() {
    if (openMatch) {
      onMatchReady(openMatch)
      return
    }
    createMutation.mutate()
  }

  return (
    <GamePageShell contentClassName="grid content-start gap-y-3">
      <PageHeader title="知識王" headline="Battle Lobby" />
      <LobbyActionCard
        title="電腦對戰"
        description="和系統控制的電腦對手進行知識王戰"
        action={
          <Button
            type="button"
            className={actionButtonClassName}
            variant="secondary"
            disabled={
              computerSettingsQuery.isPending ||
              createComputerMutation.isPending ||
              !computerSettingsQuery.data?.enabled
            }
            onClick={() => createComputerMutation.mutate()}
          >
            <GameFeatureIcon name="battle" className="size-4" />
            {computerSettingsQuery.isPending
              ? "同步中"
              : computerSettingsQuery.data?.enabled
                ? createComputerMutation.isPending
                  ? "建立中"
                  : "跟電腦對戰"
                : "電腦對戰未開放"}
          </Button>
        }
      >
        沒有真人對手時，也可以完成對戰並取得結算獎勵。
      </LobbyActionCard>

      <LobbyActionCard
        title="快速開始"
        description={
          openMatch ? "回到尚未結束的知識王對戰" : "建立一個雙人知識王房間"
        }
        action={
          <Button
            type="button"
            className={actionButtonClassName}
            disabled={openMatchQuery.isPending || createMutation.isPending}
            onClick={handleQuickStart}
          >
            <GameFeatureIcon name="battle" className="size-4" />
            {openMatchQuery.isPending
              ? "同步中"
              : openMatch
                ? "重新加入對戰"
                : createMutation.isPending
                  ? "建立中"
                  : "建立房間"}
          </Button>
        }
      >
        {openMatch
          ? "對戰已經開始或仍在等待房，可以直接回到原本的對戰。"
          : "建立房間後，把房號分享給另一位學員加入對戰。"}
      </LobbyActionCard>

      <LobbyActionCard
        title="多人連線"
        description="使用房號加入或回到對戰"
        action={
          <div className="grid w-full grid-cols-[minmax(0,1fr)_44px_minmax(104px,1fr)] items-center gap-2">
            <Input
              id="input-room-id"
              type="text"
              className="h-11 rounded-[14px] px-3 text-[15px] font-black"
              value={code}
              onChange={(event) =>
                setCode(normalizeMatchCode(event.target.value))
              }
              placeholder="請輸入房號"
            />
            <Button
              className="size-11 rounded-[14px]"
              size="icon"
              type="button"
              aria-label="掃描房號 QR Code"
              disabled={joinMutation.isPending}
              onClick={() => setScannerOpen(true)}
            >
              <ScanQrCode />
            </Button>
            <Button
              className="h-11 rounded-[14px] px-3 text-[15px] font-black"
              variant="secondary"
              type="button"
              disabled={joinMutation.isPending}
              onClick={handleJoin}
            >
              {joinMutation.isPending ? "加入中" : "加入房間"}
              <ArrowRight />
            </Button>
          </div>
        }
      >
        和其他學員連線對戰，比拼誰才是知識王。
      </LobbyActionCard>
      {scannerOpen ? (
        <MatchCodeScannerDialog
          open={scannerOpen}
          onOpenChange={setScannerOpen}
          onCode={(scannedCode) => {
            setCode(scannedCode)
            handleJoinCode(scannedCode)
          }}
        />
      ) : null}
    </GamePageShell>
  )
}
