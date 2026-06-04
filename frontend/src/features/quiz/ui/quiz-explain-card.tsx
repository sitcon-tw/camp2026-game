import { Button } from "@/shared/ui/button"
import { type QuizQuestionType } from "../model/quiz.schema"
import { cn } from "@/shared/utils/cn"
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/shared/ui/card"
import { Check } from "lucide-react"

type QuizExplainCardType = {
  quizData: QuizQuestionType
  className: string
}

export function QuizExplainCard({
  quizData,
  className,
}: QuizExplainCardType) {
  return (
    <Card className={cn(className)}>
      <CardHeader>
        <CardTitle className="text-center text-2xl">{quizData.question}</CardTitle>
        <CardDescription className="text-center">選項解析</CardDescription>
      </CardHeader>
      <CardContent>
        <div className="grid grid-cols-1 gap-4">
          <Button variant="secondary" size="lg">
            {quizData.explain[0]}
          </Button>
          <Button variant="secondary" size="lg">
            {quizData.explain[1]}
          </Button>
          <Button variant="secondary" size="lg">
            {quizData.explain[2]}
          </Button>
          <Button variant="secondary" size="lg">
            {quizData.explain[3]}
          </Button>
        </div>
      </CardContent>
      <CardFooter className="flex justify-end">
        <Button>完成<Check /></Button>
      </CardFooter>
    </Card>
  )
}
