import { createFileRoute } from "@tanstack/react-router"

import { CommunityStandDisplayPage } from "@/pages/community/ui/community-stand-display-page"

export const Route = createFileRoute("/community/$standId")({
  component: CommunityStandDisplayRoute,
})

function CommunityStandDisplayRoute() {
  const { standId } = Route.useParams()
  return <CommunityStandDisplayPage standID={standId} />
}
