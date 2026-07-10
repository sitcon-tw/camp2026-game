import type { FrontCommandKind, FrontSitone } from "@/shared/api/game"

const baseAffectedCells: Partial<Record<FrontCommandKind, number>> = {
  expand: 8,
  attack: 6,
  reinforce: 8,
}

const baseEventScores: Partial<Record<FrontCommandKind, number>> = {
  repair: 30,
  rescue: 30,
  support: 10,
  answer_challenge: 20,
}

export type FrontSitonePreview = {
  selectedCount: number
  squadBonusPercent: number
  affinityBonusPercent: number
  totalBonusPercent: number
  affectedCells?: number
  affectedCellBonus: number
  defense?: number
  defenseBonus: number
  score?: number
  scoreBonus: number
}

export function calculateFrontSitonePreview(
  sitones: FrontSitone[],
  kind: FrontCommandKind,
): FrontSitonePreview {
  const selectedCount = sitones.length
  const squadBonusPercent = Math.max(0, selectedCount - 1) * 5
  const affinityBonusPercent = Math.min(
    40,
    sitones.reduce(
      (total, sitone) =>
        sitone.frontAffinityCommands.includes(kind)
          ? total + sitone.abilityValue
          : total,
      0,
    ),
  )
  const totalBonusPercent = Math.min(
    60,
    squadBonusPercent + affinityBonusPercent,
  )
  const baseCells = baseAffectedCells[kind]
  const affectedCellBonus = scaledBonus(baseCells, totalBonusPercent)
  const baseScore = baseEventScores[kind]
  const scoreBonus = scaledBonus(baseScore, totalBonusPercent)
  const defenseBonus =
    kind === "repair" ? scaledBonus(25, totalBonusPercent) : 0

  return {
    selectedCount,
    squadBonusPercent,
    affinityBonusPercent,
    totalBonusPercent,
    affectedCells:
      baseCells === undefined ? undefined : baseCells + affectedCellBonus,
    affectedCellBonus,
    defense: kind === "repair" ? 25 + defenseBonus : undefined,
    defenseBonus,
    score: baseScore === undefined ? undefined : baseScore + scoreBonus,
    scoreBonus,
  }
}

function scaledBonus(base: number | undefined, percent: number) {
  if (!base || percent <= 0) return 0
  return Math.ceil((base * percent) / 100)
}
