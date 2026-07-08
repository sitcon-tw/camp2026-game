import { useMemo } from "react"

import { createQrMatrix } from "@/features/battle-qr/lib/qr-code"
import { cn } from "@/shared/utils"

type MatchCodeQrProps = {
  value: string
  className?: string
  label?: string
}

const quietZone = 4

export function MatchCodeQr({
  value,
  className,
  label = "現場配對 QR Code",
}: MatchCodeQrProps) {
  const matrix = useMemo(() => {
    if (!value) return null
    return createQrMatrix(value)
  }, [value])

  if (!matrix) {
    return (
      <div
        className={cn(
          "bg-surface-raised border-ink grid aspect-square place-items-center rounded-lg border-2 text-xs font-black",
          className,
        )}
      >
        QR
      </div>
    )
  }

  const viewBoxSize = matrix.length + quietZone * 2

  return (
    <svg
      role="img"
      aria-label={label}
      viewBox={`0 0 ${viewBoxSize} ${viewBoxSize}`}
      className={cn(
        "bg-card border-ink aspect-square rounded-lg border-2 p-1",
        className,
      )}
      shapeRendering="crispEdges"
    >
      <rect width={viewBoxSize} height={viewBoxSize} fill="white" />
      {matrix.map((row, rowIndex) =>
        row.map((dark, colIndex) =>
          dark ? (
            <rect
              key={`${rowIndex}-${colIndex}`}
              x={colIndex + quietZone}
              y={rowIndex + quietZone}
              width="1"
              height="1"
              fill="currentColor"
            />
          ) : null,
        ),
      )}
    </svg>
  )
}
