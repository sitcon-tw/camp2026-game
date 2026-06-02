import { Button } from "@/shared/ui/button"
import { type CombatStone } from "../model/combat.schema"

// TODO: 串接 API
const teamStones: CombatStone[] = [
  {
    id: "aaa",
    name: "A 小石",
    type: "explore",
    pictureSrc: "https://placehold.co/65x65/svg",
  },
  {
    id: "bbb",
    name: "A 小石",
    type: "wise",
    pictureSrc: "https://placehold.co/65x65/svg",
  },
  {
    id: "ccc",
    name: "A 小石",
    type: "collab",
    pictureSrc: "https://placehold.co/65x65/svg",
  },
  {
    id: "ddd",
    name: "A 小石",
    type: "engine",
    pictureSrc: "https://placehold.co/65x65/svg",
  },
  {
    id: "eee",
    name: "A 小石",
    type: "entertain",
    pictureSrc: "https://placehold.co/65x65/svg",
  },
]

export function CombactTeamManager() {
  return (
    <div className="flex w-full min-w-0 gap-2">
      {teamStones.map((stone) => {
        return (
          <div
            key={stone.id}
            className="aspect-square min-w-0 flex-1 basis-0 border"
          >
            <Button
              type="button"
              variant="ghost"
              className="border-primary size-full rounded-none border p-0"
            >
              <img
                src={stone.pictureSrc}
                alt={stone.name}
                className="size-full object-contain"
              />
            </Button>
          </div>
        )
      })}
    </div>
  )
}

// NOTE: 好想貓貓
