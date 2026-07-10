import { useMutation, useQuery } from "@tanstack/react-query"
import {
  CheckCircle2Icon,
  RotateCcwIcon,
  ScanLineIcon,
  SearchIcon,
} from "lucide-react"
import { QRCodeSVG } from "qrcode.react"
import { type FormEvent, useMemo, useState } from "react"
import { toast } from "sonner"

import { AppError } from "@/shared/api/error"
import {
  gameApi,
  type StaffPlayer,
  type StaffRoomTeamTokenResponse,
} from "@/shared/api/game"
import { PlayerQrScannerDialog } from "@/features/staff-rewards"
import {
  formatTeamName,
  groupDormRooms,
  roomDisplayName,
} from "@/shared/lib/game-labels"
import { roomTeamQrValue } from "@/shared/lib/room-team-qr"
import { Button } from "@/shared/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/shared/ui/card"
import { GameFeatureIcon } from "@/shared/ui/game-feature-icon"
import { Input } from "@/shared/ui/input"
import { PlayerAvatar } from "@/shared/ui/player-avatar"
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectLabel,
  SelectSeparator,
  SelectTrigger,
  SelectValue,
} from "@/shared/ui/select"

function errorMessage(error: unknown, fallback: string) {
  return error instanceof AppError ? error.message : fallback
}

type TargetPlayer = {
  playerId: string
  nickname: string
  team?: StaffPlayer["team"]
  avatarUrl?: string
}

function RoomTeamTokenCard({ token }: { token: StaffRoomTeamTokenResponse }) {
  const roomLabel = roomDisplayName(token.room.roomNumber)
  const expiresLabel = new Intl.DateTimeFormat("zh-TW", {
    hour: "2-digit",
    minute: "2-digit",
  }).format(new Date(token.qrTokenExpiresAt))

  return (
    <div className="border-border grid justify-items-center gap-4 border-t pt-5 text-center">
      <div className="bg-paper border-ink grid aspect-square w-full max-w-[272px] place-items-center rounded-[18px] border-4 p-4">
        <QRCodeSVG
          aria-label={`${roomLabel} 宿舍 QR Code`}
          bgColor="var(--paper)"
          className="h-full w-full"
          fgColor="var(--ink)"
          level="M"
          marginSize={4}
          role="img"
          size={240}
          title={`${roomLabel} 宿舍 QR Code`}
          value={roomTeamQrValue(token.qrToken)}
        />
      </div>
      <div>
        <h2 className="text-[24px] leading-tight font-black">{roomLabel}</h2>
        <p className="text-muted-foreground mt-1 text-sm font-bold">
          有效至 {expiresLabel}
        </p>
      </div>
    </div>
  )
}

export function StaffRoomTeamsPanel() {
  const [qrRoomNumber, setQRRoomNumber] = useState("")
  const [assignmentRoomNumber, setAssignmentRoomNumber] = useState("")
  const [scannerOpen, setScannerOpen] = useState(false)
  const [manualToken, setManualToken] = useState("")
  const [playerSearch, setPlayerSearch] = useState("")
  const [targetPlayer, setTargetPlayer] = useState<TargetPlayer | null>(null)
  const statusQuery = useQuery({
    queryKey: ["me", "status"],
    queryFn: gameApi.status,
  })
  const isStaff = statusQuery.data?.role === "staff"
  const playerSearchKeyword = playerSearch.trim()
  const roomTeamsQuery = useQuery({
    queryKey: ["staff", "room-teams"],
    queryFn: gameApi.staffRoomTeams,
    enabled: isStaff,
  })
  const groupedRoomTeams = useMemo(
    () => groupDormRooms(roomTeamsQuery.data ?? []),
    [roomTeamsQuery.data],
  )
  const assignmentRoomLabel = assignmentRoomNumber
    ? roomDisplayName(assignmentRoomNumber)
    : "宿舍"
  const playersQuery = useQuery({
    queryKey: ["staff", "players", playerSearchKeyword],
    queryFn: () => gameApi.staffPlayers(playerSearchKeyword),
    enabled: isStaff && playerSearchKeyword.length > 0,
  })
  const roomTokenMutation = useMutation({
    mutationFn: gameApi.createStaffRoomTeamToken,
    onSuccess: (result) => {
      const roomName = roomDisplayName(result.room.roomNumber)
      toast.success(`已產生 ${roomName} 宿舍 QR Code`)
    },
    onError: (error) => {
      toast.error(errorMessage(error, "宿舍 QR Code 產生失敗"))
    },
  })
  const resolveMutation = useMutation({
    mutationFn: gameApi.resolveQRCode,
    onSuccess: (player, token) => {
      setManualToken(token)
      setTargetPlayer(player)
    },
    onError: (error) => {
      setTargetPlayer(null)
      toast.error(errorMessage(error, "無法確認學員 QR Code"))
    },
  })
  const roomAssignmentMutation = useMutation({
    mutationFn: ({
      roomNumber: targetRoomNumber,
      playerId,
    }: {
      roomNumber: string
      playerId: string
    }) => gameApi.assignStaffRoomTeamMember(targetRoomNumber, { playerId }),
    onSuccess: (result) => {
      const roomName = roomDisplayName(result.room.roomNumber)
      setManualToken("")
      setTargetPlayer({
        playerId: result.player.playerId,
        nickname: result.player.nickname,
        avatarUrl: result.player.avatarUrl,
      })
      toast.success(
        result.added
          ? `已將 ${result.player.nickname} 分配至 ${roomName}`
          : `${result.player.nickname} 已在 ${roomName}`,
      )
    },
    onError: (error) => {
      toast.error(errorMessage(error, "手動分配失敗"))
    },
  })

  if (statusQuery.isSuccess && !isStaff) {
    return (
      <Card className="border-ink rounded-[22px] border-2">
        <CardHeader>
          <CardTitle className="text-xl font-black">沒有 staff 權限</CardTitle>
        </CardHeader>
        <CardContent className="text-muted-foreground leading-relaxed">
          這個頁面只開放工作人員使用。
        </CardContent>
      </Card>
    )
  }

  function handleManualAssignment(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (!assignmentRoomNumber || !targetPlayer) return
    roomAssignmentMutation.mutate({
      roomNumber: assignmentRoomNumber,
      playerId: targetPlayer.playerId,
    })
  }

  function resolveToken(token: string) {
    const normalized = token.trim()
    setManualToken(normalized)
    setTargetPlayer(null)
    if (!normalized) return
    resolveMutation.mutate(normalized)
  }

  function selectTargetPlayer(player: StaffPlayer) {
    setManualToken("")
    setTargetPlayer(player)
  }

  return (
    <section className="grid gap-3" aria-label="宿舍管理">
      <Card className="border-ink rounded-[22px] border-2">
        <CardHeader className="px-5">
          <CardTitle className="flex items-center gap-2 text-xl font-black">
            <GameFeatureIcon name="dorm" className="size-5" />
            QR Code 生成
          </CardTitle>
        </CardHeader>
        <CardContent className="grid gap-5 px-5">
          <div className="grid grid-cols-[minmax(0,1fr)_auto] gap-2">
            <Select
              value={qrRoomNumber}
              onValueChange={(value) => {
                setQRRoomNumber(value)
                roomTokenMutation.reset()
                roomTokenMutation.mutate(value)
              }}
              disabled={roomTeamsQuery.isPending}
            >
              <SelectTrigger className="h-12 w-full">
                <SelectValue
                  placeholder={
                    roomTeamsQuery.isPending ? "讀取宿舍中" : "選擇宿舍房號"
                  }
                />
              </SelectTrigger>
              <SelectContent>
                {groupedRoomTeams.map((group, index) => (
                  <SelectGroup key={group.key}>
                    {index > 0 ? <SelectSeparator /> : null}
                    <SelectLabel className="px-2 py-2 text-[11px] font-black normal-case opacity-70">
                      {group.label}
                    </SelectLabel>
                    {group.rooms.map((room) => (
                      <SelectItem key={room.roomId} value={room.roomNumber}>
                        {roomDisplayName(room.roomNumber)}
                      </SelectItem>
                    ))}
                  </SelectGroup>
                ))}
              </SelectContent>
            </Select>
            <Button
              type="button"
              variant="outline"
              size="icon"
              title="重新生成 QR Code"
              aria-label="重新生成 QR Code"
              onClick={() => roomTokenMutation.mutate(qrRoomNumber)}
              disabled={!qrRoomNumber || roomTokenMutation.isPending}
            >
              <RotateCcwIcon className="size-4" aria-hidden />
            </Button>
          </div>

          {roomTeamsQuery.isError ? (
            <p className="text-destructive px-1 text-sm font-bold">
              {errorMessage(roomTeamsQuery.error, "讀取宿舍失敗")}
            </p>
          ) : null}

          {roomTokenMutation.isPending ? (
            <p className="text-muted-foreground px-1 text-sm font-bold">
              產生宿舍 QR Code 中
            </p>
          ) : roomTokenMutation.data ? (
            <RoomTeamTokenCard token={roomTokenMutation.data} />
          ) : null}
        </CardContent>
      </Card>

      <Card className="border-ink rounded-[22px] border-2">
        <CardHeader className="px-5">
          <CardTitle className="flex items-center gap-2 text-xl font-black">
            <GameFeatureIcon name="team" className="size-5" />
            手動分配
          </CardTitle>
        </CardHeader>
        <CardContent className="grid gap-3 px-5">
          <Select
            value={assignmentRoomNumber}
            onValueChange={setAssignmentRoomNumber}
            disabled={roomTeamsQuery.isPending}
          >
            <SelectTrigger className="h-12 w-full">
              <SelectValue
                placeholder={
                  roomTeamsQuery.isPending ? "讀取宿舍中" : "選擇分配房號"
                }
              />
            </SelectTrigger>
            <SelectContent>
              {groupedRoomTeams.map((group, index) => (
                <SelectGroup key={group.key}>
                  {index > 0 ? <SelectSeparator /> : null}
                  <SelectLabel className="px-2 py-2 text-[11px] font-black normal-case opacity-70">
                    {group.label}
                  </SelectLabel>
                  {group.rooms.map((room) => (
                    <SelectItem key={room.roomId} value={room.roomNumber}>
                      {roomDisplayName(room.roomNumber)}
                    </SelectItem>
                  ))}
                </SelectGroup>
              ))}
            </SelectContent>
          </Select>
          <form
            className="grid grid-cols-[minmax(0,1fr)_auto_auto] gap-2"
            onSubmit={(event) => {
              event.preventDefault()
              resolveToken(manualToken)
            }}
          >
            <Input
              value={manualToken}
              onChange={(event) => setManualToken(event.target.value)}
              placeholder="學員個人 QR 識別碼"
              autoComplete="off"
              inputMode="text"
              aria-label="學員個人 QR 識別碼"
            />
            <Button
              type="submit"
              disabled={!manualToken.trim() || resolveMutation.isPending}
            >
              確認
            </Button>
            <Button
              type="button"
              variant="secondary"
              size="icon"
              title="掃描學員 QR Code"
              aria-label="掃描學員 QR Code"
              onClick={() => setScannerOpen(true)}
            >
              <ScanLineIcon className="size-4" aria-hidden />
            </Button>
          </form>

          <div className="relative">
            <SearchIcon
              className="text-muted-foreground pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2"
              aria-hidden
            />
            <Input
              value={playerSearch}
              onChange={(event) => setPlayerSearch(event.target.value)}
              placeholder="搜尋學員暱稱或 ID"
              autoComplete="off"
              aria-label="搜尋學員暱稱或 ID"
              className="pl-9"
            />
          </div>

          {playerSearchKeyword ? (
            <div className="grid gap-2" aria-label="學員搜尋結果">
              {playersQuery.isPending ? (
                <p className="text-muted-foreground px-1 text-sm font-bold">
                  搜尋中
                </p>
              ) : playersQuery.isError ? (
                <p className="text-destructive px-1 text-sm font-bold">
                  {errorMessage(playersQuery.error, "搜尋失敗")}
                </p>
              ) : (playersQuery.data ?? []).length > 0 ? (
                (playersQuery.data ?? []).map((player) => {
                  const selected = targetPlayer?.playerId === player.playerId
                  return (
                    <Button
                      key={player.playerId}
                      type="button"
                      variant={selected ? "secondary" : "outline"}
                      className="h-auto w-full justify-start rounded-[16px] px-3 py-2 text-left whitespace-normal shadow-none"
                      onClick={() => selectTargetPlayer(player)}
                    >
                      <span className="grid w-full min-w-0 grid-cols-[minmax(0,1fr)_auto] items-center gap-2">
                        <span className="flex min-w-0 items-center gap-2.5">
                          <PlayerAvatar
                            playerId={player.playerId}
                            nickname={player.nickname}
                            avatarUrl={player.avatarUrl}
                            className="border-ink size-9 rounded-[13px] border-2"
                          />
                          <span className="min-w-0 flex-1">
                            <span className="block truncate text-sm leading-tight font-black">
                              {player.nickname}
                            </span>
                            <span className="text-muted-foreground mt-1 block text-xs leading-snug font-bold break-all whitespace-normal">
                              {player.team?.name
                                ? formatTeamName(player.team.name)
                                : "未分組"}{" "}
                              · {player.playerId}
                            </span>
                          </span>
                        </span>
                        {selected ? (
                          <CheckCircle2Icon className="size-4" aria-hidden />
                        ) : null}
                      </span>
                    </Button>
                  )
                })
              ) : (
                <p className="text-muted-foreground px-1 text-sm font-bold">
                  找不到學員
                </p>
              )}
            </div>
          ) : null}

          <div className="bg-surface-raised border-border grid min-h-[88px] grid-cols-[52px_minmax(0,1fr)] items-center gap-3 rounded-[18px] border-2 p-3">
            <PlayerAvatar
              playerId={targetPlayer?.playerId}
              nickname={targetPlayer?.nickname}
              avatarUrl={targetPlayer?.avatarUrl}
              className="border-ink size-[52px] rounded-[18px] border-2"
            />
            <div className="min-w-0">
              <p className="text-muted-foreground text-xs font-black">
                {resolveMutation.isPending
                  ? "確認 QR Code 中"
                  : targetPlayer?.team?.name
                    ? formatTeamName(targetPlayer.team.name)
                    : "尚未選擇學員"}
              </p>
              <strong className="mt-1 block text-[22px] leading-tight font-black break-words">
                {targetPlayer?.nickname ?? "等待選擇"}
              </strong>
              {targetPlayer ? (
                <p className="text-muted-foreground mt-1 text-xs font-bold break-all">
                  {targetPlayer.playerId}
                </p>
              ) : null}
            </div>
          </div>

          <form onSubmit={handleManualAssignment}>
            <Button
              type="submit"
              className="h-12 w-full text-base"
              disabled={
                !assignmentRoomNumber ||
                !targetPlayer ||
                roomAssignmentMutation.isPending
              }
            >
              {roomAssignmentMutation.isPending
                ? "分配中"
                : `分配至 ${assignmentRoomLabel}`}
            </Button>
          </form>
        </CardContent>
      </Card>

      <PlayerQrScannerDialog
        open={scannerOpen}
        onOpenChange={setScannerOpen}
        onToken={resolveToken}
      />
    </section>
  )
}
