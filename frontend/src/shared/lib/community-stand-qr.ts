const COMMUNITY_STAND_QR_PREFIX = "camp2026-community-stand|"

export function communityStandQrValue(qrToken: string) {
  return `${COMMUNITY_STAND_QR_PREFIX}${encodeURIComponent(qrToken)}`
}

export function parseCommunityStandQRToken(value: string) {
  const trimmed = value.trim()
  if (!trimmed) return null

  if (trimmed.startsWith(COMMUNITY_STAND_QR_PREFIX)) {
    return cleanCommunityStandQRToken(
      trimmed.slice(COMMUNITY_STAND_QR_PREFIX.length),
    )
  }

  return cleanCommunityStandQRToken(trimmed)
}

function cleanCommunityStandQRToken(value: string) {
  let decoded = value.trim()
  try {
    decoded = decodeURIComponent(decoded).trim()
  } catch {
    return null
  }
  return /^cst_[A-Za-z0-9_-]{8,128}$/.test(decoded) ? decoded : null
}
