import { Badge } from "@/shared/ui/badge"
import { Card, CardContent } from "@/shared/ui/card"
import { PlayerAvatar } from "@/shared/ui/player-avatar"
import { Check, Loader2 } from "lucide-react"

type BattleWaitingPlayerCardType = {
  playerId: string
  name: string
  avatarUrl?: string
  kind?: "human" | "computer"
  team: string
  ready: boolean
  loadoutCount?: number
}

export function BattleWaitingPlayerCard({
  playerId,
  name,
  avatarUrl,
  kind,
  team,
  ready,
  loadoutCount = 0,
}: BattleWaitingPlayerCardType) {
  return (
    <Card>
      <CardContent className="flex items-center justify-between">
        <PlayerAvatar
          playerId={playerId}
          nickname={name}
          avatarUrl={avatarUrl}
          kind={kind}
          className="bg-pebble-spark border-ink size-20 rounded-[22px] border-2"
        />
        <div className="flex flex-col items-center justify-center">
          <span className="text-2xl font-bold">{name}</span>
          <span className="text-muted-foreground text-lg">{team}</span>
          <span className="text-muted-foreground text-sm font-bold">
            {loadoutCount > 0 ? `${loadoutCount} 顆小石` : "尚未選小石"}
          </span>
        </div>
        <Badge
          variant={ready ? "default" : "outline"}
          className={ready ? "bg-status-success" : "animate-pulse"}
        >
          {ready ? <Check /> : <Loader2 className="animate-spin" />}
          {ready ? "已準備" : "等待中"}
        </Badge>
      </CardContent>
    </Card>
  )
}
