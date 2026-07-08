import { Button } from "@/shared/ui/button"
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/shared/ui/card"
import { GameFeatureIcon } from "@/shared/ui/game-feature-icon"
import { PageHeader } from "@/shared/ui/page-header"
import { QrCode, ScanQrCode } from "lucide-react"

export function BattleIndexPage() {
  return (
    <main className="mx-auto grid w-full max-w-sm gap-y-2 py-4">
      <PageHeader title="知識王" headline="Battle Lobby" />
      <Card>
        <CardHeader>
          <CardTitle>現場配對</CardTitle>
          <CardDescription>用短效 QR Code 當面加入雙人知識王</CardDescription>
        </CardHeader>
        <CardContent>
          <span>
            一位玩家顯示 QR Code，另一位玩家在現場掃描加入；QR Code 會持續更新。
          </span>
        </CardContent>
        <CardFooter className="grid grid-cols-2 gap-2">
          <Button className="w-full">
            <QrCode />
            顯示配對 QR
          </Button>
          <Button className="w-full" variant="secondary">
            <ScanQrCode />
            掃描配對 QR
          </Button>
        </CardFooter>
      </Card>
      <Card>
        <CardHeader>
          <CardTitle>電腦對戰</CardTitle>
          <CardDescription>與 CPU 對決</CardDescription>
        </CardHeader>
        <CardContent>
          <span>與 SITCON 電腦進行對決，複習上課知識！</span>
        </CardContent>
        <CardFooter>
          <Button className="w-full" variant="secondary">
            <GameFeatureIcon name="battle" className="size-4" />
            跟電腦對戰
          </Button>
        </CardFooter>
      </Card>
    </main>
  )
}
