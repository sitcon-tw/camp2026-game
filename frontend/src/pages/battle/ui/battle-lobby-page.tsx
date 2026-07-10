import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { Link, useNavigate } from "@tanstack/react-router"
import { Lock, QrCode, ScanQrCode, UsersRound } from "lucide-react"
import { type ReactNode, useEffect, useState } from "react"
import { toast } from "sonner"

import { MatchCodeScannerDialog } from "@/features/battle-qr"
import {
  AppError,
  isBattleOpeningLockedError,
  isTerminalClientError,
} from "@/shared/api/error"
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
import { PageHeader } from "@/shared/ui/page-header"

function storeMatch(match: MatchState) {
  if (typeof window !== "undefined") {
    window.localStorage.setItem("camp2026.currentMatchId", match.matchId)
  }
}

function clearStoredMatch() {
  if (typeof window !== "undefined") {
    window.localStorage.removeItem("camp2026.currentMatchId")
  }
}

const actionButtonClassName = "h-11 w-full rounded-[14px] text-base font-black"

function LobbyActionCard({
  title,
  description,
  children,
  action,
  locked = false,
}: {
  title: string
  description: string
  children: ReactNode
  action: ReactNode
  locked?: boolean
}) {
  return (
    <Card
      aria-disabled={locked || undefined}
      className={[
        "gap-0 rounded-[22px] px-[15px] py-[15px]",
        locked ? "opacity-65 grayscale" : "",
      ].join(" ")}
    >
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
      !isTerminalClientError(error) && failureCount < 2,
    staleTime: 15_000,
  })
  const openMatch =
    openMatchQuery.data?.status === "waiting" ||
    openMatchQuery.data?.status === "active"
      ? openMatchQuery.data
      : null
  const openMatchIsComputer = openMatch?.mode === "computer"
  const openMatchIsMultiplayer = openMatch?.mode === "multiplayer"
  const openMatchChecking =
    openMatchQuery.isPending || openMatchQuery.isFetching
  useEffect(() => {
    if (!openMatchQuery.isFetchedAfterMount || openMatchQuery.isError) {
      return
    }
    if (openMatch) {
      storeMatch(openMatch)
      return
    }
    clearStoredMatch()
  }, [openMatch, openMatchQuery.isError, openMatchQuery.isFetchedAfterMount])
  const createPairingMutation = useMutation({
    mutationFn: gameApi.createMatchPairing,
    onSuccess: (pairing) => onMatchReady(pairing.match),
    onError: (error) => {
      if (isBattleOpeningLockedError(error)) {
        toast.error("禁止開局")
        return
      }
      if (error instanceof AppError && error.status === 409) {
        queryClient.invalidateQueries({ queryKey: ["matches", "open"] })
        toast.error("已有進行中的對戰，請重新加入")
        return
      }
      toast.error(error instanceof Error ? error.message : "建立現場配對失敗")
    },
  })
  const createMultiplayerMutation = useMutation({
    mutationFn: gameApi.createMultiplayerPairing,
    onSuccess: (pairing) => onMatchReady(pairing.match),
    onError: (error) => {
      if (isBattleOpeningLockedError(error)) {
        toast.error("禁止開局")
        return
      }
      if (error instanceof AppError && error.status === 409) {
        queryClient.invalidateQueries({ queryKey: ["matches", "open"] })
        toast.error("已有進行中的對戰，請重新加入")
        return
      }
      toast.error(error instanceof Error ? error.message : "建立多人對戰失敗")
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
      if (isBattleOpeningLockedError(error)) {
        toast.error("禁止開局")
        return
      }
      if (error instanceof AppError && error.status === 409) {
        queryClient.invalidateQueries({ queryKey: ["matches", "open"] })
        toast.error("已有進行中的對戰，請重新加入")
        return
      }
      toast.error(error instanceof Error ? error.message : "建立電腦對戰失敗")
    },
  })
  const scanPairingMutation = useMutation({
    mutationFn: gameApi.scanMatchPairingToken,
    onSuccess: onMatchReady,
    onError: (error) => {
      if (isBattleOpeningLockedError(error)) {
        toast.error("禁止開局")
        return
      }
      if (error instanceof AppError && error.status === 409) {
        queryClient.invalidateQueries({ queryKey: ["matches", "open"] })
        toast.error("已有進行中的對戰，請重新加入")
        return
      }
      if (error instanceof AppError && error.status === 404) {
        toast.error("QR Code 已失效，請掃描新的對戰 QR")
        return
      }
      toast.error(error instanceof Error ? error.message : "加入對戰失敗")
    },
  })
  const battleActionPending =
    createPairingMutation.isPending ||
    createMultiplayerMutation.isPending ||
    createComputerMutation.isPending ||
    scanPairingMutation.isPending
  const battleOpeningLocked =
    computerSettingsQuery.data?.battleOpeningLocked === true
  const battleOpeningChecking = !openMatch && computerSettingsQuery.isPending
  const battleOpeningBlocked = !openMatch && battleOpeningLocked

  function handleStartPairing() {
    if (battleActionPending) return
    if (openMatch) {
      onMatchReady(openMatch)
      return
    }
    if (battleOpeningLocked) {
      toast.error("禁止開局")
      return
    }
    createPairingMutation.mutate()
  }

  function handleScanToken(token: string) {
    if (battleActionPending) return
    if (openMatch) {
      onMatchReady(openMatch)
      return
    }
    if (battleOpeningLocked) {
      toast.error("禁止開局")
      return
    }
    scanPairingMutation.mutate(token)
  }

  function handleComputerBattle() {
    if (battleActionPending) return
    if (openMatch) {
      onMatchReady(openMatch)
      return
    }
    if (battleOpeningLocked) {
      toast.error("禁止開局")
      return
    }
    createComputerMutation.mutate()
  }

  function handleMultiplayerBattle() {
    if (battleActionPending) return
    if (openMatch) {
      onMatchReady(openMatch)
      return
    }
    if (battleOpeningLocked) {
      toast.error("禁止開局")
      return
    }
    createMultiplayerMutation.mutate()
  }

  return (
    <GamePageShell contentClassName="grid content-start gap-y-3">
      <PageHeader title="知識王" headline="Battle Lobby" />
      <LobbyActionCard
        title="現場配對"
        locked={battleOpeningBlocked}
        description={
          openMatch
            ? "回到尚未結束的知識王對戰"
            : battleOpeningLocked
              ? "目前禁止開局"
              : "用短效 QR Code 當面加入雙人知識王"
        }
        action={
          <div className="grid w-full gap-2 sm:grid-cols-2">
            <Button
              type="button"
              className={actionButtonClassName}
              disabled={
                openMatchChecking ||
                battleOpeningChecking ||
                battleActionPending ||
                battleOpeningBlocked
              }
              onClick={handleStartPairing}
            >
              {battleOpeningBlocked ? (
                <Lock className="size-4" />
              ) : (
                <QrCode className="size-4" />
              )}
              {openMatchChecking || battleOpeningChecking
                ? "同步中"
                : openMatch
                  ? "回到目前對戰"
                  : battleOpeningLocked
                    ? "禁止進入"
                    : createPairingMutation.isPending
                      ? "建立中"
                      : "顯示配對 QR"}
            </Button>
            <Button
              type="button"
              className={actionButtonClassName}
              variant="secondary"
              disabled={
                Boolean(openMatch) ||
                openMatchChecking ||
                battleOpeningChecking ||
                battleActionPending ||
                battleOpeningBlocked
              }
              onClick={() => setScannerOpen(true)}
            >
              {battleOpeningBlocked ? (
                <Lock className="size-4" />
              ) : (
                <ScanQrCode className="size-4" />
              )}
              {battleOpeningBlocked
                ? "禁止進入"
                : scanPairingMutation.isPending
                  ? "加入中"
                  : "掃描配對 QR"}
            </Button>
          </div>
        }
      >
        {openMatch
          ? "對戰已經開始或仍在等待房，可以直接回到原本的對戰。"
          : battleOpeningLocked
            ? "目前禁止開局，請等待工作人員重新開放知識王。"
            : "一位玩家顯示 QR Code，另一位玩家在現場掃描加入；QR Code 會持續更新。"}
      </LobbyActionCard>
      <LobbyActionCard
        title="多人對戰"
        locked={battleOpeningBlocked}
        description={
          openMatch
            ? openMatchIsMultiplayer
              ? "回到尚未結束的多人對戰"
              : "已有尚未結束的知識王對戰"
            : battleOpeningLocked
              ? "目前禁止開局"
              : "四人一局的知識王對戰"
        }
        action={
          <div className="grid w-full gap-2 sm:grid-cols-2">
            <Button
              type="button"
              className={actionButtonClassName}
              variant="secondary"
              disabled={
                openMatchChecking ||
                battleOpeningChecking ||
                battleActionPending ||
                battleOpeningBlocked
              }
              onClick={handleMultiplayerBattle}
            >
              {battleOpeningBlocked ? (
                <Lock className="size-4" />
              ) : (
                <UsersRound className="size-4" />
              )}
              {openMatchChecking || battleOpeningChecking
                ? "同步中"
                : openMatch
                  ? openMatchIsMultiplayer
                    ? "重新加入多人對戰"
                    : "回到目前對戰"
                  : battleOpeningLocked
                    ? "禁止進入"
                    : createMultiplayerMutation.isPending
                      ? "建立中"
                      : "建立多人房"}
            </Button>
            <Button
              type="button"
              className={actionButtonClassName}
              variant="secondary"
              disabled={
                Boolean(openMatch) ||
                openMatchChecking ||
                battleOpeningChecking ||
                battleActionPending ||
                battleOpeningBlocked
              }
              onClick={() => setScannerOpen(true)}
            >
              {battleOpeningBlocked ? (
                <Lock className="size-4" />
              ) : (
                <ScanQrCode className="size-4" />
              )}
              {battleOpeningBlocked
                ? "禁止進入"
                : scanPairingMutation.isPending
                  ? "加入中"
                  : "掃描多人 QR"}
            </Button>
          </div>
        }
      >
        {openMatch
          ? openMatchIsMultiplayer
            ? "多人對戰已經開始或仍在等待房，可以直接回到原本的對戰。"
            : "你目前有尚未結束的對戰，先回到原本的對戰再開始多人對戰。"
          : battleOpeningLocked
            ? "目前禁止開局，請等待工作人員重新開放知識王。"
            : "四位玩家進入同一局，比拚答題速度與正確率。"}
      </LobbyActionCard>
      <LobbyActionCard
        title="電腦對戰"
        locked={battleOpeningBlocked}
        description={
          openMatch
            ? openMatchIsComputer
              ? "回到尚未結束的電腦對戰"
              : "已有尚未結束的知識王對戰"
            : battleOpeningLocked
              ? "目前禁止開局"
              : "和系統控制的電腦對手進行知識王戰"
        }
        action={
          <Button
            type="button"
            className={actionButtonClassName}
            variant="secondary"
            disabled={
              openMatchChecking ||
              battleOpeningChecking ||
              battleActionPending ||
              battleOpeningBlocked ||
              (!openMatch && !computerSettingsQuery.data?.enabled)
            }
            onClick={handleComputerBattle}
          >
            {battleOpeningBlocked ? (
              <Lock className="size-4" />
            ) : (
              <GameFeatureIcon name="battle" className="size-4" />
            )}
            {openMatchChecking || battleOpeningChecking
              ? "同步中"
              : openMatch
                ? openMatchIsComputer
                  ? "重新加入電腦對戰"
                  : "回到目前對戰"
                : battleOpeningLocked
                  ? "禁止進入"
                  : computerSettingsQuery.data?.enabled
                    ? createComputerMutation.isPending
                      ? "建立中"
                      : "跟電腦對戰"
                    : "電腦對戰未開放"}
          </Button>
        }
      >
        {openMatch
          ? openMatchIsComputer
            ? "電腦對戰已經開始或仍在等待房，可以直接回到原本的對戰。"
            : "你目前有尚未結束的對戰，先回到原本的對戰再開始電腦對戰。"
          : battleOpeningLocked
            ? "目前禁止開局，電腦對戰也會暫停建立新房。"
            : "沒有真人對手時，也可以完成對戰並取得結算獎勵。"}
      </LobbyActionCard>
      <LobbyActionCard
        title="開源戰線"
        description="和小隊一起爭奪地圖領土"
        action={
          <Button asChild className={actionButtonClassName} variant="secondary">
            <Link to="/front">
              <GameFeatureIcon name="front" className="size-5" />
              前往開源戰線
            </Link>
          </Button>
        }
      >
        使用開源力部署行動，和隊友一起推進戰線、爭奪領土。
      </LobbyActionCard>
      {scannerOpen ? (
        <MatchCodeScannerDialog
          open={scannerOpen}
          onOpenChange={setScannerOpen}
          onCode={(scannedToken) => {
            setScannerOpen(false)
            handleScanToken(scannedToken)
          }}
        />
      ) : null}
    </GamePageShell>
  )
}
