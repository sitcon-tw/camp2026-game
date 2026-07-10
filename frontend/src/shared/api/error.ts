import { z } from "zod"

const ProblemDetailsSchema = z.object({
  title: z.string().optional(),
  detail: z.string().optional(),
  status: z.number().optional(),
  code: z.string().optional(),
  type: z.string().optional(),
  instance: z.string().optional(),
})

export const battleOpeningLockedMessage = "禁止開局"

const detailMessageMap: Record<string, string> = {
  "answer failed": "送出答案失敗，請稍後再試。",
  "authentication required": "請先登入後再繼續。",
  "avatar update failed": "頭貼更新失敗，請稍後再試。",
  "avatar unavailable": "目前無法讀取頭貼資料，請稍後再試。",
  "battle opening is locked": battleOpeningLockedMessage,
  "choice has been eliminated": "這個選項已被小石效果排除，請選擇其他答案。",
  "community stand claim failed": "領取攤位獎勵失敗，請稍後再試。",
  "community stand not found": "找不到這個攤位。",
  "community stand reward already claimed": "這個攤位的獎勵已經領取過了。",
  "community stand unavailable": "目前無法讀取攤位資料，請稍後再試。",
  "computer battles are disabled": "目前未開放電腦對戰。",
  "content store is unavailable": "遊戲資料暫時無法讀取，請稍後再試。",
  "database is unavailable": "資料庫暫時無法連線，請稍後再試。",
  "event stream is unavailable": "即時更新暫時無法連線，請重新整理頁面。",
  "fill missing materials failed": "自動補齊材料失敗，請稍後再試。",
  "fusion failed": "合成失敗，請稍後再試。",
  "fusion inventory unavailable": "目前無法讀取合成材料，請稍後再試。",
  "fusion recipe is inconsistent": "這個合成配方資料有異常，請通知工作人員。",
  "fusion recipe not found": "找不到這個合成配方。",
  "home summary unavailable": "首頁資料暫時無法讀取，請稍後再試。",
  "insufficient fusion materials": "材料不足，無法合成。",
  "insufficient open power": "開源力不足，無法購買。",
  "invalid page parameter": "頁碼格式不正確。",
  "invalid request body": "送出的資料格式不正確，請重新操作一次。",
  "item inventory is inconsistent": "道具資料有異常，請通知工作人員。",
  "items unavailable": "目前無法讀取道具資料，請稍後再試。",
  "loadout update failed": "小石陣容更新失敗，請稍後再試。",
  "match creation failed": "建立對戰失敗，請稍後再試。",
  "match has already started": "這場對戰已經開始，無法離開。",
  "match is full": "這個房間已滿，無法加入。",
  "match is not active": "這場對戰目前不在答題中。",
  "match is not joinable": "這個房間目前不能加入。",
  "match is not open": "這場對戰已經不是開放狀態。",
  "match is not waiting for ready": "這場對戰目前不能準備。",
  "match join failed": "加入對戰失敗，請稍後再試。",
  "match loadout is locked": "對戰已鎖定小石陣容，不能再修改。",
  "match lookup failed": "目前無法讀取對戰資料，請稍後再試。",
  "match not found": "找不到這場對戰。",
  "match question is unavailable": "目前無法讀取題目資料，請通知工作人員。",
  "match start failed": "開始對戰失敗，請稍後再試。",
  "match state unavailable": "對戰狀態暫時無法更新，請稍後再試。",
  "match was updated; retry": "對戰狀態剛剛更新了，請再試一次。",
  "matches unavailable": "目前無法讀取對戰紀錄，請稍後再試。",
  "player already has an open match":
    "你已經有進行中的對戰，請先回到原本的對戰。",
  "player is already ready": "你已經準備好了。",
  "player pair match limit reached":
    "你和這位玩家的對戰次數已達上限，請找其他玩家對戰。",
  "qrcode is unavailable": "玩家 QR Code 暫時無法產生，請稍後再試。",
  "qr code scan cooldown active": "QRCode 掃描冷卻中，請稍後再試。",
  "qr code not found": "找不到這個 QR Code，請確認是否掃描正確。",
  "qr resolve failed": "確認 QR Code 失敗，請稍後再試。",
  "question already answered": "這題已經作答過了。",
  "question is not current": "這不是目前正在作答的題目。",
  "quiz questions are unavailable": "題庫暫時無法使用，請通知工作人員。",
  "recipe materials are already complete": "材料已經足夠，不需要自動補齊。",
  "room team token not found": "找不到這個宿舍 QR Code，請確認是否已過期或掃描正確。",
  "round is not accepting answers": "目前不是可作答時間。",
  "same-team battles are disabled": "目前不開放同隊玩家對戰。",
  "select at least one sitone": "請至少選擇一顆小石。",
  "select at least one sitone before ready": "準備前請至少選擇一顆小石。",
  "shop item is locked": "這個商品尚未解鎖。",
  "shop item not found": "找不到這個商品。",
  "shop item unavailable": "目前無法讀取商品資料，請稍後再試。",
  "shop items unavailable": "目前無法讀取商店資料，請稍後再試。",
  "sitone has no avatar icon": "這顆小石目前不能設為頭貼。",
  "sitone inventory is inconsistent": "小石資料有異常，請通知工作人員。",
  "sitone is not owned": "你尚未擁有這顆小石，不能設為頭貼。",
  "sitone loadout contains unavailable sitone": "小石陣容包含尚未持有的小石。",
  "sitone loadout contains unknown sitone": "小石陣容包含不存在的小石。",
  "sitone loadout exceeds owned quantity": "小石陣容超過你持有的數量。",
  "sitone loadout unavailable": "目前無法讀取小石陣容，請稍後再試。",
  "sitone loadout update failed": "小石陣容更新失敗，請稍後再試。",
  "sitones unavailable": "目前無法讀取小石資料，請稍後再試。",
  "status unavailable": "玩家狀態暫時無法讀取，請稍後再試。",
  "player has no team": "你目前沒有小隊，不能設定小隊頭貼。",
  "player team not found": "找不到你的小隊，請通知工作人員。",
  "team avatar update failed": "小隊頭貼更新失敗，請稍後再試。",
  "unknown sitone": "找不到這顆小石。",
}

const codeMessageMap: Record<string, string> = {
  community_stand_claim_failed: "領取攤位獎勵失敗，請稍後再試。",
  community_stand_lookup_failed: "目前無法讀取攤位資料，請稍後再試。",
  fusion_create_failed: "合成失敗，請稍後再試。",
  match_join_limit_lookup_failed: "目前無法確認對戰次數上限，請稍後再試。",
  me_avatar_save_failed: "頭貼更新失敗，請稍後再試。",
  me_team_avatar_save_failed: "小隊頭貼更新失敗，請稍後再試。",
  room_team_join_failed: "加入宿舍失敗，請稍後再試。",
  room_team_token_lookup_failed: "確認宿舍 QR Code 失敗，請稍後再試。",
  shop_purchase_failed: "購買失敗，請稍後再試。",
}

export class AppError extends Error {
  readonly status: number
  readonly code: string
  readonly requestId?: string
  readonly retryable: boolean

  constructor(input: {
    status: number
    code: string
    message: string
    requestId?: string
    retryable?: boolean
  }) {
    super(input.message)
    this.name = "AppError"
    this.status = input.status
    this.code = input.code
    this.requestId = input.requestId
    this.retryable =
      input.retryable ?? (input.status >= 500 || input.status === 429)
  }
}

export function createAppError(input: {
  status: number
  body: unknown
  fallbackMessage?: string
}) {
  const parsed = ProblemDetailsSchema.safeParse(input.body)
  const status = parsed.success
    ? (parsed.data.status ?? input.status)
    : input.status
  const code = parsed.success
    ? (parsed.data.code ?? parsed.data.type ?? "HTTP_ERROR")
    : "HTTP_ERROR"
  const rawMessage = parsed.success
    ? (parsed.data.detail ?? parsed.data.title ?? input.fallbackMessage)
    : input.fallbackMessage

  return new AppError({
    status,
    code,
    message: localizedErrorMessage(status, code, rawMessage),
    requestId: parsed.success ? parsed.data.instance : undefined,
  })
}

export function isTerminalClientError(error: unknown) {
  return error instanceof AppError && error.status >= 400 && !error.retryable
}

export function isBattleOpeningLockedError(error: unknown) {
  return (
    error instanceof AppError && error.message === battleOpeningLockedMessage
  )
}

function localizedErrorMessage(status: number, code: string, message?: string) {
  if (message && detailMessageMap[message]) {
    return detailMessageMap[message]
  }
  if (codeMessageMap[code]) {
    return codeMessageMap[code]
  }
  if (message && !isLikelyEnglishMessage(message)) {
    return message
  }
  return statusFallbackMessage(status, message)
}

function isLikelyEnglishMessage(message: string) {
  return /^[A-Za-z0-9 ,.;:'"!?()/_-]+$/.test(message)
}

function statusFallbackMessage(status: number, message?: string) {
  switch (status) {
    case 400:
      return "送出的資料格式不正確，請重新操作一次。"
    case 401:
      return "請先登入後再繼續。"
    case 403:
      return "目前不能執行這個操作。"
    case 404:
      return "找不到你要的資料。"
    case 409:
      return "目前狀態不允許這個操作，請重新整理後再試。"
    case 422:
      return "送出的資料內容不符合規則，請檢查後再試。"
    case 423:
      return "這個項目尚未解鎖。"
    case 429:
      return "操作太頻繁，請稍後再試。"
    case 500:
      return "伺服器發生錯誤，請稍後再試。"
    case 503:
      return "服務暫時無法使用，請稍後再試。"
    default:
      return message ?? "操作失敗，請稍後再試。"
  }
}
