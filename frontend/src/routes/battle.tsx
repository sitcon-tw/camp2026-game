import { Outlet, createFileRoute, redirect } from "@tanstack/react-router"

import { battleOpeningQueryOptions } from "@/shared/api/battle-opening.query"

export const Route = createFileRoute("/battle")({
  loader: async ({ context }) => {
    const settings = await context.queryClient.ensureQueryData(
      battleOpeningQueryOptions(),
    )
    if (settings.battleOpeningLocked) {
      throw redirect({ to: "/", replace: true })
    }
  },
  component: Outlet,
})
