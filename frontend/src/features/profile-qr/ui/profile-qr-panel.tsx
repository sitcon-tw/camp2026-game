import { useQuery } from "@tanstack/react-query"
import { Link } from "@tanstack/react-router"
import { ScanLine } from "lucide-react"
import { QRCodeSVG } from "qrcode.react"
import { useState } from "react"

import { AppError } from "@/shared/api/error"
import { gameApi } from "@/shared/api/game"
import { Button } from "@/shared/ui/button"
import { Card, CardContent } from "@/shared/ui/card"
import { PlayerAvatar } from "@/shared/ui/player-avatar"
import { Skeleton } from "@/shared/ui/skeleton"

import { CommunityStandScannerDialog } from "./community-stand-scanner-dialog"

export function ProfileQrPanel() {
  const [communityScannerOpen, setCommunityScannerOpen] = useState(false)
  const statusQuery = useQuery({
    queryKey: ["me", "status"],
    queryFn: gameApi.status,
  })
  const qrQuery = useQuery({
    queryKey: ["me", "qrcode"],
    queryFn: gameApi.qrcode,
  })
  const isUnauthorized =
    (statusQuery.error instanceof AppError &&
      statusQuery.error.status === 401) ||
    (qrQuery.error instanceof AppError && qrQuery.error.status === 401)
  const profile = statusQuery.data
  const qrcodeToken = qrQuery.data?.qrcodeToken

  if (isUnauthorized) {
    return (
      <Card className="border-ink rounded-[var(--radius)] border-2 py-0 shadow-[4px_4px_0_rgba(23,35,58,0.14)]">
        <CardContent className="p-5">
          <h2 className="mb-2 text-[24px] font-black">請先登入</h2>
          <p className="text-muted-foreground mb-4 leading-relaxed">
            登入後才能產生個人 QR Code。
          </p>
          <Button asChild className="w-full">
            <Link to="/login">前往登入</Link>
          </Button>
        </CardContent>
      </Card>
    )
  }

  return (
    <div className="flex flex-col gap-3.5">
      <Card className="border-ink rounded-[var(--radius)] border-2 py-0 shadow-[4px_4px_0_rgba(23,35,58,0.14)]">
        <CardContent className="grid grid-cols-[70px_1fr] items-center gap-3.5 p-4">
          <PlayerAvatar
            playerId={profile?.playerId}
            nickname={profile?.nickname}
            size="lg"
            className="bg-power border-ink h-[70px] w-[70px] rounded-[24px] border-2 text-3xl"
          />
          <div>
            <span className="text-muted-foreground block text-xs font-black tracking-widest uppercase">
              玩家身份
            </span>
            <h2 className="mb-1 text-[28px] leading-none font-black tracking-tight">
              {profile?.nickname ?? "同步中"}
            </h2>
            <p className="text-muted-foreground">
              {profile
                ? `${profile.team?.name ?? (profile.role === "staff" ? "工作人員" : "未分組")} · 開源力 ${profile.openPower}`
                : "讀取玩家資料"}
            </p>
          </div>
        </CardContent>
      </Card>

      <Card className="border-ink rounded-[32px] border-2 py-0 shadow-[5px_5px_0_rgba(23,35,58,0.16)]">
        <CardContent className="px-[22px] pt-7 pb-6 text-center">
          <div className="bg-paper border-ink mx-auto mb-5 grid aspect-square w-full max-w-[306px] place-items-center rounded-[18px] border-4 p-[18px]">
            {qrcodeToken ? (
              <QRCodeSVG
                aria-label="玩家身份 QR Code"
                bgColor="var(--paper)"
                className="h-full w-full"
                fgColor="var(--ink)"
                level="M"
                marginSize={4}
                role="img"
                size={256}
                title="玩家身份 QR Code"
                value={qrcodeToken}
              />
            ) : qrQuery.isError ? (
              <div className="flex h-full w-full flex-col items-center justify-center gap-3 text-center">
                <p className="text-muted-foreground text-sm font-bold">
                  QR Code 暫時無法產生
                </p>
                <Button
                  size="sm"
                  variant="secondary"
                  onClick={() => void qrQuery.refetch()}
                >
                  重新整理
                </Button>
              </div>
            ) : (
              <Skeleton className="h-full w-full rounded-[12px]" />
            )}
          </div>
          <h2 className="mb-2 text-[26px] font-black tracking-tight">
            請掃描這個 QR Code
          </h2>
          <p className="text-muted-foreground mx-auto max-w-[15rem] leading-relaxed">
            出示給工作人員掃描，用來確認身份與任務紀錄。
          </p>
        </CardContent>
      </Card>

      <Card className="border-ink rounded-[24px] border-2 py-0 shadow-[4px_4px_0_rgba(23,35,58,0.14)]">
        <CardContent className="grid gap-3 p-4">
          <div>
            <p className="text-muted-foreground mb-1 text-xs font-black tracking-[0.08em] uppercase">
              COMMUNITY
            </p>
            <h2 className="text-[24px] leading-tight font-black">
              社群攤位獎勵
            </h2>
            <p className="text-muted-foreground mt-1 leading-relaxed">
              掃描攤位 QR Code，查看社群資訊並領取對應道具。
            </p>
          </div>
          <Button
            type="button"
            variant="secondary"
            className="w-full"
            onClick={() => setCommunityScannerOpen(true)}
          >
            <ScanLine className="size-4" aria-hidden />
            掃描社群攤位
          </Button>
        </CardContent>
      </Card>

      <CommunityStandScannerDialog
        open={communityScannerOpen}
        onOpenChange={setCommunityScannerOpen}
      />
    </div>
  )
}
