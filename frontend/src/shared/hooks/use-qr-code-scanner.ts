import jsQR from "jsqr"
import { useEffect, useState, type RefObject } from "react"

type BarcodeDetectorResult = {
  rawValue?: string
}

type BarcodeDetectorInstance = {
  detect(source: HTMLCanvasElement): Promise<BarcodeDetectorResult[]>
}

type BarcodeDetectorConstructor = new (options: {
  formats: string[]
}) => BarcodeDetectorInstance

export type QrCodeScannerStatus =
  | "idle"
  | "starting"
  | "scanning"
  | "secure-context-required"
  | "camera-unavailable"
  | "permission-denied"
  | "scan-error"

type UseQrCodeScannerOptions = {
  open: boolean
  videoRef: RefObject<HTMLVideoElement | null>
  canvasRef: RefObject<HTMLCanvasElement | null>
  onResult: (value: string) => void
  scanIntervalMs?: number
}

function getBarcodeDetector() {
  return (
    window as typeof window & {
      BarcodeDetector?: BarcodeDetectorConstructor
    }
  ).BarcodeDetector
}

function createBarcodeDetector() {
  const BarcodeDetector = getBarcodeDetector()
  if (!BarcodeDetector) return null

  try {
    return new BarcodeDetector({ formats: ["qr_code"] })
  } catch {
    return null
  }
}

function isPermissionError(error: unknown) {
  return (
    error instanceof DOMException &&
    (error.name === "NotAllowedError" || error.name === "SecurityError")
  )
}

function stopStream(stream: MediaStream | null) {
  stream?.getTracks().forEach((track) => track.stop())
}

async function decodeQrCode(
  canvas: HTMLCanvasElement,
  context: CanvasRenderingContext2D,
  detector: BarcodeDetectorInstance | null,
) {
  if (detector) {
    try {
      const detected = await detector.detect(canvas)
      const rawValue = detected[0]?.rawValue?.trim()
      if (rawValue) return rawValue
    } catch {
      // Fall through to jsQR when the native detector is present but fails.
    }
  }

  const imageData = context.getImageData(0, 0, canvas.width, canvas.height)
  return (
    jsQR(imageData.data, imageData.width, imageData.height, {
      inversionAttempts: "attemptBoth",
    })?.data.trim() ?? ""
  )
}

export function useQrCodeScanner({
  open,
  videoRef,
  canvasRef,
  onResult,
  scanIntervalMs = 350,
}: UseQrCodeScannerOptions) {
  const [status, setStatus] = useState<QrCodeScannerStatus>("idle")

  useEffect(() => {
    if (!open) return

    let cancelled = false
    let timer = 0
    let stream: MediaStream | null = null
    let activeVideo: HTMLVideoElement | null = null

    const cleanup = () => {
      window.clearTimeout(timer)
      stopStream(stream)
      if (activeVideo) {
        activeVideo.srcObject = null
      }
    }

    async function startScanner() {
      setStatus("starting")

      if (typeof window === "undefined" || typeof navigator === "undefined") {
        setStatus("camera-unavailable")
        return
      }

      if (!window.isSecureContext) {
        setStatus("secure-context-required")
        return
      }

      if (!navigator.mediaDevices?.getUserMedia) {
        setStatus("camera-unavailable")
        return
      }

      try {
        const detector = createBarcodeDetector()
        stream = await navigator.mediaDevices.getUserMedia({
          audio: false,
          video: { facingMode: { ideal: "environment" } },
        })
        if (cancelled) {
          stopStream(stream)
          return
        }

        const video = videoRef.current
        if (!video) {
          stopStream(stream)
          stream = null
          setStatus("camera-unavailable")
          return
        }

        activeVideo = video
        video.muted = true
        video.playsInline = true
        video.srcObject = stream
        await video.play()
        if (cancelled) return

        setStatus("scanning")

        const scan = async () => {
          if (cancelled) return

          try {
            const canvas = canvasRef.current
            const context = canvas?.getContext("2d", {
              willReadFrequently: true,
            })
            if (
              canvas &&
              context &&
              video.readyState >= HTMLMediaElement.HAVE_CURRENT_DATA &&
              video.videoWidth > 0 &&
              video.videoHeight > 0
            ) {
              canvas.width = video.videoWidth
              canvas.height = video.videoHeight
              context.drawImage(video, 0, 0, canvas.width, canvas.height)

              const value = await decodeQrCode(canvas, context, detector)
              if (value) {
                cancelled = true
                cleanup()
                onResult(value)
                return
              }
            }
          } catch {
            setStatus("scan-error")
          }

          timer = window.setTimeout(scan, scanIntervalMs)
        }

        await scan()
      } catch (error) {
        cleanup()
        if (cancelled) return

        setStatus(
          isPermissionError(error) ? "permission-denied" : "camera-unavailable",
        )
      }
    }

    void startScanner()

    return () => {
      cancelled = true
      cleanup()
    }
  }, [canvasRef, onResult, open, scanIntervalMs, videoRef])

  return status
}
