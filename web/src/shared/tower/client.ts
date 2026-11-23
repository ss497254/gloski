import type {
  DashboardSummary,
  Deployment,
  DeployRequest,
  Diagnosis,
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

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(path, {
    ...init,
    headers: {
      'Content-Type': 'application/json',
      ...init?.headers,
    },
  })
  const payload = (await response.json().catch(() => null)) as Envelope<T> | null
  if (!response.ok || !payload?.success) {
    throw new Error(payload?.error || `Tower request failed (${response.status})`)
  }
  return payload.data
}

/** Detect whether this origin is a control tower. Returns null when it is not. */
export async function detectTower(): Promise<TowerInfo | null> {
  try {
    const response = await fetch('/api/tower/info', { cache: 'no-store' })
    if (!response.ok) return null
    const payload = (await response.json()) as Envelope<TowerInfo>
    if (!payload.success || payload.data?.mode !== 'master') return null
    return payload.data
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
