/**
 * Error thrown by Gloski SDK for API errors
 */
export class GloskiError extends Error {
  /** HTTP status code (0 for network errors) */
  readonly status: number
  /** Error code from device (if provided) */
  readonly code?: string

  constructor(status: number, message: string, code?: string) {
    super(message)
    this.name = 'GloskiError'
    this.status = status
    this.code = code

    // Maintains proper stack trace in V8 environments
    const ErrorWithCapture = Error as typeof Error & {
      captureStackTrace?: (error: Error, constructor?: object) => void
    }
    ErrorWithCapture.captureStackTrace?.(this, GloskiError)
  }

  /** Returns true if the error is a 401 Unauthorized */
  get isUnauthorized(): boolean {
    return this.status === 401
  }

  /** Returns true if the error is a 403 Forbidden */
  get isForbidden(): boolean {
    return this.status === 403
  }

  /** Returns true if the error is a 404 Not Found */
  get isNotFound(): boolean {
    return this.status === 404
  }

  /** Returns true if the error is a network error (status 0) */
  get isNetworkError(): boolean {
    return this.status === 0
  }

  /** Returns true if the error is a device error (5xx) */
  get isServerError(): boolean {
    return this.status >= 500 && this.status < 600
  }

  /** Returns true if the error is a client error (4xx) */
  get isClientError(): boolean {
    return this.status >= 400 && this.status < 500
  }

  /** Serialize to a plain object (used by JSON.stringify) */
  toJSON(): { name: string; status: number; message: string; code?: string } {
    return {
      name: this.name,
      status: this.status,
      message: this.message,
      ...(this.code !== undefined && { code: this.code }),
    }
  }
}

/**
 * Wraps a promise into a Result, catching errors into a discriminated union
 */
export async function safe<T>(promise: Promise<T>): Promise<import('./types').Result<T>> {
  try {
    return { data: await promise, error: null }
  } catch (error) {
    if (error instanceof GloskiError) return { data: null, error }
    return { data: null, error: new GloskiError(0, getErrorMessage(error)) }
  }
}

/**
 * Unwrap a Result, returning the data or throwing the error.
 * Useful in the web app where errors are handled by try/catch.
 */
export function unwrap<T>(result: import('./types').Result<T>): T {
  if (result.error) throw result.error
  return result.data
}

/**
 * Well-known HTTP error status codes for use with GloskiError.status
 */
export const ErrorCode = {
  NETWORK_ERROR: 0,
  UNAUTHORIZED: 401,
  FORBIDDEN: 403,
  NOT_FOUND: 404,
  CONFLICT: 409,
  RATE_LIMITED: 429,
  SERVER_ERROR: 500,
  BAD_GATEWAY: 502,
  SERVICE_UNAVAILABLE: 503,
  GATEWAY_TIMEOUT: 504,
} as const

export type ErrorCode = (typeof ErrorCode)[keyof typeof ErrorCode]

/**
 * Type guard: checks if an unknown error is a GloskiError
 */
export function isGloskiError(error: unknown): error is GloskiError {
  return error instanceof GloskiError
}

/**
 * Type guard: checks if an unknown error is a GloskiError caused by a network failure (status 0)
 */
export function isNetworkError(error: unknown): error is GloskiError {
  return error instanceof GloskiError && error.status === 0
}

/**
 * Get a user-friendly error message from various error types
 */
export function getErrorMessage(error: unknown): string {
  if (error instanceof TypeError) {
    // Network errors (CORS, DNS, connection refused, etc.)
    if (error.message.includes('Failed to fetch')) {
      return 'Cannot connect to device. Please check that the device is running and the URL is correct.'
    }
    if (error.message.includes('NetworkError')) {
      return 'Network error. Check your internet connection.'
    }
    return `Network error: ${error.message}`
  }

  if (error instanceof Error) {
    if (error.name === 'AbortError') {
      return 'Request timed out. The device may be slow or unreachable.'
    }
    return error.message
  }

  return 'An unexpected error occurred'
}
