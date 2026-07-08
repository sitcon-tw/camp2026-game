import { useQuery } from "@tanstack/react-query"
import { useEffect, useMemo, useState } from "react"

import {
  gameApi,
  type LeaderboardEntry,
  type LeaderboardPlayerInventoryResponse,
  type LeaderboardScope,
  type LeaderboardTeamPlayer,
  type LeaderboardTeamPlayersResponse,
  type PlayerItem,
  type PlayerSitone,
} from "@/shared/api/game"
import {
  itemTypeClass,
  itemTypeLabel,
  rarityLabel,
  sitoneMeta,
} from "@/shared/lib/game-labels"
import { Button } from "@/shared/ui/button"
import { Card } from "@/shared/ui/card"
import { GameFeatureIcon } from "@/shared/ui/game-feature-icon"
import { GamePageShell } from "@/shared/ui/game-page-shell"
import { GameIcon } from "@/shared/ui/game-icon"
import { PageHeader } from "@/shared/ui/page-header"
import { PlayerAvatar } from "@/shared/ui/player-avatar"
import { imageSrcCandidates } from "@/shared/utils/image-src"

const TABS: { key: LeaderboardScope; label: string }[] = [
  { key: "teams", label: "團隊" },
  { key: "players", label: "隊員" },
]

const rowColors = [
  "bg-pebble-spark",
  "bg-pebble-engineer",
  "bg-pebble-resonate",
  "bg-pebble-explore",
  "bg-pebble-play",
]

type Selection =
  | { kind: "team"; teamID: string }
  | { kind: "player"; playerID: string; returnTeamID?: string }
  | null
type InventoryTab = "sitones" | "items"

function scopeLabel(scope: LeaderboardScope) {
  return scope === "teams" ? "你的隊伍" : "你的排名"
}

function emptyCurrentLabel(scope: LeaderboardScope) {
  return scope === "teams" ? "目前沒有隊伍排名" : "目前沒有隊員排名"
}

function currentDescription(
  rank: number,
  gapToPrevious: number,
  scope: LeaderboardScope,
) {
  if (gapToPrevious > 0) {
    return `距離前一名還差 ${gapToPrevious} 顆小石。`
  }
  if (rank === 1) {
    return scope === "teams"
      ? "你的隊伍目前已經在團隊榜領先。"
      : "你目前已經在隊員榜領先。"
  }
  return "小石數量已追平前一名，開源力會作為下一個排序。"
}

function statLabel(value: number, label: string) {
  return `${value} ${label}`
}

export function LeaderBoardPage() {
  const [activeScope, setActiveScope] = useState<LeaderboardScope>("teams")
  const [selection, setSelection] = useState<Selection>(null)
  const { data, isPending } = useQuery({
    queryKey: ["leaderboards", activeScope],
    queryFn: () => gameApi.leaderboard(activeScope),
  })
  const ranks = data?.entries ?? []
  const currentEntry = data?.currentEntry

  if (selection?.kind === "team") {
    return (
      <GamePageShell contentClassName="grid content-start gap-y-2">
        <PageHeader
          title="排行榜"
          headline="Leaderboard"
          onBack={() => setSelection(null)}
        />
        <TeamPlayersPanel
          teamID={selection.teamID}
          onSelectPlayer={(playerID) =>
            setSelection({
              kind: "player",
              playerID,
              returnTeamID: selection.teamID,
            })
          }
        />
      </GamePageShell>
    )
  }

  if (selection?.kind === "player") {
    const handleBack = () =>
      selection.returnTeamID
        ? setSelection({ kind: "team", teamID: selection.returnTeamID })
        : setSelection(null)

    return (
      <GamePageShell contentClassName="grid content-start gap-y-2">
        <PageHeader title="排行榜" headline="Leaderboard" onBack={handleBack} />
        <PlayerInventoryPanel playerID={selection.playerID} />
      </GamePageShell>
    )
  }

  return (
    <GamePageShell contentClassName="grid content-start gap-y-2">
      <PageHeader title="排行榜" headline="Leaderboard" />

      <Card className="bg-ink text-primary-foreground grid grid-cols-[1fr_78px] items-center gap-3 rounded-[26px] px-5 py-4">
        <div>
          <p className="text-primary-foreground/70 mb-1 text-xs font-extrabold tracking-[0.08em] uppercase">
            {scopeLabel(activeScope)}
          </p>
          <h2 className="mb-1.5 text-2xl leading-tight font-extrabold">
            {currentEntry
              ? `${currentEntry.name} 目前第 ${currentEntry.rank} 名`
              : isPending
                ? "正在同步排名"
                : emptyCurrentLabel(activeScope)}
          </h2>
          <p className="text-primary-foreground/70 text-sm leading-relaxed">
            {currentEntry && data
              ? currentDescription(
                  currentEntry.rank,
                  data.gapToPrevious,
                  activeScope,
                )
              : "完成活動與知識王戰後，排行榜會自動更新。"}
          </p>
        </div>
        <strong className="bg-pebble-spark text-ink border-primary-foreground grid h-[78px] place-items-center rounded-[22px] border-2 text-3xl font-extrabold">
          {currentEntry ? `#${currentEntry.rank}` : "-"}
        </strong>
      </Card>

      <div
        className="grid grid-cols-2 gap-2"
        role="tablist"
        aria-label="排行榜分類"
      >
        {TABS.map(({ key, label }) => (
          <Button
            key={key}
            role="tab"
            aria-selected={activeScope === key}
            variant={activeScope === key ? "default" : "outline"}
            className="min-h-11 rounded-2xl font-extrabold"
            onClick={() => {
              setActiveScope(key)
              setSelection(null)
            }}
          >
            {label}
          </Button>
        ))}
      </div>

      <div className="grid gap-2" role="tabpanel">
        {isPending ? (
          <Card className="rounded-[18px] px-3 py-4">
            <span className="text-muted-foreground text-sm font-extrabold">
              正在同步排行榜
            </span>
          </Card>
        ) : ranks.length > 0 ? (
          ranks.map((entry, index) => (
            <LeaderboardRow
              key={`${activeScope}-${entry.id}`}
              activeScope={activeScope}
              entry={entry}
              index={index}
              onSelect={() => {
                if (activeScope === "teams") {
                  setSelection({
                    kind: "team",
                    teamID: entry.teamId ?? entry.id,
                  })
                  return
                }
                setSelection({ kind: "player", playerID: entry.id })
              }}
            />
          ))
        ) : (
          <Card className="rounded-[18px] px-3 py-4">
            <span className="text-muted-foreground text-sm font-extrabold">
              目前沒有排行榜資料
            </span>
          </Card>
        )}
      </div>
    </GamePageShell>
  )
}

function LeaderboardRow({
  activeScope,
  entry,
  index,
  onSelect,
}: {
  activeScope: LeaderboardScope
  entry: LeaderboardEntry
  index: number
  onSelect: () => void
}) {
  return (
    <button
      type="button"
      className={[
        "bg-card border-ink text-card-foreground grid cursor-pointer grid-cols-[32px_42px_1fr_auto] items-center gap-3 rounded-[18px] border-2 px-3 py-3 text-left shadow-[4px_4px_0_rgba(23,35,58,0.12)] transition-transform active:translate-x-px active:translate-y-px",
        entry.current ? "bg-surface-raised" : "",
      ].join(" ")}
      onClick={onSelect}
    >
      <span className="text-sm font-extrabold">#{entry.rank}</span>
      {activeScope === "players" ? (
        <PlayerAvatar
          playerId={entry.id}
          nickname={entry.name}
          avatarUrl={entry.avatarUrl}
          className="border-ink size-[42px] rounded-[14px] border-2"
        />
      ) : (
        <TeamAvatar
          avatarUrl={entry.avatarUrl}
          fallbackClassName={rowColors[index % rowColors.length]}
        />
      )}
      <div className="min-w-0">
        <p className="truncate font-bold">{entry.name}</p>
        <p className="text-muted-foreground truncate text-xs font-semibold">
          {entry.current
            ? activeScope === "teams"
              ? "你的隊伍"
              : "你"
            : activeScope === "teams"
              ? "團隊總計"
              : (entry.teamName ?? "隊員")}
        </p>
      </div>
      <div className="text-right">
        <strong className="block text-sm font-extrabold whitespace-nowrap">
          {statLabel(entry.sitoneCount, "小石")}
        </strong>
        <span className="text-muted-foreground block text-xs font-bold whitespace-nowrap">
          {entry.openPower} OP
        </span>
      </div>
    </button>
  )
}

function TeamAvatar({
  avatarUrl,
  fallbackClassName,
}: {
  avatarUrl?: string
  fallbackClassName: string
}) {
  const avatarSrcs = useMemo(() => imageSrcCandidates(avatarUrl), [avatarUrl])
  const [avatarSrcIndex, setAvatarSrcIndex] = useState(0)
  const currentAvatarSrc = avatarSrcs[avatarSrcIndex]

  useEffect(() => {
    setAvatarSrcIndex(0)
  }, [avatarUrl])

  return (
    <div
      className={[
        "border-ink grid size-[42px] rotate-[-6deg] place-items-center overflow-hidden rounded-[14px_18px_12px_16px] border-2",
        currentAvatarSrc ? "bg-card p-1" : fallbackClassName,
      ].join(" ")}
      aria-hidden
    >
      {currentAvatarSrc ? (
        <img
          src={currentAvatarSrc}
          alt=""
          draggable={false}
          className="size-full object-contain"
          onError={() => setAvatarSrcIndex((index) => index + 1)}
        />
      ) : null}
    </div>
  )
}

function TeamPanelIcon({
  avatarUrl,
  fallback,
}: {
  avatarUrl?: string
  fallback: React.ReactNode
}) {
  const avatarSrcs = useMemo(() => imageSrcCandidates(avatarUrl), [avatarUrl])
  const [avatarSrcIndex, setAvatarSrcIndex] = useState(0)
  const currentAvatarSrc = avatarSrcs[avatarSrcIndex]

  useEffect(() => {
    setAvatarSrcIndex(0)
  }, [avatarUrl])

  if (!currentAvatarSrc) return fallback

  return (
    <img
      src={currentAvatarSrc}
      alt=""
      draggable={false}
      className="size-full object-contain p-0.5"
      onError={() => setAvatarSrcIndex((index) => index + 1)}
    />
  )
}

function TeamPlayersPanel({
  teamID,
  onSelectPlayer,
}: {
  teamID: string
  onSelectPlayer: (playerID: string) => void
}) {
  const { data, isPending, isError } = useQuery({
    queryKey: ["leaderboards", "team", teamID, "players"],
    queryFn: () => gameApi.leaderboardTeamPlayers(teamID),
  })

  return (
    <>
      <PanelTitle
        icon={
          <TeamPanelIcon
            avatarUrl={data?.team.avatarUrl}
            fallback={<GameFeatureIcon name="team" className="size-4" />}
          />
        }
        title={data?.team.name ?? "隊伍成員"}
        subtitle={
          isPending
            ? "正在同步隊伍"
            : data
              ? `${data.players.length} 位隊員`
              : "隊伍明細"
        }
      />
      {isPending ? (
        <PanelMessage text="正在同步隊員列表" />
      ) : isError ? (
        <PanelMessage text="隊伍資料讀取失敗" />
      ) : data ? (
        <TeamPlayerList data={data} onSelectPlayer={onSelectPlayer} />
      ) : null}
    </>
  )
}

function TeamPlayerList({
  data,
  onSelectPlayer,
}: {
  data: LeaderboardTeamPlayersResponse
  onSelectPlayer: (playerID: string) => void
}) {
  if (data.players.length === 0) {
    return <PanelMessage text="這個隊伍目前沒有隊員" />
  }

  return (
    <div className="grid gap-2">
      {data.players.map((player) => (
        <TeamPlayerRow
          key={player.playerId}
          player={player}
          onSelect={() => onSelectPlayer(player.playerId)}
        />
      ))}
    </div>
  )
}

function TeamPlayerRow({
  player,
  onSelect,
}: {
  player: LeaderboardTeamPlayer
  onSelect: () => void
}) {
  return (
    <button
      type="button"
      className={[
        "border-border bg-surface-raised grid cursor-pointer grid-cols-[1fr_auto] items-center gap-3 rounded-[16px] border-2 px-3 py-2.5 text-left",
        player.current ? "border-ink" : "",
      ].join(" ")}
      onClick={onSelect}
    >
      <div className="flex min-w-0 items-center gap-2.5">
        <PlayerAvatar
          playerId={player.playerId}
          nickname={player.nickname}
          avatarUrl={player.avatarUrl}
          className="border-ink size-9 rounded-[13px] border-2"
        />
        <div className="min-w-0">
          <p className="truncate font-black">{player.nickname}</p>
          <p className="text-muted-foreground truncate text-xs font-bold">
            {player.current ? "你" : "隊員"} ·{" "}
            {statLabel(player.itemCount, "道具")}
          </p>
        </div>
      </div>
      <div className="text-right">
        <strong className="block text-sm font-black whitespace-nowrap">
          {statLabel(player.sitoneCount, "小石")}
        </strong>
        <span className="text-muted-foreground block text-xs font-bold whitespace-nowrap">
          {player.openPower} OP
        </span>
      </div>
    </button>
  )
}

function PlayerInventoryPanel({ playerID }: { playerID: string }) {
  const { data, isPending, isError } = useQuery({
    queryKey: ["leaderboards", "player", playerID, "inventory"],
    queryFn: () => gameApi.leaderboardPlayerInventory(playerID),
  })

  return (
    <>
      <PanelTitle
        icon={
          data ? (
            <PlayerAvatar
              playerId={data.player.playerId}
              nickname={data.player.nickname}
              avatarUrl={data.player.avatarUrl}
              className="size-7 rounded-[10px]"
            />
          ) : (
            <GameFeatureIcon name="backpack" className="size-4" />
          )
        }
        title={data?.player.nickname ?? "隊員背包"}
        subtitle={
          isPending
            ? "正在同步背包"
            : data
              ? `${data.team.name} · ${data.player.openPower} OP`
              : "背包明細"
        }
      />
      {isPending ? (
        <PanelMessage text="正在同步隊員背包" />
      ) : isError ? (
        <PanelMessage text="隊員背包讀取失敗" />
      ) : data ? (
        <PlayerInventory data={data} />
      ) : null}
    </>
  )
}

function PlayerInventory({
  data,
}: {
  data: LeaderboardPlayerInventoryResponse
}) {
  const [activeTab, setActiveTab] = useState<InventoryTab>("sitones")
  const tabs: { key: InventoryTab; label: string; count: number }[] = [
    { key: "sitones", label: "小石", count: data.sitones.length },
    { key: "items", label: "道具", count: data.items.length },
  ]

  return (
    <div className="grid gap-3">
      <div
        className="grid grid-cols-2 gap-2"
        role="tablist"
        aria-label="背包分類"
      >
        {tabs.map((tab) => (
          <Button
            key={tab.key}
            type="button"
            role="tab"
            aria-selected={activeTab === tab.key}
            variant={activeTab === tab.key ? "default" : "outline"}
            className="min-h-11 rounded-2xl text-sm font-extrabold"
            onClick={() => setActiveTab(tab.key)}
          >
            {tab.label}
            <span className="text-xs opacity-70">{tab.count}</span>
          </Button>
        ))}
      </div>

      <p className="text-muted-foreground px-1 text-xs font-bold">
        {data.player.sitoneCount} 顆小石 · {data.player.itemCount} 個道具 ·{" "}
        {data.player.openPower} OP
      </p>

      {activeTab === "sitones" ? (
        <InventorySection title="小石" emptyText="目前沒有小石">
          {data.sitones.map((sitone) => (
            <SitoneInventoryRow key={sitone.id} sitone={sitone} />
          ))}
        </InventorySection>
      ) : (
        <InventorySection title="道具" emptyText="目前沒有道具">
          {data.items.map((item) => (
            <ItemInventoryRow key={item.id} item={item} />
          ))}
        </InventorySection>
      )}
    </div>
  )
}

function PanelTitle({
  icon,
  title,
  subtitle,
}: {
  icon: React.ReactNode
  title: string
  subtitle: string
}) {
  return (
    <div className="flex items-center gap-3 px-1 py-1">
      <span
        className="bg-surface-raised border-border grid size-7 shrink-0 place-items-center rounded-[10px] border-2"
        aria-hidden
      >
        {icon}
      </span>
      <div className="min-w-0">
        <h3 className="truncate text-xl font-black">{title}</h3>
        <p className="text-muted-foreground mt-1 truncate text-xs font-bold">
          {subtitle}
        </p>
      </div>
    </div>
  )
}

function PanelMessage({ text }: { text: string }) {
  return (
    <div className="border-border bg-surface-raised rounded-[16px] border-2 px-3 py-4">
      <span className="text-muted-foreground text-sm font-bold">{text}</span>
    </div>
  )
}

function InventorySection({
  title,
  emptyText,
  children,
}: {
  title: string
  emptyText: string
  children: React.ReactNode
}) {
  const items = Array.isArray(children) ? children.filter(Boolean) : children
  const empty = Array.isArray(items) ? items.length === 0 : !items

  return (
    <section className="grid gap-3" aria-label={title}>
      {empty ? <PanelMessage text={emptyText} /> : items}
    </section>
  )
}

function ItemInventoryRow({ item }: { item: PlayerItem }) {
  return (
    <Card className="border-ink grid grid-cols-[72px_1fr] items-start gap-3 rounded-[22px] border-2 p-3.5">
      <InventoryItemIcon className={itemTypeClass(item.item.type)}>
        <GameIcon
          iconPath={item.item.iconPath}
          imageClassName="p-2"
          fallback={<span className="text-xs font-black">ITM</span>}
        />
        <strong className="bg-card border-ink absolute right-1.5 bottom-[5px] min-w-[28px] rounded-full border-2 px-1.5 py-px text-center text-[13px]">
          {item.quantity}
        </strong>
      </InventoryItemIcon>
      <div className="min-w-0">
        <div className="mb-1.5 flex items-start justify-between gap-2">
          <h3 className="text-[18px] leading-tight font-black tracking-normal">
            {item.item.name}
          </h3>
          <strong className="text-primary shrink-0 text-[18px] font-black">
            x{item.quantity}
          </strong>
        </div>
        <div className="mb-1.5 flex flex-wrap gap-1.5">
          {[itemTypeLabel(item.item.type), rarityLabel(item.item.rarity)].map(
            (tag) => (
              <span
                key={tag}
                className="bg-surface-raised border-border text-muted-foreground rounded-full border-[1.5px] px-2 py-0.5 text-xs font-black"
              >
                {tag}
              </span>
            ),
          )}
        </div>
        <InventoryDetails>
          <p>{item.item.description}</p>
        </InventoryDetails>
      </div>
    </Card>
  )
}

function SitoneInventoryRow({ sitone }: { sitone: PlayerSitone }) {
  const meta = sitoneMeta(sitone.sitone.type)

  return (
    <Card className="border-ink grid grid-cols-[72px_1fr] items-start gap-3 rounded-[22px] border-2 p-3.5">
      <InventoryItemIcon className={meta.bgClassName}>
        <GameIcon
          iconPath={sitone.sitone.iconPath}
          imageClassName="p-2"
          fallback={<span className="text-xs font-black">{meta.short}</span>}
        />
        <strong className="bg-card border-ink absolute right-1.5 bottom-[5px] min-w-[28px] rounded-full border-2 px-1.5 py-px text-center text-[13px]">
          {sitone.quantity}
        </strong>
      </InventoryItemIcon>
      <div className="min-w-0">
        <div className="mb-1.5 flex items-start justify-between gap-2">
          <h3 className="text-[18px] leading-tight font-black tracking-normal">
            {sitone.sitone.name}
          </h3>
          <strong className="text-primary shrink-0 text-[18px] font-black">
            x{sitone.quantity}
          </strong>
        </div>
        <div className="mb-1.5 flex flex-wrap gap-1.5">
          {[`${meta.label}小石`, rarityLabel(sitone.sitone.rarity)].map(
            (tag) => (
              <span
                key={tag}
                className="bg-surface-raised border-border text-muted-foreground rounded-full border-[1.5px] px-2 py-0.5 text-xs font-black"
              >
                {tag}
              </span>
            ),
          )}
        </div>
        <InventoryDetails>
          <p>{sitone.sitone.description}</p>
          <p className="mt-1">
            <strong className="text-ink">{sitone.sitone.abilityName}</strong>：
            {sitone.sitone.abilityDescription}
          </p>
        </InventoryDetails>
      </div>
    </Card>
  )
}

function InventoryDetails({ children }: { children: React.ReactNode }) {
  return (
    <details className="group mt-2">
      <summary className="text-primary inline-flex cursor-pointer list-none items-center text-xs font-black">
        詳細
        <span className="ml-1 transition-transform group-open:rotate-180">
          ▾
        </span>
      </summary>
      <div className="text-muted-foreground mt-1.5 text-sm leading-[1.62] font-semibold">
        {children}
      </div>
    </details>
  )
}

function InventoryItemIcon({
  className,
  children,
}: {
  className: string
  children: React.ReactNode
}) {
  return (
    <div
      className={[
        "border-ink relative grid h-[72px] place-items-center overflow-hidden rounded-[20px] border-2",
        className,
      ].join(" ")}
      aria-hidden
    >
      {children}
    </div>
  )
}
