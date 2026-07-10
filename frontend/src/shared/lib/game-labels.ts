export type SitoneToneKey = "explore" | "spark" | "echo" | "build" | "play"

type SitoneMeta = {
  key: SitoneToneKey
  label: string
  short: string
  bgClassName: string
}

const sitoneTypeMap: Record<string, SitoneMeta> = {
  exploration: {
    key: "explore",
    label: "探索",
    short: "EXP",
    bgClassName: "bg-pebble-explore",
  },
  inspiration: {
    key: "spark",
    label: "靈光",
    short: "SPK",
    bgClassName: "bg-pebble-spark",
  },
  resonance: {
    key: "echo",
    label: "共鳴",
    short: "ECO",
    bgClassName: "bg-pebble-resonate",
  },
  engineering: {
    key: "build",
    label: "工程",
    short: "BLD",
    bgClassName: "bg-pebble-engineer",
  },
  entertainment: {
    key: "play",
    label: "娛樂",
    short: "PLY",
    bgClassName: "bg-pebble-play",
  },
}

const fallbackSitoneMeta: SitoneMeta = {
  key: "explore",
  label: "小石",
  short: "STN",
  bgClassName: "bg-primary",
}

const itemTypeLabels: Record<string, string> = {
  material: "素材",
  charm: "御守",
  cosmetic: "外觀",
  event: "活動紀念",
}

const itemTypeClasses: Record<string, string> = {
  material: "bg-pebble-engineer",
  charm: "bg-pebble-explore",
  cosmetic: "bg-pebble-spark",
  event: "bg-pebble-resonate",
}

const rarityLabels: Record<string, string> = {
  base: "基礎",
  common: "常見",
  rare: "稀有",
  limited: "限定",
}

const itemSourceLabels: Record<string, string> = {
  shop: "商店",
  drop: "掉落",
  both: "商店／掉落",
  event: "活動",
}

const workerDormRoomNumbers = new Set([
  "118",
  "119",
  "125",
  "128",
  "201",
  "202",
  "203",
  "204",
  "205",
  "206",
  "207",
])

const chineseDigits = [
  "零",
  "一",
  "二",
  "三",
  "四",
  "五",
  "六",
  "七",
  "八",
  "九",
]

export type DormRoomGroupKey =
  | "student-male"
  | "student-female"
  | "worker-male"
  | "worker-female"

export const dormRoomGroupOrder: DormRoomGroupKey[] = [
  "student-male",
  "student-female",
  "worker-male",
  "worker-female",
]

const dormRoomGroupLabels: Record<DormRoomGroupKey, string> = {
  "student-male": "學員宿舍 - 男宿",
  "student-female": "學員宿舍 - 女宿",
  "worker-male": "工人宿舍 - 男宿",
  "worker-female": "工人宿舍 - 女宿",
}

export function sitoneMeta(type: string): SitoneMeta {
  return sitoneTypeMap[type] ?? fallbackSitoneMeta
}

export function itemTypeLabel(type: string) {
  return itemTypeLabels[type] ?? type
}

export function itemTypeClass(type: string) {
  return itemTypeClasses[type] ?? "bg-primary"
}

export function rarityLabel(rarity: string) {
  return rarityLabels[rarity] ?? rarity
}

export function itemSourceLabel(source: string | undefined) {
  if (!source) return ""
  return itemSourceLabels[source] ?? source
}

export function rarityToneClass(rarity: string) {
  switch (rarity) {
    case "rare":
    case "稀有":
      return "bg-pebble-explore-muted"
    case "limited":
    case "限定":
      return "bg-pebble-play-muted"
    default:
      return "bg-pebble-engineer-muted"
  }
}

export function formatTeamName(name: string) {
  const normalized = name.trim()
  const match = normalized.match(/^Team\s+0*(\d+)$/i)
  if (!match) return name

  const number = Number(match[1])
  if (!Number.isInteger(number) || number <= 0 || number >= 100) return name
  return `第${formatChineseNumber(number)}小隊`
}

export function roomDisplayName(roomNumber: string) {
  const normalized = roomNumber.trim()
  if (normalized.startsWith("2")) return `男宿 ${normalized} 房`
  if (normalized.startsWith("1")) return `女宿 ${normalized} 房`
  return `${normalized} 房`
}

export function dormRoomGroupKey(roomNumber: string): DormRoomGroupKey {
  const normalized = roomNumber.trim()
  const worker = workerDormRoomNumbers.has(normalized)
  const male = normalized.startsWith("2")

  if (worker && male) return "worker-male"
  if (worker) return "worker-female"
  if (male) return "student-male"
  return "student-female"
}

export function dormRoomGroupLabel(roomNumber: string) {
  return dormRoomGroupLabels[dormRoomGroupKey(roomNumber)]
}

export function groupDormRooms<T extends { roomNumber: string }>(
  rooms: readonly T[],
) {
  return dormRoomGroupOrder
    .map((key) => ({
      key,
      label: dormRoomGroupLabels[key],
      rooms: rooms
        .filter((room) => dormRoomGroupKey(room.roomNumber) === key)
        .slice()
        .sort(compareRoomNumbers),
    }))
    .filter((group) => group.rooms.length > 0)
}

function compareRoomNumbers(
  left: { roomNumber: string },
  right: { roomNumber: string },
) {
  return (
    Number(left.roomNumber) - Number(right.roomNumber) ||
    left.roomNumber.localeCompare(right.roomNumber)
  )
}

function formatChineseNumber(number: number) {
  if (number < 10) return chineseDigits[number]
  if (number === 10) return "十"
  if (number < 20) return `十${chineseDigits[number % 10]}`

  const tens = Math.floor(number / 10)
  const ones = number % 10
  return `${chineseDigits[tens]}十${ones === 0 ? "" : chineseDigits[ones]}`
}
