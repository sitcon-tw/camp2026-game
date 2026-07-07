import { Icon } from "@iconify/react"
import { Suspense } from "react"

import { HomeStatusPanel } from "@/features/home-status"
import { HomeStatusSkeleton } from "@/features/home-status/ui/home-status-skeleton"
import { Badge } from "@/shared/ui/badge"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/shared/ui/card"
import { IconBadge } from "@/shared/ui/icon-badge"
import {
  GameFeatureIcon,
  type GameFeatureIconName,
} from "@/shared/ui/game-feature-icon"
import { MetricCard } from "@/shared/ui/metric-card"

function metricFeatureIcon(name: GameFeatureIconName) {
  return function MetricFeatureIcon({ className }: { className?: string }) {
    return <GameFeatureIcon name={name} className={className} />
  }
}

const campMetrics = [
  {
    label: "今日任務",
    value: "12",
    tone: "success",
    icon: metricFeatureIcon("stones"),
  },
  {
    label: "知識王戰",
    value: "5",
    tone: "info",
    icon: metricFeatureIcon("battle"),
  },
  {
    label: "世界魔王",
    value: "68%",
    tone: "magic",
    icon: metricFeatureIcon("battle"),
  },
] as const

const pebbleTypes = [
  {
    name: "探索",
    icon: "solar:map-point-wave-bold-duotone",
    tone: "explore",
  },
  {
    name: "靈光",
    icon: "solar:lightbulb-bolt-bold-duotone",
    tone: "spark",
  },
  {
    name: "共鳴",
    icon: "solar:hand-heart-bold-duotone",
    tone: "resonate",
  },
  {
    name: "工程",
    icon: "solar:code-square-bold-duotone",
    tone: "engineer",
  },
  {
    name: "娛樂",
    icon: "solar:confetti-minimalistic-bold-duotone",
    tone: "play",
  },
] as const

export function HomePage() {
  return (
    <main className="bg-background text-foreground min-h-dvh">
      <div className="mx-auto flex w-full max-w-6xl flex-col gap-6 px-5 py-6 sm:px-8 lg:py-10">
        <section className="bg-card flex flex-col gap-4 rounded-lg border px-5 py-5 shadow-sm sm:flex-row sm:items-center sm:justify-between sm:px-6">
          <div className="space-y-2">
            <Badge variant="secondary" className="w-fit">
              SITCON Camp 2026
            </Badge>
            <div>
              <h1 className="text-2xl font-semibold sm:text-3xl">營隊基地</h1>
              <p className="text-muted-foreground mt-1 max-w-2xl text-sm sm:text-base">
                小石、任務與知識王戰會集中在這裡呈現。
              </p>
            </div>
          </div>
          <div className="grid grid-cols-3 gap-3 sm:min-w-80">
            {campMetrics.map((metric) => (
              <MetricCard key={metric.label} {...metric} />
            ))}
          </div>
        </section>

        <section className="grid gap-6 lg:grid-cols-[1.2fr_0.8fr]">
          <Suspense fallback={<HomeStatusSkeleton />}>
            <HomeStatusPanel />
          </Suspense>

          <Card>
            <CardHeader>
              <CardTitle>小石題色</CardTitle>
              <CardDescription>
                首版先保留展示資料，之後接後端題組與玩家收藏。
              </CardDescription>
            </CardHeader>
            <CardContent className="flex flex-wrap gap-2">
              {pebbleTypes.map((type) => (
                <IconBadge
                  key={type.name}
                  label={type.name}
                  tone={type.tone}
                  icon={
                    <Icon icon={type.icon} className="size-4" aria-hidden />
                  }
                />
              ))}
            </CardContent>
          </Card>
        </section>
      </div>
    </main>
  )
}
