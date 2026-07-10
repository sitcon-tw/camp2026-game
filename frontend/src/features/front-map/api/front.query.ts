import { mutationOptions, queryOptions } from "@tanstack/react-query"

import { gameApi, type FrontCommandInput } from "@/shared/api/game"

export const frontCurrentQueryKey = ["fronts", "current"] as const

export function frontSnapshotQueryKey(frontID: string) {
  return ["fronts", "snapshot", frontID] as const
}

export function frontCurrentQueryOptions() {
  return queryOptions({
    queryKey: frontCurrentQueryKey,
    queryFn: () => gameApi.currentFront(),
    staleTime: 5_000,
  })
}

export function frontSnapshotQueryOptions(frontID: string) {
  return queryOptions({
    queryKey: frontSnapshotQueryKey(frontID),
    queryFn: () => gameApi.frontSnapshot(frontID),
    staleTime: 30_000,
  })
}

export function frontCommandMutationOptions(frontID: string) {
  return mutationOptions({
    mutationFn: (command: FrontCommandInput) =>
      gameApi.createFrontCommand(frontID, command),
  })
}
