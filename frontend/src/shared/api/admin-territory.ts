import { z } from "zod"

import { apiClient } from "./client"

const nullableArray = <T extends z.ZodType>(schema: T) =>
  z
    .array(schema)
    .nullish()
    .transform((value) => value ?? [])

const jsonRecord = z
  .record(z.string(), z.unknown())
  .nullish()
  .transform((value) => value ?? undefined)

const AdminAttackMissionSchema = z.object({
  missionId: z.string(),
  attackerTeamId: z.string(),
  defenderTeamId: z.string(),
  initiatorPlayerId: z.string(),
  beneficiaryPlayerId: z.string().optional(),
  status: z.string(),
  selectedSitoneIds: nullableArray(z.string()),
  costOpenPower: z.number(),
  voteDeadline: z.string().optional(),
  startedAt: z.string().optional(),
  resolveAt: z.string().optional(),
  resolvedAt: z.string().optional(),
  attackerTier: z.number().optional(),
  defenderTier: z.number().optional(),
  randomSeedRef: z.string().optional(),
  resultSummary: jsonRecord,
  createdAt: z.string(),
  updatedAt: z.string(),
})

const AdminAttackMissionsResponseSchema = z.object({
  missions: nullableArray(AdminAttackMissionSchema),
})

const AdminCancelMissionResponseSchema = z.object({
  mission: AdminAttackMissionSchema,
  releasedInstances: z.number(),
})

const AdminSitoneInstanceSchema = z.object({
  instanceId: z.string(),
  playerId: z.string().optional(),
  teamId: z.string().optional(),
  originPlayerId: z.string().optional(),
  originTeamId: z.string().optional(),
  sitoneId: z.string(),
  status: z.string(),
  missionId: z.string().optional(),
  fatigue: z.number(),
  cooldownUntil: z.string().optional(),
  createdAt: z.string(),
  updatedAt: z.string(),
})

const AdminSitoneInstancesResponseSchema = z.object({
  instances: nullableArray(AdminSitoneInstanceSchema),
})

const AdminYansanProcessSchema = z.object({
  processId: z.string(),
  kind: z.string(),
  sitoneId: z.string(),
  instanceId: z.string().optional(),
  actorPlayerId: z.string(),
  beneficiaryPlayerId: z.string().optional(),
  teamId: z.string(),
  costOpenPower: z.number(),
  status: z.string(),
  startedAt: z.string(),
  resolveAt: z.string().optional(),
  resolvedAt: z.string().optional(),
  result: jsonRecord,
})

const AdminYansanProcessesResponseSchema = z.object({
  processes: nullableArray(AdminYansanProcessSchema),
})

const AdminCaptiveRecordSchema = z.object({
  captiveId: z.string(),
  sitoneId: z.string(),
  originalTeamId: z.string(),
  currentTeamId: z.string(),
  instanceId: z.string().optional(),
  status: z.string(),
  capturedAt: z.string(),
  cooldownUntil: z.string().optional(),
  convertedAt: z.string().optional(),
  convertProcesses: nullableArray(AdminYansanProcessSchema),
})

const AdminCaptiveRecordsResponseSchema = z.object({
  captives: nullableArray(AdminCaptiveRecordSchema),
})

const AdminEventLogSchema = z.object({
  eventId: z.string(),
  eventType: z.string(),
  actorPlayerId: z.string().optional(),
  targetPlayerId: z.string().optional(),
  teamId: z.string().optional(),
  relatedSitoneId: z.string().optional(),
  relatedMissionId: z.string().optional(),
  payload: jsonRecord,
  createdAt: z.string(),
})

const AdminEventLogsResponseSchema = z.object({
  events: nullableArray(AdminEventLogSchema),
})

export type AdminAttackMission = z.infer<typeof AdminAttackMissionSchema>
export type AdminSitoneInstance = z.infer<typeof AdminSitoneInstanceSchema>
export type AdminYansanProcess = z.infer<typeof AdminYansanProcessSchema>
export type AdminCaptiveRecord = z.infer<typeof AdminCaptiveRecordSchema>
export type AdminEventLog = z.infer<typeof AdminEventLogSchema>

function compactParams(params: Record<string, string | number | undefined>) {
  const entries = Object.entries(params).filter(
    ([, value]) => value !== undefined && value !== "",
  )
  return entries.length > 0 ? Object.fromEntries(entries) : undefined
}

export const adminTerritoryApi = {
  async missions(filters: { status?: string; teamId?: string; limit?: number } = {}) {
    const json = await apiClient.get("/api/admin/territory/missions", {
      searchParams: compactParams(filters),
    })
    return AdminAttackMissionsResponseSchema.parse(json).missions
  },

  async cancelMission(missionID: string) {
    const json = await apiClient.post(
      `/api/admin/territory/missions/${encodeURIComponent(missionID)}/cancel`,
    )
    return AdminCancelMissionResponseSchema.parse(json)
  },

  async instances(filters: { status?: string; teamId?: string; limit?: number } = {}) {
    const json = await apiClient.get("/api/admin/territory/instances", {
      searchParams: compactParams(filters),
    })
    return AdminSitoneInstancesResponseSchema.parse(json).instances
  },

  async captives(filters: { limit?: number } = {}) {
    const json = await apiClient.get("/api/admin/territory/captives", {
      searchParams: compactParams(filters),
    })
    return AdminCaptiveRecordsResponseSchema.parse(json).captives
  },

  async events(
    filters: {
      type?: string
      teamId?: string
      playerId?: string
      limit?: number
    } = {},
  ) {
    const json = await apiClient.get("/api/admin/territory/events", {
      searchParams: compactParams(filters),
    })
    return AdminEventLogsResponseSchema.parse(json).events
  },

  async yansanProcesses(
    filters: { status?: string; kind?: string; limit?: number } = {},
  ) {
    const json = await apiClient.get("/api/admin/yansan/processes", {
      searchParams: compactParams(filters),
    })
    return AdminYansanProcessesResponseSchema.parse(json).processes
  },
}
