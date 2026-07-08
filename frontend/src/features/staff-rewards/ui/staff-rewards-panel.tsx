import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import {
  CheckCircle2Icon,
  MinusIcon,
  PlusIcon,
  ScanLineIcon,
  SearchIcon,
  SendIcon,
  UsersIcon,
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
  SelectGroup,
  SelectItem,
  SelectLabel,
  SelectSeparator,
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
  functionLabel?: string
  detailLabel?: string
  toneClass: string
  sortTags: StoneSortTag[]
  sortRank: number
}

type TargetPlayer = {
  playerId: string
  nickname: string
  team?: PlayerStatus["team"]
  avatarUrl?: string
}

type TargetMode = "player" | "team"

type StoneSortTag = "base" | "checkpoint" | "level1" | "level2"

type ItemEvolutionStage = "level1" | "level2"

type ItemFunctionMeta = {
  functionLabel: string
  detailLabel?: string
  sortRank: number
}

const ALL_PLAYERS_TEAM_ID = "__all_players__"

const BASE_STONE_IDS = new Set([
  "stone_engineering_base",
  "stone_entertainment_base",
  "stone_explorer_base",
  "stone_inspiration_base",
  "stone_resonance_base",
])

const CHECKPOINT_STONE_IDS = new Set([
  "stone_command_blind_trip",
  "stone_human_llm",
  "stone_von_neumann",
  "stone_packet_rescue",
  "stone_prompt_injection",
])

const LEVEL1_STONE_IDS = new Set([
  "stone_2022_maze",
  "stone_camp_backpack",
  "stone_booth",
  "stone_2019_blackbox",
  "stone_2021_abacus",
  "stone_prompt",
  "stone_sticky_note",
  "stone_2024_finite",
  "stone_community_handshake",
  "stone_consensus_draft",
  "stone_contribution",
  "stone_2020_tour_flag",
  "stone_docker",
  "stone_gpg",
  "stone_test",
  "stone_terminal",
  "stone_microphone",
  "stone_p5js",
  "stone_espresso",
  "stone_2024_ribbon",
])

const LEVEL2_STONE_IDS = new Set([
  "stone_2022_break_wall_cat",
  "stone_2026_camp_explorer",
  "stone_star_village_guide",
  "stone_2019_unboxed_algorithm",
  "stone_2021_abacus_descendant",
  "stone_human_after_all",
  "stone_course_shared_notes",
  "stone_2024_infinite_inspiration",
  "stone_star_village_exchange",
  "stone_student_autonomy",
  "stone_open_source_route",
  "stone_2020_sitcon_tour_group",
  "stone_kubernetes",
  "stone_crypto_gatekeeper",
  "stone_clean_code",
  "stone_system_intern",
  "stone_lightning_talk",
  "stone_tech_art",
  "stone_2024_last_night_polaroid",
])

const ITEM_EVOLUTION_META: Record<
  string,
  { stage: ItemEvolutionStage; type: Sitone["type"] }
> = {
  item_adventure_backpack: { stage: "level1", type: "exploration" },
  item_black_box_sticker: { stage: "level1", type: "exploration" },
  item_booth_sticker: { stage: "level1", type: "exploration" },
  item_canvas_code: { stage: "level1", type: "entertainment" },
  item_charter_draft: { stage: "level1", type: "resonance" },
  item_container_sticker: { stage: "level1", type: "engineering" },
  item_contribution_sticker: { stage: "level1", type: "resonance" },
  item_espresso_cup: { stage: "level1", type: "entertainment" },
  item_finite_label: { stage: "level1", type: "inspiration" },
  item_maze_map: { stage: "level1", type: "exploration" },
  item_microphone: { stage: "level1", type: "entertainment" },
  item_prompt_card: { stage: "level1", type: "inspiration" },
  item_public_key_tag: { stage: "level1", type: "engineering" },
  item_ribbon: { stage: "level1", type: "entertainment" },
  item_sticky_note: { stage: "level1", type: "inspiration" },
  item_student_community_card: { stage: "level1", type: "resonance" },
  item_terminal_cursor: { stage: "level1", type: "engineering" },
  item_test_sticker: { stage: "level1", type: "engineering" },
  item_tour_flag: { stage: "level1", type: "resonance" },
  item_wooden_abacus: { stage: "level1", type: "inspiration" },
  item_cat_paw_print: { stage: "level2", type: "exploration" },
  item_clean_spec: { stage: "level2", type: "engineering" },
  item_cluster_core: { stage: "level2", type: "engineering" },
  item_essence_timer: { stage: "level2", type: "entertainment" },
  item_human_label: { stage: "level2", type: "inspiration" },
  item_infinite_star_map: { stage: "level2", type: "inspiration" },
  item_lightning_talk_script: { stage: "level2", type: "entertainment" },
  item_mission_map: { stage: "level2", type: "exploration" },
  item_open_source_roadmap: { stage: "level2", type: "resonance" },
  item_pixel_paint: { stage: "level2", type: "entertainment" },
  item_polaroid_film: { stage: "level2", type: "entertainment" },
  item_predecessor_notes: { stage: "level2", type: "inspiration" },
  item_shared_notes_link: { stage: "level2", type: "inspiration" },
  item_signature_inkpad: { stage: "level2", type: "engineering" },
  item_star_village_badge: { stage: "level2", type: "resonance" },
  item_star_village_signpost: { stage: "level2", type: "exploration" },
  item_system_docs: { stage: "level2", type: "engineering" },
  item_toolbox_key: { stage: "level2", type: "exploration" },
  item_transparent_proposal: { stage: "level2", type: "resonance" },
  item_venue_route: { stage: "level2", type: "resonance" },
}

const ITEM_FUNCTION_META: Record<string, ItemFunctionMeta> = {
  item_charm_connection: {
    functionLabel: "功能道具",
    detailLabel: "探索型小石掉落率 +15%",
    sortRank: 2,
  },
  item_charm_debug: {
    functionLabel: "功能道具",
    detailLabel: "工程型答對分數 +10%",
    sortRank: 2,
  },
  item_charm_all_nighter: {
    functionLabel: "功能道具",
    detailLabel: "靈光型刪錯選項機率 +20%",
    sortRank: 2,
  },
  item_charm_success: {
    functionLabel: "功能道具",
    detailLabel: "娛樂型勝利開源力 +20%",
    sortRank: 2,
  },
  item_charm_harmony: {
    functionLabel: "功能道具",
    detailLabel: "共鳴型勝利開源力 +20%",
    sortRank: 2,
  },
  item_postcard_sitcon2024: { functionLabel: "無功能道具", sortRank: 3 },
  item_postcard_sitcon2026: { functionLabel: "無功能道具", sortRank: 3 },
  item_postcard_star_village: { functionLabel: "無功能道具", sortRank: 3 },
  item_tshirt_2026: { functionLabel: "無功能道具", sortRank: 3 },
  item_wooden_plank: {
    functionLabel: "合成素材",
    detailLabel: "可合成星手村路標",
    sortRank: 1,
  },
}

function sitoneSortTags(sitoneID: string): StoneSortTag[] {
  if (BASE_STONE_IDS.has(sitoneID)) return ["base"]
  if (CHECKPOINT_STONE_IDS.has(sitoneID)) return ["checkpoint"]
  if (LEVEL1_STONE_IDS.has(sitoneID)) return ["level1"]
  if (LEVEL2_STONE_IDS.has(sitoneID)) return ["level2"]
  return []
}

function stoneSortTagLabel(tag: StoneSortTag) {
  switch (tag) {
    case "base":
      return "基礎小石"
    case "checkpoint":
      return "闖關活動小石"
    case "level1":
      return "Level 1 小石"
    case "level2":
      return "Level 2 小石"
  }
}

function stoneSortRank(tags: StoneSortTag[]) {
  const tag = tags[0]
  if (tag === "base") return 0
  if (tag === "checkpoint") return 1
  if (tag === "level1") return 2
  if (tag === "level2") return 3
  return 4
}

function itemEvolutionStageLabel(stage: ItemEvolutionStage) {
  return stage === "level1" ? "一階" : "二階"
}

function itemFunctionMeta(item: Item): ItemFunctionMeta {
  const evolutionMeta = ITEM_EVOLUTION_META[item.id]
  if (evolutionMeta) {
    return {
      functionLabel: `${itemEvolutionStageLabel(evolutionMeta.stage)}${sitoneMeta(evolutionMeta.type).label}型進化道具`,
      sortRank: evolutionMeta.stage === "level1" ? 0 : 1,
    }
  }
  return (
    ITEM_FUNCTION_META[item.id] ?? {
      functionLabel: itemTypeLabel(item.type),
      sortRank: 4,
    }
  )
}

function primaryStoneSortTag(option: RewardOption) {
  return option.sortTags[0]
}

function sitoneOption(sitone: Sitone): RewardOption {
  const meta = sitoneMeta(sitone.type)
  const sortTags = sitoneSortTags(sitone.id)
  return {
    id: sitone.id,
    name: sitone.name,
    description: sitone.description,
    typeLabel: meta.label,
    rarityLabel: rarityLabel(sitone.rarity),
    toneClass: meta.bgClassName,
    sortTags,
    sortRank: stoneSortRank(sortTags),
  }
}

function itemOption(item: Item): RewardOption {
  const functionMeta = itemFunctionMeta(item)
  return {
    id: item.id,
    name: item.name,
    description: item.description,
    typeLabel: itemTypeLabel(item.type),
    rarityLabel: rarityLabel(item.rarity),
    functionLabel: functionMeta.functionLabel,
    detailLabel: functionMeta.detailLabel,
    toneClass: itemTypeClass(item.type),
    sortTags: [],
    sortRank: functionMeta.sortRank,
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

function clampOpenPowerAmount(value: number) {
  if (!Number.isFinite(value)) return 10
  return Math.max(1, Math.min(99999, Math.floor(value)))
}

export function StaffRewardsPanel() {
  const queryClient = useQueryClient()
  const [scannerOpen, setScannerOpen] = useState(false)
  const [targetMode, setTargetMode] = useState<TargetMode>("player")
  const [manualToken, setManualToken] = useState("")
  const [playerSearch, setPlayerSearch] = useState("")
  const [targetPlayer, setTargetPlayer] = useState<TargetPlayer | null>(null)
  const [teamSearch, setTeamSearch] = useState("")
  const [selectedTeamID, setSelectedTeamID] = useState("")
  const [rewardKind, setRewardKind] = useState<StaffRewardKind>("sitone")
  const [selectedRefIDs, setSelectedRefIDs] = useState<
    Record<StaffRewardKind, string>
  >({ item: "", sitone: "", open_power: "" })
  const [quantity, setQuantity] = useState(1)
  const [openPowerAmount, setOpenPowerAmount] = useState(100)
  const [search, setSearch] = useState("")

  const statusQuery = useQuery({
    queryKey: ["me", "status"],
    queryFn: gameApi.status,
  })
  const isStaff = statusQuery.data?.role === "staff"
  const playerSearchKeyword = playerSearch.trim()
  const teamSearchKeyword = teamSearch.trim()
  const playersQuery = useQuery({
    queryKey: ["staff", "players", playerSearchKeyword],
    queryFn: () => gameApi.staffPlayers(playerSearchKeyword),
    enabled: isStaff && playerSearchKeyword.length > 0,
  })
  const teamsQuery = useQuery({
    queryKey: ["staff", "teams", teamSearchKeyword],
    queryFn: () => gameApi.staffTeams(teamSearchKeyword || undefined),
    enabled: isStaff,
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
    () =>
      (sitonesQuery.data ?? [])
        .map(sitoneOption)
        .sort(
          (left, right) =>
            left.sortRank - right.sortRank ||
            left.name.localeCompare(right.name),
        ),
    [sitonesQuery.data],
  )
  const itemOptions = useMemo(
    () =>
      (itemsQuery.data ?? [])
        .map(itemOption)
        .sort(
          (left, right) =>
            left.sortRank - right.sortRank ||
            left.name.localeCompare(right.name),
        ),
    [itemsQuery.data],
  )
  const rewardOptions = useMemo(
    () =>
      rewardKind === "sitone"
        ? sitoneOptions
        : rewardKind === "item"
          ? itemOptions
          : [],
    [itemOptions, rewardKind, sitoneOptions],
  )
  const playerOptions = playersQuery.data ?? []
  const teamOptions = teamsQuery.data ?? []
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
            option.typeLabel.toLowerCase().includes(keyword) ||
            option.functionLabel?.toLowerCase().includes(keyword) ||
            option.detailLabel?.toLowerCase().includes(keyword) ||
            option.sortTags.some((tag) =>
              stoneSortTagLabel(tag).toLowerCase().includes(keyword),
            ),
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
  const selectedTeam =
    teamOptions.find((team) => team.teamId === selectedTeamID) ?? null
  const allPlayersSelected = selectedTeamID === ALL_PLAYERS_TEAM_ID
  const groupedVisibleSitoneOptions = useMemo(
    () =>
      [
        { tag: "base" as const, label: "基礎小石" },
        { tag: "checkpoint" as const, label: "闖關活動" },
        { tag: "level1" as const, label: "Level 1" },
        { tag: "level2" as const, label: "Level 2" },
      ]
        .map((group) => ({
          ...group,
          options: visibleOptions.filter(
            (option) => primaryStoneSortTag(option) === group.tag,
          ),
        }))
        .filter((group) => group.options.length > 0),
    [visibleOptions],
  )

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
      if (result.allPlayers) {
        toast.success(
          result.reward.kind === "open_power"
            ? `已發送 ${result.reward.amount} 開源力給所有人 (${result.grantedCount} 人)`
            : `已發送 ${result.reward.name} x${result.reward.quantity} 給所有人 (${result.grantedCount} 人)`,
        )
      } else if (result.team) {
        toast.success(
          result.reward.kind === "open_power"
            ? `已發送 ${result.reward.amount} 開源力給 ${result.team.name} 全組 (${result.grantedCount} 人)`
            : `已發送 ${result.reward.name} x${result.reward.quantity} 給 ${result.team.name} 全組 (${result.grantedCount} 人)`,
        )
      } else if (result.player) {
        toast.success(
          result.reward.kind === "open_power"
            ? `已發送 ${result.reward.amount} 開源力給 ${result.player.nickname}`
            : `已發送 ${result.reward.name} x${result.reward.quantity} 給 ${result.player.nickname}`,
        )
      }
      queryClient.invalidateQueries({ queryKey: ["me"] })
    },
    onError: (error) => {
      toast.error(errorMessage(error, "發送失敗"))
    },
  })

  const catalogsPending = sitonesQuery.isPending || itemsQuery.isPending
  const canSend =
    isStaff &&
    (targetMode === "player"
      ? !!targetPlayer?.playerId
      : allPlayersSelected || !!selectedTeam?.teamId) &&
    (rewardKind === "open_power" ? openPowerAmount >= 1 : !!selectedOption) &&
    (rewardKind === "open_power" || quantity >= 1) &&
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
    setTargetMode("player")
    setTargetPlayer(player)
  }

  function handleManualSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    resolveToken(manualToken)
  }

  function handleRewardSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (targetMode === "player") {
      if (!targetPlayer) return
      if (rewardKind === "open_power") {
        rewardMutation.mutate({
          playerId: targetPlayer.playerId,
          kind: rewardKind,
          amount: openPowerAmount,
        })
        return
      }
      if (!selectedOption) return
      rewardMutation.mutate({
        playerId: targetPlayer.playerId,
        kind: rewardKind,
        refId: selectedOption.id,
        quantity,
      })
      return
    }
    if (allPlayersSelected) {
      if (rewardKind === "open_power") {
        rewardMutation.mutate({
          allPlayers: true,
          kind: rewardKind,
          amount: openPowerAmount,
        })
        return
      }
      if (!selectedOption) return
      rewardMutation.mutate({
        allPlayers: true,
        kind: rewardKind,
        refId: selectedOption.id,
        quantity,
      })
      return
    }
    if (!selectedTeam) return
    if (rewardKind === "open_power") {
      rewardMutation.mutate({
        teamId: selectedTeam.teamId,
        kind: rewardKind,
        amount: openPowerAmount,
      })
      return
    }
    if (!selectedOption) return
    rewardMutation.mutate({
      teamId: selectedTeam.teamId,
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
                {targetMode === "player" ? (
                  <ScanLineIcon className="size-5" aria-hidden />
                ) : (
                  <UsersIcon className="size-5" aria-hidden />
                )}
                選擇發放對象
              </CardTitle>
            </div>
            <Tabs
              value={targetMode}
              onValueChange={(value) => setTargetMode(value as TargetMode)}
            >
              <TabsList className="w-full">
                <TabsTrigger value="player" className="w-full">
                  單人
                </TabsTrigger>
                <TabsTrigger value="team" className="w-full">
                  整組
                </TabsTrigger>
              </TabsList>
            </Tabs>
          </CardHeader>
          <CardContent className="grid gap-3 px-5">
            {targetMode === "player" ? (
              <>
                <div className="flex justify-end">
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
                      placeholder="搜尋 nickname、ID 或第幾小隊"
                      autoComplete="off"
                      aria-label="搜尋學員 nickname、ID 或第幾小隊"
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
                                      <span className="break-words">
                                        {player.team?.name ?? "未分組"}
                                      </span>{" "}
                                      · {player.playerId}
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
                        : targetPlayer
                          ? (targetPlayer.team?.name ?? "未分組")
                          : "尚未選擇學員"}
                    </p>
                    <strong className="mt-1 block text-[22px] leading-tight font-black break-words">
                      {targetPlayer?.nickname ?? "等待選擇"}
                    </strong>
                    {targetPlayer ? (
                      <p className="text-muted-foreground mt-1 text-xs leading-snug font-bold break-all">
                        {targetPlayer.playerId}
                      </p>
                    ) : null}
                  </div>
                </div>
              </>
            ) : (
              <>
                <div className="relative">
                  <SearchIcon
                    className="text-muted-foreground pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2"
                    aria-hidden
                  />
                  <Input
                    value={teamSearch}
                    onChange={(event) => setTeamSearch(event.target.value)}
                    placeholder="搜尋第幾小隊、組別名稱或 ID"
                    autoComplete="off"
                    aria-label="搜尋第幾小隊、組別名稱或 ID"
                    className="pl-9"
                  />
                </div>

                <Select
                  value={selectedTeamID}
                  onValueChange={setSelectedTeamID}
                  disabled={teamsQuery.isPending}
                >
                  <SelectTrigger className="h-12 w-full">
                    <SelectValue
                      placeholder={
                        teamsQuery.isPending ? "同步組別中" : "選擇組別"
                      }
                    />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value={ALL_PLAYERS_TEAM_ID}>所有人</SelectItem>
                    {teamOptions.length > 0 ? <SelectSeparator /> : null}
                    {teamOptions.map((team) => (
                      <SelectItem key={team.teamId} value={team.teamId}>
                        {team.name} ({team.memberCount} 人)
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>

                {teamsQuery.isError ? (
                  <p className="text-destructive px-1 text-sm font-bold">
                    {errorMessage(teamsQuery.error, "讀取組別失敗")}
                  </p>
                ) : null}

                <div className="bg-surface-raised border-border grid min-h-[88px] grid-cols-[52px_minmax(0,1fr)] items-center gap-3 rounded-[18px] border-2 p-3">
                  <div className="bg-card border-ink flex size-[52px] items-center justify-center rounded-[18px] border-2">
                    <UsersIcon className="size-6" aria-hidden />
                  </div>
                  <div className="min-w-0">
                    <p className="text-muted-foreground text-xs font-black">
                      {allPlayersSelected
                        ? "所有玩家"
                        : selectedTeam
                          ? `${selectedTeam.memberCount} 人`
                          : "尚未選擇組別"}
                    </p>
                    <strong className="mt-1 block text-[22px] leading-tight font-black break-words">
                      {allPlayersSelected
                        ? "所有人"
                        : (selectedTeam?.name ?? "等待選擇")}
                    </strong>
                  </div>
                </div>
              </>
            )}
          </CardContent>
        </Card>

        <form className="grid gap-3" onSubmit={handleRewardSubmit}>
          <Card className="border-ink rounded-[22px] border-2">
            <CardHeader className="gap-3 px-5">
              <CardTitle className="flex items-center gap-2 text-xl font-black">
                <GameFeatureIcon name="shop" className="size-5" />
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
                  <TabsTrigger value="open_power" className="w-full">
                    開源力
                  </TabsTrigger>
                </TabsList>
              </Tabs>
            </CardHeader>
            <CardContent className="grid gap-3 px-5">
              {rewardKind === "open_power" ? (
                <>
                  <div className="bg-surface-raised border-border grid min-h-[112px] gap-3 rounded-[18px] border-2 p-4">
                    <div className="min-w-0">
                      <strong className="block text-[18px] leading-tight font-black">
                        開源力
                      </strong>
                      <p className="text-muted-foreground mt-1 text-sm leading-[1.55]">
                        直接發放指定數量的開源力給學員。
                      </p>
                    </div>
                    <Input
                      value={openPowerAmount}
                      onChange={(event) =>
                        setOpenPowerAmount(
                          clampOpenPowerAmount(Number(event.target.value)),
                        )
                      }
                      type="number"
                      min={1}
                      max={99999}
                      inputMode="numeric"
                      aria-label="開源力數量"
                      className="h-12 text-center text-lg font-black"
                    />
                    <div className="grid grid-cols-4 gap-2">
                      {[10, 50, 100, 500].map((value) => (
                        <Button
                          key={value}
                          type="button"
                          variant="outline"
                          onClick={() => setOpenPowerAmount(value)}
                        >
                          {value}
                        </Button>
                      ))}
                    </div>
                  </div>
                </>
              ) : (
                <>
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
                        placeholder={
                          catalogsPending ? "同步清單中" : "選擇內容"
                        }
                      />
                    </SelectTrigger>
                    <SelectContent>
                      {rewardKind === "sitone" ? (
                        <>
                          {groupedVisibleSitoneOptions.map((group, index) => (
                            <SelectGroup key={group.tag}>
                              {index > 0 ? <SelectSeparator /> : null}
                              <SelectLabel className="px-2 py-2 text-[11px] font-black normal-case opacity-70">
                                {group.label}
                              </SelectLabel>
                              {group.options.map((option) => (
                                <SelectItem key={option.id} value={option.id}>
                                  <div className="flex min-w-0 flex-wrap items-center gap-1.5 pr-4">
                                    <span className="font-black">
                                      {option.name}
                                    </span>
                                  </div>
                                </SelectItem>
                              ))}
                            </SelectGroup>
                          ))}
                        </>
                      ) : (
                        <>
                          {visibleOptions.map((option) => (
                            <SelectItem key={option.id} value={option.id}>
                              <div className="flex min-w-0 flex-wrap items-center gap-1.5 pr-4">
                                <span className="font-black">
                                  {option.name}
                                </span>
                                {option.functionLabel ? (
                                  <span className="text-muted-foreground text-xs font-bold">
                                    {option.functionLabel}
                                  </span>
                                ) : null}
                                {option.detailLabel ? (
                                  <span className="text-muted-foreground text-xs font-bold">
                                    {option.detailLabel}
                                  </span>
                                ) : null}
                              </div>
                            </SelectItem>
                          ))}
                        </>
                      )}
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
                        {[
                          selectedOption?.typeLabel,
                          selectedOption?.functionLabel,
                          selectedOption?.detailLabel,
                          selectedOption?.rarityLabel,
                        ]
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
                      <strong className="block text-[18px] leading-tight font-black break-words">
                        {selectedOption?.name ?? "尚未選擇"}
                      </strong>
                      <p className="text-muted-foreground mt-1 line-clamp-2 text-sm leading-[1.55]">
                        {selectedOption?.description ??
                          "清單同步完成後即可選擇。"}
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
                </>
              )}
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
                {rewardMutation.data.reward.kind === "open_power"
                  ? `${rewardMutation.data.reward.name} +${rewardMutation.data.reward.amount}`
                  : `${rewardMutation.data.reward.name} x${rewardMutation.data.reward.quantity}`}
              </strong>
              <p className="text-muted-foreground text-sm font-bold">
                {rewardMutation.data.team ? (
                  <>
                    {rewardMutation.data.team.name} ·{" "}
                    {rewardMutation.data.grantedCount} 人
                  </>
                ) : rewardMutation.data.player ? (
                  <>
                    <span className="inline-flex items-center gap-2 align-middle">
                      <PlayerAvatar
                        playerId={rewardMutation.data.player.playerId}
                        nickname={rewardMutation.data.player.nickname}
                        avatarUrl={rewardMutation.data.player.avatarUrl}
                        className="border-ink size-6 rounded-[9px] border"
                      />
                      {rewardMutation.data.player.nickname}
                    </span>{" "}
                    {rewardMutation.data.player.team?.name
                      ? `· ${rewardMutation.data.player.team.name}`
                      : ""}
                  </>
                ) : null}
              </p>
              {rewardMutation.data.reward.kind === "open_power" ? (
                <strong className="text-power text-lg font-black">
                  +{rewardMutation.data.reward.amount} OP
                </strong>
              ) : null}
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
