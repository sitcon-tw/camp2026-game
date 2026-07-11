import { createFileRoute } from "@tanstack/react-router"

import { OpenPowerTransferPage } from "@/pages/open-power-transfer/ui/open-power-transfer-page"

export const Route = createFileRoute("/open-power-transfer")({
  component: OpenPowerTransferPage,
})
