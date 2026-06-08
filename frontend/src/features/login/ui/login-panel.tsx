import { useState } from "react"

import sitconLogo from "@/assets/sitcon-logo.svg"
import { Badge } from "@/shared/ui/badge"
import { Button } from "@/shared/ui/button"
import { Input } from "@/shared/ui/input"

export const stoneTypes = [
  {
    name: "探索",
    color: "#4F8CC9",
    ink: "#153A5D",
    x: "7%",
    y: "18%",
    rotate: "-10deg",
  },
  {
    name: "靈光",
    color: "#F4C84A",
    ink: "#5A3E05",
    x: "78%",
    y: "13%",
    rotate: "12deg",
  },
  {
    name: "共鳴",
    color: "#E96F86",
    ink: "#5B1F2E",
    x: "83%",
    y: "63%",
    rotate: "-7deg",
  },
  {
    name: "工程",
    color: "#31A886",
    ink: "#123F35",
    x: "10%",
    y: "72%",
    rotate: "9deg",
  },
  {
    name: "娛樂",
    color: "#9A75D6",
    ink: "#37215E",
    x: "67%",
    y: "82%",
    rotate: "-14deg",
  },
] as const

export function LoginPanel() {
  const [code, setCode] = useState("")
  const [hasTried, setHasTried] = useState(true)
  const isError = hasTried && code.length < 8

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setHasTried(true)
  }

  return (
    <section
      className="bg-card border-ink w-full rounded-[32px] border-2 px-7 py-8"
      style={{ boxShadow: "0 8px 0 rgba(23,35,58,0.16)" }}
      aria-labelledby="login-title"
    >
      <div className="mb-10 flex items-start justify-between gap-6">
        <CampBadge />
        <div className="game-mark" aria-hidden="true">
          <span className="game-mark-base" />
          <StoneGlyph type={stoneTypes[0]} />
          <StoneGlyph type={stoneTypes[1]} tiny />
        </div>
      </div>

      <div className="mb-10">
        <Badge className="mb-3 rounded-full border-2 border-ink bg-secondary px-2.5 py-1 text-[0.8rem] font-black text-foreground hover:bg-secondary">
          營隊遊戲工具
        </Badge>
        <h1
          id="login-title"
          className="font-black leading-[1.04] tracking-[-0.045em]"
          style={{ fontSize: "clamp(2.25rem, 12vw, 3.18rem)" }}
        >
          開源小石基地
        </h1>
      </div>

      <form onSubmit={handleSubmit} noValidate className="grid gap-6">
        <div className="grid gap-3">
          <label htmlFor="camp-login-code" className="text-[0.95rem] font-black">
            營隊登入碼
          </label>
          <Input
            id="camp-login-code"
            value={code}
            onChange={(e) => setCode(e.target.value.toUpperCase())}
            placeholder="例如 CAMP26-A7K2"
            autoComplete="one-time-code"
            inputMode="text"
            aria-describedby={isError ? "login-error" : undefined}
            aria-invalid={isError ? "true" : "false"}
            className="border-ink h-[54px] rounded-[var(--radius)] border-2 bg-white text-[1.08rem] font-bold tracking-wide shadow-none focus-visible:shadow-[0_0_0_4px_rgba(244,200,74,0.72)] focus-visible:ring-0 aria-invalid:bg-[#FFF7EF]"
          />
          {isError && (
            <p
              id="login-error"
              role="alert"
              className="border-ink bg-destructive rounded-[14px] border-2 px-2.5 py-1.5 text-[0.82rem] font-bold leading-snug text-white"
            >
              找不到這組登入資訊，請確認代碼或詢問隊輔。
            </p>
          )}
        </div>
        <Button
          type="submit"
          className="border-ink h-[56px] w-full border-2 text-[1.05rem] font-black shadow-[0_4px_0_rgba(23,35,58,0.22)] active:translate-y-0.5 active:shadow-[0_2px_0_rgba(23,35,58,0.22)]"
        >
          進入基地
        </Button>
      </form>

    </section>
  )
}

type StoneType = (typeof stoneTypes)[number]

export function StoneGlyph({
  type,
  tiny = false,
}: {
  type: StoneType
  tiny?: boolean
}) {
  return (
    <span
      className={`stone-glyph ${tiny ? "stone-glyph-tiny" : ""}`}
      style={
        {
          "--stone": type.color,
          "--stone-ink": type.ink,
        } as React.CSSProperties
      }
      aria-hidden="true"
    >
      <span className="stone-cut stone-cut-a" />
      <span className="stone-cut stone-cut-b" />
    </span>
  )
}

function CampBadge() {
  return (
    <div
      className="bg-surface-raised border-border inline-flex min-h-[52px] items-center gap-2.5 rounded-[18px] border-2 px-3 py-2"
      aria-label="SITCON Camp 2026 遊戲徽章"
    >
      <img
        src={sitconLogo}
        alt=""
        className="h-8 w-8 shrink-0 object-contain"
        aria-hidden="true"
      />
      <div>
        <strong className="block text-[0.92rem] font-black leading-[1.05] tracking-tight">
          SITCON
        </strong>
        <span className="text-muted-foreground mt-0.5 block text-[0.78rem] font-extrabold leading-[1.05]">
          Camp 2026
        </span>
      </div>
    </div>
  )
}
