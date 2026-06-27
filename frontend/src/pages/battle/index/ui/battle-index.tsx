import { Button } from "@/shared/ui/button"
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/shared/ui/card"
import { Field } from "@/shared/ui/field"
import { GameFeatureIcon } from "@/shared/ui/game-feature-icon"
import { Input } from "@/shared/ui/input"
import { PageHeader } from "@/shared/ui/page-header"
import { ArrowRight, DoorOpen, ScanQrCode } from "lucide-react"

export function BattleIndexPage() {
  return (
    <main className="mx-auto grid w-full max-w-sm gap-y-2 py-4">
      <PageHeader title="知識王" headline="Battle Lobby" />
      {/* 單人遊戲 */}
      <Card>
        <CardHeader>
          <CardTitle>快速開始</CardTitle>
          <CardDescription>與 CPU 對決</CardDescription>
        </CardHeader>
        <CardContent>
          <span>與 SITCON 電腦進行對決，複習上課知識！</span>
        </CardContent>
        <CardFooter>
          <Button className="w-full">
            <GameFeatureIcon name="battle" className="size-5" />
            開始遊戲
          </Button>
        </CardFooter>
      </Card>
      {/* 多人連線 */}
      <Card>
        <CardHeader>
          <CardTitle>多人連線</CardTitle>
          <CardDescription>見面就是 Yo Battle!</CardDescription>
        </CardHeader>
        <CardContent>
          <span>和其他學員連線對戰，比拼誰才是知識王！</span>
        </CardContent>
        <CardFooter className="grid gap-2">
          {/* 加入房間 */}
          <Field orientation="horizontal">
            <Input id="input-room-id" type="text" placeholder="請輸入房號" />
            <Button size="icon-lg">
              <ScanQrCode />
            </Button>
            <Button className="w-full flex-1" size="lg" variant="secondary">
              加入房間
              <ArrowRight />
            </Button>
          </Field>
          {/* - 或 - */}
          <span className="text-muted-foreground text-center">或</span>
          {/* 創建房間 */}
          <Button className="w-full flex-1">
            <DoorOpen />
            創建房間
          </Button>
        </CardFooter>
      </Card>
    </main>
  )
}
