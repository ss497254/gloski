import type { GloskiClientConfig } from './types'
import { GloskiError, getErrorMessage } from './errors'

const DEFAULT_TIMEOUT = 30000
const DEFAULT_API_PREFIX = '/api'

export interface RequestOptions {
  method?: 'GET' | 'POST' | 'PUT' | 'DELETE'
  body?: unknown
  headers?: Record<string, string>
  timeout?: number
}

export interface ClientCallbacks {
  onUnauthorized?: () => void
  onOffline?: () => void
  onOnline?: () => void
}

/**
 * HTTP client for making API requests
 */
export class HttpClient {
  private wasOffline = false
  private config: GloskiClientConfig
  private callbacks: ClientCallbacks
  private apiPrefix: string

  constructor(config: GloskiClientConfig, callbacks: ClientCallbacks = {}) {
    this.config = config
    this.callbacks = callbacks
    this.apiPrefix = config.apiPrefix ?? DEFAULT_API_PREFIX
  }

  /**
   * Build full endpoint path with API prefix
   */
  private buildEndpoint(path: string): string {
    // If path already starts with the prefix, don't add it again
    if (path.startsWith(this.apiPrefix)) {
      return path
    }
    return `${this.apiPrefix}${path}`
  }

  /**
   * Get the full URL for an endpoint
   */
  getFullUrl(endpoint: string): string {
    return this.buildAuthUrl(endpoint)
  }

  /**
   * GET request
   */
  async get<T>(endpoint: string): Promise<T> {
    return this.request<T>(endpoint, { method: 'GET' })
  }

  /**
   * POST request
   */
  async post<T>(endpoint: string, body: unknown): Promise<T> {
    return this.request<T>(endpoint, { method: 'POST', body })
  }

  /**
   * DELETE request
   */
  async delete<T>(endpoint: string): Promise<T> {
    return this.request<T>(endpoint, { method: 'DELETE' })
  }

  /**
   * Make an authenticated API request
   */
  async request<T>(endpoint: string, options: RequestOptions = {}): Promise<T> {
    const fullEndpoint = this.buildEndpoint(endpoint)
    const url = `${this.config.url}${fullEndpoint}`
    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
      ...options.headers,
    }

    // Add authentication header
    if (this.config.apiKey) {
      headers['X-API-Key'] = this.config.apiKey
    } else if (this.config.token) {
      headers['Authorization'] = `Bearer ${this.config.token}`
    }

    return this.doRequest<T>(url, {
      ...options,
      headers,
    })
  }

  /**
   * Make an unauthenticated API request (for health check, etc.)
   */
  async requestNoAuth<T>(endpoint: string, options: RequestOptions = {}): Promise<T> {
    const fullEndpoint = this.buildEndpoint(endpoint)
    const url = `${this.config.url}${fullEndpoint}`
    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
      ...options.headers,
    }

    return this.doRequest<T>(url, {
      ...options,
      headers,
    }, false)
  }

  /**
   * Upload a file using multipart/form-data
   */
  async upload<T>(endpoint: string, formData: FormData): Promise<T> {
    const fullEndpoint = this.buildEndpoint(endpoint)
    const url = `${this.config.url}${fullEndpoint}`
    const headers: Record<string, string> = {}

    // Add authentication header
    if (this.config.apiKey) {
      headers['X-API-Key'] = this.config.apiKey
    } else if (this.config.token) {
      headers['Authorization'] = `Bearer ${this.config.token}`
    }

    const controller = new AbortController()
    const timeoutId = setTimeout(
      () => controller.abort(),
      this.config.timeout || DEFAULT_TIMEOUT
    )

    try {
      const response = await fetch(url, {
        method: 'POST',
        headers,
        body: formData,
        signal: controller.signal,
      })

      clearTimeout(timeoutId)
      this.handleOnline()

      if (response.status === 401) {
        this.callbacks.onUnauthorized?.()
        throw new GloskiError(401, 'Unauthorized')
      }

      if (!response.ok) {
        const data = await response.json().catch(() => ({ error: 'Unknown error' }))
        throw new GloskiError(response.status, data.error || `HTTP ${response.status}`)
      }

      const json = await response.json()
      return json.data ?? json
    } catch (error) {
      clearTimeout(timeoutId)

      if (error instanceof GloskiError) throw error

      this.handleOffline()
      const message = getErrorMessage(error)
      throw new GloskiError(0, message)
    }
  }

  /**
   * Build an authenticated URL for downloads, WebSockets, etc.
   */
  buildAuthUrl(endpoint: string, params: Record<string, string> = {}): string {
    const fullEndpoint = this.buildEndpoint(endpoint)
    const url = new URL(fullEndpoint, this.config.url)

    // Add auth param
    if (this.config.apiKey) {
      url.searchParams.set('api_key', this.config.apiKey)
    } else if (this.config.token) {
      url.searchParams.set('token', this.config.token)
    }

    // Add additional params
    for (const [key, value] of Object.entries(params)) {
      url.searchParams.set(key, value)
    }

    return url.toString()
  }

  /**
   * Build a WebSocket URL with authentication
   */
  buildWebSocketUrl(endpoint: string, params: Record<string, string> = {}): string {
    const httpUrl = this.buildAuthUrl(endpoint, params)
    return httpUrl.replace(/^http/, 'ws')
  }

  /**
   * Internal request method
   */
  private async doRequest<T>(
    url: string,
    options: RequestOptions & { headers: Record<string, string> },
    handleAuthErrors = true
  ): Promise<T> {
    const controller = new AbortController()
    const timeoutId = setTimeout(
      () => controller.abort(),
      options.timeout || this.config.timeout || DEFAULT_TIMEOUT
    )

    try {
      const response = await fetch(url, {
        method: options.method || 'GET',
        headers: options.headers,
        body: options.body ? JSON.stringify(options.body) : undefined,
        signal: controller.signal,
      })

      clearTimeout(timeoutId)
      this.handleOnline()

      if (response.status === 401 && handleAuthErrors) {
        this.callbacks.onUnauthorized?.()
        throw new GloskiError(401, 'Unauthorized')
      }

      if (!response.ok) {
        let errorMessage = `HTTP ${response.status}`
        try {
          const data = await response.json()
          errorMessage = data.error || data.message || errorMessage
        } catch {
          if (response.statusText) {
            errorMessage = `${response.status} ${response.statusText}`
          }
        }

        // Add context for common errors
        if (response.status === 404) {
          errorMessage = `Endpoint not found (404). Make sure you're connecting to a Gloski server.`
        } else if (response.status === 403) {
          errorMessage = 'Access denied'
        } else if (response.status >= 500) {
          errorMessage = `Server error (${response.status}). The server may be experiencing issues.`
        }

        throw new GloskiError(response.status, errorMessage)
      }

      return response.json()
    } catch (error) {
      clearTimeout(timeoutId)

      if (error instanceof GloskiError) throw error

      this.handleOffline()
      const message = getErrorMessage(error)
      throw new GloskiError(0, message)
    }
  }

  /**
   * Handle server coming back online
   */
  private handleOnline(): void {
    if (this.wasOffline) {
      this.wasOffline = false
      this.callbacks.onOnline?.()
    }
  }

  /**
   * Handle server going offline
   */
  private handleOffline(): void {
    if (!this.wasOffline) {
      this.wasOffline = true
      this.callbacks.onOffline?.()
    }
  }
}

/**
 * Make a standalone health check request (no client needed)
 */
export async function checkHealth(
  url: string,
  timeout = 5000,
  apiPrefix = DEFAULT_API_PREFIX
): Promise<{ status: string; version?: string; features?: Record<string, boolean> }> {
  const controller = new AbortController()
  const timeoutId = setTimeout(() => controller.abort(), timeout)

  try {
    const response = await fetch(`${url}${apiPrefix}/health`, {
      method: 'GET',
      headers: { 'Content-Type': 'application/json' },
      signal: controller.signal,
    })

    clearTimeout(timeoutId)

    if (!response.ok) {
      throw new GloskiError(response.status, `HTTP ${response.status}`)
    }

    return response.json()
  } catch (error) {
    clearTimeout(timeoutId)

    if (error instanceof GloskiError) throw error

    const message = getErrorMessage(error)
    throw new GloskiError(0, message)
  }
}
