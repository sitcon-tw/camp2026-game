import { z } from "zod"

export const CombatStoneSchema = z.object({
  id: z.string(),
  name: z.string(),
  type: z.string(),
  pictureSrc: z.string(),
})

export type CombatStone = z.infer<typeof CombatStoneSchema>
