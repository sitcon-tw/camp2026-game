import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { Link, useNavigate } from "@tanstack/react-router"
import { Check, DoorOpen, Plus, X } from "lucide-react"
import { useCallback, useEffect, useState } from "react"
import { toast } from "sonner"

import { MatchCodeQr } from "@/features/battle-qr"
import { BattleWaitingPlayerCard } from "@/features/battle-waiting/ui/battle-waiting-player-card"
import {
  useMatchDeadlineRefresh,
  useMatchEvents,
} from "@/features/game/use-match-events"
import { AppError } from "@/shared/api/error"
import { gameApi, type PlayerSitone } from "@/shared/api/game"
import { Button } from "@/shared/ui/button"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/shared/ui/card"
import { GamePageShell } from "@/shared/ui/game-page-shell"
import { PageHeader } from "@/shared/ui/page-header"
import { SitoneIcon } from "@/shared/ui/sitone-icon"
import { cn } from "@/shared/utils"

function getStoredMatchID() {
  if (typeof window === "undefined") return ""
  return window.localStorage.getItem("camp2026.currentMatchId") ?? ""
}

function clearStoredMatchID() {
  if (typeof window !== "undefined") {
    window.localStorage.removeItem("camp2026.currentMatchId")
  }
}

const maxLoadoutSize = 5

function sameIDs(a: string[], b: string[]) {
  return a.length === b.length && a.every((id, index) => id === b[index])
}

function isPlayerSitone(
  record: PlayerSitone | undefined,
): record is PlayerSitone {
  return Boolean(record)
}

function getOwnedQuantityByID(ownedSitones: PlayerSitone[]) {
  const quantityByID = new Map<string, number>()
  for (const record of ownedSitones) {
    quantityByID.set(
      record.sitoneId,
      (quantityByID.get(record.sitoneId) ?? 0) + record.quantity,
    )
  }
  return quantityByID
}

function countSelectedSitone(sitoneIDs: string[], sitoneID: string) {
  return sitoneIDs.reduce(
    (count, currentID) => count + (currentID === sitoneID ? 1 : 0),
    0,
  )
}

function validSelection(source: string[], quantityByID: Map<string, number>) {
  const selected: string[] = []
  const used = new Map<string, number>()

  for (const sitoneID of source) {
    const ownedQuantity = quantityByID.get(sitoneID) ?? 0
    const usedQuantity = used.get(sitoneID) ?? 0
    if (ownedQuantity <= 0 || usedQuantity >= ownedQuantity) continue

    selected.push(sitoneID)
    used.set(sitoneID, usedQuantity + 1)
    if (selected.length >= maxLoadoutSize) break
  }

  return selected
}

function defaultSelection(input: {
  ownedSitones: PlayerSitone[]
  matchSitoneIDs: string[]
  defaultSitoneIDs: string[]
}) {
  const quantityByID = getOwnedQuantityByID(input.ownedSitones)
  const sources = [input.matchSitoneIDs, input.defaultSitoneIDs]
  for (const source of sources) {
    const validIDs = validSelection(source, quantityByID)
    if (validIDs.length > 0) return validIDs
  }
  return []
}

export function BattleWaitingRoomPage() {
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const [matchID] = useState(getStoredMatchID)
  const [manualSitoneIDs, setManualSitoneIDs] = useState<string[] | null>(null)
  const handleMatchDeleted = useCallback(() => {
    clearStoredMatchID()
    navigate({ to: "/battle", replace: true })
  }, [navigate])
  const matchQuery = useQuery({
    queryKey: ["matches", matchID],
    queryFn: () => gameApi.getMatch(matchID),
    enabled: matchID.length > 0,
  })
  useMatchEvents(matchID, {
    enabled: matchID.length > 0,
    onDeleted: handleMatchDeleted,
  })
  const statusQuery = useQuery({
    queryKey: ["me", "status"],
    queryFn: gameApi.status,
  })
  const ownedSitonesQuery = useQuery({
    queryKey: ["me", "sitones"],
    queryFn: gameApi.playerSitones,
  })
  const defaultLoadoutQuery = useQuery({
    queryKey: ["me", "sitone-loadout"],
    queryFn: gameApi.sitoneLoadout,
  })
  const saveLoadoutMutation = useMutation({
    mutationFn: (sitoneIDs: string[]) =>
      gameApi.updateMatchLoadout(matchID, sitoneIDs),
    onSuccess: (match, sitoneIDs) => {
      queryClient.setQueryData(["matches", matchID], match)
      queryClient.setQueryData(["me", "sitone-loadout"], {
        sitoneIds: sitoneIDs,
      })
      toast.success("已套用本場小石")
    },
    onError: (error) => {
      toast.error(error instanceof Error ? error.message : "小石套用失敗")
    },
  })
  const leaveMutation = useMutation({
    mutationFn: () => gameApi.leaveMatch(matchID),
    onSuccess: () => {
      clearStoredMatchID()
      queryClient.invalidateQueries({ queryKey: ["matches", "open"] })
      navigate({ to: "/battle", replace: true })
    },
    onError: (error) => {
      toast.error(error instanceof Error ? error.message : "離開房間失敗")
    },
  })
  const readyMutation = useMutation({
    mutationFn: async () => {
      if (selectedSitoneIDs.length === 0) {
        throw new Error("請至少選擇一顆小石")
      }
      const currentLoadout = currentPlayer?.sitoneIds ?? []
      if (!sameIDs(selectedSitoneIDs, currentLoadout)) {
        await gameApi.updateMatchLoadout(matchID, selectedSitoneIDs)
      }
      return gameApi.readyMatch(matchID)
    },
    onSuccess: (match) => {
      queryClient.setQueryData(["matches", matchID], match)
      queryClient.setQueryData(["me", "sitone-loadout"], {
        sitoneIds: selectedSitoneIDs,
      })
    },
    onError: (error) => {
      toast.error(error instanceof Error ? error.message : "準備失敗")
    },
  })
  const match = matchQuery.data
  useMatchDeadlineRefresh(matchID, match)
  const ownedSitones = ownedSitonesQuery.data ?? []
  const ownedQuantityByID = getOwnedQuantityByID(ownedSitones)
  const currentPlayer = match?.players.find(
    (player) => player.playerId === statusQuery.data?.playerId,
  )
  const selectedSitoneIDs =
    manualSitoneIDs ??
    defaultSelection({
      ownedSitones,
      matchSitoneIDs: currentPlayer?.sitoneIds ?? [],
      defaultSitoneIDs: defaultLoadoutQuery.data?.sitoneIds ?? [],
    })
  const ownedSitoneByID = new Map(
    ownedSitones.map((record) => [record.sitoneId, record]),
  )
  const selectedSitones = selectedSitoneIDs
    .map((sitoneID) => ownedSitoneByID.get(sitoneID))
    .filter(isPlayerSitone)
  const loadoutSlots = Array.from(
    { length: maxLoadoutSize },
    (_, index) => selectedSitoneIDs[index] ?? "",
  )
  const loadoutLocked = currentPlayer?.ready === true
  const loadoutDirty = !sameIDs(
    selectedSitoneIDs,
    currentPlayer?.sitoneIds ?? [],
  )
  const hasSelectedSitones = selectedSitoneIDs.length > 0
  const canSaveLoadout =
    !loadoutLocked &&
    hasSelectedSitones &&
    !saveLoadoutMutation.isPending &&
    loadoutDirty
  const loadoutPending =
    ownedSitonesQuery.isPending ||
    defaultLoadoutQuery.isPending ||
    statusQuery.isPending

  function addSitone(record: PlayerSitone) {
    if (loadoutLocked) return
    setManualSitoneIDs((current) => {
      const selected = current ?? selectedSitoneIDs
      if (selected.length >= maxLoadoutSize) {
        toast.error(`最多選擇 ${maxLoadoutSize} 顆小石`)
        return selected
      }
      const ownedQuantity =
        ownedQuantityByID.get(record.sitoneId) ?? record.quantity
      const selectedQuantity = countSelectedSitone(selected, record.sitoneId)
      if (selectedQuantity >= ownedQuantity) {
        toast.error(`${record.sitone.name} 已經放滿`)
        return selected
      }
      return [...selected, record.sitoneId]
    })
  }

  function removeSitoneAt(slotIndex: number) {
    if (loadoutLocked) return
    setManualSitoneIDs((current) => {
      const selected = [...(current ?? selectedSitoneIDs)]
      selected.splice(slotIndex, 1)
      return selected
    })
  }

  function handleSaveLoadout() {
    if (selectedSitoneIDs.length === 0) {
      toast.error("請至少選擇一顆小石")
      return
    }
    saveLoadoutMutation.mutate(selectedSitoneIDs)
  }

  useEffect(() => {
    if (
      matchQuery.error instanceof AppError &&
      matchQuery.error.status === 404
    ) {
      clearStoredMatchID()
      navigate({ to: "/battle", replace: true })
      return
    }
    if (match?.status === "active") {
      navigate({ to: "/battle/question" })
    }
    if (match?.status === "completed") {
      navigate({ to: "/battle/result" })
    }
  }, [match?.status, matchQuery.error, navigate])

  if (!matchID) {
    return (
      <GamePageShell contentClassName="grid content-start gap-y-2">
        <PageHeader title="等待房間" headline="Battle Room" />
        <Card>
          <CardContent className="grid gap-3">
            <h2 className="text-2xl font-bold">找不到目前房間</h2>
            <Button asChild>
              <Link to="/battle">回到知識王大廳</Link>
            </Button>
          </CardContent>
        </Card>
      </GamePageShell>
    )
  }

  const computerMatch = match?.mode === "computer"

  return (
    <GamePageShell contentClassName="grid content-start gap-y-2">
      <PageHeader title="等待房間" headline="Battle Room" />
      {computerMatch ? null : (
        <Card>
          <CardHeader>
            <CardTitle>房號</CardTitle>
            <CardDescription>
              將這個房號分享給其他學員，加入後兩位玩家都準備即可開始。
            </CardDescription>
          </CardHeader>
          <CardContent className="grid justify-items-center gap-4">
            <span className="block pl-[0.5rem] text-4xl font-bold tracking-[0.5rem]">
              {match?.code ?? "------"}
            </span>
            <MatchCodeQr
              value={match?.code ?? ""}
              className="size-[168px] rounded-[24px] p-2"
            />
          </CardContent>
        </Card>
      )}

      <Card>
        <CardHeader>
          <CardTitle>本場小石</CardTitle>
          <CardDescription>
            每一題都會套用整組小石；這組選擇會保存為下次預設。
          </CardDescription>
        </CardHeader>
        <CardContent className="grid gap-3">
          {loadoutPending ? (
            <span className="text-muted-foreground font-bold">
              正在同步小石
            </span>
          ) : ownedSitones.length === 0 ? (
            <span className="text-muted-foreground font-bold">
              目前沒有可用小石，無法準備。
            </span>
          ) : (
            <>
              <div className="grid grid-cols-5 gap-2" aria-label="本場小石欄位">
                {loadoutSlots.map((sitoneID, index) => {
                  const record = ownedSitoneByID.get(sitoneID)
                  return (
                    <button
                      key={`${index}-${sitoneID || "empty"}`}
                      type="button"
                      className={cn(
                        "border-ink relative grid min-h-[72px] place-items-center rounded-[18px] border-2 px-1.5 py-2 text-center shadow-[3px_3px_0_rgba(23,35,58,0.12)] transition-all active:translate-x-px active:translate-y-px disabled:cursor-not-allowed",
                        record
                          ? "bg-card"
                          : "bg-surface-raised text-muted-foreground border-dashed",
                        !loadoutLocked && record
                          ? "hover:-translate-y-0.5"
                          : "",
                      )}
                      disabled={loadoutLocked || !record}
                      onClick={() => removeSitoneAt(index)}
                      aria-label={
                        record
                          ? `移除第 ${index + 1} 格的 ${record.sitone.name}`
                          : `第 ${index + 1} 格空位`
                      }
                    >
                      {record ? (
                        <span className="grid justify-items-center gap-1">
                          {!loadoutLocked ? (
                            <span
                              className="bg-surface-raised border-ink absolute top-1.5 right-1.5 grid size-5 place-items-center rounded-full border-2"
                              aria-hidden
                            >
                              <X className="size-3" />
                            </span>
                          ) : null}
                          <SitoneIcon
                            type={record.sitone.type}
                            iconPath={record.sitone.iconPath}
                          />
                          <span className="text-[11px] leading-none font-black">
                            第 {index + 1} 格
                          </span>
                        </span>
                      ) : (
                        <span className="grid justify-items-center gap-1 text-[11px] leading-none font-black">
                          <Plus className="size-4" />
                          {index + 1}
                        </span>
                      )}
                    </button>
                  )
                })}
              </div>

              <div className="grid grid-cols-2 gap-2">
                {ownedSitones.map((record) => {
                  const selectedCount = countSelectedSitone(
                    selectedSitoneIDs,
                    record.sitoneId,
                  )
                  const ownedQuantity =
                    ownedQuantityByID.get(record.sitoneId) ?? record.quantity
                  const selected = selectedCount > 0
                  const canAddSitone =
                    !loadoutLocked &&
                    selectedSitoneIDs.length < maxLoadoutSize &&
                    selectedCount < ownedQuantity
                  return (
                    <Button
                      key={record.id}
                      type="button"
                      variant={selected ? "default" : "outline"}
                      className="h-auto min-w-0 items-start justify-start rounded-2xl px-3 py-2 text-left whitespace-normal"
                      disabled={!canAddSitone}
                      onClick={() => addSitone(record)}
                      aria-pressed={selected}
                    >
                      <SitoneIcon
                        type={record.sitone.type}
                        iconPath={record.sitone.iconPath}
                        className="size-8"
                      />
                      <span className="min-w-0 flex-1 text-left">
                        <strong className="block truncate">
                          {record.sitone.name}
                        </strong>
                        <span className="block text-xs leading-none opacity-80">
                          已放 {selectedCount}/{ownedQuantity}
                        </span>
                        <span className="mt-1 block text-xs leading-snug break-words opacity-80">
                          {record.sitone.abilityName}：
                          {record.sitone.abilityDescription}
                        </span>
                      </span>
                    </Button>
                  )
                })}
              </div>
              <div className="flex items-center justify-between gap-2">
                <span className="text-muted-foreground text-sm font-bold">
                  已選 {selectedSitoneIDs.length}/{maxLoadoutSize} 顆
                  {loadoutLocked ? "，已鎖定" : ""}
                </span>
                <Button
                  type="button"
                  size="sm"
                  variant={canSaveLoadout ? "default" : "outline"}
                  disabled={!canSaveLoadout}
                  onClick={handleSaveLoadout}
                >
                  <Check />
                  {selectedSitoneIDs.length === 0
                    ? "先選小石"
                    : saveLoadoutMutation.isPending
                      ? "套用中"
                      : "套用小石"}
                </Button>
              </div>
              {selectedSitones.length > 0 ? (
                <div className="text-muted-foreground grid gap-1 text-xs font-bold">
                  <span>
                    本場使用：
                    {selectedSitones
                      .map((record) => record.sitone.name)
                      .join("、")}
                  </span>
                  <span>
                    被動：
                    {selectedSitones
                      .map((record) => record.sitone.abilityName)
                      .join("、")}
                  </span>
                </div>
              ) : null}
            </>
          )}
        </CardContent>
      </Card>

      {matchQuery.isPending ? (
        <Card>
          <CardContent>
            <span className="text-muted-foreground font-bold">
              正在同步房間狀態
            </span>
          </CardContent>
        </Card>
      ) : (
        (match?.players ?? []).map((player) => (
          <BattleWaitingPlayerCard
            key={player.playerId}
            playerId={player.playerId}
            name={player.nickname}
            kind={player.kind}
            team={
              player.kind === "computer"
                ? "電腦對手"
                : player.playerId === match?.hostPlayerId
                  ? "房主"
                  : "挑戰者"
            }
            ready={player.ready}
            loadoutCount={player.sitoneIds.length}
          />
        ))
      )}

      <Card>
        <CardContent className="grid grid-cols-2 gap-2">
          <Button
            variant="outline"
            size="lg"
            disabled={leaveMutation.isPending || readyMutation.isPending}
            onClick={() => leaveMutation.mutate()}
          >
            <DoorOpen />
            {leaveMutation.isPending ? "離開中" : "離開房間"}
          </Button>
          <Button
            size="lg"
            disabled={
              readyMutation.isPending ||
              saveLoadoutMutation.isPending ||
              !match ||
              selectedSitoneIDs.length === 0 ||
              loadoutLocked
            }
            onClick={() => readyMutation.mutate()}
          >
            <Check />
            {readyMutation.isPending ? "同步中" : "準備完成"}
          </Button>
        </CardContent>
      </Card>
    </GamePageShell>
  )
}
