"use client"

import type { ReactNode } from "react"

import { cn } from "@/shared/utils"
import { Button } from "@/shared/ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/shared/ui/dialog"

type ZoomableQRCodeProps = {
  label: string
  children: ReactNode
  preview?: ReactNode
  className?: string
  previewClassName?: string
}

export function ZoomableQRCode({
  label,
  children,
  preview,
  className,
  previewClassName,
}: ZoomableQRCodeProps) {
  return (
    <Dialog>
      <DialogTrigger asChild>
        <Button
          type="button"
          variant="ghost"
          aria-label={`放大檢視 ${label}`}
          className={cn(
            "focus-visible:ring-ring/50 mx-auto h-auto p-0 shadow-none hover:bg-transparent focus-visible:ring-[4px]",
            className,
          )}
        >
          {children}
        </Button>
      </DialogTrigger>
      <DialogContent className="max-w-[min(calc(100%-2rem),40rem)] p-5 sm:p-6">
        <DialogHeader className="sr-only">
          <DialogTitle>{label}</DialogTitle>
          <DialogDescription>放大檢視 QR Code</DialogDescription>
        </DialogHeader>
        <div
          className={cn(
            "bg-paper border-ink mx-auto grid aspect-square w-full max-w-[min(78vw,32rem)] place-items-center rounded-[24px] border-4 p-5",
            previewClassName,
          )}
        >
          {preview ?? children}
        </div>
      </DialogContent>
    </Dialog>
  )
}
