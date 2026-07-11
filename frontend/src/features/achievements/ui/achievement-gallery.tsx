import { useQuery } from "@tanstack/react-query"
import { Clock, Lock, Trophy } from "lucide-react"
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
  const collectedSitoneCount = data?.collectedSitoneCount ?? 0
  const totalSitoneCount = data?.totalSitoneCount ?? 0
  const nextAchievement = achievements.find(
    (achievement) => !achievement.unlocked,
  )
  const featuredAchievement =
    nextAchievement ?? achievements[achievements.length - 1]
  const featuredAsset = featuredAchievement
    ? achievementAssetFor(featuredAchievement.key)
    : null
  const remainingSitones = nextAchievement
    ? Math.max(nextAchievement.requiredSitoneCount - collectedSitoneCount, 0)
    : 0
  const earnedOpenPower = achievements.reduce(
    (total, achievement) =>
      total + (achievement.unlocked ? (achievement.openPowerReward ?? 0) : 0),
    0,
  )
  const collectionProgress = totalSitoneCount
    ? Math.min(100, (collectedSitoneCount / totalSitoneCount) * 100)
    : 0

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
      <section
        className="bg-surface-raised border-ink mt-[18px] overflow-hidden rounded-[22px] border-2"
        style={{ boxShadow: "4px 4px 0 rgba(23,35,58,.14)" }}
        aria-label="成就收藏摘要"
      >
        <div className="grid grid-cols-[58px_minmax(0,1fr)_auto] items-center gap-3 p-4 pb-3">
          <div
            className={cn(
              "border-ink grid size-[58px] place-items-center overflow-hidden rounded-[18px] border-2",
              featuredAchievement
                ? toneForAchievement(featuredAchievement)
                : "bg-pebble-spark",
            )}
          >
            {featuredAsset ? (
              <GameIcon
                iconPath={featuredAsset.iconPath}
                alt=""
                className="size-full border-0 bg-transparent"
                imageClassName="p-1.5"
                fallback={<Trophy className="size-7" aria-hidden />}
              />
            ) : (
              <Trophy className="text-muted-foreground size-7" aria-hidden />
            )}
          </div>
          <div className="min-w-0">
            <p className="text-primary text-[11px] font-black tracking-[0.08em] uppercase">
              Achievement
            </p>
            <strong className="mt-0.5 block truncate text-[19px] leading-tight font-black">
              {isPending
                ? "同步成就中"
                : nextAchievement?.name || "所有成就已完成"}
            </strong>
            <p className="text-muted-foreground mt-1 text-xs font-bold">
              {isPending
                ? "讀取收藏紀錄"
                : nextAchievement
                  ? remainingSitones > 0
                    ? `再收集 ${remainingSitones} 種小石`
                    : "已達成，等待解鎖"
                  : "石全石美"}
            </p>
          </div>
          <div className="text-right">
            <span className="bg-card border-ink inline-flex min-h-7 items-center rounded-full border-2 px-2.5 text-xs font-black whitespace-nowrap">
              {isPending ? "-/-" : `${unlockedCount}/${achievements.length}`}
            </span>
            <strong className="mt-1 block text-[14px] font-black whitespace-nowrap">
              {nextAchievement?.openPowerReward
                ? `${nextAchievement.openPowerReward} OP`
                : isPending
                  ? "-"
                  : "完成"}
            </strong>
          </div>
        </div>

        <div className="px-4 pb-3.5">
          <div className="mb-1.5 flex items-center justify-between gap-3 text-xs font-bold">
            <span className="text-muted-foreground">小石圖鑑進度</span>
            <strong>
              {isPending
                ? "-/-"
                : `${collectedSitoneCount}/${totalSitoneCount}`}
            </strong>
          </div>
          <div
            className="bg-card border-ink h-3.5 overflow-hidden rounded-full border-2"
            role="progressbar"
            aria-label="小石圖鑑進度"
            aria-valuemin={0}
            aria-valuemax={totalSitoneCount || 1}
            aria-valuenow={collectedSitoneCount}
          >
            <div
              className="bg-primary h-full rounded-full transition-[width]"
              style={{ width: `${collectionProgress}%` }}
            />
          </div>
        </div>

        <div className="border-border bg-card grid grid-cols-3 border-t-2">
          <SummaryMetric
            label="歷史收集"
            value={
              isPending ? "-/-" : `${collectedSitoneCount}/${totalSitoneCount}`
            }
          />
          <SummaryMetric
            label="達成成就"
            value={isPending ? "-" : `${unlockedCount} 個`}
            className="border-border border-x-2"
          />
          <SummaryMetric
            label="成就獎勵"
            value={isPending ? "-" : `${earnedOpenPower} OP`}
          />
        </div>
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

function SummaryMetric({
  label,
  value,
  className,
}: {
  label: string
  value: string
  className?: string
}) {
  return (
    <div
      className={cn(
        "grid min-h-[66px] content-center gap-0.5 px-2 text-center",
        className,
      )}
    >
      <span className="text-muted-foreground text-[11px] font-bold">
        {label}
      </span>
      <strong className="text-[15px] font-black whitespace-nowrap">
        {value}
      </strong>
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

function AchievementCard({ achievement }: { achievement: Achievement }) {
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
        {achievement.unlocked && unlockedAt ? (
          <time
            dateTime={achievement.unlockedAt}
            className="flex items-center gap-1 text-[12px] leading-4 font-extrabold"
          >
            <Clock className="size-3 shrink-0" aria-hidden />
            {unlockedAt}
          </time>
        ) : (
          <span className="text-muted-foreground text-[13px] font-extrabold">
            成就未收錄
          </span>
        )}
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
  const emptyStateAsset = achievementAssetFor("codex_tier_01")

  return (
    <section className="border-ink bg-card grid justify-items-center gap-2.5 rounded-[22px] border-2 border-dashed px-[18px] py-6 text-center">
      <div className="border-ink bg-pebble-spark grid size-28 place-items-center overflow-hidden rounded-[22px] border-2">
        <GameIcon
          iconPath={emptyStateAsset.iconPath}
          alt="石來運轉成就"
          className="size-full border-0 bg-transparent"
          imageClassName="p-2 opacity-60 grayscale drop-shadow-[0_3px_0_rgba(23,35,58,0.18)]"
          fallback={<AchievementShape tier={1} unlocked={false} />}
        />
      </div>
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
