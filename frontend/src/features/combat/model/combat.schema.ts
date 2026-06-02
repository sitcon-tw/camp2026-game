import { z } from "zod"

export const CombatStoneSchema = z.object({
  id: z.string(),
  name: z.string(),
  type: z.enum(["探索", "靈光", "共鳴", "工程", "娛樂"]),
  pictureSrc: z.string(),
})

export type CombatStone = z.infer<typeof CombatStoneSchema>
