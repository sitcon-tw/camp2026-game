import { createFileRoute } from "@tanstack/react-router"

import {
  frontCurrentQueryOptions,
  frontSnapshotQueryOptions,
} from "@/features/front-map/api/front.query"
import { FrontPage } from "@/pages/front/ui/front-page"

export const Route = createFileRoute("/front")({
  loader: async ({ context }) => {
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
