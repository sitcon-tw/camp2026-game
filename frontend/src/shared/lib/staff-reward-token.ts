const STAFF_REWARD_TOKEN_PREFIX = "camp2026-staff-reward|"

export function staffRewardTokenQrValue(token: string) {
  return `${STAFF_REWARD_TOKEN_PREFIX}${encodeURIComponent(token)}`
}

export function parseStaffRewardToken(value: string) {
  const trimmed = value.trim()
  if (!trimmed) return null

  if (trimmed.startsWith(STAFF_REWARD_TOKEN_PREFIX)) {
    return cleanStaffRewardToken(
      trimmed.slice(STAFF_REWARD_TOKEN_PREFIX.length),
    )
  }

  return cleanStaffRewardToken(trimmed)
}

function cleanStaffRewardToken(value: string) {
  let decoded = value.trim()
  try {
    decoded = decodeURIComponent(decoded).trim()
  } catch {
    return null
  }
  return /^srt_[A-Za-z0-9_-]{8,128}$/.test(decoded) ? decoded : null
}
