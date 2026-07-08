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
  avatarUrl: z.string().optional(),
})

const TeamMemberSchema = z.object({
  playerId: z.string(),
  nickname: z.string(),
  avatarUrl: z.string().optional(),
  role: z.string().optional(),
})

const PlayerStatusSchema = z.object({
  playerId: z.string(),
  nickname: z.string(),
  team: TeamSchema.optional(),
  teamMembers: nullableArray(TeamMemberSchema),
  openPower: z.number(),
  avatarUrl: z.string().optional(),
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
  playerCount: z.number(),
  sitoneCount: z.number(),
  averageSitones: z.number(),
  openPower: z.number(),
  averageOpenPower: z.number(),
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

const UpdateAvatarResponseSchema = z.object({
  avatarUrl: z.string().optional(),
})

const UpdateNicknameResponseSchema = z.object({
  nickname: z.string(),
})

const UpdateTeamAvatarResponseSchema = z.object({
  team: TeamSchema,
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
  ownedQuantity: z.number().optional(),
  missingQuantity: z.number().optional(),
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

const FusionFillMaterialResultSchema = z.object({
  kind: z.enum(["item", "sitone"]),
  id: z.string(),
  name: z.string(),
  quantity: z.number(),
  action: z.enum(["purchase", "fusion"]).optional(),
  source: z.string().optional(),
})

const FusionFillMaterialFailureSchema = FusionFillMaterialResultSchema.extend({
  reason: z.string(),
})

const FusionFillMissingMaterialsResponseSchema = z.object({
  recipe: FusionRecipeSchema,
  filledMaterials: nullableArray(FusionFillMaterialResultSchema),
  failedMaterials: nullableArray(FusionFillMaterialFailureSchema),
  priceOpenPowerSpent: z.number(),
})

const LeaderboardEntrySchema = z.object({
  rank: z.number(),
  id: z.string(),
  name: z.string(),
  teamId: z.string().optional(),
  teamName: z.string().optional(),
  avatarUrl: z.string().optional(),
  playerCount: z.number().optional(),
  sitoneCount: z.number(),
  averageSitones: z.number().optional(),
  openPower: z.number(),
  averageOpenPower: z.number().optional(),
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
  avatarUrl: z.string().optional(),
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
  avatarUrl: z.string().optional(),
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
  avatarUrl: z.string().optional(),
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
  avatarUrl: z.string().optional(),
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
  avatarUrl: z.string().optional(),
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
  page: z.number(),
  perPage: z.number(),
  total: z.number(),
  totalPages: z.number(),
})

const AnswerAcceptedSchema = z.object({
  accepted: z.boolean(),
})

const ComputerBattleSettingsSchema = z.object({
  enabled: z.boolean(),
})

const MatchPairingResponseSchema = z.object({
  match: MatchStateSchema,
  token: z.string(),
  expiresAt: z.string(),
})

const StaffRewardKindSchema = z.enum(["item", "sitone", "open_power"])

const StaffPlayerSchema = z.object({
  playerId: z.string(),
  nickname: z.string(),
  team: TeamSchema.optional(),
  avatarUrl: z.string().optional(),
})

const StaffTeamSchema = z.object({
  teamId: z.string(),
  name: z.string(),
  avatarUrl: z.string().optional(),
  memberCount: z.number(),
})

const StaffPlayersResponseSchema = z.object({
  players: nullableArray(StaffPlayerSchema),
})

const StaffTeamsResponseSchema = z.object({
  teams: nullableArray(StaffTeamSchema),
})

const StaffRewardResponseSchema = z.object({
  rewardIds: nullableArray(z.string()),
  grantedCount: z.number(),
  allPlayers: z.boolean().optional(),
  player: z
    .object({
      playerId: z.string(),
      nickname: z.string(),
      avatarUrl: z.string().optional(),
      team: TeamSchema.optional(),
    })
    .optional(),
  team: TeamSchema.optional(),
  reward: z.object({
    kind: StaffRewardKindSchema,
    id: z.string().optional(),
    name: z.string(),
    quantity: z.number().optional(),
    amount: z.number().optional(),
  }),
})

const StaffRewardTokenResponseSchema = z.object({
  token: z.string(),
  expiresAt: z.string(),
  reward: z.object({
    kind: StaffRewardKindSchema,
    id: z.string().optional(),
    name: z.string(),
    quantity: z.number().optional(),
    amount: z.number().optional(),
  }),
})

const StaffRewardTokenClaimResponseSchema = z.object({
  rewardId: z.string(),
  reward: z.object({
    kind: StaffRewardKindSchema,
    id: z.string().optional(),
    name: z.string(),
    quantity: z.number().optional(),
    amount: z.number().optional(),
  }),
  staff: z.object({
    playerId: z.string(),
    nickname: z.string(),
  }),
})

const CommunityStandRewardSchema = z.object({
  kind: StaffRewardKindSchema,
  refId: z.string().optional(),
  name: z.string(),
  quantity: z.number().optional(),
  amount: z.number().optional(),
  iconPath: z.string().optional(),
})

const CommunityStandSchema = z.object({
  standId: z.string(),
  name: z.string(),
  description: z.string(),
  logoUrl: z.string().optional(),
  websiteUrl: z.string().optional(),
  reward: CommunityStandRewardSchema,
})

const CommunityStandDetailResponseSchema = z.object({
  stand: CommunityStandSchema,
  claimed: z.boolean(),
})

const CommunityStandDisplayResponseSchema = z.object({
  stand: CommunityStandSchema,
  visitCount: z.number(),
  claimCount: z.number(),
  qrToken: z.string(),
  qrTokenExpiresAt: z.string(),
})

const CommunityStandClaimResponseSchema = z.object({
  claimId: z.string(),
  stand: CommunityStandSchema,
  reward: CommunityStandRewardSchema,
  claimed: z.boolean(),
})

const AdminLoginResponseSchema = z.object({
  authenticated: z.boolean(),
})

const AdminSettingsSchema = z.object({
  computerBattlesEnabled: z.boolean(),
  sameTeamBattlesEnabled: z.boolean(),
  computerEasyAccuracy: z.number(),
  computerNormalAccuracy: z.number(),
  computerHardAccuracy: z.number(),
  classTimeBattleLockEnabled: z.boolean().default(false),
  classTimeBattleLockStart: z.string().default("09:00"),
  classTimeBattleLockEnd: z.string().default("17:00"),
  battleOpeningOverride: z
    .enum(["schedule", "force_open", "force_closed"])
    .default("schedule"),
  battleOpeningLocked: z.boolean().default(false),
})

const AdminCommunityStandSchema = CommunityStandSchema.extend({
  enabled: z.boolean(),
  visitCount: z.number(),
  claimCount: z.number(),
  createdAt: z.string(),
  updatedAt: z.string(),
})

const AdminCommunityStandsResponseSchema = z.object({
  stands: nullableArray(AdminCommunityStandSchema),
})

const AdminCommunityStandClaimSchema = z.object({
  claimId: z.string(),
  standId: z.string(),
  standName: z.string().optional(),
  playerId: z.string(),
  playerNickname: z.string().optional(),
  reward: CommunityStandRewardSchema,
  createdAt: z.string(),
})

const AdminCommunityStandClaimsResponseSchema = z.object({
  claims: nullableArray(AdminCommunityStandClaimSchema),
})

const AdminCommunityStandRewardInputSchema = z.object({
  kind: StaffRewardKindSchema,
  refId: z.string().optional(),
  quantity: z.number().optional(),
  amount: z.number().optional(),
})

const AdminCommunityStandUpdateInputSchema = z.object({
  name: z.string(),
  description: z.string(),
  logoUrl: z.string().optional(),
  websiteUrl: z.string().optional(),
  enabled: z.boolean(),
  reward: AdminCommunityStandRewardInputSchema,
})

const AdminCommunityStandCreateInputSchema =
  AdminCommunityStandUpdateInputSchema

const AdminDashboardTeamSummarySchema = z.object({
  teamId: z.string(),
  name: z.string(),
  avatarUrl: z.string().optional(),
})

const AdminDashboardPlayerSchema = z.object({
  rank: z.number(),
  playerId: z.string(),
  nickname: z.string(),
  avatarUrl: z.string().optional(),
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
  avatarUrl: z.string().optional(),
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
  ownerPercent: z.number(),
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

const AdminDashboardHistoryPointSchema = z.object({
  timestamp: z.string(),
  sitoneCount: z.number(),
  openPower: z.number(),
  sitoneDelta: z.number(),
  openPowerDelta: z.number(),
})

const AdminDashboardHistorySchema = z.object({
  generatedAt: z.string(),
  bucket: z.enum(["hour", "day"]),
  sitoneBaseline: z.number(),
  openPowerStart: z.number(),
  points: nullableArray(AdminDashboardHistoryPointSchema),
})

const GiftHistoryEntrySchema = z.object({
  rewardId: z.string(),
  kind: StaffRewardKindSchema,
  refId: z.string().optional(),
  name: z.string(),
  iconPath: z.string().optional(),
  quantity: z.number().optional(),
  amount: z.number().optional(),
  staffPlayerId: z.string(),
  staffNickname: z.string().optional(),
  recipientPlayerId: z.string(),
  recipientNickname: z.string().optional(),
  createdAt: z.string(),
})

const GiftHistoryResponseSchema = z.object({
  entries: nullableArray(GiftHistoryEntrySchema),
})

const AdminStudentChangeEntrySchema = z.object({
  changeId: z.string(),
  source: z.string(),
  sourceLabel: z.string(),
  playerId: z.string(),
  playerNickname: z.string().optional(),
  teamId: z.string().optional(),
  kind: StaffRewardKindSchema,
  refId: z.string().optional(),
  name: z.string(),
  iconPath: z.string().optional(),
  delta: z.number(),
  note: z.string().optional(),
  createdAt: z.string(),
})

const AdminStudentChangesResponseSchema = z.object({
  entries: nullableArray(AdminStudentChangeEntrySchema),
})

const AdminPlayerBalanceResponseSchema = z.object({
  trimId: z.string().optional(),
  rank: z.number(),
  playerId: z.string(),
  nickname: z.string(),
  teamId: z.string().optional(),
  sitoneBefore: z.number(),
  sitoneTrimmed: z.number(),
  sitoneAfter: z.number(),
  openPowerBefore: z.number(),
  openPowerTrimmed: z.number(),
  openPowerAfter: z.number(),
})

export type PlayerStatus = z.infer<typeof PlayerStatusSchema>
export type Team = z.infer<typeof TeamSchema>
export type TeamMember = z.infer<typeof TeamMemberSchema>
export type HomeResponse = z.infer<typeof HomeResponseSchema>
export type SitoneLoadoutResponse = z.infer<typeof SitoneLoadoutResponseSchema>
export type Sitone = z.infer<typeof SitoneSchema>
export type Item = z.infer<typeof ItemSchema>
export type PlayerSitone = z.infer<typeof PlayerSitoneSchema>
export type PlayerItem = z.infer<typeof PlayerItemSchema>
export type ShopItem = z.infer<typeof ShopItemSchema>
export type FusionRecipe = z.infer<typeof FusionRecipeSchema>
export type FusionFillMissingMaterialsResponse = z.infer<
  typeof FusionFillMissingMaterialsResponseSchema
>
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
export type MatchPairing = z.infer<typeof MatchPairingResponseSchema>
export type MatchPlayer = z.infer<typeof MatchPlayerSchema>
export type MatchQuestion = z.infer<typeof MatchQuestionSchema>
export type MatchQuestionResult = z.infer<typeof MatchQuestionResultSchema>
export type CompletedMatch = z.infer<typeof CompletedMatchSchema>
export type MatchChoice = "A" | "B" | "C" | "D"
export type StaffRewardKind = z.infer<typeof StaffRewardKindSchema>
export type StaffPlayer = z.infer<typeof StaffPlayerSchema>
export type StaffTeam = z.infer<typeof StaffTeamSchema>
export type StaffRewardResponse = z.infer<typeof StaffRewardResponseSchema>
export type StaffRewardTokenResponse = z.infer<
  typeof StaffRewardTokenResponseSchema
>
export type StaffRewardTokenClaimResponse = z.infer<
  typeof StaffRewardTokenClaimResponseSchema
>
export type CommunityStandReward = z.infer<typeof CommunityStandRewardSchema>
export type CommunityStand = z.infer<typeof CommunityStandSchema>
export type CommunityStandDetailResponse = z.infer<
  typeof CommunityStandDetailResponseSchema
>
export type CommunityStandDisplayResponse = z.infer<
  typeof CommunityStandDisplayResponseSchema
>
export type CommunityStandClaimResponse = z.infer<
  typeof CommunityStandClaimResponseSchema
>
export type AdminSettings = z.infer<typeof AdminSettingsSchema>
export type AdminCommunityStand = z.infer<typeof AdminCommunityStandSchema>
export type AdminCommunityStandClaim = z.infer<
  typeof AdminCommunityStandClaimSchema
>
export type AdminCommunityStandUpdateInput = z.input<
  typeof AdminCommunityStandUpdateInputSchema
>
export type AdminCommunityStandCreateInput = z.input<
  typeof AdminCommunityStandCreateInputSchema
>
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
export type AdminDashboardHistory = z.infer<typeof AdminDashboardHistorySchema>
export type AdminDashboardHistoryPoint = z.infer<
  typeof AdminDashboardHistoryPointSchema
>
export type GiftHistoryEntry = z.infer<typeof GiftHistoryEntrySchema>
export type AdminStudentChangeEntry = z.infer<
  typeof AdminStudentChangeEntrySchema
>
export type AdminPlayerBalanceResponse = z.infer<
  typeof AdminPlayerBalanceResponseSchema
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

  async communityStand(standID: string) {
    const json = await apiClient.get(
      `/api/community/${encodeURIComponent(standID)}`,
    )
    return CommunityStandDetailResponseSchema.parse(json)
  },

  async communityStandDisplay(standID: string) {
    const json = await apiClient.get(
      `/api/community/${encodeURIComponent(standID)}/display`,
    )
    return CommunityStandDisplayResponseSchema.parse(json)
  },

  async claimCommunityStand(standID: string) {
    const json = await apiClient.post(
      `/api/community/${encodeURIComponent(standID)}/claim`,
    )
    return CommunityStandClaimResponseSchema.parse(json)
  },
  async communityStandByQRToken(qrToken: string) {
    const json = await apiClient.get(
      `/api/community/scans/${encodeURIComponent(qrToken)}`,
    )
    return CommunityStandDetailResponseSchema.parse(json)
  },

  async claimCommunityStandByQRToken(qrToken: string) {
    const json = await apiClient.post(
      `/api/community/scans/${encodeURIComponent(qrToken)}/claim`,
    )
    return CommunityStandClaimResponseSchema.parse(json)
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

  async updateAvatar(sitoneId: string | null) {
    const json = await apiClient.put("/api/me/avatar", {
      json: { sitoneId },
    })
    return UpdateAvatarResponseSchema.parse(json)
  },

  async updateNickname(nickname: string) {
    const json = await apiClient.put("/api/me/nickname", {
      json: { nickname },
    })
    return UpdateNicknameResponseSchema.parse(json)
  },

  async updateTeamAvatar(sitoneId: string | null) {
    const json = await apiClient.put("/api/me/team/avatar", {
      json: { sitoneId },
    })
    return UpdateTeamAvatarResponseSchema.parse(json)
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

  async teamSitones() {
    const json = await apiClient.get("/api/me/team/sitones")
    return PlayerSitonesResponseSchema.parse(json).sitones
  },

  async playerItems() {
    const json = await apiClient.get("/api/me/items")
    return PlayerItemsResponseSchema.parse(json).items
  },

  async completedMatches(page = 1) {
    const json = await apiClient.get("/api/me/matches", {
      searchParams: { page },
    })
    return CompletedMatchesResponseSchema.parse(json)
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

  async fillMissingFusionMaterials(recipeID: string) {
    const json = await apiClient.post(
      `/api/fusions/${encodeURIComponent(recipeID)}/fill-missing-materials`,
    )
    return FusionFillMissingMaterialsResponseSchema.parse(json)
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

  async createMatchPairing() {
    const json = await apiClient.post("/api/matches/pairings")
    return MatchPairingResponseSchema.parse(json)
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

  async scanMatchPairingToken(token: string) {
    const json = await apiClient.post("/api/matches/pairings/scan", {
      json: { token },
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

  async createMatchPairingToken(matchID: string) {
    const json = await apiClient.post(
      `/api/matches/${encodeURIComponent(matchID)}/pairing-token`,
    )
    return MatchPairingResponseSchema.parse(json)
  },

  async readyMatch(matchID: string) {
    const json = await apiClient.post(`/api/matches/${matchID}/ready`)
    return MatchStateSchema.parse(json)
  },

  async leaveMatch(matchID: string) {
    await apiClient.post(`/api/matches/${matchID}/leave`)
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
    teamId?: string
    allPlayers?: boolean
    kind: StaffRewardKind
    refId?: string
    quantity?: number
    amount?: number
  }) {
    const json = await apiClient.post("/api/staff/rewards", {
      json: input,
    })
    return StaffRewardResponseSchema.parse(json)
  },

  async createStaffRewardToken(input: {
    kind: StaffRewardKind
    refId?: string
    quantity?: number
    amount?: number
  }) {
    const json = await apiClient.post("/api/staff/reward-tokens", {
      json: input,
    })
    return StaffRewardTokenResponseSchema.parse(json)
  },

  async claimStaffRewardToken(token: string) {
    const json = await apiClient.post(
      `/api/staff/reward-tokens/${encodeURIComponent(token)}/claim`,
    )
    return StaffRewardTokenClaimResponseSchema.parse(json)
  },

  async staffPlayers(query: string) {
    const json = await apiClient.get("/api/staff/players", {
      searchParams: { query },
    })
    return StaffPlayersResponseSchema.parse(json).players
  },

  async staffTeams(query?: string) {
    const json = await apiClient.get("/api/staff/teams", {
      searchParams: query ? { query } : undefined,
    })
    return StaffTeamsResponseSchema.parse(json).teams
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

  async adminHistory(bucket: "hour" | "day" = "hour") {
    const json = await apiClient.get("/api/admin/history", {
      searchParams: { bucket },
    })
    return AdminDashboardHistorySchema.parse(json)
  },

  async giftHistory() {
    const json = await apiClient.get("/api/me/gift-history")
    return GiftHistoryResponseSchema.parse(json).entries
  },

  async adminGiftHistory() {
    const json = await apiClient.get("/api/admin/gift-history")
    return GiftHistoryResponseSchema.parse(json).entries
  },

  async adminStudentChanges(limit = 300) {
    const json = await apiClient.get("/api/admin/student-changes", {
      searchParams: { limit },
    })
    return AdminStudentChangesResponseSchema.parse(json).entries
  },

  async adminCommunityStands() {
    const json = await apiClient.get("/api/admin/community-stands")
    return AdminCommunityStandsResponseSchema.parse(json).stands
  },

  async adminCommunityStandClaims(standID?: string) {
    const json = await apiClient.get("/api/admin/community-stand-claims", {
      searchParams: standID ? { standId: standID } : undefined,
    })
    return AdminCommunityStandClaimsResponseSchema.parse(json).claims
  },

  async updateAdminSettings(settings: AdminSettings) {
    const { battleOpeningLocked, ...input } = settings
    void battleOpeningLocked
    const json = await apiClient.put("/api/admin/settings", {
      json: input,
    })
    return AdminSettingsSchema.parse(json)
  },

  async createAdminCommunityStand(input: AdminCommunityStandCreateInput) {
    const json = await apiClient.post("/api/admin/community-stands", {
      json: AdminCommunityStandCreateInputSchema.parse(input),
    })
    return AdminCommunityStandSchema.parse(json)
  },

  async updateAdminCommunityStand(
    standID: string,
    input: AdminCommunityStandUpdateInput,
  ) {
    const json = await apiClient.put(
      `/api/admin/community-stands/${encodeURIComponent(standID)}`,
      {
        json: AdminCommunityStandUpdateInputSchema.parse(input),
      },
    )
    return AdminCommunityStandSchema.parse(json)
  },

  async deleteAdminCommunityStand(standID: string) {
    await apiClient.delete(
      `/api/admin/community-stands/${encodeURIComponent(standID)}`,
    )
  },

  async updateAdminTeam(
    teamID: string,
    input: { name: string; avatarUrl?: string },
  ) {
    const json = await apiClient.put(
      `/api/admin/teams/${encodeURIComponent(teamID)}`,
      {
        json: input,
      },
    )
    return TeamSchema.parse(json)
  },

  async updateAdminPlayerBalance(
    playerID: string,
    input: { sitoneCount: number; openPower: number; message?: string },
  ) {
    const json = await apiClient.put(
      `/api/admin/players/${encodeURIComponent(playerID)}/balance`,
      {
        json: input,
      },
    )
    return AdminPlayerBalanceResponseSchema.parse(json)
  },
}
