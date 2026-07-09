import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { toast } from "sonner"

import { useNow } from "@/features/territory/model/use-now"
import { gameApi } from "@/shared/api/game"
import { territoryApi, type Captive } from "@/shared/api/territory"
import { sitoneMeta } from "@/shared/lib/game-labels"
import {
  captiveStatusLabel,
  formatCountdown,
} from "@/shared/lib/territory-labels"
import { Badge } from "@/shared/ui/badge"
import { Button } from "@/shared/ui/button"
import { Card } from "@/shared/ui/card"
import { GameIcon } from "@/shared/ui/game-icon"
import { GamePageShell } from "@/shared/ui/game-page-shell"
import { PageHeader } from "@/shared/ui/page-header"

export function TerritoryCaptivesPage() {
  const captivesQuery = useQuery({
    queryKey: ["territory", "captives"],
    queryFn: territoryApi.captives,
    refetchInterval: 30000,
  })
  const catalogQuery = useQuery({
    queryKey: ["catalog", "sitones"],
    queryFn: gameApi.catalogSitones,
  })
  const standingsQuery = useQuery({
    queryKey: ["territory", "standings"],
    queryFn: territoryApi.standings,
  })

  const captives = captivesQuery.data ?? []
  const sitoneOf = (sitoneID: string) =>
    catalogQuery.data?.find((sitone) => sitone.id === sitoneID)
  const teamNameOf = (teamID: string) =>
    standingsQuery.data?.teams.find((team) => team.teamId === teamID)?.name ??
    "其他小隊"

  return (
    <GamePageShell contentClassName="grid content-start gap-y-3">
      <PageHeader title="俘虜營" headline="Captives" backTo="/territory" />

      <Card className="bg-ink text-primary-foreground gap-1.5 rounded-[24px] px-5 py-4">
        <p className="text-primary-foreground/70 text-xs font-black tracking-[0.08em] uppercase">
          POW Camp
        </p>
        <h2 className="text-xl font-black">戰俘名冊</h2>
        <p className="text-primary-foreground/75 text-sm leading-relaxed">
          俘虜的小石只能發揮 50%
          數值，冷卻期間原隊可優先搶回；冷卻結束後可送研三舍申請轉正，成功即正式歸隊。
        </p>
      </Card>

      {captivesQuery.isPending ? (
        <Card className="rounded-[18px] px-3 py-4">
          <span className="text-muted-foreground text-sm font-extrabold">
            正在同步俘虜名冊
          </span>
        </Card>
      ) : captivesQuery.isError ? (
        <Card className="rounded-[18px] px-3 py-4">
          <span className="text-muted-foreground text-sm font-extrabold">
            俘虜資料讀取失敗，請稍後再試
          </span>
        </Card>
      ) : captives.length === 0 ? (
        <Card className="rounded-[18px] px-3 py-4">
          <span className="text-muted-foreground text-sm font-extrabold">
            目前沒有俘虜的小石，發起攻擊贏得戰果吧！
          </span>
        </Card>
      ) : (
        <div className="grid gap-2">
          {captives.map((captive) => (
            <CaptiveRow
              key={captive.id}
              captive={captive}
              sitone={sitoneOf(captive.sitoneId)}
              originTeamName={teamNameOf(captive.originalTeamId)}
            />
          ))}
        </div>
      )}
    </GamePageShell>
  )
}

function CaptiveRow({
  captive,
  sitone,
  originTeamName,
}: {
  captive: Captive
  sitone?: { name: string; type: string; iconPath?: string }
  originTeamName: string
}) {
  const queryClient = useQueryClient()
  const now = useNow()
  const meta = sitoneMeta(sitone?.type ?? "")
  const cooldown = formatCountdown(captive.cooldownUntil, now)
  const cooldownActive =
    captive.cooldownUntil != null &&
    new Date(captive.cooldownUntil).getTime() > now
  const canConvert = captive.status === "captured" && !cooldownActive

  const convertMutation = useMutation({
    mutationFn: () => territoryApi.convertCaptive(captive.id),
    onSuccess: () => {
      toast.success("已送往研三舍申請轉正，結果稍後揭曉")
      queryClient.invalidateQueries({ queryKey: ["territory", "captives"] })
      queryClient.invalidateQueries({ queryKey: ["yansan"] })
    },
    onError: (error) => {
      toast.error(error instanceof Error ? error.message : "轉正申請失敗")
    },
  })

  return (
    <Card className="gap-2.5 rounded-[20px] px-3.5 py-3">
      <div className="grid grid-cols-[52px_1fr_auto] items-center gap-3">
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
          <p className="truncate font-black">{sitone?.name ?? "神秘小石"}</p>
          <p className="text-muted-foreground truncate text-xs font-bold">
            來自 {originTeamName}
          </p>
        </div>
        <Badge variant="secondary">50% 戰力</Badge>
      </div>

      <div className="flex flex-wrap items-center justify-between gap-2">
        <div className="flex flex-wrap items-center gap-1.5">
          <span className="bg-surface-raised border-border text-muted-foreground rounded-full border-[1.5px] px-2 py-0.5 text-xs font-black">
            {captiveStatusLabel(captive.status)}
          </span>
          {cooldownActive && cooldown ? (
            <span className="bg-pebble-spark-muted text-pebble-spark-foreground border-border rounded-full border-[1.5px] px-2 py-0.5 text-xs font-black tabular-nums">
              冷卻 {cooldown}
            </span>
          ) : null}
        </div>
        {captive.status === "convert_pending" ? (
          <span className="text-muted-foreground text-xs font-black">
            研三舍處理中
          </span>
        ) : (
          <Button
            size="sm"
            disabled={!canConvert || convertMutation.isPending}
            onClick={() => convertMutation.mutate()}
          >
            {convertMutation.isPending
              ? "送出中"
              : cooldownActive
                ? "冷卻中無法轉正"
                : "送研三轉正"}
          </Button>
        )}
      </div>
    </Card>
  )
}
