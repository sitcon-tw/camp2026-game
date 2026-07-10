import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import {
  CheckCircle2,
  Clock,
  ExternalLink,
  Gift,
  RotateCcw,
  ScanLine,
} from "lucide-react"
import { type FormEvent, useCallback, useEffect, useRef, useState } from "react"
import { toast } from "sonner"

import { AppError } from "@/shared/api/error"
import { gameApi, type CommunityStandReward } from "@/shared/api/game"
import { parseCommunityStandQRToken } from "@/shared/lib/community-stand-qr"
import { parseRoomTeamQRToken } from "@/shared/lib/room-team-qr"
import { parseStaffRewardToken } from "@/shared/lib/staff-reward-token"
import {
  useQrCodeScanner,
  type QrCodeScannerStatus,
} from "@/shared/hooks/use-qr-code-scanner"
import { Button } from "@/shared/ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/shared/ui/dialog"
import { Input } from "@/shared/ui/input"
import { Skeleton } from "@/shared/ui/skeleton"
import { toOptimizedImageSrc } from "@/shared/utils/image-src"

type CommunityStandScannerDialogProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
}

const statusMessages: Record<QrCodeScannerStatus, string> = {
  idle: "正在啟動相機",
  starting: "正在啟動相機",
  scanning: "對準 QR Code",
  "secure-context-required":
    "瀏覽器需要 HTTPS 或 localhost 才能開啟相機，請輸入 QR Code 內容。",
  "camera-unavailable": "這個瀏覽器無法開啟相機，請輸入 QR Code 內容。",
  "permission-denied": "相機權限未開啟，請輸入 QR Code 內容。",
  "scan-error": "相機已開啟，但無法讀取 QR Code，請輸入 QR Code 內容。",
}

export function CommunityStandScannerDialog({
  open,
  onOpenChange,
}: CommunityStandScannerDialogProps) {
  const queryClient = useQueryClient()
  const videoRef = useRef<HTMLVideoElement>(null)
  const canvasRef = useRef<HTMLCanvasElement>(null)
  const [manualValue, setManualValue] = useState("")
  const [manualMessage, setManualMessage] = useState<string | null>(null)
  const [qrToken, setQRToken] = useState<string | null>(null)
  const [staffRewardToken, setStaffRewardToken] = useState<string | null>(null)
  const [qrClaimCooldownUntil, setQRClaimCooldownUntil] = useState<
    string | null
  >(null)
  const [qrClaimCooldownActive, setQRClaimCooldownActive] = useState(false)
  const [roomTeamToken, setRoomTeamToken] = useState<string | null>(null)
  const activeQRToken = qrToken ?? ""
  const activeStaffRewardToken = staffRewardToken ?? ""
  const activeRoomTeamToken = roomTeamToken ?? ""

  useEffect(() => {
    if (!qrClaimCooldownUntil) return
    const expiresAt = Date.parse(qrClaimCooldownUntil)
    if (!Number.isFinite(expiresAt)) return
    const delay = Math.max(0, expiresAt - Date.now())

    const timer = window.setTimeout(() => {
      setQRClaimCooldownUntil(null)
      setQRClaimCooldownActive(false)
    }, delay)
    return () => window.clearTimeout(timer)
  }, [qrClaimCooldownUntil])

  const activateQRClaimCooldown = useCallback((until?: string) => {
    if (!until) {
      setQRClaimCooldownUntil(null)
      setQRClaimCooldownActive(false)
      return
    }
    const expiresAt = Date.parse(until)
    if (!Number.isFinite(expiresAt) || expiresAt <= Date.now()) {
      setQRClaimCooldownUntil(null)
      setQRClaimCooldownActive(false)
      return
    }
    setQRClaimCooldownUntil(until)
    setQRClaimCooldownActive(true)
  }, [])

  const resetScanner = useCallback(() => {
    setQRToken(null)
    setStaffRewardToken(null)
    setRoomTeamToken(null)
    setManualValue("")
    setManualMessage(null)
  }, [])

  const updateOpen = useCallback(
    (nextOpen: boolean) => {
      if (!nextOpen) resetScanner()
      onOpenChange(nextOpen)
    },
    [onOpenChange, resetScanner],
  )

  const staffRewardClaimMutation = useMutation({
    mutationFn: () => gameApi.claimStaffRewardToken(activeStaffRewardToken),
    onSuccess: (result) => {
      activateQRClaimCooldown(result.qrCodeScanCooldownUntil)
      void queryClient.invalidateQueries({ queryKey: ["me"] })
      toast.success(`已領取 ${rewardText(result.reward)}`)
    },
    onError: (error) => {
      if (error instanceof AppError && error.status === 409) {
        toast.error("這個獎勵 Token 已經領取過了")
        return
      }
      toast.error(error instanceof AppError ? error.message : "領取失敗")
    },
  })
  const roomTeamJoinMutation = useMutation({
    mutationFn: (token: string) => gameApi.joinRoomTeamByQRToken(token),
    onSuccess: (result) => {
      toast.success(
        result.joined
          ? `已加入 ${result.room.roomNumber} 宿舍群組`
          : `已在 ${result.room.roomNumber} 宿舍群組`,
      )
    },
    onError: (error) => {
      toast.error(error instanceof AppError ? error.message : "加入宿舍失敗")
    },
  })
  const activeRoomTeamJoinPending =
    roomTeamJoinMutation.isPending &&
    roomTeamJoinMutation.variables === activeRoomTeamToken
  const activeRoomTeamJoined =
    roomTeamJoinMutation.isSuccess &&
    roomTeamJoinMutation.variables === activeRoomTeamToken
  const activeRoomTeamJoinError =
    roomTeamJoinMutation.isError &&
    roomTeamJoinMutation.variables === activeRoomTeamToken
  const openScannedValue = useCallback(
    (value: string) => {
      const parsedRewardToken = parseStaffRewardToken(value)
      if (parsedRewardToken) {
        setManualMessage(null)
        setStaffRewardToken(parsedRewardToken)
        return
      }

      const parsedRoomTeamToken = parseRoomTeamQRToken(value)
      if (parsedRoomTeamToken) {
        setManualMessage(null)
        setRoomTeamToken(parsedRoomTeamToken)
        roomTeamJoinMutation.mutate(parsedRoomTeamToken)
        return
      }

      const parsedQRToken = parseCommunityStandQRToken(value)
      if (!parsedQRToken) {
        setManualMessage("找不到 QR Code 內容，請確認後再手動輸入。")
        return
      }
      setManualMessage(null)
      setQRToken(parsedQRToken)
    },
    [roomTeamJoinMutation],
  )
  const scannerStatus = useQrCodeScanner({
    open: open && !qrToken && !staffRewardToken && !roomTeamToken,
    videoRef,
    canvasRef,
    onResult: openScannedValue,
  })
  const standQuery = useQuery({
    queryKey: ["community", "stand", "scan", activeQRToken],
    queryFn: () => gameApi.communityStandByQRToken(activeQRToken),
    enabled: open && activeQRToken.length > 0,
  })
  const claimMutation = useMutation({
    mutationFn: () => gameApi.claimCommunityStandByQRToken(activeQRToken),
    onSuccess: (result) => {
      activateQRClaimCooldown(result.qrCodeScanCooldownUntil)
      queryClient.setQueryData(["community", "stand", "scan", activeQRToken], {
        stand: result.stand,
        claimed: true,
      })
      void queryClient.invalidateQueries({ queryKey: ["me"] })
      toast.success(`已領取 ${rewardText(result.reward)}`)
    },
    onError: (error) => {
      if (error instanceof AppError && error.status === 409) {
        queryClient.setQueryData(
          ["community", "stand", "scan", activeQRToken],
          (current: unknown) => {
            if (!current || typeof current !== "object") return current
            return { ...current, claimed: true }
          },
        )
        toast.error("這個攤位的獎勵已經領取過了")
        return
      }
      toast.error(error instanceof AppError ? error.message : "領取失敗")
    },
  })

  function handleManualSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    openScannedValue(manualValue)
  }

  return (
    <Dialog open={open} onOpenChange={updateOpen}>
      <DialogContent className="gap-4">
        {!qrToken && !staffRewardToken && !roomTeamToken ? (
          <>
            <DialogHeader>
              <DialogTitle>QRCode 掃描器</DialogTitle>
              <DialogDescription>
                {manualMessage ?? statusMessages[scannerStatus]}
              </DialogDescription>
            </DialogHeader>

            <div className="bg-ink border-ink aspect-square overflow-hidden rounded-[18px] border-2">
              <video
                ref={videoRef}
                className="h-full w-full object-cover"
                playsInline
                muted
              />
              <canvas ref={canvasRef} className="hidden" />
            </div>

            <form className="grid gap-3" onSubmit={handleManualSubmit}>
              <Input
                value={manualValue}
                onChange={(event) => {
                  setManualValue(event.target.value)
                  setManualMessage(null)
                }}
                placeholder="輸入 QR Code 內容"
                autoComplete="off"
                inputMode="text"
              />
              <DialogFooter>
                <Button type="submit" variant="secondary" className="w-full">
                  確認 QR Code
                </Button>
              </DialogFooter>
            </form>
          </>
        ) : staffRewardToken ? (
          <>
            <DialogHeader>
              <DialogTitle>工作人員獎勵</DialogTitle>
              <DialogDescription>
                掃描成功，確認後領取工作人員設定的獎勵。
              </DialogDescription>
            </DialogHeader>

            <div className="bg-surface-raised border-ink rounded-[18px] border-2 p-4">
              <p className="text-muted-foreground mb-1 text-xs font-black tracking-[0.08em] uppercase">
                TOKEN
              </p>
              <code className="block text-sm break-all">
                {staffRewardToken}
              </code>
            </div>

            <DialogFooter className="grid grid-cols-1 gap-2 sm:grid-cols-[auto_1fr]">
              <Button type="button" variant="secondary" onClick={resetScanner}>
                <RotateCcw className="size-4" aria-hidden />
                重掃
              </Button>
              <Button
                type="button"
                disabled={
                  staffRewardClaimMutation.isPending ||
                  staffRewardClaimMutation.isSuccess ||
                  qrClaimCooldownActive
                }
                onClick={() => staffRewardClaimMutation.mutate()}
              >
                {qrClaimCooldownActive ? (
                  <>
                    <Clock className="size-4" aria-hidden />
                    領取冷卻中
                  </>
                ) : staffRewardClaimMutation.isSuccess ? (
                  <>
                    <CheckCircle2 className="size-4" aria-hidden />
                    已領取
                  </>
                ) : staffRewardClaimMutation.isPending ? (
                  "領取中"
                ) : (
                  <>
                    <Gift className="size-4" aria-hidden />
                    領取獎勵
                  </>
                )}
              </Button>
            </DialogFooter>
          </>
        ) : roomTeamToken ? (
          <>
            <DialogHeader>
              <DialogTitle>宿舍房號</DialogTitle>
              <DialogDescription>
                {activeRoomTeamJoinPending
                  ? "正在加入宿舍群組"
                  : activeRoomTeamJoined
                    ? "已加入宿舍群組"
                    : activeRoomTeamJoinError
                      ? "宿舍群組加入失敗"
                      : "掃描成功"}
              </DialogDescription>
            </DialogHeader>

            <div className="bg-surface-raised border-ink rounded-[18px] border-2 p-4">
              <p className="text-muted-foreground mb-1 text-xs font-black tracking-[0.08em] uppercase">
                ROOM TOKEN
              </p>
              <code className="block text-sm break-all">{roomTeamToken}</code>
            </div>

            <DialogFooter className="grid grid-cols-1 gap-2 sm:grid-cols-[auto_1fr]">
              <Button type="button" variant="secondary" onClick={resetScanner}>
                <RotateCcw className="size-4" aria-hidden />
                重掃
              </Button>
              {activeRoomTeamJoinError ? (
                <Button
                  type="button"
                  onClick={() =>
                    roomTeamJoinMutation.mutate(activeRoomTeamToken)
                  }
                >
                  <RotateCcw className="size-4" aria-hidden />
                  重試加入
                </Button>
              ) : (
                <Button type="button" disabled>
                  {activeRoomTeamJoined ? (
                    <>
                      <CheckCircle2 className="size-4" aria-hidden />
                      已加入
                    </>
                  ) : activeRoomTeamJoinPending ? (
                    "加入中"
                  ) : (
                    <>
                      <CheckCircle2 className="size-4" aria-hidden />
                      準備加入
                    </>
                  )}
                </Button>
              )}
            </DialogFooter>
          </>
        ) : (
          <>
            <DialogHeader>
              <DialogTitle>社群攤位</DialogTitle>
              <DialogDescription>
                {standQuery.isPending
                  ? "正在讀取攤位資訊"
                  : standQuery.error
                    ? "無法讀取這個攤位"
                    : "掃描成功"}
              </DialogDescription>
            </DialogHeader>

            {standQuery.isPending ? (
              <div className="grid justify-items-center gap-4 py-2">
                <Skeleton className="h-36 w-full max-w-sm rounded-[24px]" />
                <Skeleton className="h-8 w-3/4" />
                <Skeleton className="h-24 w-full" />
              </div>
            ) : standQuery.error ? (
              <div className="grid gap-3">
                <div className="bg-surface-raised border-ink rounded-[18px] border-2 p-4">
                  <h3 className="text-xl font-black">找不到這個攤位</h3>
                  <p className="text-muted-foreground mt-2 leading-relaxed">
                    請確認 QR Code 是否為本次活動的社群攤位 QR Code。
                  </p>
                </div>
                <DialogFooter>
                  <Button
                    type="button"
                    variant="secondary"
                    className="w-full"
                    onClick={resetScanner}
                  >
                    <ScanLine className="size-4" aria-hidden />
                    重新掃描
                  </Button>
                </DialogFooter>
              </div>
            ) : (
              <div className="grid gap-4">
                <div className="grid justify-items-center gap-3 text-center">
                  {standQuery.data.stand.logoUrl ? (
                    <div className="bg-surface-raised border-ink w-full max-w-sm overflow-hidden rounded-[24px] border-2 p-3">
                      <img
                        src={standQuery.data.stand.logoUrl}
                        alt=""
                        className="block h-28 w-full object-contain"
                      />
                    </div>
                  ) : (
                    <div className="bg-surface-raised border-ink grid size-24 place-items-center rounded-[24px] border-2">
                      <Gift className="size-9" aria-hidden />
                    </div>
                  )}
                  <div>
                    <h3 className="text-[26px] leading-tight font-black">
                      {standQuery.data.stand.name}
                    </h3>
                    <p className="text-muted-foreground mt-2 leading-relaxed whitespace-pre-line">
                      {standQuery.data.stand.description}
                    </p>
                  </div>
                </div>

                {standQuery.data.stand.websiteUrl ? (
                  <Button asChild variant="outline" className="w-full">
                    <a
                      href={standQuery.data.stand.websiteUrl}
                      target="_blank"
                      rel="noreferrer"
                    >
                      <ExternalLink className="size-4" aria-hidden />
                      社群網站
                    </a>
                  </Button>
                ) : null}

                <div className="bg-surface-raised border-ink rounded-[18px] border-2 p-4">
                  <p className="text-muted-foreground mb-1 text-xs font-black tracking-[0.08em] uppercase">
                    REWARD
                  </p>
                  <div className="flex items-center gap-3">
                    {standQuery.data.stand.reward.iconPath ? (
                      <img
                        src={toOptimizedImageSrc(
                          standQuery.data.stand.reward.iconPath,
                        )}
                        alt=""
                        className="border-ink bg-card size-12 rounded-[14px] border-2 object-cover"
                      />
                    ) : (
                      <div className="border-ink bg-card grid size-12 place-items-center rounded-[14px] border-2">
                        <Gift className="size-5" aria-hidden />
                      </div>
                    )}
                    <div className="min-w-0">
                      <p className="font-black">
                        {rewardText(standQuery.data.stand.reward)}
                      </p>
                      <p className="text-muted-foreground text-sm font-bold">
                        每位學員每個攤位限領一次
                      </p>
                    </div>
                  </div>
                </div>

                <DialogFooter className="grid grid-cols-1 gap-2 sm:grid-cols-[auto_1fr]">
                  <Button
                    type="button"
                    variant="secondary"
                    onClick={resetScanner}
                  >
                    <RotateCcw className="size-4" aria-hidden />
                    重掃
                  </Button>
                  <Button
                    type="button"
                    disabled={
                      standQuery.data.claimed ||
                      claimMutation.isPending ||
                      qrClaimCooldownActive
                    }
                    onClick={() => claimMutation.mutate()}
                  >
                    {qrClaimCooldownActive ? (
                      <>
                        <Clock className="size-4" aria-hidden />
                        領取冷卻中
                      </>
                    ) : standQuery.data.claimed ? (
                      <>
                        <CheckCircle2 className="size-4" aria-hidden />
                        已領取
                      </>
                    ) : claimMutation.isPending ? (
                      "領取中"
                    ) : (
                      <>
                        <Gift className="size-4" aria-hidden />
                        領取獎勵
                      </>
                    )}
                  </Button>
                </DialogFooter>
              </div>
            )}
          </>
        )}
      </DialogContent>
    </Dialog>
  )
}

function rewardText(reward: CommunityStandReward) {
  if (reward.kind === "open_power") return `${reward.amount ?? 0} 開源力`
  return `${reward.name} x${reward.quantity ?? 1}`
}
