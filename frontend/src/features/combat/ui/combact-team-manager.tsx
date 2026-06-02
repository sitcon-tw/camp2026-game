import { Button } from "@/shared/ui/button"
import { type CombatStone } from "../model/combat.schema"

// TODO: 串接 API
const teamStones: CombatStone[] = [
  {
    id: "aaa",
    name: "A 小石",
    type: "探索",
    pictureSrc: "https://placehold.co/65x65/svg",
  },
  {
    id: "bbb",
    name: "A 小石",
    type: "靈光",
    pictureSrc: "https://placehold.co/65x65/svg",
  },
  {
    id: "ccc",
    name: "A 小石",
    type: "共鳴",
    pictureSrc: "https://placehold.co/65x65/svg",
  },
  {
    id: "ddd",
    name: "A 小石",
    type: "工程",
    pictureSrc: "https://placehold.co/65x65/svg",
  },
  {
    id: "eee",
    name: "A 小石",
    type: "娛樂",
    pictureSrc: "https://placehold.co/65x65/svg",
  },
]

export function CombactTeamManager() {
  return (
    <div className="flex w-full flex-col gap-4">
      {teamStones.map((stone) => {
        return (
          <div key={stone.id} className="">
            <Button
              type="button"
              variant="secondary"
              className="flex h-fit w-full p-2"
            >
              <img
                src={stone.pictureSrc}
                alt={stone.name}
                className="aspect-square rounded-lg object-contain"
              />
              <div className="flex flex-1 flex-col">
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
  )
}

// NOTE: 好想貓貓
