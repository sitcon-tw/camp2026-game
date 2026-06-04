import { useState } from "react"
import { Badge } from "@/shared/ui/badge"
import { Button } from "@/shared/ui/button"
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/shared/ui/dialog"
import { type CombatStone } from "../model/combat.schema"

// TODO: 串接 API
const initialTeamStones: CombatStone[] = [
  {
    id: "aaa",
    name: "A 小石",
    type: "探索",
    pictureSrc: "https://placehold.co/65x65/svg",
  },
  {
    id: "bbb",
    name: "B 小石",
    type: "靈光",
    pictureSrc: "https://placehold.co/65x65/svg",
  },
  {
    id: "ccc",
    name: "C 小石",
    type: "共鳴",
    pictureSrc: "https://placehold.co/65x65/svg",
  },
  {
    id: "ddd",
    name: "D 小石",
    type: "工程",
    pictureSrc: "https://placehold.co/65x65/svg",
  },
  {
    id: "eee",
    name: "E 小石",
    type: "娛樂",
    pictureSrc: "https://placehold.co/65x65/svg",
  },
]

const initialBackpackStones: CombatStone[] = [
  {
    id: "fff",
    name: "F 小石",
    type: "探索",
    pictureSrc: "https://placehold.co/65x65/svg",
  },
  {
    id: "ggg",
    name: "G 小石",
    type: "靈光",
    pictureSrc: "https://placehold.co/65x65/svg",
  },
  {
    id: "hhh",
    name: "H 小石",
    type: "共鳴",
    pictureSrc: "https://placehold.co/65x65/svg",
  },
  {
    id: "iii",
    name: "I 小石",
    type: "工程",
    pictureSrc: "https://placehold.co/65x65/svg",
  },
  {
    id: "jjj",
    name: "J 小石",
    type: "娛樂",
    pictureSrc: "https://placehold.co/65x65/svg",
  },
]

export function CombactTeamManager() {
  const [teamStones, setTeamStones] = useState<CombatStone[]>(initialTeamStones)
  const [backpackStones, setBackpackStones] = useState<CombatStone[]>(
    initialBackpackStones,
  )
  const [selectedTeamStoneId, setSelectedTeamStoneId] = useState<string | null>(
    null,
  )

  const selectedTeamStone = teamStones.find(
    (stone) => stone.id === selectedTeamStoneId,
  )
  const occupiedTeamStoneIds = new Set(
    teamStones
      .filter((stone) => stone.id !== selectedTeamStoneId)
      .map((stone) => stone.id),
  )

  const closePopup = () => {
    setSelectedTeamStoneId(null)
  }

  const swapStone = (backpackStone: CombatStone) => {
    if (!selectedTeamStone) {
      return
    }

    if (occupiedTeamStoneIds.has(backpackStone.id)) {
      return
    }

    setTeamStones((currentStones) =>
      currentStones.map((stone) =>
        stone.id === selectedTeamStone.id ? backpackStone : stone,
      ),
    )
    setBackpackStones((currentStones) =>
      currentStones.map((stone) =>
        stone.id === backpackStone.id ? selectedTeamStone : stone,
      ),
    )
    closePopup()
  }

  return (
    <>
      <div className="flex w-full flex-col gap-4">
        {teamStones.map((stone) => {
          return (
            <div key={stone.id}>
              <Button
                type="button"
                variant="secondary"
                className="flex h-fit w-full justify-start p-2 text-left"
                onClick={() => {
                  setSelectedTeamStoneId(stone.id)
                }}
              >
                <img
                  src={stone.pictureSrc}
                  alt={stone.name}
                  className="aspect-square size-16 rounded-lg object-contain"
                />
                <div className="flex flex-1 flex-col items-start gap-1">
                  <div>
                    <span className="text-lg">{stone.name}</span>{" "}
                    <span className="text-muted-foreground text-sm">
                      #{stone.id}
                    </span>
                  </div>
                  <span>{stone.type}型小石</span>
                </div>
              </Button>
            </div>
          )
        })}
      </div>

      <Dialog
        open={selectedTeamStone !== undefined}
        onOpenChange={(open) => {
          if (!open) {
            closePopup()
          }
        }}
      >
        {selectedTeamStone ? (
          <DialogContent className="max-h-[85dvh] overflow-hidden p-0">
            <DialogHeader className="border-b p-4 pr-12">
              <DialogTitle>更換小石</DialogTitle>
              <DialogDescription>
                選擇背包小石替換 {selectedTeamStone.name}
              </DialogDescription>
            </DialogHeader>

            <div className="grid max-h-[55dvh] gap-3 overflow-y-auto p-4">
              {backpackStones.map((stone) => {
                const isAlreadyInTeam = occupiedTeamStoneIds.has(stone.id)

                return (
                  <Button
                    key={stone.id}
                    type="button"
                    variant="outline"
                    className="h-fit justify-start p-2 text-left"
                    disabled={isAlreadyInTeam}
                    onClick={() => {
                      swapStone(stone)
                    }}
                  >
                    <img
                      src={stone.pictureSrc}
                      alt={stone.name}
                      className="aspect-square size-14 rounded-lg object-contain"
                    />
                    <div className="flex min-w-0 flex-1 flex-col items-start gap-1">
                      <div className="min-w-0">
                        <span className="text-base font-medium">
                          {stone.name}
                        </span>{" "}
                        <span className="text-muted-foreground text-sm">
                          #{stone.id}
                        </span>
                      </div>
                      <div className="flex flex-wrap gap-2">
                        <Badge variant="secondary">{stone.type}型小石</Badge>
                        {isAlreadyInTeam ? (
                          <Badge variant="outline">已上場</Badge>
                        ) : null}
                      </div>
                    </div>
                  </Button>
                )
              })}
            </div>

            <DialogFooter className="border-t p-4">
              <DialogClose asChild>
                <Button type="button" variant="outline">
                  取消
                </Button>
              </DialogClose>
            </DialogFooter>
          </DialogContent>
        ) : null}
      </Dialog>
    </>
  )
}

// NOTE: 好想貓貓
