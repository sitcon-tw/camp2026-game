const MISSION_STORAGE_KEY = "camp2026.territoryMissionId"

export function getStoredMissionID() {
  if (typeof window === "undefined") return ""
  return window.localStorage.getItem(MISSION_STORAGE_KEY) ?? ""
}

export function storeMissionID(missionID: string) {
  window.localStorage.setItem(MISSION_STORAGE_KEY, missionID)
}

export function clearStoredMissionID() {
  window.localStorage.removeItem(MISSION_STORAGE_KEY)
}
