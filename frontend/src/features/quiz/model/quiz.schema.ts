import { z } from "zod"

export const QuizQuestionSchema = z.object({
  id: z.number(),
  question: z.string(),
  answer: z.array(z.string()),
  correctAnswer: z.number(),
  explain: z.array(z.string()),
})

export type QuizQuestionType = z.infer<typeof QuizQuestionSchema>
