import { useCallback, useRef, useState } from "react"

import { normalizeMatchCode } from "@/features/battle-qr/lib/match-code"
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

type MatchCodeScannerDialogProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  onCode: (code: string) => void
}

const statusMessages: Record<QrCodeScannerStatus, string> = {
  idle: "正在啟動相機",
  starting: "正在啟動相機",
  scanning: "對準房號 QR Code",
  "secure-context-required":
    "瀏覽器需要 HTTPS 或 localhost 才能開啟相機，請輸入房號。",
  "camera-unavailable": "這個瀏覽器無法開啟相機，請輸入房號。",
  "permission-denied": "相機權限未開啟，請輸入房號。",
  "scan-error": "相機已開啟，但無法讀取 QR Code，請輸入房號。",
}

export function MatchCodeScannerDialog({
  open,
  onOpenChange,
  onCode,
}: MatchCodeScannerDialogProps) {
  const videoRef = useRef<HTMLVideoElement>(null)
  const canvasRef = useRef<HTMLCanvasElement>(null)
  const [manualCode, setManualCode] = useState("")
  const [manualMessage, setManualMessage] = useState<string | null>(null)

  const updateOpen = useCallback(
    (nextOpen: boolean) => {
      if (!nextOpen) setManualMessage(null)
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

  const message = manualMessage ?? statusMessages[scannerStatus]

  function updateManualCode(value: string) {
    setManualCode(normalizeMatchCode(value))
    setManualMessage(null)
  }

  function handleManualSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault()
    const code = normalizeMatchCode(manualCode)
    if (!code) {
      setManualMessage("請輸入房號。")
      return
    }
    onCode(code)
    updateOpen(false)
  }

  return (
    <Dialog open={open} onOpenChange={updateOpen}>
      <DialogContent className="gap-4">
        <DialogHeader>
          <DialogTitle>掃描房號 QR Code</DialogTitle>
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

        <form className="grid gap-3" onSubmit={handleManualSubmit}>
          <Input
            value={manualCode}
            onChange={(event) => updateManualCode(event.target.value)}
            placeholder="手動輸入房號"
            autoComplete="off"
            inputMode="text"
          />
          <DialogFooter>
            <Button type="submit" variant="secondary" className="w-full">
              加入房間
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
