import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { Link } from "@tanstack/react-router"
import { Check, Pencil, RefreshCcw, RotateCcw, ScanLine, X } from "lucide-react"
import { QRCodeSVG } from "qrcode.react"
import { FormEvent, useMemo, useState } from "react"
import { toast } from "sonner"

import { AppError } from "@/shared/api/error"
import {
  gameApi,
  type HomeResponse,
  type PlayerSitone,
  type PlayerStatus,
} from "@/shared/api/game"
import { Button } from "@/shared/ui/button"
import { Card, CardContent } from "@/shared/ui/card"
import { Input } from "@/shared/ui/input"
import { PlayerAvatar } from "@/shared/ui/player-avatar"
import { Skeleton } from "@/shared/ui/skeleton"
import { toOptimizedImageSrc } from "@/shared/utils/image-src"

import { CommunityStandScannerDialog } from "./community-stand-scanner-dialog"

export function ProfileQrPanel() {
  const [communityScannerOpen, setCommunityScannerOpen] = useState(false)
  const [nicknameEditing, setNicknameEditing] = useState(false)
  const [nicknameValue, setNicknameValue] = useState("")
  const queryClient = useQueryClient()
  const statusQuery = useQuery({
    queryKey: ["me", "status"],
    queryFn: gameApi.status,
  })
  const qrcodeQuery = useQuery({
    queryKey: ["me", "qrcode"],
    queryFn: gameApi.qrcode,
    enabled: statusQuery.isSuccess,
    throwOnError: false,
  })
  const sitonesQuery = useQuery({
    queryKey: ["me", "sitones"],
    queryFn: gameApi.playerSitones,
  })
  const teamSitonesQuery = useQuery({
    queryKey: ["me", "team", "sitones"],
    queryFn: gameApi.teamSitones,
    enabled: Boolean(statusQuery.data?.team),
  })
  const avatarMutation = useMutation({
    mutationFn: gameApi.updateAvatar,
    onSuccess: (result) => {
      const nextAvatarUrl = result.avatarUrl
      queryClient.setQueryData<PlayerStatus>(["me", "status"], (current) =>
        current ? { ...current, avatarUrl: nextAvatarUrl } : current,
      )
      queryClient.setQueryData<HomeResponse>(["me", "home"], (current) =>
        current
          ? {
              ...current,
              player: { ...current.player, avatarUrl: nextAvatarUrl },
            }
          : current,
      )
      void queryClient.invalidateQueries({ queryKey: ["me"] })
      void queryClient.invalidateQueries({ queryKey: ["leaderboards"] })
      void queryClient.invalidateQueries({ queryKey: ["staff", "players"] })
      void queryClient.invalidateQueries({ queryKey: ["matches"] })
      toast.success("頭貼已更新")
    },
    onError: (error) => {
      toast.error(error instanceof AppError ? error.message : "頭貼更新失敗")
    },
  })
  const nicknameMutation = useMutation({
    mutationFn: gameApi.updateNickname,
    onSuccess: (result) => {
      setNicknameEditing(false)
      setNicknameValue(result.nickname)
      queryClient.setQueryData<PlayerStatus>(["me", "status"], (current) =>
        current ? { ...current, nickname: result.nickname } : current,
      )
      void queryClient.invalidateQueries({ queryKey: ["me", "status"] })
      void queryClient.invalidateQueries({ queryKey: ["me", "home"] })
      void queryClient.invalidateQueries({ queryKey: ["leaderboards"] })
      toast.success("暱稱已更新")
    },
    onError: (error) => {
      toast.error(error instanceof AppError ? error.message : "暱稱更新失敗")
    },
  })
  const teamAvatarMutation = useMutation({
    mutationFn: gameApi.updateTeamAvatar,
    onSuccess: (result) => {
      queryClient.setQueryData<PlayerStatus>(["me", "status"], (current) =>
        current ? { ...current, team: result.team } : current,
      )
      queryClient.setQueryData<HomeResponse>(["me", "home"], (current) =>
        current
          ? {
              ...current,
              player: { ...current.player, team: result.team },
            }
          : current,
      )
      void queryClient.invalidateQueries({ queryKey: ["me"] })
      void queryClient.invalidateQueries({ queryKey: ["leaderboards"] })
      void queryClient.invalidateQueries({ queryKey: ["staff", "teams"] })
      toast.success("小隊頭貼已更新")
    },
    onError: (error) => {
      toast.error(
        error instanceof AppError ? error.message : "小隊頭貼更新失敗",
      )
    },
  })
  const isUnauthorized =
    statusQuery.error instanceof AppError && statusQuery.error.status === 401
  const profile = statusQuery.data
  const nicknameInputLength = [...nicknameValue.trim()].length

  function startNicknameEdit() {
    setNicknameValue(profile?.nickname ?? "")
    setNicknameEditing(true)
  }

  function cancelNicknameEdit() {
    setNicknameValue(profile?.nickname ?? "")
    setNicknameEditing(false)
  }

  function submitNickname(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    const nickname = nicknameValue.trim()
    const length = [...nickname].length
    if (length < 1) {
      toast.error("請輸入暱稱")
      return
    }
    if (length > 20) {
      toast.error("暱稱最多 20 個字")
      return
    }
    if (nickname === profile?.nickname) {
      setNicknameEditing(false)
      return
    }
    nicknameMutation.mutate(nickname)
  }

  if (isUnauthorized) {
    return (
      <Card className="border-ink rounded-[var(--radius)] border-2 py-0 shadow-[4px_4px_0_rgba(23,35,58,0.14)]">
        <CardContent className="p-5">
          <h2 className="mb-2 text-[24px] font-black">請先登入</h2>
          <p className="text-muted-foreground mb-4 leading-relaxed">
            登入後才能開啟 QRCode 掃描器。
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
      <Card className="border-ink rounded-[32px] border-2 py-0 shadow-[5px_5px_0_rgba(23,35,58,0.16)]">
        <CardContent className="grid gap-5 px-[22px] pt-7 pb-6 text-center">
          <PersonalQRCode
            token={qrcodeQuery.data?.qrcodeToken}
            loading={statusQuery.isPending || qrcodeQuery.isPending}
            error={qrcodeQuery.error}
            retrying={qrcodeQuery.isFetching}
            onRetry={() => void qrcodeQuery.refetch()}
          />
          <div>
            <p className="text-muted-foreground mb-1 text-xs font-black tracking-[0.08em] uppercase">
              PLAYER QR
            </p>
            <h2 className="mb-2 text-[26px] font-black tracking-tight">
              個人 QR Code
            </h2>
            <p className="text-muted-foreground mx-auto max-w-[16rem] leading-relaxed">
              給工作人員掃描，將獎勵發送到你的通行證。
            </p>
          </div>
        </CardContent>
      </Card>

      <Card className="border-ink rounded-[24px] border-2 py-0 shadow-[4px_4px_0_rgba(23,35,58,0.14)]">
        <CardContent className="grid gap-3 p-4">
          <div>
            <p className="text-muted-foreground mb-1 text-xs font-black tracking-[0.08em] uppercase">
              SCANNER
            </p>
            <h2 className="mb-2 text-[26px] font-black tracking-tight">
              開啟 QRCode 掃描器
            </h2>
            <p className="text-muted-foreground leading-relaxed">
              掃描工作人員或社群攤位 QR Code，領取對應獎勵。
            </p>
          </div>
          <Button
            type="button"
            className="w-full"
            onClick={() => setCommunityScannerOpen(true)}
          >
            <ScanLine className="size-4" aria-hidden />
            開啟掃描器
          </Button>
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

      <Card className="border-ink rounded-[var(--radius)] border-2 py-0 shadow-[4px_4px_0_rgba(23,35,58,0.14)]">
        <CardContent className="grid grid-cols-[70px_1fr] items-center gap-3.5 p-4">
          <PlayerAvatar
            playerId={profile?.playerId}
            nickname={profile?.nickname}
            avatarUrl={profile?.avatarUrl}
            size="lg"
            className="bg-power border-ink h-[70px] w-[70px] rounded-[24px] border-2 text-3xl"
          />
          <div className="min-w-0">
            <span className="text-muted-foreground block text-xs font-black tracking-widest uppercase">
              玩家身份
            </span>
            {nicknameEditing ? (
              <form className="mt-1 grid gap-2" onSubmit={submitNickname}>
                <Input
                  aria-label="暱稱"
                  maxLength={40}
                  value={nicknameValue}
                  disabled={nicknameMutation.isPending}
                  onChange={(event) => setNicknameValue(event.target.value)}
                />
                <div className="flex items-center justify-between gap-2">
                  <span
                    className={
                      nicknameInputLength > 20
                        ? "text-destructive text-xs font-bold"
                        : "text-muted-foreground text-xs font-bold"
                    }
                  >
                    {nicknameInputLength}/20
                  </span>
                  <div className="flex gap-2">
                    <Button
                      type="button"
                      variant="outline"
                      size="icon"
                      aria-label="取消修改暱稱"
                      disabled={nicknameMutation.isPending}
                      onClick={cancelNicknameEdit}
                    >
                      <X className="size-4" aria-hidden />
                    </Button>
                    <Button
                      type="submit"
                      size="icon"
                      aria-label="儲存暱稱"
                      disabled={nicknameMutation.isPending}
                    >
                      <Check className="size-4" aria-hidden />
                    </Button>
                  </div>
                </div>
              </form>
            ) : (
              <div className="mb-1 flex min-w-0 items-center gap-2">
                <h2 className="min-w-0 truncate text-[28px] leading-none font-black tracking-tight">
                  {profile?.nickname ?? "同步中"}
                </h2>
                <Button
                  type="button"
                  variant="ghost"
                  size="icon"
                  aria-label="修改暱稱"
                  disabled={!profile}
                  onClick={startNicknameEdit}
                >
                  <Pencil className="size-4" aria-hidden />
                </Button>
              </div>
            )}
            <p className="text-muted-foreground">
              {profile
                ? `${profile.team?.name ?? (profile.role === "staff" ? "工作人員" : "未分組")} · 開源力 ${profile.openPower}`
                : "讀取玩家資料"}
            </p>
          </div>
        </CardContent>
      </Card>

      <AvatarPickerCard
        currentAvatarUrl={profile?.avatarUrl}
        sitones={sitonesQuery.data ?? []}
        loading={sitonesQuery.isLoading}
        pending={avatarMutation.isPending}
        onSelect={(sitoneId) => avatarMutation.mutate(sitoneId)}
        onReset={() => avatarMutation.mutate(null)}
      />

      {profile?.team ? (
        <TeamAvatarPickerCard
          currentAvatarUrl={profile.team.avatarUrl}
          sitones={teamSitonesQuery.data ?? []}
          loading={teamSitonesQuery.isLoading}
          pending={teamAvatarMutation.isPending}
          teamName={profile.team.name}
          onSelect={(sitoneId) => teamAvatarMutation.mutate(sitoneId)}
          onReset={() => teamAvatarMutation.mutate(null)}
        />
      ) : null}

      <CommunityStandScannerDialog
        open={communityScannerOpen}
        onOpenChange={setCommunityScannerOpen}
      />
    </div>
  )
}

type PersonalQRCodeProps = {
  token?: string
  loading: boolean
  error: unknown
  retrying: boolean
  onRetry: () => void
}

function PersonalQRCode({
  token,
  loading,
  error,
  retrying,
  onRetry,
}: PersonalQRCodeProps) {
  if (loading) {
    return (
      <div
        className="bg-paper border-ink mx-auto grid aspect-square w-full max-w-[306px] place-items-center rounded-[18px] border-4 p-[18px]"
        aria-label="載入個人 QR Code"
      >
        <Skeleton className="h-full w-full rounded-[12px]" />
      </div>
    )
  }

  if (error || !token) {
    return (
      <div className="bg-paper border-ink mx-auto grid aspect-square w-full max-w-[306px] place-items-center rounded-[18px] border-4 p-[18px]">
        <div className="bg-surface-raised border-ink grid h-full w-full place-items-center rounded-[12px] border-2 p-5 text-center">
          <div className="grid justify-items-center gap-3">
            <p className="text-[20px] leading-tight font-black">
              QR Code 暫時無法顯示
            </p>
            <p className="text-muted-foreground text-sm leading-relaxed">
              {error instanceof AppError ? error.message : "請稍後再重新整理。"}
            </p>
            <Button
              type="button"
              variant="secondary"
              size="sm"
              disabled={retrying}
              onClick={onRetry}
            >
              <RefreshCcw className="size-4" aria-hidden />
              重新整理
            </Button>
          </div>
        </div>
      </div>
    )
  }

  return (
    <div className="bg-paper border-ink mx-auto grid aspect-square w-full max-w-[306px] place-items-center rounded-[18px] border-4 p-[18px]">
      <QRCodeSVG
        aria-label="個人 QR Code"
        bgColor="var(--paper)"
        className="h-full w-full"
        fgColor="var(--ink)"
        level="M"
        marginSize={4}
        role="img"
        size={256}
        title="個人 QR Code"
        value={token}
      />
    </div>
  )
}

type AvatarPickerCardProps = {
  currentAvatarUrl?: string
  sitones: PlayerSitone[]
  loading: boolean
  pending: boolean
  onSelect: (sitoneId: string) => void
  onReset: () => void
}

function AvatarPickerCard({
  currentAvatarUrl,
  sitones,
  loading,
  pending,
  onSelect,
  onReset,
}: AvatarPickerCardProps) {
  const avatarSitones = useMemo(
    () => sitones.filter((record) => record.sitone.iconPath),
    [sitones],
  )

  return (
    <Card className="border-ink rounded-[24px] border-2 py-0 shadow-[4px_4px_0_rgba(23,35,58,0.14)]">
      <CardContent className="grid gap-3 p-4">
        <div className="flex items-start justify-between gap-3">
          <div>
            <p className="text-muted-foreground mb-1 text-xs font-black tracking-[0.08em] uppercase">
              AVATAR
            </p>
            <h2 className="text-[24px] leading-tight font-black">頭貼小石</h2>
          </div>
          <Button
            type="button"
            variant={currentAvatarUrl ? "outline" : "secondary"}
            size="sm"
            disabled={pending || !currentAvatarUrl}
            onClick={onReset}
          >
            <RotateCcw className="size-4" aria-hidden />
            預設
          </Button>
        </div>

        {loading ? (
          <div className="grid grid-cols-4 gap-2" aria-label="載入頭貼小石">
            {Array.from({ length: 4 }).map((_, index) => (
              <Skeleton key={index} className="aspect-square rounded-[18px]" />
            ))}
          </div>
        ) : avatarSitones.length > 0 ? (
          <div
            className="grid grid-cols-4 gap-2"
            role="group"
            aria-label="選擇頭貼小石"
          >
            {avatarSitones.map((record) => {
              const iconPath = record.sitone.iconPath
              const selected = iconPath === currentAvatarUrl

              return (
                <button
                  key={record.id}
                  type="button"
                  className={[
                    "border-ink bg-card grid h-[76px] min-w-0 place-items-center overflow-hidden rounded-[18px] border-2 p-1.5 shadow-[2px_2px_0_rgba(23,35,58,0.12)] transition",
                    selected ? "ring-primary ring-4" : "",
                  ].join(" ")}
                  aria-label={`選擇${record.sitone.name}作為頭貼`}
                  aria-pressed={selected}
                  disabled={pending || selected}
                  onClick={() => onSelect(record.sitoneId)}
                >
                  <img
                    src={toOptimizedImageSrc(iconPath)}
                    alt=""
                    aria-hidden="true"
                    draggable={false}
                    className="h-full max-h-[66px] w-full object-contain"
                  />
                  <span className="sr-only">{record.sitone.name}</span>
                </button>
              )
            })}
          </div>
        ) : (
          <p className="text-muted-foreground rounded-[18px] border-2 border-dashed p-4 text-center text-sm font-bold">
            取得小石後就能把牠設為頭貼。
          </p>
        )}
      </CardContent>
    </Card>
  )
}

type TeamAvatarPickerCardProps = {
  currentAvatarUrl?: string
  sitones: PlayerSitone[]
  loading: boolean
  pending: boolean
  teamName: string
  onSelect: (sitoneId: string) => void
  onReset: () => void
}

function TeamAvatarPickerCard({
  currentAvatarUrl,
  sitones,
  loading,
  pending,
  teamName,
  onSelect,
  onReset,
}: TeamAvatarPickerCardProps) {
  const avatarSitones = useMemo(
    () => sitones.filter((record) => record.sitone.iconPath),
    [sitones],
  )

  return (
    <Card className="border-ink rounded-[24px] border-2 py-0 shadow-[4px_4px_0_rgba(23,35,58,0.14)]">
      <CardContent className="grid gap-3 p-4">
        <div className="flex items-start justify-between gap-3">
          <div>
            <p className="text-muted-foreground mb-1 text-xs font-black tracking-[0.08em] uppercase">
              TEAM AVATAR
            </p>
            <h2 className="text-[24px] leading-tight font-black">小隊頭貼</h2>
            <p className="text-muted-foreground mt-1 text-sm font-bold">
              {teamName} · 隊員持有小石可選
            </p>
          </div>
          <Button
            type="button"
            variant={currentAvatarUrl ? "outline" : "secondary"}
            size="sm"
            disabled={pending || !currentAvatarUrl}
            onClick={onReset}
          >
            <RotateCcw className="size-4" aria-hidden />
            預設
          </Button>
        </div>

        {loading ? (
          <div className="grid grid-cols-4 gap-2" aria-label="載入小隊頭貼小石">
            {Array.from({ length: 8 }).map((_, index) => (
              <Skeleton key={index} className="aspect-square rounded-[18px]" />
            ))}
          </div>
        ) : avatarSitones.length > 0 ? (
          <div
            className="grid max-h-[306px] grid-cols-4 gap-2 overflow-y-auto pr-1"
            role="group"
            aria-label="選擇小隊頭貼小石"
          >
            {avatarSitones.map((record) => {
              const iconPath = record.sitone.iconPath
              const selected = iconPath === currentAvatarUrl

              return (
                <button
                  key={record.sitoneId}
                  type="button"
                  className={[
                    "border-ink bg-card grid h-[76px] min-w-0 place-items-center overflow-hidden rounded-[18px] border-2 p-1.5 shadow-[2px_2px_0_rgba(23,35,58,0.12)] transition",
                    selected ? "ring-primary ring-4" : "",
                  ].join(" ")}
                  aria-label={`選擇${record.sitone.name}作為小隊頭貼`}
                  aria-pressed={selected}
                  disabled={pending || selected}
                  onClick={() => onSelect(record.sitoneId)}
                >
                  <img
                    src={toOptimizedImageSrc(iconPath)}
                    alt=""
                    aria-hidden="true"
                    draggable={false}
                    className="h-full max-h-[66px] w-full object-contain"
                  />
                  <span className="sr-only">{record.sitone.name}</span>
                </button>
              )
            })}
          </div>
        ) : (
          <p className="text-muted-foreground rounded-[18px] border-2 border-dashed p-4 text-center text-sm font-bold">
            隊員取得小石後就能把牠設為小隊頭貼。
          </p>
        )}
      </CardContent>
    </Card>
  )
}
