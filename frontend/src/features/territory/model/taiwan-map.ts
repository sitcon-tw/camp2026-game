/**
 * 領地地圖幾何資料 — 以陽明交大光復校區公開校園地圖為比例參考，
 * 重新繪製成遊戲用的互動校園平面圖。
 *
 * 這份資料不是官方地圖向量檔轉製，而是依公開校園地圖中的相對位置
 * 自行繪製：西側運動場與綠地、東北竹湖、中央教學區、南側宿舍區。
 * 領地以大面積相鄰分區呈現，接近 OpenFront / Territorial 類戰略圖。
 */

export type TerritoryRegion = {
  id: string
  /** 對應校園領地名稱 */
  dorm: string
  /** 校園區位提示 */
  county: string
  /** 玩家隊伍對應順位（0-9，依 standings 回傳順序）；Boss 區不使用 */
  slot: number
  isBoss: boolean
  path: string
  labelX: number
  labelY: number
}

export const TAIWAN_MAP_VIEW_BOX = "0 0 420 420"

export const TAIWAN_TERRITORY_REGIONS: TerritoryRegion[] = [
  {
    id: "dorm-7",
    dorm: "七舍",
    county: "南宿舍區",
    slot: 0,
    isBoss: false,
    path: "M170 278 L244 248 L328 276 L292 340 L214 358 L142 344 Z",
    labelX: 225,
    labelY: 309,
  },
  {
    id: "dorm-8",
    dorm: "八舍",
    county: "北門宿舍區",
    slot: 1,
    isBoss: false,
    path: "M252 34 L334 42 L386 76 L360 126 L316 130 L252 116 Z",
    labelX: 316,
    labelY: 82,
  },
  {
    id: "dorm-9",
    dorm: "九舍",
    county: "北門宿舍區",
    slot: 2,
    isBoss: false,
    path: "M92 26 L206 20 L252 34 L252 116 L152 126 L118 92 Z",
    labelX: 186,
    labelY: 72,
  },
  {
    id: "bamboo-house",
    dorm: "竹軒",
    county: "竹湖旁",
    slot: 3,
    isBoss: false,
    path: "M252 116 L316 130 L302 176 L344 220 L328 276 L244 248 L226 174 Z",
    labelX: 289,
    labelY: 198,
  },
  {
    id: "dorm-10",
    dorm: "十舍",
    county: "南宿舍區",
    slot: 4,
    isBoss: false,
    path: "M152 126 L226 174 L244 248 L170 278 L92 218 L152 194 Z",
    labelX: 171,
    labelY: 213,
  },
  {
    id: "dorm-11",
    dorm: "十一舍",
    county: "北門宿舍區",
    slot: 5,
    isBoss: false,
    path: "M360 126 L386 76 L390 68 L392 214 L344 220 L302 176 L316 130 Z",
    labelX: 355,
    labelY: 151,
  },
  {
    id: "dorm-12",
    dorm: "十二舍",
    county: "南門宿舍區",
    slot: 6,
    isBoss: false,
    path: "M44 192 L92 218 L170 278 L142 344 L82 370 L36 328 L36 234 Z",
    labelX: 92,
    labelY: 289,
  },
  {
    id: "dorm-13",
    dorm: "十三舍",
    county: "南門宿舍區",
    slot: 7,
    isBoss: false,
    path: "M82 370 L142 344 L214 358 L232 402 L102 382 Z",
    labelX: 155,
    labelY: 371,
  },
  {
    id: "women-dorm-2",
    dorm: "女二舍",
    county: "北門餐廳區",
    slot: 8,
    isBoss: false,
    path: "M36 82 L92 26 L118 92 L152 126 L152 194 L92 218 L44 192 L36 132 Z",
    labelX: 91,
    labelY: 139,
  },
  {
    id: "south-dorm",
    dorm: "南舍",
    county: "南門",
    slot: 9,
    isBoss: false,
    path: "M214 358 L292 340 L342 386 L232 402 Z",
    labelX: 268,
    labelY: 375,
  },
  {
    id: "yansan",
    dorm: "研三舍",
    county: "竹湖餐廳區",
    slot: 10,
    isBoss: true,
    path: "M328 276 L392 214 L392 330 L342 386 L292 340 Z",
    labelX: 351,
    labelY: 313,
  },
]
