import { useQueryClient } from "@tanstack/react-query"
import { useEffect, useState } from "react"

import { isTerminalClientError } from "@/shared/api/error"
import {
  FrontSnapshotSchema,
  gameApi,
  type FrontSnapshot,
} from "@/shared/api/game"

import { frontSnapshotQueryKey } from "./front.query"

const frontEventNames = [
  "front_updated",
  "garrison_updated",
  "trade_started",
  "trade_completed",
  "rail_updated",
  "train_started",
  "train_arrived",
  "train_cancelled",
] as const

export type FrontConnectionState = "connecting" | "live" | "reconnecting"

export function useFrontEvents(frontID: string, enabled = true) {
  const queryClient = useQueryClient()
  const [connectionState, setConnectionState] =
    useState<FrontConnectionState>("connecting")

  useEffect(() => {
    if (!enabled || !frontID || typeof window === "undefined") return

    let source: EventSource | null = null
    let reconnectTimeout: number | null = null
    let watchdogInterval: number | null = null
    let reconnectAttempts = 0
    let lastEventAt = Date.now()
    let disposed = false

    const closeSource = () => {
      source?.close()
      source = null
    }
    const clearReconnect = () => {
      if (reconnectTimeout == null) return
      window.clearTimeout(reconnectTimeout)
      reconnectTimeout = null
    }
    const refresh = async () => {
      try {
        const snapshot = await gameApi.frontSnapshot(frontID)
        queryClient.setQueryData(frontSnapshotQueryKey(frontID), snapshot)
        return true
      } catch (error) {
        return !isTerminalClientError(error)
      }
    }
    const scheduleReconnect = () => {
      if (disposed || reconnectTimeout != null) return
      setConnectionState("reconnecting")
      const delay = Math.min(5_000, 1_000 * 2 ** reconnectAttempts)
      reconnectAttempts += 1
      reconnectTimeout = window.setTimeout(() => {
        reconnectTimeout = null
        connect()
      }, delay)
    }
    const handleSnapshot = (event: MessageEvent<string>) => {
      lastEventAt = Date.now()
      try {
        const snapshot = FrontSnapshotSchema.parse(JSON.parse(event.data))
        queryClient.setQueryData<FrontSnapshot>(
          frontSnapshotQueryKey(frontID),
          (current) =>
            current?.revision !== undefined &&
            current.revision > (snapshot.revision ?? 0)
              ? current
              : snapshot,
        )
      } catch {
        // Ignore malformed events and keep the stream connected.
      }
    }
    const handleError = () => {
      if (disposed) return
      closeSource()
      void refresh().then((shouldReconnect) => {
        if (shouldReconnect) scheduleReconnect()
      })
    }
    const connect = () => {
      if (disposed) return
      clearReconnect()
      closeSource()
      setConnectionState(reconnectAttempts > 0 ? "reconnecting" : "connecting")
      source = new EventSource(
        `/api/fronts/${encodeURIComponent(frontID)}/events`,
        { withCredentials: true },
      )
      source.onopen = () => {
        reconnectAttempts = 0
        lastEventAt = Date.now()
        setConnectionState("live")
      }
      source.onerror = handleError
      source.addEventListener("keepalive", () => {
        lastEventAt = Date.now()
      })
      for (const eventName of frontEventNames) {
        source.addEventListener(eventName, handleSnapshot)
      }
    }
    const handleVisibility = () => {
      if (document.visibilityState !== "visible" || disposed) return
      clearReconnect()
      closeSource()
      reconnectAttempts = 0
      connect()
    }

    watchdogInterval = window.setInterval(() => {
      if (disposed || source == null || Date.now() - lastEventAt <= 60_000)
        return
      closeSource()
      void refresh().then((shouldReconnect) => {
        if (shouldReconnect) scheduleReconnect()
      })
    }, 20_000)
    document.addEventListener("visibilitychange", handleVisibility)
    connect()

    return () => {
      disposed = true
      clearReconnect()
      closeSource()
      if (watchdogInterval != null) window.clearInterval(watchdogInterval)
      document.removeEventListener("visibilitychange", handleVisibility)
    }
  }, [enabled, frontID, queryClient])

  return connectionState
}
