/**
 * 領地地圖幾何資料 — 以臺灣本島輪廓象徵十一處宿舍領地。
 *
 * 由北至南切成十一塊帶狀領地，對應真實縣市的南北順序：
 * 臺北 → 新北 → 桃園 → 新竹 → 苗栗 → 臺中 → 彰化 → 雲林 → 嘉義 → 臺南 → 南臺灣(研三舍 Boss)。
 *
 * 幾何來源說明：此為本專案自行繪製的「簡化臺灣剪影」原創幾何
 * （參考公開的臺灣行政區輪廓比例，非直接取用任何第三方向量檔），
 * 相鄰領地共享邊界線，可整片拼合成臺灣島形。
 */

export type TerritoryRegion = {
  id: string
  /** 對應宿舍領地名稱 */
  dorm: string
  /** 象徵的臺灣縣市 */
  county: string
  /** 玩家隊伍對應順位（0-9，依 standings 回傳順序）；Boss 區不使用 */
  slot: number
  isBoss: boolean
  path: string
  labelX: number
  labelY: number
}

export const TAIWAN_MAP_VIEW_BOX = "0 0 300 460"

export const TAIWAN_TERRITORY_REGIONS: TerritoryRegion[] = [
  {
    id: "taipei",
    dorm: "七舍",
    county: "臺北",
    slot: 0,
    isBoss: false,
    path: "M134 62 L150 45 L178 25 L205 18 L258 52 L259 62 Z",
    labelX: 196,
    labelY: 40,
  },
  {
    id: "new-taipei",
    dorm: "八舍",
    county: "新北",
    slot: 1,
    isBoss: false,
    path: "M102 105 L118 80 L134 62 L259 62 L262 92 L260 105 Z",
    labelX: 181,
    labelY: 85,
  },
  {
    id: "taoyuan",
    dorm: "九舍",
    county: "桃園",
    slot: 2,
    isBoss: false,
    path: "M83 150 L92 120 L102 105 L260 105 L252 150 Z",
    labelX: 171,
    labelY: 129,
  },
  {
    id: "hsinchu",
    dorm: "竹軒",
    county: "新竹",
    slot: 3,
    isBoss: false,
    path: "M76 195 L78 165 L83 150 L252 150 L242 195 Z",
    labelX: 163,
    labelY: 173,
  },
  {
    id: "miaoli",
    dorm: "十舍",
    county: "苗栗",
    slot: 4,
    isBoss: false,
    path: "M79 240 L74 210 L76 195 L242 195 L240 205 L235 240 Z",
    labelX: 158,
    labelY: 218,
  },
  {
    id: "taichung",
    dorm: "十一舍",
    county: "臺中",
    slot: 5,
    isBoss: false,
    path: "M93 285 L82 255 L79 240 L235 240 L232 260 L227 285 Z",
    labelX: 157,
    labelY: 262,
  },
  {
    id: "changhua",
    dorm: "十二舍",
    county: "彰化",
    slot: 6,
    isBoss: false,
    path: "M107 320 L98 300 L93 285 L227 285 L222 315 L220 320 Z",
    labelX: 160,
    labelY: 302,
  },
  {
    id: "yunlin",
    dorm: "十三舍",
    county: "雲林",
    slot: 7,
    isBoss: false,
    path: "M123 355 L118 345 L107 320 L220 320 L210 355 Z",
    labelX: 165,
    labelY: 337,
  },
  {
    id: "chiayi",
    dorm: "女二舍",
    county: "嘉義",
    slot: 8,
    isBoss: false,
    path: "M144 395 L142 390 L123 355 L210 355 L205 370 L197 395 Z",
    labelX: 168,
    labelY: 374,
  },
  {
    id: "tainan",
    dorm: "南舍",
    county: "臺南",
    slot: 9,
    isBoss: false,
    path: "M144 395 L197 395 L192 410 L185 425 L160 425 L152 412 Z",
    labelX: 172,
    labelY: 407,
  },
  {
    id: "yansan",
    dorm: "研三舍",
    county: "南臺灣",
    slot: 10,
    isBoss: true,
    path: "M160 425 L185 425 L178 448 Z",
    labelX: 172,
    labelY: 434,
  },
]
