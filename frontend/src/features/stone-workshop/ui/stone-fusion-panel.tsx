import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { useMemo, useState } from "react"
import { Check, X } from "lucide-react"
import { toast } from "sonner"

import { gameApi, type FusionRecipe } from "@/shared/api/game"
import {
  rarityLabel,
  sitoneMeta,
  itemTypeClass,
} from "@/shared/lib/game-labels"
import { Button } from "@/shared/ui/button"
import { Card } from "@/shared/ui/card"
import { GameFeatureIcon } from "@/shared/ui/game-feature-icon"
import { GameIcon } from "@/shared/ui/game-icon"
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/shared/ui/dialog"
import { cn } from "@/shared/utils"

function componentTone(component: FusionRecipe["inputs"][number]) {
  if (component.kind === "sitone" && component.type) {
    return sitoneMeta(component.type).bgClassName
  }
  return component.type ? itemTypeClass(component.type) : "bg-primary"
}

function canFuse(recipe: FusionRecipe) {
  return recipe.enabled && recipe.available
}

function ComponentPill({
  component,
}: {
  component: FusionRecipe["inputs"][number]
}) {
  return (
    <div className="bg-surface-raised border-border grid grid-cols-[36px_1fr_auto] items-center gap-2 rounded-[16px] border-2 p-2">
      <span
        className={[
          "border-ink grid size-9 place-items-center overflow-hidden rounded-[12px_16px_10px_14px] border-2",
          componentTone(component),
        ].join(" ")}
        aria-hidden
      >
        <GameIcon
          iconPath={component.iconPath}
          imageClassName="p-0.5"
          fallback={null}
        />
      </span>
      <span className="min-w-0">
        <strong className="block truncate text-sm font-extrabold">
          {component.name}
        </strong>
        <small className="text-muted-foreground block text-xs font-semibold">
          {component.rarity ? rarityLabel(component.rarity) : component.kind}
        </small>
        {component.kind === "sitone" && component.abilityDescription ? (
          <small className="text-muted-foreground mt-0.5 block text-[11px] leading-4 font-semibold">
            {component.abilityName}：{component.abilityDescription}
          </small>
        ) : null}
      </span>
      <strong className="text-sm font-extrabold">x{component.quantity}</strong>
    </div>
  )
}

function ComponentPreview({
  component,
}: {
  component: FusionRecipe["inputs"][number]
}) {
  return (
    <span className="border-border bg-surface-raised inline-flex min-w-0 items-center gap-1.5 rounded-full border px-2 py-1 text-xs font-bold">
      <span
        className={cn(
          "border-ink grid size-4 shrink-0 place-items-center overflow-hidden rounded-[6px] border",
          componentTone(component),
        )}
        aria-hidden
      >
        <GameIcon
          iconPath={component.iconPath}
          imageClassName="p-px"
          fallback={null}
        />
      </span>
      <span className="truncate">{component.name}</span>
      <span className="shrink-0">x{component.quantity}</span>
    </span>
  )
}

function ComponentPreviewRow({
  label,
  components,
}: {
  label: string
  components: FusionRecipe["inputs"]
}) {
  return (
    <div className="grid grid-cols-[40px_minmax(0,1fr)] items-start gap-2">
      <span className="text-muted-foreground pt-1 text-xs font-extrabold">
        {label}
      </span>
      <div className="flex min-w-0 flex-wrap gap-1.5">
        {components.map((component) => (
          <ComponentPreview
            key={`${label}-${component.kind}-${component.id}`}
            component={component}
          />
        ))}
      </div>
    </div>
  )
}

function RecipeListItem({
  recipe,
  onSelect,
  pending,
}: {
  recipe: FusionRecipe
  onSelect: () => void
  pending: boolean
}) {
  const output = recipe.outputs[0]
  const ready = canFuse(recipe)

  return (
    <button
      type="button"
      className={cn(
        "bg-card border-ink grid w-full gap-3 rounded-[22px] border-2 p-3 text-left shadow-[4px_4px_0_rgba(23,35,58,0.12)] transition-all active:translate-x-px active:translate-y-px",
        ready ? "hover:-translate-y-0.5" : "opacity-75",
      )}
      onClick={onSelect}
    >
      <div className="grid grid-cols-[64px_minmax(0,1fr)_auto] items-center gap-3">
        <div
          className={[
            "border-ink grid size-16 rotate-[-6deg] place-items-center rounded-[20px_24px_18px_22px] border-2",
            output ? componentTone(output) : "bg-pebble-engineer",
          ].join(" ")}
          aria-hidden
        >
          <GameIcon
            iconPath={output?.iconPath}
            imageClassName="p-1.5"
            fallback={<GameFeatureIcon name="forge" className="size-12" />}
          />
        </div>
        <div className="min-w-0">
          <h2 className="truncate text-xl leading-tight font-extrabold tracking-normal">
            {recipe.name}
          </h2>
          <p className="text-muted-foreground mt-1 line-clamp-2 text-sm leading-5 font-medium">
            {recipe.description}
          </p>
        </div>
        <span
          className={cn(
            "border-ink rounded-full border px-2.5 py-1 text-xs font-extrabold whitespace-nowrap",
            ready ? "bg-secondary" : "bg-surface-raised",
          )}
        >
          {pending ? "合成中" : ready ? "可合成" : "材料不足"}
        </span>
      </div>

      <div className="grid gap-1.5">
        <ComponentPreviewRow label="需要" components={recipe.inputs} />
        <ComponentPreviewRow label="產出" components={recipe.outputs} />
      </div>
    </button>
  )
}

function RecipeSection({
  title,
  count,
  children,
}: {
  title: string
  count: number
  children: React.ReactNode
}) {
  if (count === 0) return null

  return (
    <section className="grid gap-2">
      <div className="flex items-center justify-between gap-2 px-1">
        <h2 className="text-[17px] font-extrabold">{title}</h2>
        <span className="text-muted-foreground text-xs font-bold">
          {count} 個配方
        </span>
      </div>
      <div className="grid gap-2">{children}</div>
    </section>
  )
}

function FusionConfirmDialog({
  recipe,
  pending,
  onOpenChange,
  onConfirm,
}: {
  recipe: FusionRecipe | null
  pending: boolean
  onOpenChange: (open: boolean) => void
  onConfirm: (recipeID: string) => void
}) {
  const ready = recipe ? canFuse(recipe) : false

  return (
    <Dialog open={Boolean(recipe)} onOpenChange={onOpenChange}>
      {recipe ? (
        <DialogContent className="gap-4">
          <DialogHeader>
            <DialogTitle>{ready ? "確認合成" : "材料不足"}</DialogTitle>
            <DialogDescription>
              {ready
                ? `確定要合成「${recipe.name}」嗎？`
                : `目前還不能合成「${recipe.name}」。`}
            </DialogDescription>
          </DialogHeader>

          <div className="bg-surface-raised grid grid-cols-[72px_1fr] items-center gap-3 rounded-[20px] p-3">
            <div
              className={[
                "border-ink grid size-[72px] rotate-[-6deg] place-items-center rounded-[24px_28px_20px_24px] border-2",
                recipe.outputs[0]
                  ? componentTone(recipe.outputs[0])
                  : "bg-pebble-engineer",
              ].join(" ")}
              aria-hidden
            >
              <GameIcon
                iconPath={recipe.outputs[0]?.iconPath}
                imageClassName="p-1.5"
                fallback={<GameFeatureIcon name="forge" className="size-14" />}
              />
            </div>
            <div className="min-w-0">
              <h3 className="text-xl leading-tight font-extrabold">
                {recipe.name}
              </h3>
              <p className="text-muted-foreground mt-1 text-sm leading-5 font-medium">
                {recipe.description}
              </p>
            </div>
          </div>

          <section className="grid gap-2" aria-label={`${recipe.name} 消耗`}>
            <div className="flex items-center justify-between gap-2">
              <h3 className="text-[17px] font-extrabold">消耗</h3>
              <span className="text-muted-foreground text-xs font-bold">
                {ready ? "材料足夠" : "材料不足"}
              </span>
            </div>
            {recipe.inputs.map((component) => (
              <ComponentPill
                key={`input-${component.kind}-${component.id}`}
                component={component}
              />
            ))}
          </section>

          <section className="grid gap-2" aria-label={`${recipe.name} 產物`}>
            <h3 className="text-[17px] font-extrabold">產物</h3>
            {recipe.outputs.map((component) => (
              <ComponentPill
                key={`output-${component.kind}-${component.id}`}
                component={component}
              />
            ))}
          </section>

          <DialogFooter className="grid grid-cols-2 gap-2 sm:grid-cols-2">
            <DialogClose asChild>
              <Button type="button" variant="outline" className="w-full">
                <X />
                取消
              </Button>
            </DialogClose>
            <Button
              type="button"
              className="w-full"
              disabled={!ready || pending}
              onClick={() => onConfirm(recipe.id)}
            >
              <Check />
              {pending ? "合成中" : ready ? "確定" : "材料不足"}
            </Button>
          </DialogFooter>
        </DialogContent>
      ) : null}
    </Dialog>
  )
}

export function StoneFusionPanel() {
  const queryClient = useQueryClient()
  const [selectedRecipe, setSelectedRecipe] = useState<FusionRecipe | null>(
    null,
  )
  const { data: recipes = [], isPending } = useQuery({
    queryKey: ["fusions", "recipes"],
    queryFn: gameApi.fusionRecipes,
  })
  const sortedRecipes = useMemo(
    () =>
      recipes
        .map((recipe, index) => ({ recipe, index }))
        .sort((first, second) => {
          const firstReady = canFuse(first.recipe)
          const secondReady = canFuse(second.recipe)
          if (firstReady !== secondReady) {
            return firstReady ? -1 : 1
          }
          return first.index - second.index
        })
        .map(({ recipe }) => recipe),
    [recipes],
  )
  const fusionMutation = useMutation({
    mutationFn: gameApi.createFusion,
    onSuccess: (result) => {
      toast.success(`已合成：${result.recipe.name}`)
      setSelectedRecipe(null)
      queryClient.invalidateQueries({ queryKey: ["fusions", "recipes"] })
      queryClient.invalidateQueries({ queryKey: ["me", "items"] })
      queryClient.invalidateQueries({ queryKey: ["me", "sitones"] })
      queryClient.invalidateQueries({ queryKey: ["me", "home"] })
    },
    onError: (error) => {
      toast.error(error instanceof Error ? error.message : "合成失敗")
    },
  })

  return (
    <div className="flex flex-1 flex-col gap-3 pb-3">
      {isPending ? (
        <Card className="rounded-[22px] p-4 py-4">
          <h2 className="text-[21px] leading-tight font-extrabold">
            正在同步合成配方
          </h2>
        </Card>
      ) : recipes.length > 0 ? (
        <>
          <RecipeSection title="全部配方" count={sortedRecipes.length}>
            {sortedRecipes.map((recipe) => (
              <RecipeListItem
                key={recipe.id}
                recipe={recipe}
                pending={
                  fusionMutation.isPending &&
                  fusionMutation.variables === recipe.id
                }
                onSelect={() => setSelectedRecipe(recipe)}
              />
            ))}
          </RecipeSection>

          <FusionConfirmDialog
            recipe={selectedRecipe}
            pending={
              fusionMutation.isPending &&
              fusionMutation.variables === selectedRecipe?.id
            }
            onOpenChange={(open) => {
              if (!open) setSelectedRecipe(null)
            }}
            onConfirm={(recipeID) => fusionMutation.mutate(recipeID)}
          />
        </>
      ) : (
        <Card className="rounded-[22px] p-4 py-4">
          <h2 className="text-[21px] leading-tight font-extrabold">
            目前沒有合成配方
          </h2>
        </Card>
      )}
    </div>
  )
}
