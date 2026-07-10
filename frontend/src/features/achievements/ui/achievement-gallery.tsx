import { useQuery } from "@tanstack/react-query"
import { Lock, Trophy } from "lucide-react"
import { useMemo, useState } from "react"

import { achievementAssetFor } from "@/features/achievements/lib/achievement-assets"
import { gameApi, type Achievement } from "@/shared/api/game"
import { Button } from "@/shared/ui/button"
import { Card } from "@/shared/ui/card"
import { GameIcon } from "@/shared/ui/game-icon"
import { cn } from "@/shared/utils"

type DisplayMode = "unlocked" | "all"

const achievementTones = [
  "bg-pebble-explore",
  "bg-pebble-spark",
  "bg-pebble-resonate",
  "bg-pebble-engineer",
  "bg-pebble-play",
] as const

const achievementStages = [
  { label: "起步", tiers: [1, 2] },
  { label: "成長", tiers: [3, 4] },
  { label: "進階", tiers: [5, 6] },
  { label: "熟練", tiers: [7, 8] },
  { label: "圓滿", tiers: [9, 10, 0] },
] as const

const dateTimeFormatter = new Intl.DateTimeFormat("zh-TW", {
  year: "numeric",
  month: "2-digit",
  day: "2-digit",
  hour: "2-digit",
  minute: "2-digit",
  hour12: false,
})

function formatUnlockedAt(value?: string) {
  if (!value) return ""
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return ""
  return dateTimeFormatter.format(date)
}

function toneForAchievement(achievement: Achievement) {
  if (achievement.key === "codex_complete") return "bg-pebble-spark"
  const tier = achievement.tier ?? 1
  return achievementTones[(tier - 1) % achievementTones.length]
}

export function AchievementGallery() {
  const [mode, setMode] = useState<DisplayMode>("unlocked")
  const { data, isPending, error } = useQuery({
    queryKey: ["me", "achievements"],
    queryFn: gameApi.achievements,
  })
  const achievements = useMemo(
    () => data?.achievements ?? [],
    [data?.achievements],
  )
  const visibleAchievements = useMemo(
    () =>
      achievements.filter(
        (achievement) => mode === "all" || achievement.unlocked,
      ),
    [achievements, mode],
  )
  const unlockedCount = data?.unlockedCount ?? 0

  if (error) {
    return (
      <Card className="border-ink bg-card grid min-h-40 content-center gap-2 rounded-[22px] border-2 p-5 text-center">
        <Trophy className="text-muted-foreground mx-auto size-9" aria-hidden />
        <h2 className="text-lg font-extrabold">成就資料暫時無法讀取</h2>
        <p className="text-muted-foreground text-sm font-semibold">
          請稍後重新整理頁面。
        </p>
      </Card>
    )
  }

  return (
    <div>
      <Card
        className="bg-surface-raised before:border-ink/25 relative mt-[18px] grid grid-cols-[1fr_116px] gap-3 overflow-hidden rounded-[28px] p-[18px] py-[18px] before:pointer-events-none before:absolute before:inset-2.5 before:rounded-[24px] before:border before:border-dashed"
        aria-label="成就收藏摘要"
      >
        <div className="relative z-10">
          <span className="text-moss text-[11px] font-extrabold tracking-[0.08em] uppercase">
            目前成就
          </span>
          <strong className="mt-1 block text-[42px] leading-none font-extrabold tracking-normal">
            {isPending ? "-/-" : `${unlockedCount}/${achievements.length}`}
          </strong>
          <p className="text-muted-foreground mt-2 max-w-[190px] text-[13px] leading-5 font-semibold">
            曾收集 {data?.collectedSitoneCount ?? 0}/
            {data?.totalSitoneCount ?? 0} 種小石，逐步完成圖鑑里程碑。
          </p>
        </div>
        <div className="relative z-10 h-28 w-[116px]" aria-hidden>
          <AchievementShape
            tier={1}
            unlocked
            className="absolute top-[12px] right-[42px] z-10 scale-[0.88] -rotate-[8deg]"
          />
          <AchievementShape
            tier={5}
            unlocked
            className="absolute top-[32px] right-1 z-20 scale-[0.78] rotate-[13deg]"
          />
          <AchievementShape
            tier={10}
            unlocked
            className="absolute top-[58px] right-[30px] z-30 scale-[0.72] -rotate-[3deg]"
          />
        </div>
      </Card>

      <section
        className="mt-3 grid grid-cols-5 gap-1.5"
        aria-label="成就階段摘要"
      >
        {achievementStages.map((stage) => {
          const entries = achievements.filter((achievement) =>
            (stage.tiers as readonly number[]).includes(achievement.tier ?? 0),
          )
          const unlocked = entries.filter(
            (achievement) => achievement.unlocked,
          ).length
          return (
            <div
              key={stage.label}
              className="border-border bg-card grid min-h-[58px] content-center gap-0.5 rounded-2xl border-2 px-1 py-2 text-center"
            >
              <span className="text-muted-foreground text-[11px] font-semibold">
                {stage.label}
              </span>
              <strong className="text-[15px] font-extrabold">
                {unlocked}/{entries.length}
              </strong>
            </div>
          )
        })}
      </section>

      <section className="mt-4" aria-label="成就篩選">
        <div
          className="border-ink bg-card grid grid-cols-2 gap-2 rounded-[20px] border-2 p-1.5"
          role="group"
          aria-label="成就顯示模式"
        >
          <SegmentButton
            active={mode === "unlocked"}
            onClick={() => setMode("unlocked")}
          >
            已獲得
          </SegmentButton>
          <SegmentButton active={mode === "all"} onClick={() => setMode("all")}>
            全成就
          </SegmentButton>
        </div>
      </section>

      <section className="mt-[18px]" aria-label="成就卡片列表">
        <div className="mb-2.5 flex items-end justify-between gap-3">
          <div>
            <span className="text-moss text-[11px] font-extrabold tracking-[0.08em] uppercase">
              COLLECTION
            </span>
            <h2 className="mt-0.5 text-[21px] leading-tight font-extrabold tracking-normal">
              {mode === "unlocked" ? "已獲得成就" : "所有成就"}
            </h2>
          </div>
          <p className="text-muted-foreground text-xs font-semibold whitespace-nowrap">
            {mode === "unlocked" ? "只看目前獲得" : "包含未解鎖剪影"}
          </p>
        </div>

        {isPending ? (
          <div className="grid grid-cols-2 gap-2.5">
            {[0, 1, 2, 3].map((item) => (
              <div
                key={item}
                className="bg-muted border-border min-h-[248px] animate-pulse rounded-[22px] border-2"
              />
            ))}
          </div>
        ) : visibleAchievements.length > 0 ? (
          <div className="grid grid-cols-2 gap-2.5">
            {visibleAchievements.map((achievement) => (
              <AchievementCard
                key={achievement.key}
                achievement={achievement}
                mode={mode}
              />
            ))}
          </div>
        ) : (
          <EmptyCollection onViewAll={() => setMode("all")} />
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

function AchievementCard({
  achievement,
  mode,
}: {
  achievement: Achievement
  mode: DisplayMode
}) {
  const complete = achievement.key === "codex_complete"
  const unlockedAt = formatUnlockedAt(achievement.unlockedAt)
  const tone = toneForAchievement(achievement)
  const asset = achievementAssetFor(achievement.key)

  return (
    <Card
      className={[
        "flex min-h-[248px] flex-col gap-2.5 rounded-[22px] p-2.5 py-2.5",
        achievement.unlocked
          ? "border-ink bg-card"
          : "border-ink bg-muted border-dashed",
      ].join(" ")}
      aria-label={`${achievement.name}，${achievement.unlocked ? "已獲得" : "未獲得"}`}
    >
      <div className="flex items-center justify-between gap-1.5">
        <span className="border-ink bg-surface-raised inline-flex min-h-6 items-center gap-1 rounded-full border-2 px-1.5 text-[11px] font-extrabold">
          <span
            className={cn("border-ink size-[9px] rounded-full border", tone)}
          />
          {complete ? "完整收藏" : `圖鑑 ${achievement.tier}`}
        </span>
        <span
          className={cn(
            "border-ink inline-flex min-h-6 items-center rounded-full border-2 px-1.5 text-[11px] font-extrabold",
            achievement.unlocked ? "bg-secondary" : "bg-muted",
          )}
        >
          {achievement.unlocked ? "已獲得" : "未解鎖"}
        </span>
      </div>

      <div className="border-border bg-surface-raised relative grid min-h-24 place-items-center overflow-hidden rounded-[18px] border-2">
        <GameIcon
          iconPath={asset.iconPath}
          alt={achievement.name}
          className="size-24 rounded-none"
          imageClassName={cn(
            "p-2.5 drop-shadow-[0_3px_0_rgba(23,35,58,0.18)]",
            !achievement.unlocked && "opacity-55 grayscale",
          )}
          fallback={
            <AchievementShape
              tier={achievement.tier ?? 0}
              unlocked={achievement.unlocked}
              complete={complete}
              tone={tone}
            />
          }
        />
        {achievement.unlocked && unlockedAt ? (
          <time
            dateTime={achievement.unlockedAt}
            className="bg-ink/90 text-primary-foreground absolute inset-x-0 bottom-0 px-1.5 py-1 text-center text-[10px] leading-tight font-extrabold"
          >
            {unlockedAt}
          </time>
        ) : null}
      </div>

      <div className="grid gap-1">
        <h3 className="text-[17px] leading-tight font-extrabold tracking-normal">
          {achievement.name}
        </h3>
        <p className="text-muted-foreground text-xs leading-5 font-semibold">
          收集 {achievement.requiredSitoneCount} 種小石
        </p>
      </div>

      <div className="border-border bg-surface-raised rounded-[14px] border px-2 py-1.5">
        <span className="block text-[12px] font-extrabold">成就獎勵</span>
        <span className="text-muted-foreground block text-[11px] leading-4 font-semibold">
          {achievement.openPowerReward
            ? `${achievement.openPowerReward} 開源力`
            : "完成全部小石圖鑑"}
        </span>
      </div>

      <div className="border-border mt-auto grid gap-0.5 border-t-2 border-dashed pt-2">
        <span className="text-[13px] font-extrabold">
          {achievement.unlocked
            ? "成就已收錄"
            : mode === "all"
              ? "尚未達成"
              : "未取得"}
        </span>
        {!achievement.unlocked ? (
          <span className="text-muted-foreground text-[11px] font-semibold">
            剪影狀態
          </span>
        ) : null}
      </div>
    </Card>
  )
}

function AchievementShape({
  tier,
  unlocked,
  complete = false,
  tone,
  className,
}: {
  tier: number
  unlocked: boolean
  complete?: boolean
  tone?: string
  className?: string
}) {
  const resolvedTone = tone ?? achievementTones[Math.max(tier - 1, 0) % 5]

  return (
    <div
      className={cn(
        "border-ink relative grid h-[58px] w-[62px] place-items-center rounded-[18px_24px_16px_26px] border-2",
        unlocked ? resolvedTone : "bg-muted",
        className,
      )}
      aria-hidden
    >
      <span className="border-ink/30 bg-card/45 absolute top-2 left-2 h-3.5 w-6 rotate-[-18deg] rounded-[12px_8px_10px_7px] border" />
      {unlocked ? (
        <Trophy className="relative z-10 size-7" />
      ) : (
        <Lock className="text-muted-foreground relative z-10 size-6" />
      )}
      <strong className="border-ink bg-card absolute right-[-7px] bottom-[-6px] z-20 grid size-[26px] place-items-center rounded-full border-2 text-xs font-extrabold">
        {complete ? "★" : unlocked ? tier : "?"}
      </strong>
    </div>
  )
}

function EmptyCollection({ onViewAll }: { onViewAll: () => void }) {
  return (
    <section className="border-ink bg-card grid justify-items-center gap-2.5 rounded-[22px] border-2 border-dashed px-[18px] py-6 text-center">
      <AchievementShape tier={1} unlocked={false} />
      <h3 className="text-lg font-extrabold">目前還沒有獲得成就</h3>
      <p className="text-muted-foreground max-w-[260px] text-[13px] leading-6 font-semibold">
        收集不同種類的小石，達成的圖鑑里程碑會收錄在這裡。
      </p>
      <Button
        type="button"
        variant="secondary"
        className="mt-1 min-h-11 rounded-2xl px-3.5 text-sm font-extrabold"
        onClick={onViewAll}
      >
        查看全部成就
      </Button>
    </section>
  )
}
