import type { Middleware, MiddlewareRequest, Result } from './types'

export interface RetryOptions {
  /** Maximum number of retry attempts (default: 3) */
  maxRetries?: number
  /** Initial delay in milliseconds (default: 1000) */
  delay?: number
  /** Maximum delay in milliseconds (default: 10000) */
  maxDelay?: number
  /** HTTP status codes that trigger a retry (default: [429, 500, 502, 503, 504]) */
  retryOn?: number[]
  /** Retry on network errors / status 0 (default: true) */
  retryOnNetworkError?: boolean
}

const DEFAULT_RETRY_ON = [429, 500, 502, 503, 504]
const SAFE_TO_RETRY_METHODS = new Set(['GET', 'HEAD', 'OPTIONS'])
// These status codes indicate the device didn't process the request, so mutations are safe to retry
const SAFE_TO_RETRY_STATUSES = new Set([429, 503])

/**
 * Creates a retry middleware with exponential backoff.
 *
 * - GET/HEAD/OPTIONS requests are retried for all configured status codes.
 * - POST/PUT/DELETE are only retried for 429 and 503 (device didn't process the request).
 * - Network errors (fetch throws) are retried for all methods.
 * - Respects the `Retry-After` header when present.
 *
 * @example
 * ```ts
 * const client = new GloskiClient({ url: '...', apiKey: '...' })
 * client.use(withRetry({ maxRetries: 3, delay: 1000 }))
 * ```
 */
export function withRetry(options?: RetryOptions): Middleware {
  const maxRetries = options?.maxRetries ?? 3
  const baseDelay = options?.delay ?? 1000
  const maxDelay = options?.maxDelay ?? 10000
  const retryOn = new Set(options?.retryOn ?? DEFAULT_RETRY_ON)
  const retryOnNetworkError = options?.retryOnNetworkError ?? true

  return async (request: MiddlewareRequest, next) => {
    let lastError: unknown
    const isSafeMethod = SAFE_TO_RETRY_METHODS.has(request.method)

    for (let attempt = 0; attempt <= maxRetries; attempt++) {
      try {
        const response = await next(request)

        if (retryOn.has(response.status) && attempt < maxRetries) {
          // For unsafe methods, only retry device-level rejections
          if (!isSafeMethod && !SAFE_TO_RETRY_STATUSES.has(response.status)) {
            return response
          }

          const retryAfter = parseRetryAfter(response.headers.get('Retry-After'))
          const delay = retryAfter ?? computeDelay(attempt, baseDelay, maxDelay)
          await sleep(delay)
          continue
        }

        return response
      } catch (error) {
        lastError = error

        if (!retryOnNetworkError || attempt >= maxRetries) {
          throw error
        }

        const delay = computeDelay(attempt, baseDelay, maxDelay)
        await sleep(delay)
      }
    }

    throw lastError
  }
}

function computeDelay(attempt: number, baseDelay: number, maxDelay: number): number {
  const exponential = baseDelay * Math.pow(2, attempt)
  const jitter = Math.random() * baseDelay * 0.5
  return Math.min(exponential + jitter, maxDelay)
}

function parseRetryAfter(value: string | null): number | null {
  if (!value) return null

  // Try as seconds
  const seconds = Number(value)
  if (!isNaN(seconds)) return seconds * 1000

  // Try as HTTP date
  const date = Date.parse(value)
  if (!isNaN(date)) return Math.max(0, date - Date.now())

  return null
}

function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms))
}

// =============================================================================
// Pagination Helper
// =============================================================================

export interface PaginateOptions {
  /** Items per page (default: 100) */
  limit?: number
  /** Starting offset (default: 0) */
  offset?: number
}

/**
 * Generic pagination helper using async generators.
 * Yields pages of results until a page returns fewer items than `limit`.
 *
 * @param fetcher - Function that fetches a page given (limit, offset)
 * @param options - Pagination options
 *
 * @example
 * ```ts
 * // Iterate over all installed packages in pages of 50
 * for await (const page of paginate(
 *   (limit, offset) => client.packages.listInstalled(limit),
 *   { limit: 50 }
 * )) {
 *   console.log(`Got ${page.length} packages`)
 * }
 * ```
 */
export async function* paginate<T>(
  fetcher: (limit: number, offset: number) => Promise<Result<T[]>>,
  options?: PaginateOptions
): AsyncGenerator<T[], void, unknown> {
  const limit = options?.limit ?? 100
  let offset = options?.offset ?? 0

  while (true) {
    const result = await fetcher(limit, offset)
    if (result.error) throw result.error
    if (result.data.length === 0) return

    yield result.data
    if (result.data.length < limit) return
    offset += limit
  }
}
