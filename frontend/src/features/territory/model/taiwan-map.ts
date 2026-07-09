/**
 * 領地地圖幾何資料 — 以陽明交大光復校區公開校園地圖為比例參考，
 * 重新繪製成遊戲用的互動校園平面圖。
 *
 * 這份資料不是官方地圖向量檔轉製，而是依公開校園地圖中的相對位置
 * 自行繪製：西側運動場與綠地、東北竹湖、中央教學區、南側宿舍區。
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
    path: "M218 320 L268 320 L268 354 L218 354 Z",
    labelX: 243,
    labelY: 333,
  },
  {
    id: "dorm-8",
    dorm: "八舍",
    county: "北門宿舍區",
    slot: 1,
    isBoss: false,
    path: "M258 68 L306 68 L306 104 L258 104 Z",
    labelX: 282,
    labelY: 82,
  },
  {
    id: "dorm-9",
    dorm: "九舍",
    county: "北門宿舍區",
    slot: 2,
    isBoss: false,
    path: "M202 64 L250 64 L250 100 L202 100 Z",
    labelX: 226,
    labelY: 78,
  },
  {
    id: "bamboo-house",
    dorm: "竹軒",
    county: "竹湖旁",
    slot: 3,
    isBoss: false,
    path: "M312 104 L350 104 L358 136 L320 146 L300 128 Z",
    labelX: 329,
    labelY: 122,
  },
  {
    id: "dorm-10",
    dorm: "十舍",
    county: "南宿舍區",
    slot: 4,
    isBoss: false,
    path: "M162 320 L210 320 L210 354 L162 354 Z",
    labelX: 186,
    labelY: 333,
  },
  {
    id: "dorm-11",
    dorm: "十一舍",
    county: "北門宿舍區",
    slot: 5,
    isBoss: false,
    path: "M314 68 L360 68 L360 104 L314 104 Z",
    labelX: 337,
    labelY: 82,
  },
  {
    id: "dorm-12",
    dorm: "十二舍",
    county: "南門宿舍區",
    slot: 6,
    isBoss: false,
    path: "M110 292 L158 292 L158 332 L110 332 Z",
    labelX: 134,
    labelY: 308,
  },
  {
    id: "dorm-13",
    dorm: "十三舍",
    county: "南門宿舍區",
    slot: 7,
    isBoss: false,
    path: "M164 364 L212 364 L212 394 L164 394 Z",
    labelX: 188,
    labelY: 374,
  },
  {
    id: "women-dorm-2",
    dorm: "女二舍",
    county: "北門餐廳區",
    slot: 8,
    isBoss: false,
    path: "M172 104 L226 104 L226 140 L172 140 Z",
    labelX: 199,
    labelY: 118,
  },
  {
    id: "south-dorm",
    dorm: "南舍",
    county: "南門",
    slot: 9,
    isBoss: false,
    path: "M260 362 L312 362 L312 394 L260 394 Z",
    labelX: 286,
    labelY: 374,
  },
  {
    id: "yansan",
    dorm: "研三舍",
    county: "竹湖餐廳區",
    slot: 10,
    isBoss: true,
    path: "M300 302 L354 302 L354 342 L300 342 Z",
    labelX: 327,
    labelY: 318,
  },
]
