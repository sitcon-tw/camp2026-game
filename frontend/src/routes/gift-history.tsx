import { createFileRoute } from "@tanstack/react-router"

import { GiftHistoryPage } from "@/pages/gift-history/ui/gift-history-page"

export const Route = createFileRoute("/gift-history")({
  component: GiftHistoryPage,
})
