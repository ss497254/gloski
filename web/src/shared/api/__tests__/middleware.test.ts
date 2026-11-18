import { describe, test, expect, mock } from 'bun:test'
import { withRetry, paginate } from '../middleware'
import { GloskiError } from '../errors'
import type { MiddlewareRequest, MiddlewareNext, Result } from '../types'

function makeRequest(method = 'GET'): MiddlewareRequest {
  return { url: 'http://localhost/api/test', method, headers: {} }
}

function okResponse(): Response {
  return new Response('ok', { status: 200 })
}

function errorResponse(status: number, headers?: Record<string, string>): Response {
  return new Response('error', { status, headers })
}

describe('withRetry', () => {
  test('passes through successful responses', async () => {
    const middleware = withRetry()
    const next = mock(() => Promise.resolve(okResponse()))

    const res = await middleware(makeRequest(), next)

    expect(res.status).toBe(200)
    expect(next).toHaveBeenCalledTimes(1)
  })

  test('retries on 500 for GET', async () => {
    const middleware = withRetry({ maxRetries: 2, delay: 1 })
    let calls = 0
    const next: MiddlewareNext = async () => {
      calls++
      if (calls < 3) return errorResponse(500)
      return okResponse()
    }

    const res = await middleware(makeRequest(), next)

    expect(res.status).toBe(200)
    expect(calls).toBe(3)
  })

  test('returns error response after exhausting retries', async () => {
    const middleware = withRetry({ maxRetries: 2, delay: 1 })
    const next: MiddlewareNext = async () => errorResponse(500)

    const res = await middleware(makeRequest(), next)

    expect(res.status).toBe(500)
  })

  test('does NOT retry POST on 500 (unsafe method)', async () => {
    const middleware = withRetry({ maxRetries: 2, delay: 1 })
    const next = mock(() => Promise.resolve(errorResponse(500)))

    const res = await middleware(makeRequest('POST'), next)

    expect(res.status).toBe(500)
    expect(next).toHaveBeenCalledTimes(1)
  })

  test('retries POST on 429 (safe to retry)', async () => {
    const middleware = withRetry({ maxRetries: 1, delay: 1 })
    let calls = 0
    const next: MiddlewareNext = async () => {
      calls++
      if (calls === 1) return errorResponse(429)
      return okResponse()
    }

    const res = await middleware(makeRequest('POST'), next)

    expect(res.status).toBe(200)
    expect(calls).toBe(2)
  })

  test('retries POST on 503 (safe to retry)', async () => {
    const middleware = withRetry({ maxRetries: 1, delay: 1 })
    let calls = 0
    const next: MiddlewareNext = async () => {
      calls++
      if (calls === 1) return errorResponse(503)
      return okResponse()
    }

    const res = await middleware(makeRequest('POST'), next)

    expect(res.status).toBe(200)
    expect(calls).toBe(2)
  })

  test('retries on network error', async () => {
    const middleware = withRetry({ maxRetries: 1, delay: 1 })
    let calls = 0
    const next: MiddlewareNext = async () => {
      calls++
      if (calls === 1) throw new TypeError('Failed to fetch')
      return okResponse()
    }

    const res = await middleware(makeRequest(), next)

    expect(res.status).toBe(200)
    expect(calls).toBe(2)
  })

  test('throws after exhausting network error retries', async () => {
    const middleware = withRetry({ maxRetries: 1, delay: 1, retryOnNetworkError: true })
    const next: MiddlewareNext = async () => {
      throw new TypeError('Failed to fetch')
    }

    expect(middleware(makeRequest(), next)).rejects.toThrow('Failed to fetch')
  })

  test('does not retry network error when retryOnNetworkError is false', async () => {
    const middleware = withRetry({ retryOnNetworkError: false, delay: 1 })
    const next = mock(async () => {
      throw new TypeError('Failed to fetch')
    }) as unknown as MiddlewareNext

    expect(middleware(makeRequest(), next)).rejects.toThrow('Failed to fetch')
    // wait for the rejection to settle
    await new Promise((r) => setTimeout(r, 10))
    expect(next).toHaveBeenCalledTimes(1)
  })

  test('respects Retry-After header (seconds)', async () => {
    const middleware = withRetry({ maxRetries: 1, delay: 10000 })
    let calls = 0
    const start = Date.now()
    const next: MiddlewareNext = async () => {
      calls++
      if (calls === 1) return errorResponse(429, { 'Retry-After': '0' })
      return okResponse()
    }

    const res = await middleware(makeRequest(), next)
    const elapsed = Date.now() - start

    expect(res.status).toBe(200)
    // Retry-After: 0 should be near-instant (< 100ms), not the 10s default
    expect(elapsed).toBeLessThan(100)
  })

  test('custom retryOn status codes', async () => {
    const middleware = withRetry({ maxRetries: 1, delay: 1, retryOn: [418] })
    let calls = 0
    const next: MiddlewareNext = async () => {
      calls++
      if (calls === 1) return errorResponse(418)
      return okResponse()
    }

    const res = await middleware(makeRequest(), next)

    expect(res.status).toBe(200)
    expect(calls).toBe(2)
  })

  test('does not retry non-matching status codes', async () => {
    const middleware = withRetry({ maxRetries: 2, delay: 1 })
    const next = mock(() => Promise.resolve(errorResponse(404)))

    const res = await middleware(makeRequest(), next)

    expect(res.status).toBe(404)
    expect(next).toHaveBeenCalledTimes(1)
  })
})

describe('paginate', () => {
  function mockFetcher(pages: number[][]): (limit: number, offset: number) => Promise<Result<number[]>> {
    let callIdx = 0
    return async () => {
      const page = pages[callIdx++] ?? []
      return { data: page, error: null }
    }
  }

  test('yields all pages until a short page', async () => {
    const fetcher = mockFetcher([[1, 2], [3, 4], [5]])
    const allPages: number[][] = []

    for await (const page of paginate(fetcher, { limit: 2 })) {
      allPages.push(page)
    }

    expect(allPages).toEqual([[1, 2], [3, 4], [5]])
  })

  test('yields single page when results < limit', async () => {
    const fetcher = mockFetcher([[1, 2, 3]])
    const allPages: number[][] = []

    for await (const page of paginate(fetcher, { limit: 10 })) {
      allPages.push(page)
    }

    expect(allPages).toEqual([[1, 2, 3]])
  })

  test('stops on empty page', async () => {
    const fetcher = mockFetcher([[]])
    const allPages: number[][] = []

    for await (const page of paginate(fetcher)) {
      allPages.push(page)
    }

    expect(allPages).toEqual([])
  })

  test('throws on error result', async () => {
    const err = new GloskiError(500, 'Server error')
    const fetcher = async () => ({ data: null, error: err }) as Result<number[]>

    const gen = paginate(fetcher)
    expect(gen.next()).rejects.toThrow('Server error')
  })

  test('respects custom offset', async () => {
    const offsets: number[] = []
    const fetcher = async (_limit: number, offset: number): Promise<Result<number[]>> => {
      offsets.push(offset)
      return { data: offset < 20 ? [1, 2] : [], error: null }
    }

    for await (const page of paginate(fetcher, { limit: 2, offset: 10 })) {
      expect(page).toEqual([1, 2])
    }

    expect(offsets[0]).toBe(10)
    expect(offsets[1]).toBe(12)
  })
})
