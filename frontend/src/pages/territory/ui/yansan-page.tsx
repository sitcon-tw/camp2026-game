import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { toast } from "sonner"

import { BossStatusBanner } from "@/features/territory/ui/boss-status-banner"
import { territoryApi } from "@/shared/api/territory"
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "@/shared/ui/alert-dialog"
import { Badge } from "@/shared/ui/badge"
import { Button } from "@/shared/ui/button"
import { Card } from "@/shared/ui/card"
import { GamePageShell } from "@/shared/ui/game-page-shell"
import { PageHeader } from "@/shared/ui/page-header"

export function YansanPage() {
  const queryClient = useQueryClient()
  const bossQuery = useQuery({
    queryKey: ["yansan", "boss", "status"],
    queryFn: territoryApi.bossStatus,
    refetchInterval: (query) =>
      query.state.data?.bossStatus === "under_attack" ? 5000 : 20000,
  })

  const bossAttackMutation = useMutation({
    mutationFn: territoryApi.attackBoss,
    onSuccess: () => {
      toast.success("已登記參與 Boss 戰！")
      queryClient.invalidateQueries({ queryKey: ["yansan", "boss"] })
    },
    onError: (error) => {
      toast.error(error instanceof Error ? error.message : "Boss 戰登記失敗")
    },
  })

  return (
    <GamePageShell contentClassName="grid content-start gap-y-3">
      <PageHeader title="研三舍" headline="Yansan Hall" backTo="/territory" />

      {bossQuery.data ? (
        <BossStatusBanner status={bossQuery.data.bossStatus} />
      ) : (
        <Card className="rounded-[22px] px-4 py-4">
          <span className="text-muted-foreground text-sm font-extrabold">
            {bossQuery.isError
              ? "Boss 戰況讀取失敗，請稍後再試"
              : "正在同步 Boss 戰況"}
          </span>
        </Card>
      )}

      {bossQuery.data?.bossStatus === "open" ? (
        <Card className="gap-2 rounded-[20px] px-4 py-3.5">
          <div className="flex items-center justify-between gap-2">
            <h2 className="text-lg font-black">集結進攻 Boss</h2>
            <Badge variant="secondary">
              {bossQuery.data.participatingTeamIds.length} /{" "}
              {bossQuery.data.requiredTeamCount ?? 9} 隊已集結
            </Badge>
          </div>
          <p className="text-muted-foreground text-xs leading-relaxed font-bold">
            Boss 戰必須全部小隊共同參與才能發動。登記後等待其他小隊集結。
          </p>
          <AlertDialog>
            <AlertDialogTrigger asChild>
              <Button disabled={bossAttackMutation.isPending}>
                {bossAttackMutation.isPending ? "登記中" : "登記參戰"}
              </Button>
            </AlertDialogTrigger>
            <AlertDialogContent size="sm">
              <AlertDialogHeader>
                <AlertDialogTitle>確認參與 Boss 戰</AlertDialogTitle>
                <AlertDialogDescription>
                  確定要代表小隊登記參與研三舍 Boss 戰嗎？
                </AlertDialogDescription>
              </AlertDialogHeader>
              <AlertDialogFooter>
                <AlertDialogCancel>取消</AlertDialogCancel>
                <AlertDialogAction
                  onClick={() => bossAttackMutation.mutate()}
                >
                  確認參戰
                </AlertDialogAction>
              </AlertDialogFooter>
            </AlertDialogContent>
          </AlertDialog>
        </Card>
      ) : null}
    </GamePageShell>
  )
}
