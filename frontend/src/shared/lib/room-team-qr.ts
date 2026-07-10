const ROOM_TEAM_QR_PREFIX = "camp2026-room-team|"

export function roomTeamQrValue(qrToken: string) {
  return `${ROOM_TEAM_QR_PREFIX}${encodeURIComponent(qrToken)}`
}

export function parseRoomTeamQRToken(value: string) {
  const trimmed = value.trim()
  if (!trimmed) return null

  if (trimmed.startsWith(ROOM_TEAM_QR_PREFIX)) {
    return cleanRoomTeamQRToken(trimmed.slice(ROOM_TEAM_QR_PREFIX.length))
  }

  return cleanRoomTeamQRToken(trimmed)
}

function cleanRoomTeamQRToken(value: string) {
  let decoded = value.trim()
  try {
    decoded = decodeURIComponent(decoded).trim()
  } catch {
    return null
  }
  return /^rmt_[A-Za-z0-9_-]{8,128}$/.test(decoded) ? decoded : null
}
