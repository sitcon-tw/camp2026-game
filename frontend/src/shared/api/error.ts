import { z } from "zod"

const ProblemDetailsSchema = z.object({
  title: z.string().optional(),
  detail: z.string().optional(),
  status: z.number().optional(),
  type: z.string().optional(),
  instance: z.string().optional(),
})

export class AppError extends Error {
  readonly status: number
  readonly code: string
  readonly requestId?: string
  readonly retryable: boolean

  constructor(input: {
    status: number
    code: string
    message: string
    requestId?: string
    retryable?: boolean
  }) {
    super(input.message)
    this.name = "AppError"
    this.status = input.status
    this.code = input.code
    this.requestId = input.requestId
    this.retryable =
      input.retryable ?? (input.status >= 500 || input.status === 429)
  }
}

export function createAppError(input: {
  status: number
  body: unknown
  fallbackMessage?: string
}) {
  const parsed = ProblemDetailsSchema.safeParse(input.body)
  const status = parsed.success
    ? (parsed.data.status ?? input.status)
    : input.status
  const message = parsed.success
    ? (parsed.data.detail ?? parsed.data.title ?? input.fallbackMessage)
    : input.fallbackMessage

  return new AppError({
    status,
    code: parsed.success ? (parsed.data.type ?? "HTTP_ERROR") : "HTTP_ERROR",
    message: message ?? "Request failed",
    requestId: parsed.success ? parsed.data.instance : undefined,
  })
}

export function isTerminalClientError(error: unknown) {
  return error instanceof AppError && error.status >= 400 && !error.retryable
}
