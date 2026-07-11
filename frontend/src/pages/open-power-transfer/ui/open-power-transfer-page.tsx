import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { useNavigate } from "@tanstack/react-router"
import { ArrowRightLeft, Sparkles } from "lucide-react"
import { type FormEvent, useEffect, useMemo, useState } from "react"
import { toast } from "sonner"

import { WorkshopPageShell } from "@/features/stone-workshop"
import { AppError } from "@/shared/api/error"
import { gameApi } from "@/shared/api/game"
import { Button } from "@/shared/ui/button"
import { Input } from "@/shared/ui/input"
import { Label } from "@/shared/ui/label"
import { PlayerAvatar } from "@/shared/ui/player-avatar"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/shared/ui/select"

export function OpenPowerTransferPage() {
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const [recipientID, setRecipientID] = useState("")
  const [amount, setAmount] = useState("")
  const homeQuery = useQuery({
    queryKey: ["me", "home"],
    queryFn: gameApi.home,
  })
  const unauthorized =
    homeQuery.error instanceof AppError && homeQuery.error.status === 401
  const player = homeQuery.data?.player
  const teamID = player?.team?.teamId
  const teamQuery = useQuery({
    queryKey: ["leaderboards", "team", teamID, "players"],
    queryFn: () => gameApi.leaderboardTeamPlayers(teamID ?? ""),
    enabled: Boolean(teamID),
  })

  useEffect(() => {
    if (unauthorized) navigate({ to: "/login", replace: true })
  }, [navigate, unauthorized])

  const openPower = homeQuery.data?.summary.openPower ?? player?.openPower ?? 0
  const recipients = useMemo(
    () =>
      (player?.teamMembers ?? []).filter(
        (member) => member.playerId !== player?.playerId,
      ),
    [player?.playerId, player?.teamMembers],
  )
  const selectedRecipientID = recipientID || recipients[0]?.playerId || ""
  const amountValue = Number(amount)
  const amountValid = Number.isInteger(amountValue) && amountValue >= 1

  const transferMutation = useMutation({
    mutationFn: gameApi.createOpenPowerTransfer,
    onSuccess: (result) => {
      setAmount("")
      setRecipientID("")
      void queryClient.invalidateQueries({ queryKey: ["me", "home"] })
      void queryClient.invalidateQueries({ queryKey: ["me", "status"] })
      void queryClient.invalidateQueries({
        queryKey: ["leaderboards", "team", teamID, "players"],
      })
      toast.success(
        `已轉帳 ${result.amount} OP 給 ${result.recipientNickname || result.recipientPlayerId}`,
      )
    },
    onError: (error) => {
      toast.error(error instanceof Error ? error.message : "開源力轉帳失敗")
    },
  })

  function submitTransfer(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (!selectedRecipientID) {
      toast.error("請選擇收款隊友")
      return
    }
    if (!amountValid) {
      toast.error("請輸入至少 1 OP")
      return
    }
    if (amountValue > openPower) {
      toast.error("開源力不足，無法轉帳")
      return
    }
    transferMutation.mutate({
      recipientPlayerId: selectedRecipientID,
      amount: amountValue,
    })
  }

  if (unauthorized) return null

  const teamOpenPower =
    teamQuery.data?.players.reduce(
      (total, member) => total + member.openPower,
      0,
    ) ?? 0

  return (
    <WorkshopPageShell eyebrow="OPEN POWER EXCHANGE" title="開源力轉帳">
      <div className="grid gap-3">
        <section
          className="bg-card border-ink rounded-[22px] border-2 p-[15px]"
          aria-label="轉帳功能"
        >
          <div className="mb-3 flex items-start justify-between gap-3">
            <div>
              <p className="text-muted-foreground mb-1 text-xs font-black tracking-[0.08em] uppercase">
                Transfer
              </p>
              <h2 className="text-[22px] font-black tracking-normal">
                轉給隊友
              </h2>
            </div>
            <span className="bg-pebble-spark border-ink rounded-full border-2 px-3 py-1 text-sm font-black whitespace-nowrap">
              {openPower} OP
            </span>
          </div>

          {homeQuery.isPending ? (
            <div className="bg-muted border-border h-32 rounded-[17px] border-2" />
          ) : recipients.length > 0 ? (
            <form className="grid gap-3" onSubmit={submitTransfer}>
              <div className="grid gap-2">
                <Label htmlFor="open-power-transfer-recipient">收款隊友</Label>
                <Select
                  value={selectedRecipientID}
                  onValueChange={setRecipientID}
                  disabled={transferMutation.isPending}
                >
                  <SelectTrigger
                    id="open-power-transfer-recipient"
                    className="w-full"
                  >
                    <SelectValue placeholder="選擇同組成員" />
                  </SelectTrigger>
                  <SelectContent>
                    {recipients.map((member) => (
                      <SelectItem key={member.playerId} value={member.playerId}>
                        {member.nickname}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>

              <div className="grid grid-cols-[minmax(0,1fr)_112px] items-end gap-2">
                <div className="grid gap-2">
                  <Label htmlFor="open-power-transfer-amount">轉帳 OP</Label>
                  <Input
                    id="open-power-transfer-amount"
                    type="number"
                    inputMode="numeric"
                    min={1}
                    max={openPower}
                    value={amount}
                    disabled={transferMutation.isPending}
                    placeholder="輸入數量"
                    onChange={(event) => setAmount(event.target.value)}
                  />
                </div>
                <Button
                  type="submit"
                  disabled={
                    transferMutation.isPending ||
                    !selectedRecipientID ||
                    !amountValid ||
                    amountValue > openPower
                  }
                  className="min-h-10"
                >
                  <ArrowRightLeft className="size-4" aria-hidden />
                  {transferMutation.isPending ? "處理中" : "轉帳"}
                </Button>
              </div>
            </form>
          ) : (
            <p className="text-muted-foreground bg-surface-raised border-border rounded-[17px] border-2 px-3 py-3 text-sm font-bold">
              目前沒有可轉帳的同組成員
            </p>
          )}
        </section>

        <section
          className="bg-card border-ink rounded-[22px] border-2 p-[15px]"
          aria-label="整隊開源力"
        >
          <div className="mb-3 flex items-start justify-between gap-3">
            <div>
              <p className="text-muted-foreground mb-1 text-xs font-black tracking-[0.08em] uppercase">
                Team Balance
              </p>
              <h2 className="text-[22px] font-black tracking-normal">
                整隊開源力
              </h2>
            </div>
            <span className="bg-surface-raised border-border flex items-center gap-1.5 rounded-full border-2 px-3 py-1 text-sm font-black whitespace-nowrap">
              <Sparkles className="size-4" aria-hidden />
              {teamQuery.isPending ? "-" : `${teamOpenPower} OP`}
            </span>
          </div>

          {teamQuery.isPending ? (
            <div className="grid gap-2">
              {[0, 1, 2].map((item) => (
                <div
                  key={item}
                  className="bg-muted border-border h-[58px] rounded-[17px] border-2"
                />
              ))}
            </div>
          ) : teamQuery.isError ? (
            <p className="text-muted-foreground bg-surface-raised border-border rounded-[17px] border-2 px-3 py-3 text-sm font-bold">
              隊伍開源力讀取失敗
            </p>
          ) : teamQuery.data && teamQuery.data.players.length > 0 ? (
            <ul className="grid gap-2">
              {teamQuery.data.players.map((member) => (
                <li
                  key={member.playerId}
                  className="bg-surface-raised border-border grid min-h-[58px] grid-cols-[40px_minmax(0,1fr)_auto] items-center gap-3 rounded-[17px] border-2 px-3 py-2"
                >
                  <PlayerAvatar
                    playerId={member.playerId}
                    nickname={member.nickname}
                    avatarUrl={member.avatarUrl}
                    size="lg"
                    className="border-ink bg-pebble-resonate border-2"
                  />
                  <div className="min-w-0">
                    <strong className="block truncate text-[16px] font-black">
                      {member.nickname}
                    </strong>
                    {member.current ? (
                      <small className="text-muted-foreground text-xs font-bold">
                        你
                      </small>
                    ) : null}
                  </div>
                  <strong className="text-[17px] font-black whitespace-nowrap">
                    {member.openPower} OP
                  </strong>
                </li>
              ))}
            </ul>
          ) : (
            <p className="text-muted-foreground bg-surface-raised border-border rounded-[17px] border-2 px-3 py-3 text-sm font-bold">
              尚未取得隊伍資料
            </p>
          )}
        </section>
      </div>
    </WorkshopPageShell>
  )
}
