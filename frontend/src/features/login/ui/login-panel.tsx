import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { useNavigate } from "@tanstack/react-router"
import { type FormEvent, useEffect, useRef, useState } from "react"

import sitconLogo from "@/assets/sitcon-logo.svg"
import { gameApi } from "@/shared/api/game"
import { Badge } from "@/shared/ui/badge"
import { Button } from "@/shared/ui/button"
import { Input } from "@/shared/ui/input"
import { cn } from "@/shared/utils"

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

type LoginPanelProps = {
  initialToken?: string
}

export function LoginPanel({ initialToken }: LoginPanelProps) {
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const autoToken = initialToken?.trim() ?? ""
  const autoSubmittedTokenRef = useRef("")
  const [code, setCode] = useState(autoToken)
  const [hasTried, setHasTried] = useState(false)
  const statusQuery = useQuery({
    queryKey: ["me", "status", "login"],
    queryFn: gameApi.status,
    gcTime: 0,
    refetchOnMount: "always",
    retry: false,
    staleTime: 0,
    throwOnError: false,
  })
  const loginMutation = useMutation({
    mutationFn: gameApi.login,
    onSuccess: (result) => {
      queryClient.setQueryData(["me", "status"], result.player)
      queryClient.invalidateQueries({ queryKey: ["me", "home"] })
      navigate({
        to: loginDestination(),
        replace: true,
      })
    },
  })
  const hasTokenFromLink = autoToken.length > 0
  const hasAttempted = hasTried || hasTokenFromLink
  const isError =
    hasAttempted && (code.trim().length === 0 || loginMutation.isError)

  useEffect(() => {
    if (
      autoToken.length === 0 ||
      autoSubmittedTokenRef.current === autoToken ||
      !statusQuery.isFetchedAfterMount ||
      !statusQuery.isError
    ) {
      return
    }
    autoSubmittedTokenRef.current = autoToken
    loginMutation.mutate(autoToken)
  }, [
    autoToken,
    loginMutation,
    statusQuery.isError,
    statusQuery.isFetchedAfterMount,
  ])

  useEffect(() => {
    if (!statusQuery.isFetchedAfterMount || !statusQuery.data) {
      return
    }
    queryClient.setQueryData(["me", "status"], statusQuery.data)
    queryClient.invalidateQueries({ queryKey: ["me", "home"] })
    navigate({
      to: loginDestination(),
      replace: true,
    })
  }, [navigate, queryClient, statusQuery.data, statusQuery.isFetchedAfterMount])

  function handleSubmit(e: FormEvent) {
    e.preventDefault()
    setHasTried(true)
    const token = code.trim()
    if (token.length === 0) return
    loginMutation.mutate(token)
  }

  return (
    <section
      className="bg-card border-ink w-full rounded-[32px] border-2 px-7 py-8"
      style={{ boxShadow: "0 8px 0 rgba(23,35,58,0.16)" }}
      aria-labelledby="login-title"
    >
      <div className="mb-10 flex items-start justify-between gap-6">
        <CampBadge />
        <div className="relative h-[76px] w-[76px] shrink-0" aria-hidden="true">
          <span className="bg-surface-raised border-ink absolute inset-2 rotate-6 rounded-[28px] border-2" />
          <StoneGlyph
            type={stoneTypes[0]}
            className="absolute top-1 left-0 -rotate-12"
          />
          <StoneGlyph
            type={stoneTypes[1]}
            tiny
            className="absolute right-0 bottom-0 rotate-12"
          />
        </div>
      </div>

      <div className="mb-10">
        <Badge className="border-ink bg-secondary text-foreground hover:bg-secondary mb-3 rounded-full border-2 px-2.5 py-1 text-[0.8rem] font-black">
          營隊遊戲工具
        </Badge>
        <h1
          id="login-title"
          className="leading-[1.04] font-black tracking-[-0.045em]"
          style={{ fontSize: "clamp(2.25rem, 12vw, 3.18rem)" }}
        >
          開源小石基地
        </h1>
      </div>

      <form onSubmit={handleSubmit} noValidate className="grid gap-6">
        <div className="grid gap-3">
          <label
            htmlFor="camp-login-code"
            className="text-[0.95rem] font-black"
          >
            營隊登入 Token
          </label>
          <Input
            id="camp-login-code"
            value={code}
            onChange={(e) => setCode(e.target.value)}
            placeholder="例如 token-1"
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
              className="border-ink bg-destructive rounded-[14px] border-2 px-2.5 py-1.5 text-[0.82rem] leading-snug font-bold text-white"
            >
              {loginMutation.isError
                ? "找不到這組登入資訊，請確認 Token 或詢問隊輔。"
                : "請輸入登入 Token。"}
            </p>
          )}
        </div>
        <Button
          type="submit"
          disabled={statusQuery.isPending || loginMutation.isPending}
          className="border-ink h-[56px] w-full border-2 text-[1.05rem] font-black shadow-[0_4px_0_rgba(23,35,58,0.22)] active:translate-y-0.5 active:shadow-[0_2px_0_rgba(23,35,58,0.22)]"
        >
          {statusQuery.isPending
            ? "確認登入狀態"
            : loginMutation.isPending
              ? "登入中"
              : "進入基地"}
        </Button>
      </form>
    </section>
  )
}

function loginDestination() {
  return "/"
}

type StoneType = (typeof stoneTypes)[number]

export function StoneGlyph({
  type,
  tiny = false,
  className,
}: {
  type: StoneType
  tiny?: boolean
  className?: string
}) {
  return (
    <span
      className={cn(
        "relative block shrink-0 border-2 border-[var(--stone-ink)] bg-[var(--stone)]",
        tiny
          ? "size-9 rounded-[14px_18px_12px_16px]"
          : "size-14 rounded-[20px_26px_16px_22px]",
        className,
      )}
      style={
        {
          "--stone": type.color,
          "--stone-ink": type.ink,
        } as React.CSSProperties
      }
      aria-hidden="true"
    >
      <span className="absolute top-1/4 left-1/4 h-[3px] w-1/2 -rotate-12 rounded-full bg-[var(--stone-ink)]/35" />
      <span className="absolute right-1/4 bottom-1/4 h-[3px] w-1/3 rotate-12 rounded-full bg-[var(--stone-ink)]/35" />
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
        <strong className="block text-[0.92rem] leading-[1.05] font-black tracking-tight">
          SITCON
        </strong>
        <span className="text-muted-foreground mt-0.5 block text-[0.78rem] leading-[1.05] font-extrabold">
          Camp 2026
        </span>
      </div>
    </div>
  )
}
