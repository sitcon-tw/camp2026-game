import { useMutation, useQueryClient } from "@tanstack/react-query"
import { toast } from "sonner"
import type { ComponentProps } from "react"

import { gameApi, type ShopItem } from "@/shared/api/game"
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogMedia,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "@/shared/ui/alert-dialog"
import { Button } from "@/shared/ui/button"
import { GameFeatureIcon } from "@/shared/ui/game-feature-icon"

type ShopPurchaseConfirmButtonProps = {
  item: ShopItem
  currentOpenPower?: number
  className?: string
  size?: ComponentProps<typeof Button>["size"]
  label?: string
}

export function ShopPurchaseConfirmButton({
  item,
  currentOpenPower,
  className,
  size,
  label = "購買",
}: ShopPurchaseConfirmButtonProps) {
  const queryClient = useQueryClient()
  const canAfford =
    currentOpenPower == null || currentOpenPower >= item.priceOpenPower
  const purchaseMutation = useMutation({
    mutationFn: gameApi.purchase,
    onSuccess: (result) => {
      toast.success(`已購買 ${result.item.name}`)
      queryClient.invalidateQueries({ queryKey: ["shop", "items"] })
      queryClient.invalidateQueries({ queryKey: ["shop", "item", item.id] })
      queryClient.invalidateQueries({ queryKey: ["me", "status"] })
      queryClient.invalidateQueries({ queryKey: ["me", "home"] })
      queryClient.invalidateQueries({ queryKey: ["me", "items"] })
    },
    onError: (error) => {
      toast.error(error instanceof Error ? error.message : "購買失敗")
    },
  })

  return (
    <AlertDialog>
      <AlertDialogTrigger asChild>
        <Button
          className={className}
          size={size}
          disabled={purchaseMutation.isPending}
        >
          <GameFeatureIcon name="shop" className="size-4" />
          {purchaseMutation.isPending ? "購買中" : label}
        </Button>
      </AlertDialogTrigger>
      <AlertDialogContent size="sm">
        <AlertDialogHeader>
          <AlertDialogMedia>
            <GameFeatureIcon name="shop" className="size-10" />
          </AlertDialogMedia>
          <AlertDialogTitle>確認購買</AlertDialogTitle>
          <AlertDialogDescription>
            將使用 {item.priceOpenPower} 開源力購買「{item.name}」。
            {currentOpenPower == null
              ? ""
              : ` 目前持有 ${currentOpenPower} 開源力。`}
          </AlertDialogDescription>
        </AlertDialogHeader>
        {!canAfford ? (
          <div className="text-destructive text-sm font-bold">
            目前開源力不足，無法完成購買。
          </div>
        ) : null}
        <AlertDialogFooter>
          <AlertDialogCancel disabled={purchaseMutation.isPending}>
            取消
          </AlertDialogCancel>
          <AlertDialogAction
            disabled={!canAfford || purchaseMutation.isPending}
            onClick={() => purchaseMutation.mutate(item.id)}
          >
            <GameFeatureIcon name="shop" className="size-4" />
            {purchaseMutation.isPending ? "購買中" : "確認購買"}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  )
}
