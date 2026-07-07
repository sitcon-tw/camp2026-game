import { useQueryClient } from "@tanstack/react-query"
import { useEffect } from "react"

import { MatchStateSchema, type MatchState } from "@/shared/api/game"

const matchEventNames = [
  "match_updated",
  "player_ready",
  "player_answered",
  "round_started",
  "round_revealed",
  "match_completed",
] as const

export function useMatchEvents(
  matchID: string,
  options: { enabled?: boolean; onDeleted?: () => void } = {},
) {
  const queryClient = useQueryClient()
  const enabled = options.enabled ?? true
  const onDeleted = options.onDeleted

  useEffect(() => {
    if (!enabled || !matchID || typeof window === "undefined") return

    let source: EventSource | null = null
    let reconnectTimeout: number | null = null
    let reconnectAttempts = 0
    let deleted = false
    let disposed = false

    const clearReconnectTimeout = () => {
      if (reconnectTimeout != null) {
        window.clearTimeout(reconnectTimeout)
        reconnectTimeout = null
      }
    }

    const closeSource = () => {
      if (source == null) return
      source.close()
      source = null
    }

    const refreshMatch = () =>
      queryClient.invalidateQueries({ queryKey: ["matches", matchID] })

    const scheduleReconnect = () => {
      if (disposed || deleted || reconnectTimeout != null) return

      const delay = Math.min(5000, 1000 * 2 ** reconnectAttempts)
      reconnectAttempts += 1
      reconnectTimeout = window.setTimeout(() => {
        reconnectTimeout = null
        connect()
      }, delay)
    }

    const handleMessage = (event: MessageEvent<string>) => {
      try {
        const match = MatchStateSchema.parse(JSON.parse(event.data))
        queryClient.setQueryData<MatchState>(["matches", matchID], match)
      } catch {
        // Ignore malformed events and let the connection keep streaming.
      }
    }

    const handleDeleted = () => {
      deleted = true
      clearReconnectTimeout()
      closeSource()
      queryClient.removeQueries({ queryKey: ["matches", matchID] })
      onDeleted?.()
    }

    const handleError = () => {
      if (disposed || deleted) return
      closeSource()
      void refreshMatch()
      scheduleReconnect()
    }

    const connect = () => {
      if (disposed || deleted) return

      clearReconnectTimeout()
      closeSource()

      source = new EventSource(
        `/api/matches/${encodeURIComponent(matchID)}/events`,
        { withCredentials: true },
      )

      source.onopen = () => {
        reconnectAttempts = 0
        void refreshMatch()
      }
      source.onerror = handleError
      for (const eventName of matchEventNames) {
        source.addEventListener(eventName, handleMessage)
      }
      source.addEventListener("match_deleted", handleDeleted)
    }

    connect()

    return () => {
      disposed = true
      clearReconnectTimeout()
      closeSource()
    }
  }, [enabled, matchID, onDeleted, queryClient])
}

export function useMatchDeadlineRefresh(
  matchID: string,
  match: MatchState | undefined,
) {
  const queryClient = useQueryClient()

  useEffect(() => {
    if (!matchID || match?.status !== "active") return

    const deadline =
      match.phase === "revealing" ? match.revealEndsAt : match.roundEndsAt
    if (!deadline) return

    const deadlineAt = new Date(deadline).getTime()
    if (!Number.isFinite(deadlineAt)) return

    const delay = Math.max(0, deadlineAt - Date.now() + 300)
    const timeout = window.setTimeout(() => {
      void queryClient.invalidateQueries({ queryKey: ["matches", matchID] })
    }, delay)

    return () => window.clearTimeout(timeout)
  }, [
    matchID,
    match?.phase,
    match?.revealEndsAt,
    match?.roundEndsAt,
    match?.status,
    queryClient,
  ])
}
