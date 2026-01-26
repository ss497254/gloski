/**
 * Error thrown by Gloski SDK for API errors
 */
export class GloskiError extends Error {
  /** HTTP status code (0 for network errors) */
  readonly status: number
  /** Error code from server (if provided) */
  readonly code?: string

  constructor(status: number, message: string, code?: string) {
    super(message)
    this.name = 'GloskiError'
    this.status = status
    this.code = code

    // Maintains proper stack trace in V8 environments
    const ErrorWithCapture = Error as typeof Error & {
      captureStackTrace?: (error: Error, constructor: Function) => void
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

  /** Returns true if the error is a server error (5xx) */
  get isServerError(): boolean {
    return this.status >= 500 && this.status < 600
  }

  /** Returns true if the error is a client error (4xx) */
  get isClientError(): boolean {
    return this.status >= 400 && this.status < 500
  }
}

/**
 * Get a user-friendly error message from various error types
 */
export function getErrorMessage(error: unknown): string {
  if (error instanceof TypeError) {
    // Network errors (CORS, DNS, connection refused, etc.)
    if (error.message.includes('Failed to fetch')) {
      return 'Cannot connect to server. Please check that the server is running and the URL is correct.'
    }
    if (error.message.includes('NetworkError')) {
      return 'Network error. Check your internet connection.'
    }
    return `Network error: ${error.message}`
  }

  if (error instanceof Error) {
    if (error.name === 'AbortError') {
      return 'Request timed out. The server may be slow or unreachable.'
    }
    return error.message
  }

  return 'An unexpected error occurred'
}
