export type AchievementAsset = {
  iconPath: string
  notificationImagePath: string
  badge: string
  detail: string
  rewardLine: string
}

const achievementAssetsByKey: Record<string, AchievementAsset> = {
  codex_tier_01: {
    iconPath: "/game-icons/achievements/achievement-codex-tier-01.png",
    notificationImagePath:
      "/game-icons/alerts/achievement-codex-tier-01-alert.png",
    badge: "圖鑑階級 1",
    detail: "小石把第一批收藏排進展示盤，圖鑑開始轉動了。",
    rewardLine: "獲得 500 OP",
  },
  codex_tier_02: {
    iconPath: "/game-icons/achievements/achievement-codex-tier-02.png",
    notificationImagePath:
      "/game-icons/alerts/achievement-codex-tier-02-alert.png",
    badge: "圖鑑階級 2",
    detail: "小石確認每個空格都有位置，下一批收藏已經在路上。",
    rewardLine: "獲得 550 OP",
  },
  codex_tier_03: {
    iconPath: "/game-icons/achievements/achievement-codex-tier-03.png",
    notificationImagePath:
      "/game-icons/alerts/achievement-codex-tier-03-alert.png",
    badge: "圖鑑階級 3",
    detail: "小石一次整理三排線索，收藏、回憶和開源力一起入袋。",
    rewardLine: "獲得 600 OP",
  },
  codex_tier_04: {
    iconPath: "/game-icons/achievements/achievement-codex-tier-04.png",
    notificationImagePath:
      "/game-icons/alerts/achievement-codex-tier-04-alert.png",
    badge: "圖鑑階級 4",
    detail: "小石替圖鑑裝上更新貼紙，收藏進度又往前了一版。",
    rewardLine: "獲得 650 OP",
  },
  codex_tier_05: {
    iconPath: "/game-icons/achievements/achievement-codex-tier-05.png",
    notificationImagePath:
      "/game-icons/alerts/achievement-codex-tier-05-alert.png",
    badge: "圖鑑階級 5",
    detail: "小石把半滿展示櫃擦亮，接下來每一步都更有把握。",
    rewardLine: "獲得 700 OP",
  },
  codex_tier_06: {
    iconPath: "/game-icons/achievements/achievement-codex-tier-06.png",
    notificationImagePath:
      "/game-icons/alerts/achievement-codex-tier-06-alert.png",
    badge: "圖鑑階級 6",
    detail: "小石立起穩穩的收藏架，三十種小石排成可靠的基座。",
    rewardLine: "獲得 750 OP",
  },
  codex_tier_07: {
    iconPath: "/game-icons/achievements/achievement-codex-tier-07.png",
    notificationImagePath:
      "/game-icons/alerts/achievement-codex-tier-07-alert.png",
    badge: "圖鑑階級 7",
    detail: "小石一格一格補完空位，慢慢收集也能留下清楚痕跡。",
    rewardLine: "獲得 800 OP",
  },
  codex_tier_08: {
    iconPath: "/game-icons/achievements/achievement-codex-tier-08.png",
    notificationImagePath:
      "/game-icons/alerts/achievement-codex-tier-08-alert.png",
    badge: "圖鑑階級 8",
    detail: "小石打開高階收藏艙，光從圖鑑縫隙裡冒了出來。",
    rewardLine: "獲得 850 OP",
  },
  codex_tier_09: {
    iconPath: "/game-icons/achievements/achievement-codex-tier-09.png",
    notificationImagePath:
      "/game-icons/alerts/achievement-codex-tier-09-alert.png",
    badge: "圖鑑階級 9",
    detail: "小石帶回遠方的收藏樣本，圖鑑多了一整段旅程。",
    rewardLine: "獲得 900 OP",
  },
  codex_tier_10: {
    iconPath: "/game-icons/achievements/achievement-codex-tier-10.png",
    notificationImagePath:
      "/game-icons/alerts/achievement-codex-tier-10-alert.png",
    badge: "圖鑑階級 10",
    detail: "小石點亮五色收藏環，離完整圖鑑只剩最後一哩。",
    rewardLine: "獲得 950 OP",
  },
  codex_complete: {
    iconPath: "/game-icons/achievements/achievement-codex-complete.png",
    notificationImagePath:
      "/game-icons/alerts/achievement-codex-complete-alert.png",
    badge: "完整圖鑑",
    detail: "小石把最後一格放回原位，整本圖鑑終於亮成星圖。",
    rewardLine: "獲得 1200 OP",
  },
}

const fallbackAchievementAsset = achievementAssetsByKey.codex_tier_01

export function achievementAssetFor(key: string) {
  return achievementAssetsByKey[key] ?? fallbackAchievementAsset
}
