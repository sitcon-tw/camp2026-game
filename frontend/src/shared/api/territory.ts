import { z } from "zod"

import { apiClient } from "./client"

const nullableArray = <T extends z.ZodType>(schema: T) =>
  z
    .array(schema)
    .nullish()
    .transform((value) => value ?? [])

export const TerritoryTierSchema = z.enum([
  "leader",
  "contested",
  "challenger",
])

const TerritoryStandingTierSchema = z.union([
  TerritoryTierSchema,
  z.literal("boss"),
])

export const RiskLevelSchema = z.enum(["low", "medium", "high"])

const TerritoryStandingTeamSchema = z.object({
  teamId: z.string(),
  name: z.string(),
  avatarUrl: z.string().optional(),
  tier: TerritoryStandingTierSchema,
  canAttack: z.boolean(),
  canBeAttacked: z.boolean(),
  isBoss: z.boolean().optional(),
})

const TerritoryStandingsResponseSchema = z.object({
  teams: nullableArray(TerritoryStandingTeamSchema),
  myTeamId: z.string(),
  myTier: TerritoryTierSchema,
  myActions: nullableArray(z.string()),
  activeMissionId: z.string().optional(),
})

const EstimatedCostSchema = z.object({
  min: z.number(),
  max: z.number(),
})

export const AttackMissionStatusSchema = z.enum([
  "voting",
  "deployed",
  "resolved",
  "cancelled",
])

const AttackMissionResultSchema = z.object({
  outcome: z.string(),
  summary: z.string().optional(),
  capturedSitoneIds: nullableArray(z.string()),
  lostSitoneIds: nullableArray(z.string()),
  missingSitoneIds: nullableArray(z.string()),
  deadSitoneIds: nullableArray(z.string()),
})

const AttackMissionSchema = z.object({
  id: z.string(),
  status: AttackMissionStatusSchema,
  attackerTeamId: z.string().optional(),
  defenderTeamId: z.string().optional(),
  initiatorPlayerId: z.string().optional(),
  sitoneIds: nullableArray(z.string()),
  voteDeadline: z.string().optional(),
  resolveAt: z.string().optional(),
  estimatedCost: EstimatedCostSchema.optional(),
  riskLevel: RiskLevelSchema.optional(),
  missingRisk: RiskLevelSchema.optional(),
  deathRisk: RiskLevelSchema.optional(),
  agreeCount: z.number().default(0),
  threshold: z.number().default(0),
  myVote: z.boolean().optional(),
  result: AttackMissionResultSchema.optional(),
})

const DefenseConfigSchema = z.object({
  sitoneIds: nullableArray(z.string()),
  cap: z.number(),
  lockedUntil: z.string().optional(),
})

export const CaptiveStatusSchema = z.enum([
  "captured",
  "captive_cooldown",
  "convert_pending",
])

const CaptiveSchema = z.object({
  id: z.string(),
  sitoneId: z.string(),
  originalTeamId: z.string(),
  cooldownUntil: z.string().optional(),
  status: CaptiveStatusSchema,
})

const CaptivesResponseSchema = z.object({
  captives: nullableArray(CaptiveSchema),
})

export const BossStatusSchema = z.enum([
  "closed",
  "open",
  "under_attack",
  "finished",
])

const ProtectedSitoneSchema = z.object({
  id: z.string(),
  sitoneId: z.string(),
  originalTeamId: z.string().optional(),
  originalPlayerId: z.string().optional(),
})

const YansanConvertProcessSchema = z.object({
  id: z.string(),
  captiveId: z.string().optional(),
  sitoneId: z.string().optional(),
  status: z.string().optional(),
})

const YansanStatusResponseSchema = z.object({
  bossStatus: BossStatusSchema,
  servicesAvailable: z.boolean(),
  protectedSitones: nullableArray(ProtectedSitoneSchema),
  teamRedeemCount: z.number(),
  convertProcesses: nullableArray(YansanConvertProcessSchema),
})

const YansanActionResultSchema = z.object({
  id: z.string().optional(),
  success: z.boolean().optional(),
  status: z.string().optional(),
  message: z.string().optional(),
})

const BossStatusClientResponseSchema = z.object({
  bossStatus: BossStatusSchema,
  participatingTeamIds: nullableArray(z.string()),
  requiredTeamCount: z.number().optional(),
  myTeamRegistered: z.boolean().optional(),
})

const BossStatusServerResponseSchema = z.object({
  status: BossStatusSchema,
  registeredTeams: nullableArray(z.string()),
  requiredTeams: z.number().optional(),
  myTeamRegistered: z.boolean().optional(),
})

const BossStatusResponseSchema = z
  .union([BossStatusClientResponseSchema, BossStatusServerResponseSchema])
  .transform((response) => {
    if ("bossStatus" in response) return response

    return {
      bossStatus: response.status,
      participatingTeamIds: response.registeredTeams,
      requiredTeamCount: response.requiredTeams,
      myTeamRegistered: response.myTeamRegistered,
    }
  })

const RescueSchema = z.object({
  id: z.string(),
  sitoneId: z.string(),
  status: z.string(),
  estimatedCost: EstimatedCostSchema.optional(),
  riskLevel: RiskLevelSchema.optional(),
})

export type TerritoryTier = z.infer<typeof TerritoryTierSchema>
export type RiskLevel = z.infer<typeof RiskLevelSchema>
export type TerritoryStandingTeam = z.infer<typeof TerritoryStandingTeamSchema>
export type TerritoryStandingsResponse = z.infer<
  typeof TerritoryStandingsResponseSchema
>
export type AttackMissionStatus = z.infer<typeof AttackMissionStatusSchema>
export type AttackMission = z.infer<typeof AttackMissionSchema>
export type DefenseConfig = z.infer<typeof DefenseConfigSchema>
export type CaptiveStatus = z.infer<typeof CaptiveStatusSchema>
export type Captive = z.infer<typeof CaptiveSchema>
export type BossStatus = z.infer<typeof BossStatusSchema>
export type ProtectedSitone = z.infer<typeof ProtectedSitoneSchema>
export type YansanStatusResponse = z.infer<typeof YansanStatusResponseSchema>
export type BossStatusResponse = z.infer<typeof BossStatusResponseSchema>
export type Rescue = z.infer<typeof RescueSchema>

export const territoryApi = {
  async standings() {
    const json = await apiClient.get("/api/territory/standings")
    return TerritoryStandingsResponseSchema.parse(json)
  },

  async createAttack(input: { defenderTeamId: string; sitoneIds: string[] }) {
    const json = await apiClient.post("/api/territory/attacks", {
      json: input,
    })
    return AttackMissionSchema.parse(json)
  },

  async attack(missionID: string) {
    const json = await apiClient.get(
      `/api/territory/attacks/${encodeURIComponent(missionID)}`,
    )
    return AttackMissionSchema.parse(json)
  },

  async voteAttack(missionID: string, agree: boolean) {
    const json = await apiClient.post(
      `/api/territory/attacks/${encodeURIComponent(missionID)}/votes`,
      { json: { agree } },
    )
    return AttackMissionSchema.parse(json)
  },

  async cancelAttack(missionID: string) {
    const json = await apiClient.post(
      `/api/territory/attacks/${encodeURIComponent(missionID)}/cancel`,
    )
    return AttackMissionSchema.parse(json)
  },

  async defense() {
    const json = await apiClient.get("/api/territory/defense")
    return DefenseConfigSchema.parse(json)
  },

  async updateDefense(sitoneIds: string[]) {
    const json = await apiClient.put("/api/territory/defense", {
      json: { sitoneIds },
    })
    return DefenseConfigSchema.parse(json)
  },

  async captives() {
    const json = await apiClient.get("/api/territory/captives")
    return CaptivesResponseSchema.parse(json).captives
  },

  async convertCaptive(captiveID: string) {
    const json = await apiClient.post(
      `/api/territory/captives/${encodeURIComponent(captiveID)}/convert`,
    )
    return YansanActionResultSchema.parse(json)
  },

  async yansanStatus() {
    const json = await apiClient.get("/api/yansan/status")
    return YansanStatusResponseSchema.parse(json)
  },

  async yansanRedeem(sitoneID: string) {
    const json = await apiClient.post("/api/yansan/redeem", {
      json: { sitoneId: sitoneID },
    })
    return YansanActionResultSchema.parse(json)
  },

  async yansanConvert(captiveID: string) {
    const json = await apiClient.post("/api/yansan/convert", {
      json: { captiveId: captiveID },
    })
    return YansanActionResultSchema.parse(json)
  },

  async bossStatus() {
    const json = await apiClient.get("/api/yansan/boss/status")
    return BossStatusResponseSchema.parse(json)
  },

  async attackBoss() {
    const json = await apiClient.post("/api/yansan/boss/attack")
    return BossStatusResponseSchema.parse(json)
  },

  async createRescue(sitoneID: string) {
    const json = await apiClient.post("/api/territory/rescues", {
      json: { sitoneId: sitoneID },
    })
    return RescueSchema.parse(json)
  },

  async rescue(rescueID: string) {
    const json = await apiClient.get(
      `/api/territory/rescues/${encodeURIComponent(rescueID)}`,
    )
    return RescueSchema.parse(json)
  },
}
