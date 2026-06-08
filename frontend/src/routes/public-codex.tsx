import { createFileRoute } from "@tanstack/react-router"

import { PublicCodexPage } from "@/pages/public-codex/ui/public-codex-page"

export const Route = createFileRoute("/public-codex")({
  component: PublicCodexPage,
})
