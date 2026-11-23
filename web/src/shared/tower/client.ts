import type {
  DashboardSummary,
  Deployment,
  DeployRequest,
  Diagnosis,
  DeviceAction,
  FleetDevice,
  SelfTestResult,
  FleetEvent,
  TailnetResponse,
  TowerInfo,
} from './types'

interface Envelope<T> {
  success: boolean
  data: T
  error?: string
}

const logLimit = 4000
const secretKey = /(token|password|secret|api_?key|authorization|credential|private_?key)/i

/** Hide credentials wherever they appear in a logged JSON value. */
function redact(value: unknown, seen = new WeakSet<object>()): unknown {
  if (Array.isArray(value)) {
    if (seen.has(value)) return '[circular]'
    seen.add(value)
    return value.map((item) => redact(item, seen))
  }
  if (value !== null && typeof value === 'object') {
    if (seen.has(value)) return '[circular]'
    seen.add(value)
    return Object.fromEntries(
      Object.entries(value).map(([key, item]) => [key, secretKey.test(key) ? '[redacted]' : redact(item, seen)]),
    )
  }
  return value
}

/** One short line describing a body; uploads and binary data are not printed. */
function describe(body: unknown): string {
  if (body === undefined || body === null) return '-'
  if (body instanceof FormData) return '<form-data>'
  if (body instanceof Blob || body instanceof ArrayBuffer) return '<binary>'
  let value = body
  if (typeof body === 'string') {
    try {
      value = JSON.parse(body)
    } catch {
      // Plain text: log it as-is.
    }
  }
  let text: string
  try {
    text = JSON.stringify(redact(value))
  } catch {
    text = '<unserializable>'
  }
  return text.length > logLimit ? `${text.slice(0, logLimit)}…` : text
}

function requestId(): string {
  return globalThis.crypto?.randomUUID?.() ?? `${Date.now().toString(36)}-${Math.random().toString(36).slice(2)}`
}

/**
 * Every dashboard API call goes through here. The request id is sent to the
 * master and returned in its response header, so the browser console and the
 * master log can be matched line for line.
 */
async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const id = requestId()
  const method = init?.method ?? 'GET'
  const headers = new Headers(init?.headers)
  headers.set('Content-Type', 'application/json')
  headers.set('X-Request-ID', id)
  console.debug(`[gloski] -> ${id} ${method} ${path}`, describe(init?.body))

  const started = performance.now()
  let response: Response
  let payload: Envelope<T> | null
  try {
    response = await fetch(path, { cache: 'no-store', ...init, headers })
    payload = (await response.json().catch(() => null)) as Envelope<T> | null
  } catch (error) {
    console.warn(`[gloski] !! ${id} ${method} ${path} ${Math.round(performance.now() - started)}ms`, error)
    throw error
  }
  const elapsed = `${Math.round(performance.now() - started)}ms`

  if (!response.ok || !payload?.success) {
    const error = new Error(payload?.error || `Tower request failed (${response.status})`)
    console.warn(`[gloski] <- ${id} ${method} ${path} ${response.status} ${elapsed}`, describe(payload))
    throw error
  }
  console.debug(`[gloski] <- ${id} ${method} ${path} ${response.status} ${elapsed}`, describe(payload))
  return payload.data
}

/** Detect whether this origin is a control tower. Returns null when it is not. */
export async function detectTower(): Promise<TowerInfo | null> {
  try {
    // Go through request() so detection has a request id in both logs too.
    // A non-master origin fails the request and is treated as "not a tower".
    const info = await request<TowerInfo>('/api/tower/info')
    return info.mode === 'master' ? info : null
  } catch {
    return null
  }
}

export async function getDashboard(): Promise<DashboardSummary> {
  return request<DashboardSummary>('/api/dashboard')
}

/** Read-only health checks of the master itself. */
export async function getSelfTest(): Promise<SelfTestResult> {
  return request<SelfTestResult>('/api/system/self-test')
}

export async function listDevices(): Promise<FleetDevice[]> {
  const data = await request<{ devices: FleetDevice[] }>('/api/devices')
  return data.devices ?? []
}

export async function deleteDevice(id: string): Promise<void> {
  await request<{ removed: boolean }>(`/api/devices/${encodeURIComponent(id)}`, { method: 'DELETE' })
}

export async function listTailnetPeers(): Promise<TailnetResponse> {
  return request<TailnetResponse>('/api/tailnet/peers')
}

export async function tailnetAvailable(): Promise<boolean> {
  try {
    const result = await listTailnetPeers()
    return result.available && result.peers.length > 0
  } catch {
    return false
  }
}

export async function deployDevice(payload: DeployRequest): Promise<Deployment> {
  return request<Deployment>('/api/devices/deploy', {
    method: 'POST',
    body: JSON.stringify(payload),
  })
}

export async function getDeployment(id: string): Promise<Deployment> {
  return request<Deployment>(`/api/deployments/${encodeURIComponent(id)}`)
}

export async function startDevice(id: string): Promise<{ pid: string; status: string }> {
  return request<{ pid: string; status: string }>(`/api/devices/${encodeURIComponent(id)}/start`, { method: 'POST' })
}

/** Accept a device's new SSH host key and install the agent again. */
export async function trustDeviceHost(id: string): Promise<{ pid: string; status: string }> {
  return request<{ pid: string; status: string }>(`/api/devices/${encodeURIComponent(id)}/trust-host`, {
    method: 'POST',
  })
}

export async function stopDevice(id: string): Promise<{ stopped: boolean; status: string }> {
  return request<{ stopped: boolean; status: string }>(`/api/devices/${encodeURIComponent(id)}/stop`, {
    method: 'POST',
  })
}

/** Walk the master-to-agent connection chain and explain the first broken link. */
export async function diagnoseDevice(id: string): Promise<Diagnosis> {
  return request<Diagnosis>(`/api/devices/${encodeURIComponent(id)}/diagnose`, { method: 'POST' })
}

/** Recent master-owned changes to a device, newest first. */
export async function listDeviceActions(id: string): Promise<DeviceAction[]> {
  const data = await request<{ actions: DeviceAction[] }>(`/api/devices/${encodeURIComponent(id)}/actions`)
  return data.actions ?? []
}

/** Restore the wanted state recorded in a reversible device action. */
export async function undoDeviceAction(id: string, actionId: string): Promise<{ undone: boolean }> {
  return request<{ undone: boolean }>(
    `/api/devices/${encodeURIComponent(id)}/actions/${encodeURIComponent(actionId)}/undo`,
    { method: 'POST' }
  )
}

/** API prefix to give the SDK when talking to a device through the master. */
export function deviceApiPrefix(id: string): string {
  return `/api/devices/${encodeURIComponent(id)}/api`
}

/** WebSocket URL for a device's stats stream, proxied by the master. */
export function deviceStatsSocketUrl(id: string): string {
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  return `${protocol}//${window.location.host}${deviceApiPrefix(id)}/system/stats/ws`
}

export interface FleetSocket {
  close: () => void
}

/** Subscribe to fleet updates. Returns a handle; reconnects automatically. */
export function connectFleetSocket(handlers: {
  onEvent: (event: FleetEvent) => void
  onStatus?: (connected: boolean) => void
}): FleetSocket {
  let ws: WebSocket | null = null
  let stopped = false
  let attempts = 0
  let timer: ReturnType<typeof setTimeout> | null = null

  const connect = () => {
    if (stopped) return
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    ws = new WebSocket(`${protocol}//${window.location.host}/api/fleet/ws`)

    ws.onopen = () => {
      attempts = 0
      handlers.onStatus?.(true)
    }
    ws.onmessage = (event) => {
      try {
        handlers.onEvent(JSON.parse(event.data) as FleetEvent)
      } catch {
        // Ignore malformed events.
      }
    }
    ws.onerror = () => handlers.onStatus?.(false)
    ws.onclose = () => {
      handlers.onStatus?.(false)
      if (stopped) return
      attempts += 1
      const delay = Math.min(1000 * 2 ** Math.min(attempts, 5), 30000)
      timer = setTimeout(connect, delay)
    }
  }

  connect()

  return {
    close: () => {
      stopped = true
      if (timer) clearTimeout(timer)
      ws?.close()
    },
  }
}
