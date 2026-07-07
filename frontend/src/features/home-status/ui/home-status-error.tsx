import { TriangleAlert } from "lucide-react"

import { AppError } from "@/shared/api/error"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/shared/ui/card"

type HomeStatusErrorProps = {
  error: unknown
}

export function HomeStatusError({ error }: HomeStatusErrorProps) {
  const message =
    error instanceof AppError
      ? error.message
      : error instanceof Error
        ? error.message
        : "無法取得基地狀態"

  return (
    <main className="bg-background text-foreground min-h-dvh px-5 py-6 sm:px-8 lg:py-10">
      <Card className="border-destructive/40 mx-auto max-w-xl">
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <TriangleAlert className="text-destructive size-5" />
            基地連線失敗
          </CardTitle>
          <CardDescription>{message}</CardDescription>
        </CardHeader>
        <CardContent className="text-muted-foreground text-sm">
          請稍後重新整理頁面。
        </CardContent>
      </Card>
    </main>
  )
}
