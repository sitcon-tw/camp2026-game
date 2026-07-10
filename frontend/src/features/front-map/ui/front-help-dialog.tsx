import {
  Flag,
  Focus,
  Gem,
  Info,
  MoveUpRight,
  ShieldPlus,
  Swords,
  Trophy,
  Zap,
} from "lucide-react"
import type { ComponentType, ReactNode } from "react"

import type { FrontMapMode } from "@/shared/api/game"
import { Badge } from "@/shared/ui/badge"
import { Button } from "@/shared/ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/shared/ui/dialog"
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/shared/ui/tooltip"

export function FrontHelpDialog({ mapMode }: { mapMode: FrontMapMode }) {
  return (
    <Dialog>
      <TooltipProvider delayDuration={100}>
        <Tooltip>
          <TooltipTrigger asChild>
            <DialogTrigger asChild>
              <Button
                type="button"
                size="icon-sm"
                variant="secondary"
                aria-label="查看戰線玩法"
              >
                <Info className="size-4" aria-hidden />
              </Button>
            </DialogTrigger>
          </TooltipTrigger>
          <TooltipContent side="bottom">戰線玩法</TooltipContent>
        </Tooltip>
      </TooltipProvider>

      <DialogContent className="max-h-[86dvh] gap-3 overflow-y-auto p-4 sm:p-6">
        <DialogHeader className="pr-10 text-left">
          <DialogTitle className="text-xl">開源戰線玩法</DialogTitle>
          <DialogDescription>
            {mapMode === "territory_grid"
              ? "擴張領土、守住邊界，和小隊一起累積排名分數。"
              : "派遣小石到節點，和小隊一起完成戰線命令。"}
          </DialogDescription>
        </DialogHeader>

        {mapMode === "territory_grid" ? (
          <div className="grid gap-0">
            <RuleRow icon={MoveUpRight} title="擴張" badge="10 開源力">
              選擇中立區域，從最接近該方向的我方邊界取得相連領土。
            </RuleRow>
            <RuleRow icon={Swords} title="攻擊" badge="15 開源力">
              選擇已與我方接壤的敵隊領土，削弱或佔領該方向的敵方邊界。
            </RuleRow>
            <RuleRow icon={ShieldPlus} title="防守" badge="最多 8 開源力">
              選擇防禦未滿的己方領土，強化該方向的邊界；防禦 100
              時不可重複投入，實際消耗依補防格數計算。
            </RuleRow>
            <RuleRow icon={Focus} title="包圍">
              封閉一整塊區域後，內部可佔領格會自動歸入我方；只要區域內含敵方基地核心，就不會自動吞併。
            </RuleRow>
            <RuleRow icon={Flag} title="永久基地">
              每隊旗幟是不可佔領的基地核心。攻擊只會影響基地周圍的一般領土。
            </RuleRow>
            <RuleRow icon={Zap} title="開源力">
              戰線與首頁、商店使用同一份個人開源力。只有執行命令的學員會支付畫面標示的開源力，小隊其他成員不會被扣款。
            </RuleRow>
            <RuleRow icon={Gem} title="前線小石">
              可從自己已擁有的小石中選擇派遣。小石不會被消耗，敵隊現有收藏也永遠不會被奪走；取得的是里程碑或救援額外發放的新小石。
            </RuleRow>
            <RuleRow icon={Trophy} title="分數與獎勵" last>
              目前領土每格值 10 戰線分，失地時分數也會下降。小隊歷史最高領土每達
              25
              格，執行該次佔領的學員獲得小石；失地後重搶不會重複發放。救援地標另獎勵
              1 顆小石。佔領資源地標後可再採集 1
              顆小石；戰線獎勵不會增加開源力。
            </RuleRow>
          </div>
        ) : (
          <div className="grid gap-0">
            <RuleRow icon={Gem} title="派遣小石">
              從自己已擁有的小石中選擇一顆，再對相鄰節點送出可用命令。
            </RuleRow>
            <RuleRow icon={Zap} title="開源力">
              戰線與首頁、商店共用個人開源力，執行命令時只扣除操作學員自己的餘額。
            </RuleRow>
            <RuleRow icon={Trophy} title="小隊成果" last>
              控制節點並完成修復、救援與挑戰，可提高小隊戰線成果與排名。
            </RuleRow>
          </div>
        )}
      </DialogContent>
    </Dialog>
  )
}

function RuleRow({
  icon: Icon,
  title,
  badge,
  last = false,
  children,
}: {
  icon: ComponentType<{ className?: string }>
  title: string
  badge?: string
  last?: boolean
  children: ReactNode
}) {
  return (
    <div
      className={
        last ? "grid gap-1 py-3" : "border-border grid gap-1 border-b py-3"
      }
    >
      <div className="flex flex-wrap items-center gap-2 text-sm font-black">
        <Icon className="text-primary size-4" aria-hidden />
        <span>{title}</span>
        {badge ? (
          <Badge variant="outline" className="ml-auto">
            {badge}
          </Badge>
        ) : null}
      </div>
      <p className="text-muted-foreground pl-6 text-sm leading-6 font-semibold">
        {children}
      </p>
    </div>
  )
}
