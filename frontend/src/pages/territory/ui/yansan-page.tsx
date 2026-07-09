import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { toast } from "sonner"

import { BossStatusBanner } from "@/features/territory/ui/boss-status-banner"
import { gameApi } from "@/shared/api/game"
import { territoryApi, type ProtectedSitone } from "@/shared/api/territory"
import { sitoneMeta } from "@/shared/lib/game-labels"
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
import { GameIcon } from "@/shared/ui/game-icon"
import { GamePageShell } from "@/shared/ui/game-page-shell"
import { PageHeader } from "@/shared/ui/page-header"

export function YansanPage() {
  const queryClient = useQueryClient()
  const yansanQuery = useQuery({
    queryKey: ["yansan", "status"],
    queryFn: territoryApi.yansanStatus,
    refetchInterval: (query) =>
      query.state.data?.bossStatus === "under_attack" ? 5000 : 20000,
  })
  const catalogQuery = useQuery({
    queryKey: ["catalog", "sitones"],
    queryFn: gameApi.catalogSitones,
  })

  const status = yansanQuery.data
  const servicesDown = status != null && !status.servicesAvailable
  const sitoneOf = (sitoneID: string) =>
    catalogQuery.data?.find((sitone) => sitone.id === sitoneID)

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
      queryClient.invalidateQueries({ queryKey: ["yansan"] })
    },
    onError: (error) => {
      toast.error(error instanceof Error ? error.message : "Boss 戰登記失敗")
    },
  })

  const redeemMutation = useMutation({
    mutationFn: territoryApi.yansanRedeem,
    onSuccess: () => {
      toast.success("贖回申請已送出，結果由研三舍公告")
      queryClient.invalidateQueries({ queryKey: ["yansan"] })
      queryClient.invalidateQueries({ queryKey: ["me", "status"] })
    },
    onError: (error) => {
      toast.error(error instanceof Error ? error.message : "贖回申請失敗")
    },
  })

  return (
    <GamePageShell contentClassName="grid content-start gap-y-3">
      <PageHeader title="研三舍" headline="Yansan Hall" backTo="/territory" />

      {status ? (
        <BossStatusBanner status={status.bossStatus} />
      ) : (
        <Card className="rounded-[22px] px-4 py-4">
          <span className="text-muted-foreground text-sm font-extrabold">
            {yansanQuery.isError
              ? "研三舍狀態讀取失敗，請稍後再試"
              : "正在同步研三舍狀態"}
          </span>
        </Card>
      )}

      {servicesDown ? (
        <Card className="border-destructive gap-1 rounded-[18px] px-4 py-3">
          <p className="text-destructive text-sm font-black">
            研三舍服務全面暫停
          </p>
          <p className="text-muted-foreground text-xs leading-relaxed font-bold">
            Boss
            戰進行期間，研三舍贖回服務暫停；進行中的申請視為失敗，已支付的開源力不予退還。
          </p>
        </Card>
      ) : null}

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
                  Boss
                  戰開打後，研三舍贖回服務將暫停，進行中的申請視為失敗且開源力不退。確定要代表小隊登記參戰嗎？
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

      <section className="grid gap-2" aria-label="保管中的一次性小石">
        <div className="flex items-center justify-between px-1">
          <h2 className="text-lg font-black">一次性小石保管庫</h2>
          {status ? (
            <Badge variant="outline">本隊已贖回 {status.teamRedeemCount} 次</Badge>
          ) : null}
        </div>
        <p className="text-muted-foreground px-1 text-xs leading-relaxed font-bold">
          一次性小石出事後會統一送到研三舍保管，可付出開源力申請贖回；整隊贖回次數越多，成本越高、成功率越低。
        </p>

        {status == null ? null : status.protectedSitones.length === 0 ? (
          <Card className="rounded-[18px] px-3 py-4">
            <span className="text-muted-foreground text-sm font-extrabold">
              目前沒有小石在研三舍保管，太好了！
            </span>
          </Card>
        ) : (
          <div className="grid gap-2">
            {status.protectedSitones.map((entry) => (
              <ProtectedSitoneRow
                key={entry.id}
                entry={entry}
                sitone={sitoneOf(entry.sitoneId)}
                disabled={servicesDown || redeemMutation.isPending}
                onRedeem={() => redeemMutation.mutate(entry.sitoneId)}
              />
            ))}
          </div>
        )}
      </section>

    </GamePageShell>
  )
}

function ProtectedSitoneRow({
  entry,
  sitone,
  disabled,
  onRedeem,
}: {
  entry: ProtectedSitone
  sitone?: { name: string; type: string; iconPath?: string }
  disabled: boolean
  onRedeem: () => void
}) {
  const meta = sitoneMeta(sitone?.type ?? "")

  return (
    <Card className="grid grid-cols-[52px_1fr_auto] items-center gap-3 rounded-[20px] px-3.5 py-3">
      <div
        className={[
          "border-ink grid size-[52px] place-items-center overflow-hidden rounded-[16px] border-2",
          meta.bgClassName,
        ].join(" ")}
        aria-hidden
      >
        <GameIcon
          iconPath={sitone?.iconPath}
          imageClassName="p-1.5"
          fallback={
            <span className="text-[10px] font-black">{meta.short}</span>
          }
        />
      </div>
      <div className="min-w-0">
        <p className="truncate font-black">{sitone?.name ?? entry.sitoneId}</p>
        <p className="text-muted-foreground truncate text-xs font-bold">
          一次性小石 · 研三舍保管中
        </p>
      </div>
      <AlertDialog>
        <AlertDialogTrigger asChild>
          <Button size="sm" disabled={disabled}>
            申請贖回
          </Button>
        </AlertDialogTrigger>
        <AlertDialogContent size="sm">
          <AlertDialogHeader>
            <AlertDialogTitle>確認申請贖回</AlertDialogTitle>
            <AlertDialogDescription>
              贖回會消耗執行者的開源力，成本與成功率由伺服器依原持有者與整隊贖回次數計算。若
              Boss 戰期間申請失敗，開源力不予退還。
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>取消</AlertDialogCancel>
            <AlertDialogAction onClick={onRedeem}>確認贖回</AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </Card>
  )
}
