import { z } from "zod"

import { apiClient } from "./client"

const nullableArray = <T extends z.ZodType>(schema: T) =>
  z
    .array(schema)
    .nullish()
    .transform((value) => value ?? [])

const TeamSchema = z.object({
  teamId: z.string(),
  name: z.string(),
})

const TeamMemberSchema = z.object({
  playerId: z.string(),
  nickname: z.string(),
  role: z.string().optional(),
})

const PlayerStatusSchema = z.object({
  playerId: z.string(),
  nickname: z.string(),
  team: TeamSchema.optional(),
  teamMembers: nullableArray(TeamMemberSchema),
  openPower: z.number(),
  role: z.string().optional(),
})

const HomeActionSchema = z.object({
  id: z.string(),
  label: z.string(),
  enabled: z.boolean(),
})

const HomeTeamRankSchema = z.object({
  rank: z.number(),
  teamId: z.string(),
  name: z.string(),
  sitoneCount: z.number(),
  openPower: z.number(),
  gapToPrevious: z.number(),
})

const HomeResponseSchema = z.object({
  player: PlayerStatusSchema,
  summary: z.object({
    openPower: z.number(),
    sitoneCount: z.number(),
    itemCount: z.number(),
  }),
  teamRank: HomeTeamRankSchema.optional(),
  actions: nullableArray(HomeActionSchema),
})

const AuthLoginResponseSchema = z.object({
  player: PlayerStatusSchema,
})

const QRCodeResponseSchema = z.object({
  qrcodeToken: z.string(),
})

const QRResolveResponseSchema = z.object({
  player: PlayerStatusSchema,
})

const SitoneLoadoutResponseSchema = z.object({
  sitoneIds: nullableArray(z.string()),
})

const SitoneSchema = z.object({
  id: z.string(),
  name: z.string(),
  type: z.string(),
  rarity: z.string(),
  style: z.string(),
  description: z.string(),
  iconPath: z.string().optional(),
  abilityName: z.string(),
  abilityKind: z.enum([
    "material_drop_rate",
    "answer_score_bonus",
    "open_power_bonus",
    "eliminate_wrong_choice",
  ]),
  abilityValue: z.number(),
  abilityCount: z.number(),
  abilityDescription: z.string(),
})

const ItemSchema = z.object({
  id: z.string(),
  name: z.string(),
  type: z.string(),
  rarity: z.string(),
  description: z.string(),
  iconPath: z.string().optional(),
  source: z.string().optional(),
})

const CatalogSitonesResponseSchema = z.object({
  sitones: nullableArray(SitoneSchema),
})

const CatalogItemsResponseSchema = z.object({
  items: nullableArray(ItemSchema),
})

const PlayerSitoneSchema = z.object({
  id: z.string(),
  sitoneId: z.string(),
  quantity: z.number(),
  sitone: SitoneSchema,
})

const PlayerItemSchema = z.object({
  id: z.string(),
  itemId: z.string(),
  quantity: z.number(),
  item: ItemSchema,
})

const PlayerSitonesResponseSchema = z.object({
  sitones: nullableArray(PlayerSitoneSchema),
})

const PlayerItemsResponseSchema = z.object({
  items: nullableArray(PlayerItemSchema),
})

const ShopItemSchema = z.object({
  id: z.string(),
  name: z.string(),
  type: z.string(),
  rarity: z.string(),
  description: z.string(),
  iconPath: z.string().optional(),
  source: z.string().optional(),
  priceOpenPower: z.number(),
  locked: z.boolean().default(false),
  redeemed: z.boolean(),
})

const ShopItemsResponseSchema = z.object({
  items: nullableArray(ShopItemSchema),
})

const ShopItemDetailResponseSchema = z.object({
  item: ShopItemSchema,
})

const PurchaseResponseSchema = z.object({
  purchaseId: z.string(),
  itemId: z.string(),
  quantity: z.number(),
  priceOpenPower: z.number(),
  openPower: z.number(),
  item: ShopItemSchema,
})

const FusionComponentSchema = z.object({
  kind: z.enum(["item", "sitone"]),
  id: z.string(),
  name: z.string(),
  type: z.string().optional(),
  rarity: z.string().optional(),
  iconPath: z.string().optional(),
  source: z.string().optional(),
  abilityName: z.string().optional(),
  abilityKind: z.string().optional(),
  abilityValue: z.number().optional(),
  abilityCount: z.number().optional(),
  abilityDescription: z.string().optional(),
  quantity: z.number(),
})

const FusionRecipeSchema = z.object({
  id: z.string(),
  branchId: z.string().optional(),
  type: z.string().optional(),
  stageFrom: z.number().optional(),
  stageTo: z.number().optional(),
  name: z.string(),
  description: z.string(),
  story: z.string().optional(),
  reviewTitle: z.string().optional(),
  reviewUrl: z.string().optional(),
  enabled: z.boolean(),
  available: z.boolean(),
  inputs: nullableArray(FusionComponentSchema),
  outputs: nullableArray(FusionComponentSchema),
})

const FusionRecipesResponseSchema = z.object({
  recipes: nullableArray(FusionRecipeSchema),
})

const FusionCreateResponseSchema = z.object({
  fusionId: z.string(),
  recipe: FusionRecipeSchema,
})

const LeaderboardEntrySchema = z.object({
  rank: z.number(),
  id: z.string(),
  name: z.string(),
  teamId: z.string().optional(),
  teamName: z.string().optional(),
  sitoneCount: z.number(),
  openPower: z.number(),
  current: z.boolean(),
})

const LeaderboardResponseSchema = z.object({
  scope: z.enum(["teams", "players"]),
  entries: nullableArray(LeaderboardEntrySchema),
  currentEntry: LeaderboardEntrySchema.optional(),
  gapToPrevious: z.number(),
})

const LeaderboardTeamPlayerSchema = z.object({
  playerId: z.string(),
  nickname: z.string(),
  sitoneCount: z.number(),
  itemCount: z.number(),
  openPower: z.number(),
  current: z.boolean(),
})

const LeaderboardTeamPlayersResponseSchema = z.object({
  team: TeamSchema,
  players: nullableArray(LeaderboardTeamPlayerSchema),
})

const LeaderboardInventoryPlayerSchema = z.object({
  playerId: z.string(),
  nickname: z.string(),
  sitoneCount: z.number(),
  itemCount: z.number(),
  openPower: z.number(),
  current: z.boolean(),
})

const LeaderboardPlayerInventoryResponseSchema = z.object({
  player: LeaderboardInventoryPlayerSchema,
  team: TeamSchema,
  items: nullableArray(PlayerItemSchema),
  sitones: nullableArray(PlayerSitoneSchema),
})

const MatchPlayerSchema = z.object({
  playerId: z.string(),
  nickname: z.string(),
  kind: z.enum(["human", "computer"]).default("human"),
  ready: z.boolean(),
  answeredCurrentQuestion: z.boolean().optional(),
  sitoneIds: nullableArray(z.string()),
  score: z.number().optional(),
  maxScore: z.number().optional(),
  answerScoreBonusPercent: z.number().optional(),
  openPowerBonusPercent: z.number().optional(),
  materialDropBonusPercent: z.number().optional(),
  eliminateChancePercent: z.number().optional(),
  eliminateCount: z.number().optional(),
  eliminatedChoices: nullableArray(z.string()),
  eliminatedBy: nullableArray(z.string()),
  baseOpenPowerReward: z.number().optional(),
  openPowerReward: z.number().optional(),
  materialDrop: z
    .object({
      dropped: z.boolean(),
      itemId: z.string().optional(),
      itemName: z.string().optional(),
      sitoneId: z.string().optional(),
      sitoneName: z.string().optional(),
      quantity: z.number().optional(),
      dropRate: z.number(),
    })
    .optional(),
})

const MatchQuestionSchema = z.object({
  questionId: z.string(),
  prompt: z.string(),
  choiceA: z.string(),
  choiceB: z.string(),
  choiceC: z.string(),
  choiceD: z.string(),
})

const MatchAnswerResultSchema = z.object({
  playerId: z.string(),
  nickname: z.string(),
  choice: z.string().optional(),
  correct: z.boolean(),
  baseScore: z.number(),
  bonusScore: z.number(),
  score: z.number(),
  elapsedMillis: z.number(),
  answeredAt: z.string().optional(),
})

const MatchQuestionResultSchema = MatchQuestionSchema.extend({
  correctChoice: z.string(),
  explanation: z.string(),
  answers: nullableArray(MatchAnswerResultSchema),
})

export const MatchStateSchema = z.object({
  matchId: z.string(),
  code: z.string().optional(),
  mode: z.enum(["pvp", "computer"]).default("pvp"),
  status: z.enum(["waiting", "active", "completed"]),
  phase: z.enum(["answering", "revealing"]).optional(),
  hostPlayerId: z.string(),
  players: nullableArray(MatchPlayerSchema),
  currentQuestionIndex: z.number().optional(),
  questionCount: z.number().optional(),
  currentQuestion: MatchQuestionSchema.optional(),
  currentQuestionResult: MatchQuestionResultSchema.optional(),
  roundStartedAt: z.string().optional(),
  roundEndsAt: z.string().optional(),
  revealEndsAt: z.string().optional(),
  createdAt: z.string(),
  startedAt: z.string().optional(),
  completedAt: z.string().optional(),
  results: nullableArray(MatchQuestionResultSchema),
})

const CompletedMatchPlayerSchema = z.object({
  playerId: z.string(),
  nickname: z.string(),
  sitoneIds: nullableArray(z.string()),
  score: z.number(),
})

const CompletedMatchSchema = z.object({
  matchId: z.string(),
  status: z.literal("completed"),
  hostPlayerId: z.string(),
  players: nullableArray(CompletedMatchPlayerSchema),
  questionCount: z.number(),
  createdAt: z.string(),
  startedAt: z.string().optional(),
  completedAt: z.string().optional(),
})

const CompletedMatchesResponseSchema = z.object({
  matches: nullableArray(CompletedMatchSchema),
})

const AnswerAcceptedSchema = z.object({
  accepted: z.boolean(),
})

const ComputerBattleSettingsSchema = z.object({
  enabled: z.boolean(),
})

const StaffRewardKindSchema = z.enum(["item", "sitone"])

const StaffPlayerSchema = z.object({
  playerId: z.string(),
  nickname: z.string(),
  team: TeamSchema.optional(),
})

const StaffPlayersResponseSchema = z.object({
  players: nullableArray(StaffPlayerSchema),
})

const StaffRewardResponseSchema = z.object({
  rewardId: z.string(),
  player: z.object({
    playerId: z.string(),
    nickname: z.string(),
    team: TeamSchema,
  }),
  reward: z.object({
    kind: StaffRewardKindSchema,
    id: z.string(),
    name: z.string(),
    quantity: z.number(),
  }),
})

const AdminLoginResponseSchema = z.object({
  authenticated: z.boolean(),
})

const AdminSettingsSchema = z.object({
  computerBattlesEnabled: z.boolean(),
  computerEasyAccuracy: z.number(),
  computerNormalAccuracy: z.number(),
  computerHardAccuracy: z.number(),
})

const AdminDashboardTeamSummarySchema = z.object({
  teamId: z.string(),
  name: z.string(),
})

const AdminDashboardPlayerSchema = z.object({
  rank: z.number(),
  playerId: z.string(),
  nickname: z.string(),
  team: AdminDashboardTeamSummarySchema.optional(),
  role: z.string().optional(),
  sitoneCount: z.number(),
  itemCount: z.number(),
  openPower: z.number(),
  matchCount: z.number(),
  completedMatchCount: z.number(),
  answerCount: z.number(),
  correctAnswerCount: z.number(),
  answerAccuracy: z.number(),
  score: z.number(),
  lastActivityAt: z.string().optional(),
})

const AdminDashboardPlayerRankSchema = AdminDashboardPlayerSchema.omit({
  role: true,
})

const AdminDashboardTeamSchema = z.object({
  rank: z.number(),
  teamId: z.string(),
  name: z.string(),
  playerCount: z.number(),
  sitoneCount: z.number(),
  itemCount: z.number(),
  openPower: z.number(),
  averageSitones: z.number(),
  averageItems: z.number(),
  averageOpenPower: z.number(),
  topPlayer: AdminDashboardPlayerRankSchema.optional(),
})

const AdminDashboardInventoryEntrySchema = z.object({
  id: z.string(),
  name: z.string(),
  type: z.string().optional(),
  rarity: z.string().optional(),
  iconPath: z.string().optional(),
  source: z.string().optional(),
  quantity: z.number(),
  ownerCount: z.number(),
  catalogMissing: z.boolean(),
})

const AdminDashboardRecentMatchSchema = z.object({
  matchId: z.string(),
  code: z.string().optional(),
  mode: z.enum(["pvp", "computer"]),
  status: z.enum(["waiting", "active", "completed"]),
  playerCount: z.number(),
  winnerPlayerId: z.string().optional(),
  winnerNickname: z.string().optional(),
  topScore: z.number(),
  createdAt: z.string().optional(),
  startedAt: z.string().optional(),
  completedAt: z.string().optional(),
})

const AdminDashboardSchema = z.object({
  generatedAt: z.string(),
  summary: z.object({
    playerCount: z.number(),
    staffCount: z.number(),
    teamCount: z.number(),
    ungroupedPlayerCount: z.number(),
    totalSitones: z.number(),
    totalItems: z.number(),
    totalOpenPower: z.number(),
    totalMatches: z.number(),
    waitingMatches: z.number(),
    activeMatches: z.number(),
    completedMatches: z.number(),
    answerCount: z.number(),
    correctAnswerCount: z.number(),
    answerAccuracy: z.number(),
    shopPurchaseCount: z.number(),
    fusionCount: z.number(),
    staffRewardCount: z.number(),
    itemDropCount: z.number(),
    droppedItemCount: z.number(),
  }),
  topPlayers: z.object({
    bySitones: nullableArray(AdminDashboardPlayerRankSchema),
    byOpenPower: nullableArray(AdminDashboardPlayerRankSchema),
    byItems: nullableArray(AdminDashboardPlayerRankSchema),
    byScore: nullableArray(AdminDashboardPlayerRankSchema),
    byAccuracy: nullableArray(AdminDashboardPlayerRankSchema),
  }),
  teams: nullableArray(AdminDashboardTeamSchema),
  players: nullableArray(AdminDashboardPlayerSchema),
  inventory: z.object({
    sitones: nullableArray(AdminDashboardInventoryEntrySchema),
    items: nullableArray(AdminDashboardInventoryEntrySchema),
  }),
  matches: z.object({
    total: z.number(),
    waiting: z.number(),
    active: z.number(),
    completed: z.number(),
    pvp: z.number(),
    computer: z.number(),
    answerCount: z.number(),
    correctAnswerCount: z.number(),
    answerAccuracy: z.number(),
    averageScore: z.number(),
    averageElapsedMillis: z.number(),
    dropAttempts: z.number(),
    dropSuccesses: z.number(),
    dropRate: z.number(),
    recent: nullableArray(AdminDashboardRecentMatchSchema),
  }),
})

export type PlayerStatus = z.infer<typeof PlayerStatusSchema>
export type TeamMember = z.infer<typeof TeamMemberSchema>
export type HomeResponse = z.infer<typeof HomeResponseSchema>
export type SitoneLoadoutResponse = z.infer<typeof SitoneLoadoutResponseSchema>
export type Sitone = z.infer<typeof SitoneSchema>
export type Item = z.infer<typeof ItemSchema>
export type PlayerSitone = z.infer<typeof PlayerSitoneSchema>
export type PlayerItem = z.infer<typeof PlayerItemSchema>
export type ShopItem = z.infer<typeof ShopItemSchema>
export type FusionRecipe = z.infer<typeof FusionRecipeSchema>
export type LeaderboardScope = z.infer<
  typeof LeaderboardResponseSchema
>["scope"]
export type LeaderboardResponse = z.infer<typeof LeaderboardResponseSchema>
export type LeaderboardEntry = z.infer<typeof LeaderboardEntrySchema>
export type LeaderboardTeamPlayer = z.infer<typeof LeaderboardTeamPlayerSchema>
export type LeaderboardTeamPlayersResponse = z.infer<
  typeof LeaderboardTeamPlayersResponseSchema
>
export type LeaderboardPlayerInventoryResponse = z.infer<
  typeof LeaderboardPlayerInventoryResponseSchema
>
export type MatchState = z.infer<typeof MatchStateSchema>
export type MatchPlayer = z.infer<typeof MatchPlayerSchema>
export type MatchQuestion = z.infer<typeof MatchQuestionSchema>
export type MatchQuestionResult = z.infer<typeof MatchQuestionResultSchema>
export type CompletedMatch = z.infer<typeof CompletedMatchSchema>
export type MatchChoice = "A" | "B" | "C" | "D"
export type StaffRewardKind = z.infer<typeof StaffRewardKindSchema>
export type StaffPlayer = z.infer<typeof StaffPlayerSchema>
export type StaffRewardResponse = z.infer<typeof StaffRewardResponseSchema>
export type AdminSettings = z.infer<typeof AdminSettingsSchema>
export type AdminDashboard = z.infer<typeof AdminDashboardSchema>
export type AdminDashboardPlayer = z.infer<typeof AdminDashboardPlayerSchema>
export type AdminDashboardPlayerRank = z.infer<
  typeof AdminDashboardPlayerRankSchema
>
export type AdminDashboardTeam = z.infer<typeof AdminDashboardTeamSchema>
export type AdminDashboardInventoryEntry = z.infer<
  typeof AdminDashboardInventoryEntrySchema
>
export type AdminDashboardRecentMatch = z.infer<
  typeof AdminDashboardRecentMatchSchema
>

export const gameApi = {
  async login(token: string) {
    const json = await apiClient.post("/api/auth/login", {
      json: { token },
    })
    return AuthLoginResponseSchema.parse(json)
  },

  async logout() {
    await apiClient.post("/api/auth/logout")
  },

  async home() {
    const json = await apiClient.get("/api/me/home")
    return HomeResponseSchema.parse(json)
  },

  async status() {
    const json = await apiClient.get("/api/me/status")
    return PlayerStatusSchema.parse(json)
  },

  async qrcode() {
    const json = await apiClient.get("/api/me/qrcode")
    return QRCodeResponseSchema.parse(json)
  },

  async resolveQRCode(qrcodeToken: string) {
    const json = await apiClient.post("/api/qr/resolve", {
      json: { qrcodeToken },
    })
    return QRResolveResponseSchema.parse(json).player
  },

  async sitoneLoadout() {
    const json = await apiClient.get("/api/me/sitone-loadout")
    return SitoneLoadoutResponseSchema.parse(json)
  },

  async updateSitoneLoadout(sitoneIds: string[]) {
    const json = await apiClient.put("/api/me/sitone-loadout", {
      json: { sitoneIds },
    })
    return SitoneLoadoutResponseSchema.parse(json)
  },

  async catalogSitones() {
    const json = await apiClient.get("/api/catalog/sitones")
    return CatalogSitonesResponseSchema.parse(json).sitones
  },

  async catalogItems() {
    const json = await apiClient.get("/api/catalog/items")
    return CatalogItemsResponseSchema.parse(json).items
  },

  async playerSitones() {
    const json = await apiClient.get("/api/me/sitones")
    return PlayerSitonesResponseSchema.parse(json).sitones
  },

  async playerItems() {
    const json = await apiClient.get("/api/me/items")
    return PlayerItemsResponseSchema.parse(json).items
  },

  async completedMatches() {
    const json = await apiClient.get("/api/me/matches")
    return CompletedMatchesResponseSchema.parse(json).matches
  },

  async shopItems() {
    const json = await apiClient.get("/api/shop/items")
    return ShopItemsResponseSchema.parse(json).items
  },

  async shopItem(itemID: string) {
    const json = await apiClient.get(`/api/shop/items/${itemID}`)
    return ShopItemDetailResponseSchema.parse(json).item
  },

  async purchase(itemID: string) {
    const json = await apiClient.post("/api/shop/purchases", {
      json: { itemId: itemID },
    })
    return PurchaseResponseSchema.parse(json)
  },

  async fusionRecipes() {
    const json = await apiClient.get("/api/fusions/recipes")
    return FusionRecipesResponseSchema.parse(json).recipes
  },

  async createFusion(recipeID: string) {
    const json = await apiClient.post("/api/fusions", {
      json: { recipeId: recipeID },
    })
    return FusionCreateResponseSchema.parse(json)
  },

  async leaderboard(scope: LeaderboardScope) {
    const json = await apiClient.get("/api/leaderboards", {
      searchParams: { scope },
    })
    return LeaderboardResponseSchema.parse(json)
  },

  async leaderboardTeamPlayers(teamID: string) {
    const json = await apiClient.get(
      `/api/leaderboards/teams/${teamID}/players`,
    )
    return LeaderboardTeamPlayersResponseSchema.parse(json)
  },

  async leaderboardPlayerInventory(playerID: string) {
    const json = await apiClient.get(
      `/api/leaderboards/players/${playerID}/inventory`,
    )
    return LeaderboardPlayerInventoryResponseSchema.parse(json)
  },

  async createMatch() {
    const json = await apiClient.post("/api/matches")
    return MatchStateSchema.parse(json)
  },

  async computerBattleSettings() {
    const json = await apiClient.get("/api/matches/computer/settings")
    return ComputerBattleSettingsSchema.parse(json)
  },

  async createComputerMatch() {
    const json = await apiClient.post("/api/matches/computer")
    return MatchStateSchema.parse(json)
  },

  async joinMatch(code: string) {
    const json = await apiClient.post("/api/matches/join", {
      json: { code },
    })
    return MatchStateSchema.parse(json)
  },

  async openMatch() {
    const json = await apiClient.get("/api/matches/open")
    return MatchStateSchema.parse(json)
  },

  async getMatch(matchID: string) {
    const json = await apiClient.get(`/api/matches/${matchID}`)
    return MatchStateSchema.parse(json)
  },

  async readyMatch(matchID: string) {
    const json = await apiClient.post(`/api/matches/${matchID}/ready`)
    return MatchStateSchema.parse(json)
  },

  async updateMatchLoadout(matchID: string, sitoneIds: string[]) {
    const json = await apiClient.put(`/api/matches/${matchID}/loadout`, {
      json: { sitoneIds },
    })
    return MatchStateSchema.parse(json)
  },

  async answerMatch(matchID: string, questionID: string, choice: MatchChoice) {
    const json = await apiClient.post(`/api/matches/${matchID}/answers`, {
      json: { questionId: questionID, choice },
    })
    return AnswerAcceptedSchema.parse(json)
  },

  async createStaffReward(input: {
    playerId?: string
    qrcodeToken?: string
    kind: StaffRewardKind
    refId: string
    quantity: number
  }) {
    const json = await apiClient.post("/api/staff/rewards", {
      json: input,
    })
    return StaffRewardResponseSchema.parse(json)
  },

  async staffPlayers(query: string) {
    const json = await apiClient.get("/api/staff/players", {
      searchParams: { query },
    })
    return StaffPlayersResponseSchema.parse(json).players
  },

  async adminLogin(password: string) {
    const json = await apiClient.post("/api/admin/login", {
      json: { password },
    })
    return AdminLoginResponseSchema.parse(json)
  },

  async adminLogout() {
    await apiClient.post("/api/admin/logout")
  },

  async adminSettings() {
    const json = await apiClient.get("/api/admin/settings")
    return AdminSettingsSchema.parse(json)
  },

  async adminDashboard() {
    const json = await apiClient.get("/api/admin/dashboard")
    return AdminDashboardSchema.parse(json)
  },

  async updateAdminSettings(settings: AdminSettings) {
    const json = await apiClient.put("/api/admin/settings", {
      json: settings,
    })
    return AdminSettingsSchema.parse(json)
  },
}
