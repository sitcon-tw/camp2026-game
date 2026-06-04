import { Button } from "@/shared/ui/button"
import { type QuizQuestionType } from "../model/quiz.schema"
import { cn } from "@/shared/utils/cn"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/shared/ui/card"

type QuizQuestionCardType = {
  quizData: QuizQuestionType
  className: string
}

const submitAnswerA = () => {
  // TODO
}

const submitAnswerB = () => {
  // TODO
}
const submitAnswerC = () => {
  // TODO
}
const submitAnswerD = () => {
  // TODO
}

export function QuizQuestionCard({
  quizData,
  className,
}: QuizQuestionCardType) {
  return (
    <Card className={cn(className)}>
      <CardHeader>
        <CardTitle className="text-center text-2xl">{quizData.question}</CardTitle>
        <CardDescription className="text-center">作答說明：單選題。</CardDescription>
      </CardHeader>
      <CardContent>
        <div className="grid grid-cols-1 gap-4">
          <Button variant="secondary" size="lg" onClick={submitAnswerA}>
            {quizData.answer[0]}
          </Button>
          <Button variant="secondary" size="lg" onClick={submitAnswerB}>
            {quizData.answer[1]}
          </Button>
          <Button variant="secondary" size="lg" onClick={submitAnswerC}>
            {quizData.answer[2]}
          </Button>
          <Button variant="secondary" size="lg" onClick={submitAnswerD}>
            {quizData.answer[3]}
          </Button>
        </div>
      </CardContent>
    </Card>
  )
}
