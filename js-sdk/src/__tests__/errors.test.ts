import { describe, test, expect } from 'bun:test'
import { GloskiError, getErrorMessage } from '../errors'

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
