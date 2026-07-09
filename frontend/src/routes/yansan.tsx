import { createFileRoute } from "@tanstack/react-router"

import { YansanPage } from "@/pages/territory/ui/yansan-page"

export const Route = createFileRoute("/yansan")({
  component: YansanPage,
})
