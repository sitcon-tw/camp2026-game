import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import {
  CheckCircle2,
  ExternalLink,
  Gift,
  RotateCcw,
  ScanLine,
} from "lucide-react"
import { type FormEvent, useCallback, useRef, useState } from "react"
import { toast } from "sonner"

import { AppError } from "@/shared/api/error"
import { gameApi, type CommunityStandReward } from "@/shared/api/game"
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

type CommunityStandScannerDialogProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
}

const statusMessages: Record<QrCodeScannerStatus, string> = {
  idle: "正在啟動相機",
  starting: "正在啟動相機",
  scanning: "對準社群攤位 QR Code",
  "secure-context-required":
    "瀏覽器需要 HTTPS 或 localhost 才能開啟相機，請輸入攤位 ID。",
  "camera-unavailable": "這個瀏覽器無法開啟相機，請輸入攤位 ID。",
  "permission-denied": "相機權限未開啟，請輸入攤位 ID。",
  "scan-error": "相機已開啟，但無法讀取 QR Code，請輸入攤位 ID。",
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
  const [standID, setStandID] = useState<string | null>(null)
  const activeStandID = standID ?? ""

  const resetScanner = useCallback(() => {
    setStandID(null)
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

  const openStand = useCallback((value: string) => {
    const parsedStandID = parseCommunityStandID(value)
    if (!parsedStandID) {
      setManualMessage("找不到攤位 ID，請確認 QR Code 或手動輸入。")
      return
    }
    setManualMessage(null)
    setStandID(parsedStandID)
  }, [])

  const scannerStatus = useQrCodeScanner({
    open: open && !standID,
    videoRef,
    canvasRef,
    onResult: openStand,
  })
  const standQuery = useQuery({
    queryKey: ["community", "stand", activeStandID],
    queryFn: () => gameApi.communityStand(activeStandID),
    enabled: open && activeStandID.length > 0,
  })
  const claimMutation = useMutation({
    mutationFn: () => gameApi.claimCommunityStand(activeStandID),
    onSuccess: (result) => {
      queryClient.setQueryData(["community", "stand", result.stand.standId], {
        stand: result.stand,
        claimed: true,
      })
      void queryClient.invalidateQueries({ queryKey: ["me"] })
      toast.success(`已領取 ${rewardText(result.reward)}`)
    },
    onError: (error) => {
      if (error instanceof AppError && error.status === 409) {
        queryClient.setQueryData(
          ["community", "stand", activeStandID],
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
    openStand(manualValue)
  }

  return (
    <Dialog open={open} onOpenChange={updateOpen}>
      <DialogContent className="gap-4">
        {!standID ? (
          <>
            <DialogHeader>
              <DialogTitle>掃描社群攤位 QR Code</DialogTitle>
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
                placeholder="輸入攤位 ID 或 QR Code 網址"
                autoComplete="off"
                inputMode="text"
              />
              <DialogFooter>
                <Button type="submit" variant="secondary" className="w-full">
                  前往攤位
                </Button>
              </DialogFooter>
            </form>
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
                <Skeleton className="size-24 rounded-[24px]" />
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
                  <div className="bg-surface-raised border-ink grid size-24 place-items-center overflow-hidden rounded-[24px] border-2">
                    {standQuery.data.stand.logoUrl ? (
                      <img
                        src={standQuery.data.stand.logoUrl}
                        alt=""
                        className="h-full w-full object-cover"
                      />
                    ) : (
                      <Gift className="size-9" aria-hidden />
                    )}
                  </div>
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
                        src={standQuery.data.stand.reward.iconPath}
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
                      standQuery.data.claimed || claimMutation.isPending
                    }
                    onClick={() => claimMutation.mutate()}
                  >
                    {standQuery.data.claimed ? (
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

function parseCommunityStandID(value: string) {
  const trimmed = value.trim()
  if (!trimmed) return null

  try {
    const url = new URL(trimmed)
    const parts = url.pathname.split("/").filter(Boolean)
    if (parts[0] === "community" && parts[1]) return cleanStandID(parts[1])
  } catch {
    // Treat non-URL QR contents as a raw stand ID.
  }

  if (trimmed.startsWith("/")) {
    const parts = trimmed.split("/").filter(Boolean)
    if (parts[0] === "community" && parts[1]) return cleanStandID(parts[1])
  }
  return cleanStandID(trimmed)
}

function cleanStandID(value: string) {
  let decoded = value.trim()
  try {
    decoded = decodeURIComponent(decoded).trim()
  } catch {
    return null
  }
  return /^[a-z0-9][a-z0-9-]{1,80}$/.test(decoded) ? decoded : null
}

function rewardText(reward: CommunityStandReward) {
  if (reward.kind === "open_power") return `${reward.amount ?? 0} 開源力`
  return `${reward.name} x${reward.quantity ?? 1}`
}
