import { createFileRoute, redirect } from "@tanstack/react-router"

import {
  frontCurrentQueryOptions,
  frontSnapshotQueryOptions,
} from "@/features/front-map/api/front.query"
import { FrontPage } from "@/pages/front/ui/front-page"
import { battleOpeningQueryOptions } from "@/shared/api/battle-opening.query"

export const Route = createFileRoute("/front")({
  loader: async ({ context }) => {
    const settings = await context.queryClient.ensureQueryData(
      battleOpeningQueryOptions(),
    )
    if (settings.battleOpeningLocked) {
      throw redirect({ to: "/", replace: true })
    }

    const current = await context.queryClient.ensureQueryData(
      frontCurrentQueryOptions(),
    )

    if (!current.front) return

    await context.queryClient.ensureQueryData(
      frontSnapshotQueryOptions(current.front.id),
    )
  },
  component: FrontPage,
})
