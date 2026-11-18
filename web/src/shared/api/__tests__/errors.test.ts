import { describe, expect, test } from 'bun:test'
import { ErrorCode, getErrorMessage, GloskiError, isGloskiError, isNetworkError, safe } from '../errors'

describe('GloskiError', () => {
  test('constructor sets properties correctly', () => {
    const error = new GloskiError(404, 'Not found', 'NOT_FOUND')
    expect(error.status).toBe(404)
    expect(error.message).toBe('Not found')
    expect(error.code).toBe('NOT_FOUND')
    expect(error.name).toBe('GloskiError')
  })

  test('constructor works without code', () => {
    const error = new GloskiError(500, 'Server error')
    expect(error.status).toBe(500)
    expect(error.message).toBe('Server error')
    expect(error.code).toBeUndefined()
  })

  test('extends Error', () => {
    const error = new GloskiError(400, 'Bad request')
    expect(error).toBeInstanceOf(Error)
    expect(error).toBeInstanceOf(GloskiError)
  })

  test('isUnauthorized', () => {
    expect(new GloskiError(401, 'Unauthorized').isUnauthorized).toBe(true)
    expect(new GloskiError(403, 'Forbidden').isUnauthorized).toBe(false)
  })

  test('isForbidden', () => {
    expect(new GloskiError(403, 'Forbidden').isForbidden).toBe(true)
    expect(new GloskiError(401, 'Unauthorized').isForbidden).toBe(false)
  })

  test('isNotFound', () => {
    expect(new GloskiError(404, 'Not found').isNotFound).toBe(true)
    expect(new GloskiError(400, 'Bad request').isNotFound).toBe(false)
  })

  test('isNetworkError', () => {
    expect(new GloskiError(0, 'Network error').isNetworkError).toBe(true)
    expect(new GloskiError(500, 'Server error').isNetworkError).toBe(false)
  })

  test('isServerError', () => {
    expect(new GloskiError(500, 'Internal').isServerError).toBe(true)
    expect(new GloskiError(502, 'Bad gateway').isServerError).toBe(true)
    expect(new GloskiError(599, 'Custom').isServerError).toBe(true)
    expect(new GloskiError(400, 'Bad request').isServerError).toBe(false)
    expect(new GloskiError(600, 'Custom').isServerError).toBe(false)
  })

  test('isClientError', () => {
    expect(new GloskiError(400, 'Bad request').isClientError).toBe(true)
    expect(new GloskiError(499, 'Custom').isClientError).toBe(true)
    expect(new GloskiError(500, 'Server error').isClientError).toBe(false)
    expect(new GloskiError(399, 'Redirect').isClientError).toBe(false)
  })

  test('toJSON() returns serializable object with code', () => {
    const error = new GloskiError(404, 'Not found', 'NOT_FOUND')
    expect(error.toJSON()).toEqual({
      name: 'GloskiError',
      status: 404,
      message: 'Not found',
      code: 'NOT_FOUND',
    })
  })

  test('toJSON() omits code when undefined', () => {
    const error = new GloskiError(500, 'Server error')
    const json = error.toJSON()
    expect(json).toEqual({ name: 'GloskiError', status: 500, message: 'Server error' })
    expect('code' in json).toBe(false)
  })

  test('JSON.stringify uses toJSON()', () => {
    const error = new GloskiError(401, 'Unauthorized')
    const parsed = JSON.parse(JSON.stringify(error))
    expect(parsed.name).toBe('GloskiError')
    expect(parsed.status).toBe(401)
    expect(parsed.message).toBe('Unauthorized')
  })
})

describe('isGloskiError', () => {
  test('returns true for GloskiError', () => {
    expect(isGloskiError(new GloskiError(404, 'Not found'))).toBe(true)
  })

  test('returns false for plain Error', () => {
    expect(isGloskiError(new Error('oops'))).toBe(false)
  })

  test('returns false for non-errors', () => {
    expect(isGloskiError('string')).toBe(false)
    expect(isGloskiError(null)).toBe(false)
    expect(isGloskiError(undefined)).toBe(false)
  })
})

describe('isNetworkError', () => {
  test('returns true for network errors (status 0)', () => {
    expect(isNetworkError(new GloskiError(0, 'Network error'))).toBe(true)
  })

  test('returns false for HTTP errors', () => {
    expect(isNetworkError(new GloskiError(500, 'Server error'))).toBe(false)
  })

  test('returns false for non-GloskiError', () => {
    expect(isNetworkError(new Error('oops'))).toBe(false)
  })
})

describe('ErrorCode', () => {
  test('contains standard HTTP codes', () => {
    expect(ErrorCode.NETWORK_ERROR).toBe(0)
    expect(ErrorCode.UNAUTHORIZED).toBe(401)
    expect(ErrorCode.FORBIDDEN).toBe(403)
    expect(ErrorCode.NOT_FOUND).toBe(404)
    expect(ErrorCode.RATE_LIMITED).toBe(429)
    expect(ErrorCode.SERVER_ERROR).toBe(500)
  })

  test('works with GloskiError status comparison', () => {
    const err = new GloskiError(404, 'Not found')
    expect(err.status === ErrorCode.NOT_FOUND).toBe(true)
    expect(err.status === ErrorCode.UNAUTHORIZED).toBe(false)
  })
})

describe('getErrorMessage', () => {
  test('handles TypeError with "Failed to fetch"', () => {
    const error = new TypeError('Failed to fetch')
    expect(getErrorMessage(error)).toContain('Cannot connect to server')
  })

  test('handles TypeError with "NetworkError"', () => {
    const error = new TypeError('NetworkError when attempting to fetch')
    expect(getErrorMessage(error)).toContain('Network error')
  })

  test('handles generic TypeError', () => {
    const error = new TypeError('Something else')
    expect(getErrorMessage(error)).toContain('Network error: Something else')
  })

  test('handles AbortError', () => {
    const error = new DOMException('The operation was aborted', 'AbortError')
    expect(getErrorMessage(error)).toContain('timed out')
  })

  test('handles generic Error', () => {
    const error = new Error('Something went wrong')
    expect(getErrorMessage(error)).toBe('Something went wrong')
  })

  test('handles non-Error values', () => {
    expect(getErrorMessage('string error')).toBe('An unexpected error occurred')
    expect(getErrorMessage(null)).toBe('An unexpected error occurred')
    expect(getErrorMessage(undefined)).toBe('An unexpected error occurred')
  })
})

describe('safe', () => {
  test('returns data on success', async () => {
    const result = await safe(Promise.resolve(42))
    expect(result).toEqual({ data: 42, error: null })
  })

  test('returns GloskiError on GloskiError throw', async () => {
    const err = new GloskiError(404, 'Not found')
    const result = await safe(Promise.reject(err))
    expect(result.data).toBeNull()
    expect(result.error).toBe(err)
  })

  test('wraps unknown errors in GloskiError with status 0', async () => {
    const result = await safe(Promise.reject(new Error('boom')))
    expect(result.data).toBeNull()
    expect(result.error).toBeInstanceOf(GloskiError)
    expect(result.error!.status).toBe(0)
    expect(result.error!.message).toBe('boom')
  })

  test('wraps non-Error throws in GloskiError', async () => {
    const result = await safe(Promise.reject('string error'))
    expect(result.data).toBeNull()
    expect(result.error).toBeInstanceOf(GloskiError)
    expect(result.error!.status).toBe(0)
    expect(result.error!.message).toBe('An unexpected error occurred')
  })
})
