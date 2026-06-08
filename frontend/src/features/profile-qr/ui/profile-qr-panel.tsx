import { Card, CardContent } from "@/shared/ui/card"

const PROFILE_QR_MOCK = {
  nickname: "阿洛",
  teamName: "松鼠小隊",
  openPower: 430,
  avatarInitial: "洛",
  qrcodeToken: "qr_token_123456",
}

function tokenBit(token: string, index: number) {
  let hash = 0x811c9dc5

  for (const char of `${token}:${index}`) {
    hash ^= char.charCodeAt(0)
    hash = Math.imul(hash, 0x01000193)
  }

  hash ^= hash >>> 13
  hash = Math.imul(hash, 0x85ebca6b)
  hash ^= hash >>> 16

  return (hash & 1) === 1
}

function createPassCodeCells(token: string) {
  return Array.from({ length: 169 }, (_, index) => {
  const x = index % 13
  const y = Math.floor(index / 13)
  const finder = (x < 4 && y < 4) || (x > 8 && y < 4) || (x < 4 && y > 8)
  const timing = (x === 6 && y % 2 === 0) || (y === 6 && x % 2 === 0)
  const quietCorner = x > 9 && y > 9

  if (finder || timing) {
    return true
  }

  if (quietCorner) {
    return false
  }

  return tokenBit(token, index)
  })
}

export function ProfileQrPanel() {
  // TODO(api): replace PROFILE_QR_MOCK with GET /api/me/status and GET /api/me/qrcode.
  const profile = PROFILE_QR_MOCK
  const passCodeCells = createPassCodeCells(profile.qrcodeToken)

  return (
    <div className="flex flex-col gap-3.5">
      <Card className="border-ink rounded-[var(--radius)] border-2 py-0 shadow-[4px_4px_0_rgba(23,35,58,0.14)]">
        <CardContent
          className="grid items-center gap-3.5 p-4"
          style={{ gridTemplateColumns: "70px 1fr" }}
        >
          <div className="bg-power border-ink flex h-[70px] w-[70px] items-center justify-center rounded-[24px] border-2 text-3xl font-black">
            {profile.avatarInitial}
          </div>
          <div>
            <span className="text-muted-foreground block text-xs font-black tracking-widest uppercase">
              玩家身份
            </span>
            <h2 className="mb-1 text-[28px] font-black leading-none tracking-tight">
              {profile.nickname}
            </h2>
            <p className="text-muted-foreground">
              {profile.teamName} · {profile.openPower} OP
            </p>
          </div>
        </CardContent>
      </Card>

      <Card className="border-ink rounded-[32px] border-2 py-0 shadow-[5px_5px_0_rgba(23,35,58,0.16)]">
        <CardContent className="px-[22px] pt-7 pb-6 text-center">
          <div
            role="img"
            aria-label="玩家身份通行碼圖樣"
            className="border-ink mx-auto mb-5 grid aspect-square w-full max-w-[306px] grid-cols-[repeat(13,1fr)] gap-1 rounded-[18px] border-4 bg-[#fffdf5] p-[18px]"
          >
            {passCodeCells.map((filled, index) => (
              <span
                key={index}
                className={`rounded-sm ${filled ? "bg-ink" : "bg-transparent"}`}
              />
            ))}
          </div>
          <h2 className="mb-2 text-[26px] font-black tracking-tight">
            請掃描這個 QR Code
          </h2>
          <p className="text-muted-foreground mx-auto max-w-[15rem] leading-relaxed">
            出示給工作人員掃描，用來確認身份與任務紀錄。
          </p>
        </CardContent>
      </Card>
    </div>
  )
}
