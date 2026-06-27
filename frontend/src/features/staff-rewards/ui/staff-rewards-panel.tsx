import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import {
  CheckCircle2Icon,
  MinusIcon,
  PlusIcon,
  ScanLineIcon,
  SearchIcon,
  SendIcon,
} from "lucide-react"
import { type FormEvent, useMemo, useState } from "react"
import { toast } from "sonner"

import { PlayerQrScannerDialog } from "./player-qr-scanner-dialog"
import { AppError } from "@/shared/api/error"
import {
  gameApi,
  type Item,
  type PlayerStatus,
  type Sitone,
  type StaffPlayer,
  type StaffRewardKind,
} from "@/shared/api/game"
import {
  itemTypeClass,
  itemTypeLabel,
  rarityLabel,
  sitoneMeta,
} from "@/shared/lib/game-labels"
import { Button } from "@/shared/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/shared/ui/card"
import { GameFeatureIcon } from "@/shared/ui/game-feature-icon"
import { Input } from "@/shared/ui/input"
import { PlayerAvatar } from "@/shared/ui/player-avatar"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/shared/ui/select"
import { Tabs, TabsList, TabsTrigger } from "@/shared/ui/tabs"
import { cn } from "@/shared/utils"

type RewardOption = {
  id: string
  name: string
  description: string
  typeLabel: string
  rarityLabel: string
  toneClass: string
}

type TargetPlayer = {
  playerId: string
  nickname: string
  team?: PlayerStatus["team"]
}

function sitoneOption(sitone: Sitone): RewardOption {
  const meta = sitoneMeta(sitone.type)
  return {
    id: sitone.id,
    name: sitone.name,
    description: sitone.description,
    typeLabel: meta.label,
    rarityLabel: rarityLabel(sitone.rarity),
    toneClass: meta.bgClassName,
  }
}

function itemOption(item: Item): RewardOption {
  return {
    id: item.id,
    name: item.name,
    description: item.description,
    typeLabel: itemTypeLabel(item.type),
    rarityLabel: rarityLabel(item.rarity),
    toneClass: itemTypeClass(item.type),
  }
}

function errorMessage(error: unknown, fallback: string) {
  if (error instanceof AppError) return error.message
  return fallback
}

function clampQuantity(value: number) {
  if (!Number.isFinite(value)) return 1
  return Math.max(1, Math.min(99, Math.floor(value)))
}

export function StaffRewardsPanel() {
  const queryClient = useQueryClient()
  const [scannerOpen, setScannerOpen] = useState(false)
  const [manualToken, setManualToken] = useState("")
  const [playerSearch, setPlayerSearch] = useState("")
  const [targetPlayer, setTargetPlayer] = useState<TargetPlayer | null>(null)
  const [rewardKind, setRewardKind] = useState<StaffRewardKind>("sitone")
  const [selectedRefIDs, setSelectedRefIDs] = useState<
    Record<StaffRewardKind, string>
  >({ item: "", sitone: "" })
  const [quantity, setQuantity] = useState(1)
  const [search, setSearch] = useState("")

  const statusQuery = useQuery({
    queryKey: ["me", "status"],
    queryFn: gameApi.status,
  })
  const isStaff = statusQuery.data?.role === "staff"
  const playerSearchKeyword = playerSearch.trim()
  const playersQuery = useQuery({
    queryKey: ["staff", "players", playerSearchKeyword],
    queryFn: () => gameApi.staffPlayers(playerSearchKeyword),
    enabled: isStaff && playerSearchKeyword.length > 0,
  })
  const sitonesQuery = useQuery({
    queryKey: ["catalog", "sitones"],
    queryFn: gameApi.catalogSitones,
  })
  const itemsQuery = useQuery({
    queryKey: ["catalog", "items"],
    queryFn: gameApi.catalogItems,
  })

  const sitoneOptions = useMemo(
    () => (sitonesQuery.data ?? []).map(sitoneOption),
    [sitonesQuery.data],
  )
  const itemOptions = useMemo(
    () => (itemsQuery.data ?? []).map(itemOption),
    [itemsQuery.data],
  )
  const rewardOptions = rewardKind === "sitone" ? sitoneOptions : itemOptions
  const playerOptions = playersQuery.data ?? []
  const selectedRefID = rewardOptions.some(
    (option) => option.id === selectedRefIDs[rewardKind],
  )
    ? selectedRefIDs[rewardKind]
    : (rewardOptions[0]?.id ?? "")
  const selectedOption = rewardOptions.find(
    (option) => option.id === selectedRefID,
  )
  const visibleOptions = useMemo(() => {
    const keyword = search.trim().toLowerCase()
    const filtered = keyword
      ? rewardOptions.filter(
          (option) =>
            option.name.toLowerCase().includes(keyword) ||
            option.id.toLowerCase().includes(keyword) ||
            option.typeLabel.toLowerCase().includes(keyword),
        )
      : rewardOptions
    if (
      selectedOption &&
      !filtered.some((option) => option.id === selectedOption.id)
    ) {
      return [selectedOption, ...filtered]
    }
    return filtered
  }, [rewardOptions, search, selectedOption])

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

  const rewardMutation = useMutation({
    mutationFn: gameApi.createStaffReward,
    onSuccess: (result) => {
      toast.success(
        `已發送 ${result.reward.name} x${result.reward.quantity} 給 ${result.player.nickname}`,
      )
      queryClient.invalidateQueries({ queryKey: ["me"] })
    },
    onError: (error) => {
      toast.error(errorMessage(error, "發送失敗"))
    },
  })

  const catalogsPending = sitonesQuery.isPending || itemsQuery.isPending
  const canSend =
    isStaff &&
    !!targetPlayer?.playerId &&
    !!selectedOption &&
    quantity >= 1 &&
    !rewardMutation.isPending

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

  function handleManualSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    resolveToken(manualToken)
  }

  function handleRewardSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (!targetPlayer || !selectedOption) return
    rewardMutation.mutate({
      playerId: targetPlayer.playerId,
      kind: rewardKind,
      refId: selectedOption.id,
      quantity,
    })
  }

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

  return (
    <>
      <section className="grid gap-3" aria-label="staff 發放流程">
        <Card className="border-ink rounded-[22px] border-2">
          <CardHeader className="gap-3 px-5">
            <div className="flex items-center justify-between gap-3">
              <CardTitle className="flex items-center gap-2 text-xl font-black">
                <ScanLineIcon className="size-5" aria-hidden />
                掃描學員
              </CardTitle>
              <Button
                type="button"
                variant="secondary"
                size="sm"
                onClick={() => setScannerOpen(true)}
              >
                <ScanLineIcon className="size-4" aria-hidden />
                掃描
              </Button>
            </div>
          </CardHeader>
          <CardContent className="grid gap-3 px-5">
            <form
              className="grid grid-cols-[1fr_auto] gap-2"
              onSubmit={handleManualSubmit}
            >
              <Input
                value={manualToken}
                onChange={(event) => setManualToken(event.target.value)}
                placeholder="QR 識別碼"
                autoComplete="off"
                inputMode="text"
                aria-label="QR 識別碼"
              />
              <Button type="submit" disabled={resolveMutation.isPending}>
                確認
              </Button>
            </form>

            <div className="grid gap-2">
              <div className="relative">
                <SearchIcon
                  className="text-muted-foreground pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2"
                  aria-hidden
                />
                <Input
                  value={playerSearch}
                  onChange={(event) => setPlayerSearch(event.target.value)}
                  placeholder="搜尋 nickname 或 ID"
                  autoComplete="off"
                  aria-label="搜尋學員 nickname 或 ID"
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
                  ) : playerOptions.length > 0 ? (
                    playerOptions.map((player) => {
                      const selected =
                        targetPlayer?.playerId === player.playerId
                      return (
                        <Button
                          key={player.playerId}
                          type="button"
                          variant={selected ? "secondary" : "outline"}
                          className="h-auto w-full justify-start rounded-[16px] px-3 py-2 text-left whitespace-normal shadow-none"
                          onClick={() => selectTargetPlayer(player)}
                        >
                          <span className="flex w-full min-w-0 items-center justify-between gap-2">
                            <span className="flex min-w-0 items-center gap-2.5">
                              <PlayerAvatar
                                playerId={player.playerId}
                                nickname={player.nickname}
                                className="border-ink size-9 rounded-[13px] border-2"
                              />
                              <span className="min-w-0">
                                <span className="block truncate text-sm leading-tight font-black">
                                  {player.nickname}
                                </span>
                                <span className="text-muted-foreground mt-1 block truncate text-xs leading-tight font-bold">
                                  {player.team?.name ?? "未分組"} ·{" "}
                                  {player.playerId}
                                </span>
                              </span>
                            </span>
                            {selected ? (
                              <CheckCircle2Icon
                                className="size-4"
                                aria-hidden
                              />
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
            </div>

            <div className="bg-surface-raised border-border grid min-h-[88px] grid-cols-[52px_1fr] items-center gap-3 rounded-[18px] border-2 p-3">
              <PlayerAvatar
                playerId={targetPlayer?.playerId}
                nickname={targetPlayer?.nickname}
                className="border-ink size-[52px] rounded-[18px] border-2"
              />
              <div>
                <p className="text-muted-foreground text-xs font-black">
                  {resolveMutation.isPending
                    ? "確認 QR Code 中"
                    : targetPlayer
                      ? (targetPlayer.team?.name ?? "未分組")
                      : "尚未選擇學員"}
                </p>
                <strong className="mt-1 block text-[22px] leading-tight font-black">
                  {targetPlayer?.nickname ?? "等待選擇"}
                </strong>
              </div>
            </div>
          </CardContent>
        </Card>

        <form className="grid gap-3" onSubmit={handleRewardSubmit}>
          <Card className="border-ink rounded-[22px] border-2">
            <CardHeader className="gap-3 px-5">
              <CardTitle className="flex items-center gap-2 text-xl font-black">
                <GameFeatureIcon name="pass" className="size-5" />
                選擇發放內容
              </CardTitle>
              <Tabs
                value={rewardKind}
                onValueChange={(value) => {
                  setRewardKind(value as StaffRewardKind)
                  setSearch("")
                }}
              >
                <TabsList className="w-full">
                  <TabsTrigger value="sitone" className="w-full">
                    小石
                  </TabsTrigger>
                  <TabsTrigger value="item" className="w-full">
                    道具
                  </TabsTrigger>
                </TabsList>
              </Tabs>
            </CardHeader>
            <CardContent className="grid gap-3 px-5">
              <Input
                value={search}
                onChange={(event) => setSearch(event.target.value)}
                placeholder="搜尋小石/道具名稱或 ID"
                autoComplete="off"
                aria-label="搜尋發放內容"
              />
              <Select
                value={selectedRefID}
                onValueChange={(value) =>
                  setSelectedRefIDs((current) => ({
                    ...current,
                    [rewardKind]: value,
                  }))
                }
                disabled={catalogsPending || rewardOptions.length === 0}
              >
                <SelectTrigger className="h-12 w-full">
                  <SelectValue
                    placeholder={catalogsPending ? "同步清單中" : "選擇內容"}
                  />
                </SelectTrigger>
                <SelectContent>
                  {visibleOptions.map((option) => (
                    <SelectItem key={option.id} value={option.id}>
                      {option.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>

              <div className="bg-surface-raised border-border grid min-h-[112px] grid-cols-[64px_1fr] gap-3 rounded-[18px] border-2 p-3">
                <div
                  className={cn(
                    "border-ink h-16 rounded-[20px_24px_16px_22px] border-2",
                    selectedOption?.toneClass ?? "bg-card",
                  )}
                  aria-hidden
                />
                <div>
                  <div className="mb-1 flex flex-wrap gap-1.5">
                    {[selectedOption?.typeLabel, selectedOption?.rarityLabel]
                      .filter(Boolean)
                      .map((tag) => (
                        <span
                          key={tag}
                          className="bg-card border-border text-muted-foreground rounded-full border px-2 py-0.5 text-xs font-black"
                        >
                          {tag}
                        </span>
                      ))}
                  </div>
                  <strong className="block text-[18px] leading-tight font-black">
                    {selectedOption?.name ?? "尚未選擇"}
                  </strong>
                  <p className="text-muted-foreground mt-1 line-clamp-2 text-sm leading-[1.55]">
                    {selectedOption?.description ?? "清單同步完成後即可選擇。"}
                  </p>
                </div>
              </div>

              <div className="grid grid-cols-[auto_1fr_auto] items-center gap-2">
                <Button
                  type="button"
                  variant="outline"
                  size="icon"
                  aria-label="減少數量"
                  onClick={() =>
                    setQuantity((value) => clampQuantity(value - 1))
                  }
                  disabled={quantity <= 1}
                >
                  <MinusIcon className="size-4" aria-hidden />
                </Button>
                <Input
                  value={quantity}
                  onChange={(event) =>
                    setQuantity(clampQuantity(Number(event.target.value)))
                  }
                  type="number"
                  min={1}
                  max={99}
                  inputMode="numeric"
                  aria-label="發放數量"
                  className="h-11 text-center text-lg font-black"
                />
                <Button
                  type="button"
                  variant="outline"
                  size="icon"
                  aria-label="增加數量"
                  onClick={() =>
                    setQuantity((value) => clampQuantity(value + 1))
                  }
                  disabled={quantity >= 99}
                >
                  <PlusIcon className="size-4" aria-hidden />
                </Button>
              </div>
            </CardContent>
          </Card>

          <Button
            type="submit"
            className="h-12 w-full text-base"
            disabled={!canSend}
          >
            <SendIcon className="size-4" aria-hidden />
            {rewardMutation.isPending ? "發送中" : "發送"}
          </Button>
        </form>

        {rewardMutation.data ? (
          <Card className="border-ink rounded-[22px] border-2">
            <CardContent className="grid gap-2 p-5">
              <p className="text-muted-foreground text-xs font-black">
                最後一次發放
              </p>
              <strong className="text-[20px] leading-tight font-black">
                {rewardMutation.data.reward.name} x
                {rewardMutation.data.reward.quantity}
              </strong>
              <p className="text-muted-foreground text-sm font-bold">
                <span className="inline-flex items-center gap-2 align-middle">
                  <PlayerAvatar
                    playerId={rewardMutation.data.player.playerId}
                    nickname={rewardMutation.data.player.nickname}
                    className="border-ink size-6 rounded-[9px] border"
                  />
                  {rewardMutation.data.player.nickname}
                </span>{" "}
                · {rewardMutation.data.player.team.name}
              </p>
            </CardContent>
          </Card>
        ) : null}
      </section>

      <PlayerQrScannerDialog
        open={scannerOpen}
        onOpenChange={setScannerOpen}
        onToken={resolveToken}
      />
    </>
  )
}
