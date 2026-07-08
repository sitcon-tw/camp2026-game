/// <reference types="vite/client" />

import { Outlet, createRootRouteWithContext } from "@tanstack/react-router"
import type { QueryClient } from "@tanstack/react-query"

import { QueryProvider } from "@/app/providers/query-provider"
import { AuthGate } from "@/features/auth/ui/auth-gate"
import { MaintenanceAnnouncement } from "@/features/maintenance/ui/maintenance-announcement"
import { GameErrorPage } from "@/pages/error/ui/game-error-page"
import { NotFoundPage } from "@/pages/not-found/ui/not-found-page"
import { AppBottomNav } from "@/shared/ui/app-bottom-nav"
import { Toaster } from "@/shared/ui/sonner"

export const Route = createRootRouteWithContext<{
  queryClient: QueryClient
}>()({
  component: RootComponent,
  errorComponent: GameErrorPage,
  notFoundComponent: NotFoundPage,
})

function RootComponent() {
  const { queryClient } = Route.useRouteContext()

  return (
    <QueryProvider queryClient={queryClient}>
      <AuthGate>
        <Outlet />
        <AppBottomNav />
      </AuthGate>

      <MaintenanceAnnouncement />
      <Toaster position="bottom-center" />
    </QueryProvider>
  )
}
