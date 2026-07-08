import { useCallback, useRef } from "react"

import { normalizeMatchCode } from "@/features/battle-qr/lib/match-code"
import {
  useQrCodeScanner,
  type QrCodeScannerStatus,
} from "@/shared/hooks/use-qr-code-scanner"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/shared/ui/dialog"

type MatchCodeScannerDialogProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  onCode: (code: string) => void
}

const statusMessages: Record<QrCodeScannerStatus, string> = {
  idle: "正在啟動相機",
  starting: "正在啟動相機",
  scanning: "對準現場配對 QR Code",
  "secure-context-required": "瀏覽器需要 HTTPS 或 localhost 才能開啟相機。",
  "camera-unavailable": "這個瀏覽器無法開啟相機。",
  "permission-denied": "相機權限未開啟。",
  "scan-error": "相機已開啟，但無法讀取 QR Code。",
}

export function MatchCodeScannerDialog({
  open,
  onOpenChange,
  onCode,
}: MatchCodeScannerDialogProps) {
  const videoRef = useRef<HTMLVideoElement>(null)
  const canvasRef = useRef<HTMLCanvasElement>(null)

  const updateOpen = useCallback(
    (nextOpen: boolean) => {
      onOpenChange(nextOpen)
    },
    [onOpenChange],
  )

  const handleScanResult = useCallback(
    (value: string) => {
      const code = normalizeMatchCode(value)
      if (!code) return

      onCode(code)
      updateOpen(false)
    },
    [onCode, updateOpen],
  )
  const scannerStatus = useQrCodeScanner({
    open,
    videoRef,
    canvasRef,
    onResult: handleScanResult,
  })

  const message = statusMessages[scannerStatus]

  return (
    <Dialog open={open} onOpenChange={updateOpen}>
      <DialogContent className="gap-4">
        <DialogHeader>
          <DialogTitle>掃描現場配對 QR Code</DialogTitle>
          <DialogDescription>{message}</DialogDescription>
        </DialogHeader>

        <div className="border-ink bg-ink aspect-square overflow-hidden rounded-[18px] border-2">
          <video
            ref={videoRef}
            className="h-full w-full object-cover"
            playsInline
            muted
          />
          <canvas ref={canvasRef} className="hidden" />
        </div>
      </DialogContent>
    </Dialog>
  )
}
