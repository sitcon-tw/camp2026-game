import { queryOptions } from "@tanstack/react-query"

import { gameApi } from "./game"

export const battleOpeningQueryKey = [
  "matches",
  "computer",
  "settings",
] as const

export function battleOpeningQueryOptions() {
  return queryOptions({
    queryKey: battleOpeningQueryKey,
    queryFn: gameApi.computerBattleSettings,
    staleTime: 5_000,
  })
}
