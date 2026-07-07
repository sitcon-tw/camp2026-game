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
    let watchdogInterval: number | null = null
    let reconnectAttempts = 0
    let lastEventAt = Date.now()
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
      lastEventAt = Date.now()
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

    const handleVisibilityChange = () => {
      if (document.visibilityState !== "visible" || disposed || deleted) return
      // Force reconnect when tab comes back into focus — CF may have silently dropped the connection.
      clearReconnectTimeout()
      closeSource()
      reconnectAttempts = 0
      connect()
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
        lastEventAt = Date.now()
        void refreshMatch()
      }
      source.onerror = handleError
      source.addEventListener("keepalive", () => {
        lastEventAt = Date.now()
      })
      for (const eventName of matchEventNames) {
        source.addEventListener(eventName, handleMessage)
      }
      source.addEventListener("match_deleted", handleDeleted)
    }

    // Watchdog: if no event (including keepalive) arrives for 60s, the connection is likely
    // silently dead (CF dropped it without triggering onerror). Force reconnect.
    watchdogInterval = window.setInterval(() => {
      if (disposed || deleted || source == null) return
      if (Date.now() - lastEventAt > 60_000) {
        closeSource()
        void refreshMatch()
        scheduleReconnect()
      }
    }, 20_000)

    document.addEventListener("visibilitychange", handleVisibilityChange)
    connect()

    return () => {
      disposed = true
      clearReconnectTimeout()
      closeSource()
      if (watchdogInterval != null) window.clearInterval(watchdogInterval)
      document.removeEventListener("visibilitychange", handleVisibilityChange)
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

    const transitionPhase = match.phase
    const transitionQuestionIndex = match.currentQuestionIndex ?? -1
    const transitionQuestionID = match.currentQuestion?.questionId ?? ""

    const delay = Math.max(0, deadlineAt - Date.now() + 100)
    const timeout = window.setTimeout(() => {
      void queryClient.invalidateQueries({ queryKey: ["matches", matchID] })
    }, delay)

    const recoveryDelays = [400, 900, 2000]
    const recoveryTimeouts = recoveryDelays.map((extraDelay) =>
      window.setTimeout(() => {
        const latest = queryClient.getQueryData<MatchState>(["matches", matchID])
        if (latest?.status !== "active") return

        const latestQuestionIndex = latest.currentQuestionIndex ?? -1
        const latestQuestionID = latest.currentQuestion?.questionId ?? ""
        const unchanged =
          latest.phase === transitionPhase &&
          latestQuestionIndex === transitionQuestionIndex &&
          latestQuestionID === transitionQuestionID

        if (!unchanged) return

        void queryClient.invalidateQueries({ queryKey: ["matches", matchID] })
      }, delay + extraDelay),
    )

    return () => {
      window.clearTimeout(timeout)
      for (const recoveryTimeout of recoveryTimeouts) {
        window.clearTimeout(recoveryTimeout)
      }
    }
  }, [
    matchID,
    match?.currentQuestion?.questionId,
    match?.currentQuestionIndex,
    match?.phase,
    match?.revealEndsAt,
    match?.roundEndsAt,
    match?.status,
    queryClient,
  ])
}
