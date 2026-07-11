import { AchievementGallery } from "@/features/achievements"
import { WorkshopPageShell } from "@/features/stone-workshop"

export function AchievementsPage() {
  return (
    <WorkshopPageShell eyebrow="ACHIEVEMENT FIELD KIT" title="成就圖鑑">
      <AchievementGallery />
    </WorkshopPageShell>
  )
}
