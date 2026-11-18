import { GloskiError, getErrorMessage } from './errors'
import type { GloskiClientConfig, HealthResponse, Middleware, MiddlewareNext, MiddlewareRequest } from './types'

const DEFAULT_TIMEOUT = 30000
const DEFAULT_API_PREFIX = '/api'

export interface RequestOptions {
  method?: 'GET' | 'POST' | 'PUT' | 'DELETE'
  body?: unknown
  headers?: Record<string, string>
  timeout?: number
  signal?: AbortSignal
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
  private isDisconnected = false
  private config: GloskiClientConfig
  private callbacks: ClientCallbacks
  private apiPrefix: string
  private pendingRequests: Map<string, Promise<unknown>> = new Map()
  private customFetch?: typeof globalThis.fetch
  private middlewares: Middleware[] = []

  constructor(config: GloskiClientConfig, callbacks: ClientCallbacks = {}) {
    this.config = config
    this.callbacks = callbacks
    this.apiPrefix = config.apiPrefix ?? DEFAULT_API_PREFIX
    this.customFetch = config.fetch
  }

  /**
   * Add a request middleware
   */
  use(middleware: Middleware): void {
    this.middlewares.push(middleware)
  }

  /**
   * Get the fetch function (custom or globalThis.fetch)
   */
  getFetch(): typeof globalThis.fetch {
    return this.customFetch ?? globalThis.fetch
  }

  /**
   * Dispose the HTTP client and clear pending requests
   */
  dispose(): void {
    this.pendingRequests.clear()
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
  async get<T>(endpoint: string, options?: RequestOptions): Promise<T> {
    return this.request<T>(endpoint, { ...options, method: 'GET' })
  }

  /**
   * POST request
   */
  async post<T>(endpoint: string, body: unknown, options?: RequestOptions): Promise<T> {
    return this.request<T>(endpoint, { ...options, method: 'POST', body })
  }

  /**
   * PUT request
   */
  async put<T>(endpoint: string, body: unknown, options?: RequestOptions): Promise<T> {
    return this.request<T>(endpoint, { ...options, method: 'PUT', body })
  }

  /**
   * DELETE request
   */
  async delete<T>(endpoint: string, options?: RequestOptions): Promise<T> {
    return this.request<T>(endpoint, { ...options, method: 'DELETE' })
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

    return this.doRequest<T>(
      url,
      {
        ...options,
        headers,
      },
      false
    )
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
    const timeoutId = setTimeout(() => controller.abort(), this.config.timeout || DEFAULT_TIMEOUT)

    try {
      const middlewareRequest: MiddlewareRequest = {
        url,
        method: 'POST',
        headers,
        body: formData,
        signal: controller.signal,
      }
      const response = await this.executeWithMiddleware(middlewareRequest)

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
   * Upload a file with progress tracking using XMLHttpRequest.
   * Falls back to regular upload() if XMLHttpRequest is not available.
   */
  async uploadWithProgress<T>(
    endpoint: string,
    formData: FormData,
    onProgress: (loaded: number, total: number | null) => void
  ): Promise<T> {
    if (typeof XMLHttpRequest === 'undefined') {
      return this.upload<T>(endpoint, formData)
    }

    const fullEndpoint = this.buildEndpoint(endpoint)
    const url = `${this.config.url}${fullEndpoint}`

    return new Promise<T>((resolve, reject) => {
      const xhr = new XMLHttpRequest()
      xhr.open('POST', url)

      // Add authentication header
      if (this.config.apiKey) {
        xhr.setRequestHeader('X-API-Key', this.config.apiKey)
      } else if (this.config.token) {
        xhr.setRequestHeader('Authorization', `Bearer ${this.config.token}`)
      }

      const timeoutId = setTimeout(() => {
        xhr.abort()
        reject(new GloskiError(0, 'Request timeout'))
      }, this.config.timeout || DEFAULT_TIMEOUT)

      xhr.upload.onprogress = (event) => {
        onProgress(event.loaded, event.lengthComputable ? event.total : null)
      }

      xhr.onload = () => {
        clearTimeout(timeoutId)
        this.handleOnline()

        if (xhr.status === 401) {
          this.callbacks.onUnauthorized?.()
          reject(new GloskiError(401, 'Unauthorized'))
          return
        }

        if (xhr.status < 200 || xhr.status >= 300) {
          try {
            const data = JSON.parse(xhr.responseText)
            reject(new GloskiError(xhr.status, data.error || `HTTP ${xhr.status}`))
          } catch {
            reject(new GloskiError(xhr.status, `HTTP ${xhr.status}`))
          }
          return
        }

        try {
          const json = JSON.parse(xhr.responseText)
          resolve((json.data ?? json) as T)
        } catch {
          reject(new GloskiError(0, 'Invalid JSON response'))
        }
      }

      xhr.onerror = () => {
        clearTimeout(timeoutId)
        this.handleOffline()
        reject(new GloskiError(0, 'Network error'))
      }

      xhr.ontimeout = () => {
        clearTimeout(timeoutId)
        reject(new GloskiError(0, 'Request timeout'))
      }

      xhr.send(formData)
    })
  }

  /**
   * Build an authenticated URL for downloads, WebSockets, etc.
   */
  buildAuthUrl(endpoint: string, params: Record<string, string> = {}): string {
    const fullEndpoint = this.buildEndpoint(endpoint)
    const url = new URL(this.config.url)

    // Ensure there's a slash between the old path and the new one
    // The URL object will automatically handle the formatting
    url.pathname = `${url.pathname}/${fullEndpoint}`.replace(/\/+/g, '/')

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
   * Build a middleware chain that ends with the given handler
   */
  private buildMiddlewareChain(finalHandler: MiddlewareNext): MiddlewareNext {
    let handler = finalHandler
    for (let i = this.middlewares.length - 1; i >= 0; i--) {
      const mw = this.middlewares[i]
      const next = handler
      handler = (req) => mw(req, next)
    }
    return handler
  }

  /**
   * Execute a fetch call through the middleware chain
   */
  private async executeWithMiddleware(request: MiddlewareRequest): Promise<Response> {
    const fetchFn = this.getFetch()
    const finalHandler: MiddlewareNext = (req) =>
      fetchFn(req.url, {
        method: req.method,
        headers: req.headers,
        body: req.body,
        signal: req.signal,
      })

    const chain = this.buildMiddlewareChain(finalHandler)
    return chain(request)
  }

  /**
   * Generate cache key for request deduplication
   */
  private getCacheKey(url: string, method: string, body?: unknown): string {
    const bodyStr = body ? JSON.stringify(body) : ''
    return `${method}:${url}:${bodyStr}`
  }

  /**
   * Internal request method
   */
  private async doRequest<T>(
    url: string,
    options: RequestOptions & { headers: Record<string, string> },
    handleAuthErrors = true
  ): Promise<T> {
    const method = options.method || 'GET'
    const cacheKey = this.getCacheKey(url, method, options.body)

    // Check if this request is already in flight (deduplication for GET only)
    // POST/PUT/DELETE are mutations and must not be deduplicated
    if (method === 'GET' && this.pendingRequests.has(cacheKey)) {
      return this.pendingRequests.get(cacheKey) as Promise<T>
    }

    const timeoutController = new AbortController()
    const timeoutId = setTimeout(
      () => timeoutController.abort(),
      options.timeout || this.config.timeout || DEFAULT_TIMEOUT
    )

    // Combine external signal with timeout signal
    let signal: AbortSignal = timeoutController.signal
    if (options.signal) {
      // Use AbortSignal.any() if available (modern browsers), otherwise use external signal priority
      if ('any' in AbortSignal && typeof AbortSignal.any === 'function') {
        signal = AbortSignal.any([options.signal, timeoutController.signal])
      } else {
        // Fallback: prefer external signal if provided
        signal = options.signal
        // Listen to external signal to clear timeout
        options.signal.addEventListener('abort', () => {
          clearTimeout(timeoutId)
          timeoutController.abort()
        })
      }
    }

    // Create the request promise and cache it
    const requestPromise = (async () => {
      try {
        const middlewareRequest: MiddlewareRequest = {
          url,
          method,
          headers: options.headers,
          body: options.body ? JSON.stringify(options.body) : undefined,
          signal,
        }
        const response = await this.executeWithMiddleware(middlewareRequest)

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

          // Only use generic messages as fallback if device didn't provide one
          if (errorMessage === `HTTP ${response.status}`) {
            if (response.status === 404) {
              errorMessage = `Endpoint not found (404). Make sure you're connecting to a Gloski device.`
            } else if (response.status === 403) {
              errorMessage = 'Access denied'
            } else if (response.status >= 500) {
              errorMessage = `Server error (${response.status}). The device may be experiencing issues.`
            }
          }

          throw new GloskiError(response.status, errorMessage)
        }

        const json = await response.json()
        // Unwrap the standard { success, data } envelope if present
        return json.data !== undefined ? json.data : json
      } catch (error) {
        clearTimeout(timeoutId)

        if (error instanceof GloskiError) throw error

        this.handleOffline()
        const message = getErrorMessage(error)
        throw new GloskiError(0, message)
      } finally {
        // Remove from cache when request completes (success or failure)
        this.pendingRequests.delete(cacheKey)
      }
    })()

    // Cache the promise
    this.pendingRequests.set(cacheKey, requestPromise)

    return requestPromise as Promise<T>
  }

  /**
   * Handle device coming back online
   */
  private handleOnline(): void {
    if (this.isDisconnected) {
      this.isDisconnected = false
      this.callbacks.onOnline?.()
    }
  }

  /**
   * Handle device going offline
   */
  private handleOffline(): void {
    if (!this.isDisconnected) {
      this.isDisconnected = true
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
): Promise<HealthResponse> {
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
