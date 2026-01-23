import { useServersStore, type Server } from '@/stores/servers'
import type {
  LoginResponse,
  AuthStatus,
  SystemStats,
  ProcessInfo,
  ListResponse,
  ReadResponse,
  SearchResult,
  Task,
  TasksResponse,
  TaskLogsResponse,
  SystemdResponse,
  SystemdLogsResponse,
  APIResponse,
} from '@/lib/types'

// =============================================================================
// API Client
// =============================================================================

const API_TIMEOUT = 30000

export class APIError extends Error {
  status: number

  constructor(status: number, message: string) {
    super(message)
    this.status = status
    this.name = 'APIError'
  }

  get isUnauthorized() {
    return this.status === 401
  }

  get isForbidden() {
    return this.status === 403
  }

  get isNotFound() {
    return this.status === 404
  }
}

interface RequestOptions {
  method?: string
  body?: unknown
  headers?: Record<string, string>
  timeout?: number
}

/**
 * Make an API request to a server
 * Uses API key if available, otherwise uses JWT token
 */
async function request<T>(
  server: Server,
  endpoint: string,
  options: RequestOptions = {}
): Promise<T> {
  const url = `${server.url}${endpoint}`
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    ...options.headers,
  }

  // Use API key if available, otherwise use JWT token
  if (server.apiKey) {
    headers['X-API-Key'] = server.apiKey
  } else if (server.token) {
    headers['Authorization'] = `Bearer ${server.token}`
  }

  const controller = new AbortController()
  const timeoutId = setTimeout(
    () => controller.abort(),
    options.timeout || API_TIMEOUT
  )

  try {
    const response = await fetch(url, {
      method: options.method || 'GET',
      headers,
      body: options.body ? JSON.stringify(options.body) : undefined,
      signal: controller.signal,
    })

    clearTimeout(timeoutId)

    if (response.status === 401) {
      // Mark server as unauthorized
      useServersStore.getState().updateServer(server.id, { status: 'unauthorized' })
      throw new APIError(401, 'Unauthorized')
    }

    if (!response.ok) {
      const data = await response.json().catch(() => ({ error: 'Unknown error' }))
      throw new APIError(response.status, data.error || `HTTP ${response.status}`)
    }

    return response.json()
  } catch (error) {
    clearTimeout(timeoutId)

    if (error instanceof APIError) throw error

    // Network error - mark server as offline
    useServersStore.getState().updateServer(server.id, { status: 'offline' })

    // Convert to user-friendly error message
    const message = getErrorMessage(error)
    throw new APIError(0, message)
  }
}

/**
 * Get a user-friendly error message from various error types
 */
function getErrorMessage(error: unknown): string {
  if (error instanceof TypeError) {
    // Network errors (CORS, DNS, connection refused, etc.)
    if (error.message.includes('Failed to fetch')) {
      return `Cannot connect to server. Please check:\n• The server is running\n• The URL is correct\n• CORS is enabled on the server`
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

/**
 * Make an unauthenticated request (for health check, login)
 */
async function requestNoAuth<T>(
  serverUrl: string,
  endpoint: string,
  options: RequestOptions = {}
): Promise<T> {
  const url = `${serverUrl}${endpoint}`
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    ...options.headers,
  }

  const controller = new AbortController()
  const timeoutId = setTimeout(
    () => controller.abort(),
    options.timeout || API_TIMEOUT
  )

  try {
    const response = await fetch(url, {
      method: options.method || 'GET',
      headers,
      body: options.body ? JSON.stringify(options.body) : undefined,
      signal: controller.signal,
    })

    clearTimeout(timeoutId)

    if (!response.ok) {
      // Try to get error message from response body
      let errorMessage = `HTTP ${response.status}`
      try {
        const data = await response.json()
        errorMessage = data.error || data.message || errorMessage
      } catch {
        // Response wasn't JSON, use status text
        if (response.statusText) {
          errorMessage = `${response.status} ${response.statusText}`
        }
      }

      // Add more context for common errors
      if (response.status === 404) {
        errorMessage = `Endpoint not found (404). Make sure you're connecting to a Gloski server.`
      } else if (response.status === 401) {
        errorMessage = 'Authentication required'
      } else if (response.status === 403) {
        errorMessage = 'Access denied'
      } else if (response.status >= 500) {
        errorMessage = `Server error (${response.status}). The server may be experiencing issues.`
      }

      throw new APIError(response.status, errorMessage)
    }

    return response.json()
  } catch (error) {
    clearTimeout(timeoutId)

    if (error instanceof APIError) throw error

    // Convert to user-friendly error message
    const message = getErrorMessage(error)
    throw new APIError(0, message)
  }
}

// =============================================================================
// Server API Factory
// =============================================================================

export function createServerApi(server: Server) {
  return {
    // Auth
    login: (password: string) =>
      requestNoAuth<LoginResponse>(server.url, '/api/auth/login', {
        method: 'POST',
        body: { password },
      }),

    status: () => request<AuthStatus>(server, '/api/auth/status'),

    // System
    stats: () => request<SystemStats>(server, '/api/system/stats'),

    processes: (limit = 100) =>
      request<{ processes: ProcessInfo[] }>(
        server,
        `/api/system/processes?limit=${limit}`
      ),

    // Files
    listFiles: (path = '/') =>
      request<ListResponse>(server, `/api/files?path=${encodeURIComponent(path)}`),

    readFile: (path: string) =>
      request<ReadResponse>(server, `/api/files/read?path=${encodeURIComponent(path)}`),

    writeFile: (path: string, content: string) =>
      request<APIResponse>(server, '/api/files/write', {
        method: 'POST',
        body: { path, content },
      }),

    mkdir: (path: string) =>
      request<APIResponse>(server, '/api/files/mkdir', {
        method: 'POST',
        body: { path },
      }),

    deleteFile: (path: string) =>
      request<APIResponse>(server, `/api/files?path=${encodeURIComponent(path)}`, {
        method: 'DELETE',
      }),

    uploadFile: async (destPath: string, file: File): Promise<{ filename: string }> => {
      const formData = new FormData()
      formData.append('path', destPath)
      formData.append('file', file)

      const headers: Record<string, string> = {}
      if (server.apiKey) {
        headers['X-API-Key'] = server.apiKey
      } else if (server.token) {
        headers['Authorization'] = `Bearer ${server.token}`
      }

      const response = await fetch(`${server.url}/api/files/upload`, {
        method: 'POST',
        headers,
        body: formData,
      })

      if (response.status === 401) {
        useServersStore.getState().updateServer(server.id, { status: 'unauthorized' })
        throw new APIError(401, 'Unauthorized')
      }

      if (!response.ok) {
        const data = await response.json().catch(() => ({ error: 'Unknown error' }))
        throw new APIError(response.status, data.error || `HTTP ${response.status}`)
      }

      const json = await response.json()
      return json.data
    },

    getDownloadUrl: (path: string): string => {
      const authParam = server.apiKey
        ? `api_key=${server.apiKey}`
        : `token=${server.token}`
      return `${server.url}/api/files/download?path=${encodeURIComponent(path)}&${authParam}`
    },

    // Search
    search: (path: string, query: string, content = false, limit = 100) => {
      const params = new URLSearchParams({ path, q: query, limit: String(limit) })
      if (content) params.set('content', 'true')
      return request<{ results: SearchResult[]; count: number }>(
        server,
        `/api/search?${params}`
      )
    },

    // Tasks
    listTasks: () => request<TasksResponse>(server, '/api/tasks'),

    startTask: (command: string, cwd?: string) =>
      request<Task>(server, '/api/tasks', {
        method: 'POST',
        body: { command, cwd },
      }),

    getTask: (id: string) => request<Task>(server, `/api/tasks/${id}`),

    getTaskLogs: (id: string) =>
      request<TaskLogsResponse>(server, `/api/tasks/${id}/logs`),

    stopTask: (id: string) =>
      request<APIResponse>(server, `/api/tasks/${id}`, {
        method: 'DELETE',
      }),

    // Systemd
    listSystemd: (userMode = true) =>
      request<SystemdResponse>(server, `/api/systemd?user=${userMode}`),

    systemdAction: (unit: string, action: string, userMode = true) =>
      request<APIResponse>(server, `/api/systemd/${unit}/${action}?user=${userMode}`, {
        method: 'POST',
      }),

    systemdLogs: (unit: string, userMode = true, lines = 100) =>
      request<SystemdLogsResponse>(
        server,
        `/api/systemd/${unit}/logs?user=${userMode}&lines=${lines}`
      ),

    // WebSocket URL for terminal
    getTerminalUrl: (cwd?: string): string => {
      const wsUrl = server.url.replace(/^http/, 'ws')
      const authParam = server.apiKey
        ? `api_key=${server.apiKey}`
        : `token=${server.token}`
      const cwdParam = cwd ? `&cwd=${encodeURIComponent(cwd)}` : ''
      return `${wsUrl}/api/terminal?${authParam}${cwdParam}`
    },
  }
}

export type ServerApi = ReturnType<typeof createServerApi>

// =============================================================================
// Health Check (no auth required)
// =============================================================================

export async function checkServerHealth(
  serverUrl: string
): Promise<{ status: string }> {
  return requestNoAuth<{ status: string }>(serverUrl, '/api/health', { timeout: 5000 })
}

// =============================================================================
// Login (no auth required)
// =============================================================================

export async function loginToServer(
  serverUrl: string,
  password: string
): Promise<LoginResponse> {
  return requestNoAuth<LoginResponse>(serverUrl, '/api/auth/login', {
    method: 'POST',
    body: { password },
  })
}
