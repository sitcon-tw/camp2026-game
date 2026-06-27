import { useQuery } from "@tanstack/react-query"
import { useMemo, useState } from "react"

import { gameApi, type PlayerSitone, type Sitone } from "@/shared/api/game"
import {
  rarityLabel,
  rarityToneClass,
  sitoneMeta,
} from "@/shared/lib/game-labels"
import { Button } from "@/shared/ui/button"
import { Card } from "@/shared/ui/card"
import { cn } from "@/shared/utils"

type StoneTypeKey = "all" | "explore" | "spark" | "echo" | "build" | "play"
type CollectionMode = "owned" | "all"

type StoneType = {
  key: StoneTypeKey
  label: string
  short: string
  bgClassName: string
}

type Stone = {
  id: string
  name: string
  type: Exclude<StoneTypeKey, "all">
  rarity: string
  count: number
  owned: boolean
  description: string
  iconPath?: string
  abilityName: string
  abilityDescription: string
}

const stoneTypes: StoneType[] = [
  { key: "all", label: "全部", short: "All", bgClassName: "bg-ink" },
  {
    key: "explore",
    label: "探索",
    short: "EXP",
    bgClassName: "bg-pebble-explore",
  },
  {
    key: "spark",
    label: "靈光",
    short: "SPK",
    bgClassName: "bg-pebble-spark",
  },
  {
    key: "echo",
    label: "共鳴",
    short: "ECO",
    bgClassName: "bg-pebble-resonate",
  },
  {
    key: "build",
    label: "工程",
    short: "BLD",
    bgClassName: "bg-pebble-engineer",
  },
  {
    key: "play",
    label: "娛樂",
    short: "PLY",
    bgClassName: "bg-pebble-play",
  },
]

function typeMeta(key: StoneTypeKey) {
  return stoneTypes.find((type) => type.key === key) ?? stoneTypes[0]
}

function buildStones(catalog: Sitone[], ownedSitones: PlayerSitone[]): Stone[] {
  const counts = new Map(
    ownedSitones.map((record) => [record.sitoneId, record.quantity] as const),
  )

  return catalog.map((sitone) => {
    const meta = sitoneMeta(sitone.type)
    const count = counts.get(sitone.id) ?? 0

    return {
      id: sitone.id,
      name: sitone.name,
      type: meta.key,
      rarity: rarityLabel(sitone.rarity),
      count,
      owned: count > 0,
      description: sitone.description,
      iconPath: sitone.iconPath,
      abilityName: sitone.abilityName,
      abilityDescription: sitone.abilityDescription,
    }
  })
}

export function StoneCollectionPanel() {
  const [activeType, setActiveType] = useState<StoneTypeKey>("all")
  const [mode, setMode] = useState<CollectionMode>("owned")
  const catalogQuery = useQuery({
    queryKey: ["catalog", "sitones"],
    queryFn: gameApi.catalogSitones,
  })
  const ownedQuery = useQuery({
    queryKey: ["me", "sitones"],
    queryFn: gameApi.playerSitones,
  })
  const stones = useMemo(
    () => buildStones(catalogQuery.data ?? [], ownedQuery.data ?? []),
    [catalogQuery.data, ownedQuery.data],
  )

  const visibleStones = useMemo(
    () =>
      stones.filter((stone) => {
        const typeMatches = activeType === "all" || stone.type === activeType
        const modeMatches = mode === "all" || stone.owned
        return typeMatches && modeMatches
      }),
    [activeType, mode, stones],
  )

  const ownedCount = stones.filter((stone) => stone.owned).length
  const totalPieces = stones.reduce((sum, stone) => sum + stone.count, 0)
  const rareOwned = stones.filter(
    (stone) => stone.owned && stone.rarity !== "常見",
  ).length

  return (
    <div>
      <Card
        className="bg-surface-raised before:border-ink/25 relative mt-[18px] grid grid-cols-[1fr_116px] gap-3 overflow-hidden rounded-[28px] p-[18px] py-[18px] before:pointer-events-none before:absolute before:inset-2.5 before:rounded-[24px] before:border before:border-dashed"
        aria-label="收藏摘要"
      >
        <div className="relative z-10">
          <span className="text-moss text-[11px] font-extrabold tracking-[0.08em] uppercase">
            目前收藏
          </span>
          <strong className="mt-1 block text-[42px] leading-none font-extrabold tracking-normal">
            {ownedCount}/{stones.length}
          </strong>
          <p className="text-muted-foreground mt-2 max-w-[190px] text-[13px] leading-5 font-semibold">
            你已收集 {totalPieces} 顆小石，其中 {rareOwned} 種是稀有以上。
          </p>
        </div>
        <div className="relative z-10 h-28 w-[116px]" aria-hidden>
          <StoneShape
            type="explore"
            owned
            count={3}
            className="absolute top-[12px] right-[42px] z-10 scale-[0.88] -rotate-[8deg]"
          />
          <StoneShape
            type="spark"
            owned
            count={2}
            className="absolute top-[32px] right-1 z-20 scale-[0.78] rotate-[13deg]"
          />
          <StoneShape
            type="echo"
            owned
            count={4}
            className="absolute top-[58px] right-[30px] z-30 scale-[0.72] -rotate-[3deg]"
          />
        </div>
      </Card>

      <section
        className="mt-3 grid grid-cols-5 gap-1.5"
        aria-label="收藏數值摘要"
      >
        {(["explore", "spark", "echo", "build", "play"] as const).map((key) => {
          const typedStones = stones.filter((stone) => stone.type === key)
          const owned = typedStones.filter((stone) => stone.owned).length
          return (
            <div
              key={key}
              className="border-border bg-card grid min-h-[58px] content-center gap-0.5 rounded-2xl border-2 px-1 py-2 text-center"
            >
              <span className="text-muted-foreground text-[11px] font-semibold">
                {typeMeta(key).label}
              </span>
              <strong className="text-[15px] font-extrabold">
                {owned}/{typedStones.length}
              </strong>
            </div>
          )
        })}
      </section>

      <section className="mt-4" aria-label="小石篩選">
        <div
          className="border-ink bg-card grid grid-cols-2 gap-2 rounded-[20px] border-2 p-1.5"
          role="group"
          aria-label="收藏顯示模式"
        >
          <SegmentButton
            active={mode === "owned"}
            onClick={() => setMode("owned")}
          >
            已擁有
          </SegmentButton>
          <SegmentButton active={mode === "all"} onClick={() => setMode("all")}>
            全圖鑑
          </SegmentButton>
        </div>

        <div
          className="mt-2.5 flex [scrollbar-width:none] gap-2 overflow-x-auto pb-1 [&::-webkit-scrollbar]:hidden"
          role="group"
          aria-label="小石分類"
        >
          {stoneTypes.map((type) => (
            <Button
              key={type.key}
              type="button"
              variant={activeType === type.key ? "secondary" : "outline"}
              className={[
                "min-h-11 shrink-0 rounded-2xl px-3 text-sm font-extrabold shadow-none",
              ].join(" ")}
              onClick={() => setActiveType(type.key)}
            >
              <span
                className={[
                  "border-ink grid h-6 min-w-[34px] place-items-center rounded-full border-2 text-[10px] tracking-normal",
                  type.bgClassName,
                  type.key === "all" ? "text-card" : "text-ink",
                ].join(" ")}
              >
                {type.short}
              </span>
              {type.label}
            </Button>
          ))}
        </div>
      </section>

      <section className="mt-[18px]" aria-label="小石卡片列表">
        <div className="mb-2.5 flex items-end justify-between gap-3">
          <div>
            <span className="text-moss text-[11px] font-extrabold tracking-[0.08em] uppercase">
              COLLECTION
            </span>
            <h2 className="mt-0.5 text-[21px] leading-tight font-extrabold tracking-normal">
              {activeType === "all"
                ? "所有分類"
                : `${typeMeta(activeType).label}小石`}
            </h2>
          </div>
          <p className="text-muted-foreground text-xs font-semibold whitespace-nowrap">
            {mode === "owned" ? "只看目前持有" : "包含未發現剪影"}
          </p>
        </div>

        {visibleStones.length > 0 ? (
          <div className="grid grid-cols-2 gap-2.5">
            {visibleStones.map((stone) => (
              <StoneCard key={stone.id} stone={stone} mode={mode} />
            ))}
          </div>
        ) : (
          <EmptyCollection
            activeType={activeType}
            onViewAll={() => setMode("all")}
          />
        )}
      </section>
    </div>
  )
}

function SegmentButton({
  active,
  onClick,
  children,
}: {
  active: boolean
  onClick: () => void
  children: React.ReactNode
}) {
  return (
    <Button
      type="button"
      variant={active ? "default" : "ghost"}
      className={[
        "min-h-11 rounded-2xl text-sm font-extrabold shadow-none",
        active ? "" : "border-transparent bg-transparent",
      ].join(" ")}
      onClick={onClick}
    >
      {children}
    </Button>
  )
}

function StoneCard({ stone, mode }: { stone: Stone; mode: CollectionMode }) {
  const meta = typeMeta(stone.type)
  return (
    <Card
      className={[
        "flex min-h-[248px] flex-col gap-2.5 rounded-[22px] p-2.5 py-2.5",
        stone.owned
          ? "border-ink bg-card"
          : "border-ink bg-muted border-dashed",
      ].join(" ")}
      aria-label={`${stone.name}，${meta.label}小石`}
    >
      <div className="flex items-center justify-between gap-1.5">
        <span className="border-ink bg-surface-raised inline-flex min-h-6 items-center gap-1 rounded-full border-2 px-1.5 text-[11px] font-extrabold">
          <span
            className={[
              "border-ink size-[9px] rounded-full border",
              meta.bgClassName,
            ].join(" ")}
          />
          {meta.label}
        </span>
        <span
          className={[
            "border-ink inline-flex min-h-6 items-center rounded-full border-2 px-1.5 text-[11px] font-extrabold",
            rarityToneClass(stone.rarity),
          ].join(" ")}
        >
          {stone.rarity}
        </span>
      </div>

      <div className="border-border bg-surface-raised grid min-h-20 place-items-center rounded-[18px] border-2">
        <StoneShape
          type={stone.type}
          owned={stone.owned}
          count={stone.count}
          iconPath={stone.iconPath}
        />
      </div>

      <div className="grid gap-1">
        <h3 className="text-[17px] leading-tight font-extrabold tracking-normal">
          {stone.owned ? stone.name : "未發現小石"}
        </h3>
        <p className="text-muted-foreground text-xs leading-5 font-semibold">
          {stone.description}
        </p>
      </div>

      <div className="border-border bg-surface-raised rounded-[14px] border px-2 py-1.5">
        <span className="block text-[12px] font-extrabold">
          {stone.abilityName}
        </span>
        <span className="text-muted-foreground block text-[11px] leading-4 font-semibold">
          {stone.abilityDescription}
        </span>
      </div>

      <div className="border-border mt-auto grid gap-0.5 border-t-2 border-dashed pt-2">
        <span className="text-[13px] font-extrabold">
          {stone.owned
            ? `持有 ${stone.count} 顆`
            : mode === "all"
              ? "尚未收集"
              : "未取得"}
        </span>
        {!stone.owned && (
          <span className="text-muted-foreground text-[11px] font-semibold">
            剪影狀態
          </span>
        )}
      </div>
    </Card>
  )
}

function StoneShape({
  type,
  owned,
  count,
  iconPath,
  className = "",
}: {
  type: Exclude<StoneTypeKey, "all">
  owned: boolean
  count: number
  iconPath?: string
  className?: string
}) {
  const meta = typeMeta(type)
  const badgeClassName = cn(
    "border-ink bg-card z-20 grid size-[26px] place-items-center rounded-full border-2 text-sm font-extrabold",
    iconPath ? "absolute right-[-7px] bottom-[-6px]" : "relative",
  )

  return (
    <div
      className={cn(
        "border-ink relative grid h-[58px] w-[62px] place-items-center rounded-[18px_24px_16px_26px] border-2",
        owned ? meta.bgClassName : "bg-muted",
        className,
      )}
      aria-hidden
    >
      {iconPath ? (
        <img
          src={iconPath}
          alt=""
          className={cn(
            "relative z-10 size-[54px] object-contain drop-shadow-[0_2px_0_rgba(23,35,58,0.18)]",
            !owned && "opacity-45 brightness-0 grayscale",
          )}
          loading="lazy"
          draggable={false}
        />
      ) : (
        <>
          <span className="border-ink/30 bg-card/45 absolute top-2 left-2 h-3.5 w-6 rotate-[-18deg] rounded-[12px_8px_10px_7px] border" />
          <span className="border-ink/30 bg-card/45 absolute right-2 bottom-2 h-3 w-[18px] rotate-[14deg] rounded-[7px_10px_8px_12px] border" />
        </>
      )}
      {owned ? (
        <strong className={badgeClassName}>{count}</strong>
      ) : (
        <span className={badgeClassName}>?</span>
      )}
    </div>
  )
}

function EmptyCollection({
  activeType,
  onViewAll,
}: {
  activeType: StoneTypeKey
  onViewAll: () => void
}) {
  const meta = typeMeta(activeType)
  return (
    <section className="border-ink bg-card grid justify-items-center gap-2.5 rounded-[22px] border-2 border-dashed px-[18px] py-6 text-center">
      <div
        className={[
          "border-ink grid h-[76px] w-[82px] place-items-center rounded-[24px_18px_26px_20px] border-2",
          meta.bgClassName,
        ].join(" ")}
      >
        <span className="border-ink bg-card size-[30px] rounded-full border-2 border-dashed" />
      </div>
      <h3 className="text-lg font-extrabold">這個分類還沒有收藏</h3>
      <p className="text-muted-foreground max-w-[260px] text-[13px] leading-6 font-semibold">
        完成營隊活動或知識王戰後，取得的小石會出現在這裡。
      </p>
      <Button
        type="button"
        variant="secondary"
        className="mt-1 min-h-11 rounded-2xl px-3.5 text-sm font-extrabold"
        onClick={onViewAll}
      >
        查看全部圖鑑
      </Button>
    </section>
  )
}
