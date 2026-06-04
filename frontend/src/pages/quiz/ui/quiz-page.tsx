import { QuizQuestionType } from "@/features/quiz/model/quiz.schema"
import { QuizExplainCard } from "@/features/quiz/ui/quiz-explain-card"
import { QuizQuestionCard } from "@/features/quiz/ui/quiz-question-card"
import { QuizTeamList } from "@/features/quiz/ui/quiz-team-list"
import { QuizUserCard } from "@/features/quiz/ui/quiz-user-card"

// TODO: 串接 API
const enemySitones = [
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

// TODO: 串接 API
const playerSitones = [
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

const quizData: QuizQuestionType = {
  id: 0,
  question: "貓貓貓貓貓貓",
  answer: ["貓", "貓", "貓", "貓"],
  correctAnswer: 0,
  explain: ["貓", "貓", "貓", "貓"],
}

export function QuizPage() {
  return (
    <div className="relative h-dvh w-dvw">
      {/* 玩家對決畫面 */}
      <>
        {/* 對手 */}
        <div className="relative top-4 left-0 z-0 grid w-dvw grid-cols-1 gap-4">
          <QuizUserCard
            userName="四十二號混凝土"
            userPower={2}
            sitoneName="東康小石"
            sitconId={1}
            sitoneType="工程"
            className="mx-auto"
          />
          <QuizTeamList
            team={enemySitones}
            highlight={1}
            reverse
            className="mx-auto w-fit"
          />
        </div>
        {/* 玩家 */}
        <div className="absolute bottom-4 left-0 grid w-dvw grid-cols-1 gap-4">
          <QuizTeamList
            team={playerSitones}
            highlight={1}
            className="mx-auto w-fit"
          />
          <QuizUserCard
            userName="義大利麵"
            userPower={0}
            sitoneName="西康小石"
            sitconId={0}
            sitoneType="靈光"
            className="mx-auto"
          />
        </div>
      </>

      {/* 顯示題目畫面 */}
      <>
        <div className="absolute top-0 left-0 z-10 h-dvh w-dvw bg-foreground/50 flex items-center px-4">
          <QuizQuestionCard quizData={quizData} className="w-full" />
        </div>
      </>

      {/* 顯示解析畫面 */}
      <>
        <div className="absolute top-0 left-0 z-20 h-dvh w-dvw bg-foreground/50 flex items-center px-4">
          <QuizExplainCard quizData={quizData} className="w-full" />
        </div>
      </>
    </div>
  )
}
