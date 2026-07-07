import { useQuery } from "@tanstack/react-query"
import { Link } from "@tanstack/react-router"
import {
  ArrowLeft,
  ExternalLink,
  Gift,
  RefreshCcw,
  UsersRound,
} from "lucide-react"
import { QRCodeSVG } from "qrcode.react"
import { useMemo } from "react"

import { gameApi, type CommunityStandReward } from "@/shared/api/game"
import { communityStandQrValue } from "@/shared/lib/community-stand-qr"
import { Button } from "@/shared/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/shared/ui/card"
import { GamePageShell } from "@/shared/ui/game-page-shell"
import { Skeleton } from "@/shared/ui/skeleton"
import { toOptimizedImageSrc } from "@/shared/utils/image-src"

type CommunityStandDisplayPageProps = {
  standID: string
}

export function CommunityStandDisplayPage({
  standID,
}: CommunityStandDisplayPageProps) {
  const standQuery = useQuery({
    queryKey: ["community", "stand", "display", standID],
    queryFn: () => gameApi.communityStandDisplay(standID),
    refetchInterval: 15_000,
  })
  const qrValue = useMemo(
    () => communityStandQrValue(standQuery.data?.qrToken ?? ""),
    [standQuery.data?.qrToken],
  )

  return (
    <GamePageShell contentClassName="gap-3">
      <header className="flex items-center gap-3">
        <Link
          to="/"
          aria-label="返回營隊基地"
          className="border-ink bg-card text-ink focus-visible:outline-power grid size-11 shrink-0 place-items-center rounded-2xl border-2 transition-transform focus-visible:outline-3 focus-visible:outline-offset-2 active:translate-y-px"
        >
          <ArrowLeft className="size-5" aria-hidden />
        </Link>
        <div>
          <p className="text-muted-foreground mb-1 text-xs font-black tracking-[0.08em] uppercase">
            COMMUNITY STAND
          </p>
          <h1 className="text-[29px] leading-[1.05] font-black tracking-tight">
            社群攤位看板
          </h1>
        </div>
      </header>

      {standQuery.isPending ? (
        <DisplayLoading />
      ) : standQuery.error ? (
        <Card className="border-ink rounded-[24px] border-2 py-0">
          <CardContent className="grid gap-3 p-5">
            <h2 className="text-2xl font-black">找不到這個攤位</h2>
            <p className="text-muted-foreground leading-relaxed">
              請確認網址是否為本次活動的社群攤位網址。
            </p>
            <Button
              type="button"
              variant="secondary"
              onClick={() => void standQuery.refetch()}
            >
              <RefreshCcw className="size-4" aria-hidden />
              重新整理
            </Button>
          </CardContent>
        </Card>
      ) : (
        <div className="grid gap-3">
          <section className="grid content-start gap-3">
            <Card className="border-ink rounded-[28px] border-2 py-0 shadow-[5px_5px_0_rgba(23,35,58,0.16)]">
              <CardContent className="grid gap-4 p-5">
                <div className="grid grid-cols-[80px_minmax(0,1fr)] items-start gap-3">
                  <div className="bg-surface-raised border-ink grid size-20 shrink-0 place-items-center overflow-hidden rounded-[22px] border-2">
                    {standQuery.data.stand.logoUrl ? (
                      <img
                        src={standQuery.data.stand.logoUrl}
                        alt=""
                        className="h-full w-full object-cover"
                      />
                    ) : (
                      <UsersRound className="size-8" aria-hidden />
                    )}
                  </div>
                  <div className="min-w-0">
                    <h2 className="text-[28px] leading-tight font-black break-words">
                      {standQuery.data.stand.name}
                    </h2>
                    <p className="text-muted-foreground mt-2 leading-relaxed break-words whitespace-pre-line">
                      {standQuery.data.stand.description}
                    </p>
                  </div>
                </div>

                {standQuery.data.stand.websiteUrl ? (
                  <Button asChild variant="outline" className="w-full">
                    <a
                      href={standQuery.data.stand.websiteUrl}
                      target="_blank"
                      rel="noreferrer"
                    >
                      <ExternalLink className="size-4" aria-hidden />
                      社群網站
                    </a>
                  </Button>
                ) : null}
              </CardContent>
            </Card>

            <div className="grid grid-cols-2 gap-3">
              <MetricCard
                label="來過學員"
                value={standQuery.data.visitCount}
                caption="掃描後載入攤位資訊"
              />
              <MetricCard
                label="已領獎勵"
                value={standQuery.data.claimCount}
                caption="完成領取的學員"
              />
            </div>

            <Card className="border-ink rounded-[22px] border-2 py-0">
              <CardContent className="flex items-center gap-3 p-4">
                {standQuery.data.stand.reward.iconPath ? (
                  <img
                    src={toOptimizedImageSrc(
                      standQuery.data.stand.reward.iconPath,
                    )}
                    alt=""
                    className="border-ink bg-card size-12 rounded-[14px] border-2 object-cover"
                  />
                ) : (
                  <div className="border-ink bg-card grid size-12 place-items-center rounded-[14px] border-2">
                    <Gift className="size-5" aria-hidden />
                  </div>
                )}
                <div className="min-w-0">
                  <p className="text-muted-foreground text-xs font-black tracking-[0.08em] uppercase">
                    REWARD
                  </p>
                  <p className="font-black">
                    {rewardText(standQuery.data.stand.reward)}
                  </p>
                </div>
              </CardContent>
            </Card>
          </section>

          <Card className="border-ink rounded-[28px] border-2 py-0 shadow-[5px_5px_0_rgba(23,35,58,0.16)]">
            <CardHeader>
              <CardTitle className="text-center text-2xl font-black">
                攤位 QR Code
              </CardTitle>
            </CardHeader>
            <CardContent className="grid gap-4 px-5 pb-5 text-center">
              <div className="bg-paper border-ink mx-auto grid aspect-square w-full max-w-[288px] place-items-center rounded-[18px] border-4 p-4">
                <QRCodeSVG
                  aria-label={`${standQuery.data.stand.name} 攤位 QR Code`}
                  bgColor="var(--paper)"
                  className="h-full w-full"
                  fgColor="var(--ink)"
                  level="M"
                  marginSize={4}
                  role="img"
                  size={256}
                  title={`${standQuery.data.stand.name} 攤位 QR Code`}
                  value={qrValue}
                />
              </div>
              <Button
                type="button"
                variant="secondary"
                className="w-full"
                onClick={() => void standQuery.refetch()}
              >
                <RefreshCcw className="size-4" aria-hidden />
                更新人數
              </Button>
            </CardContent>
          </Card>
        </div>
      )}
    </GamePageShell>
  )
}

function DisplayLoading() {
  return (
    <div className="grid gap-3">
      <Card className="border-ink rounded-[28px] border-2 py-0">
        <CardContent className="grid gap-4 p-5">
          <Skeleton className="size-20 rounded-[22px]" />
          <Skeleton className="h-8 w-2/3" />
          <Skeleton className="h-24 w-full" />
        </CardContent>
      </Card>
      <Card className="border-ink rounded-[28px] border-2 py-0">
        <CardContent className="grid gap-4 p-5">
          <Skeleton className="aspect-square w-full rounded-[18px]" />
          <Skeleton className="h-10 w-full" />
        </CardContent>
      </Card>
    </div>
  )
}

function MetricCard({
  label,
  value,
  caption,
}: {
  label: string
  value: number
  caption: string
}) {
  return (
    <Card className="border-ink rounded-[20px] border-2 py-0">
      <CardContent className="p-4 text-center">
        <p className="text-muted-foreground text-xs font-black tracking-[0.08em] uppercase">
          {label}
        </p>
        <strong className="block text-[38px] leading-none font-black">
          {value}
        </strong>
        <p className="text-muted-foreground mt-2 text-sm font-bold">
          {caption}
        </p>
      </CardContent>
    </Card>
  )
}

function rewardText(reward: CommunityStandReward) {
  if (reward.kind === "open_power") return `${reward.amount ?? 0} 開源力`
  return `${reward.name} x${reward.quantity ?? 1}`
}
