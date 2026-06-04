import { QuizPage } from '@/pages/quiz/ui/quiz-page'
import { createFileRoute } from '@tanstack/react-router'

export const Route = createFileRoute('/quiz')({
  component: QuizPage,
})
