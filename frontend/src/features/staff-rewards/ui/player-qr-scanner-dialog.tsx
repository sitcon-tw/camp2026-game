import { type FormEvent, useCallback, useRef, useState } from "react"

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

type PlayerQrScannerDialogProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  onToken: (token: string) => void
}

function normalizeToken(value: string) {
  return value.trim()
}

const statusMessages: Record<QrCodeScannerStatus, string> = {
  idle: "正在啟動相機",
  starting: "正在啟動相機",
  scanning: "對準學員的個人 QR Code",
  "secure-context-required":
    "瀏覽器需要 HTTPS 或 localhost 才能開啟相機，請輸入 QR 識別碼。",
  "camera-unavailable": "這個瀏覽器無法開啟相機，請輸入 QR 識別碼。",
  "permission-denied": "相機權限未開啟，請輸入 QR 識別碼。",
  "scan-error": "相機已開啟，但無法讀取 QR Code，請輸入 QR 識別碼。",
}

export function PlayerQrScannerDialog({
  open,
  onOpenChange,
  onToken,
}: PlayerQrScannerDialogProps) {
  const videoRef = useRef<HTMLVideoElement>(null)
  const canvasRef = useRef<HTMLCanvasElement>(null)
  const [manualToken, setManualToken] = useState("")
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
      const token = normalizeToken(value)
      if (!token) return

      onToken(token)
      updateOpen(false)
    },
    [onToken, updateOpen],
  )
  const scannerStatus = useQrCodeScanner({
    open,
    videoRef,
    canvasRef,
    onResult: handleScanResult,
  })

  const message = manualMessage ?? statusMessages[scannerStatus]

  function updateManualToken(value: string) {
    setManualToken(value)
    setManualMessage(null)
  }

  function handleManualSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    const token = normalizeToken(manualToken)
    if (!token) {
      setManualMessage("請輸入 QR 識別碼。")
      return
    }
    onToken(token)
    updateOpen(false)
  }

  return (
    <Dialog open={open} onOpenChange={updateOpen}>
      <DialogContent className="gap-4">
        <DialogHeader>
          <DialogTitle>掃描學員 QR Code</DialogTitle>
          <DialogDescription>{message}</DialogDescription>
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
            value={manualToken}
            onChange={(event) => updateManualToken(event.target.value)}
            placeholder="手動輸入 QR 識別碼"
            autoComplete="off"
            inputMode="text"
          />
          <DialogFooter>
            <Button type="submit" variant="secondary" className="w-full">
              確認學員
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
