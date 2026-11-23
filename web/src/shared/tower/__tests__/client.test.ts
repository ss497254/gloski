import { afterEach, beforeEach, describe, expect, mock, test } from 'bun:test'
import { deployDevice } from '../client'
import type { DeployRequest } from '../types'

describe('tower client request logs', () => {
  const originalFetch = globalThis.fetch
  const originalDebug = console.debug

  beforeEach(() => {
    globalThis.fetch = mock(() =>
      Promise.resolve(
        new Response(JSON.stringify({ success: true, data: { id: 'deployment-1' } }), {
          status: 200,
          headers: { 'Content-Type': 'application/json' },
        }),
      ),
    )
    console.debug = mock(() => {})
  })

  afterEach(() => {
    globalThis.fetch = originalFetch
    console.debug = originalDebug
  })

  test('redacts credentials and sends a request id', async () => {
    const payload = {
      name: 'ct',
      host: '100.x.y.z',
      user: 'root',
      ssh_port: 22,
      join_token: 'super-secret-token',
      ssh_key: 'super-secret-key',
    } as DeployRequest & { join_token: string; ssh_key: string }
    await deployDevice(payload)

    const lines = (console.debug as ReturnType<typeof mock>).mock.calls
      .map((args: unknown[]) => args.map((arg) => String(arg)).join(' '))
      .join('\n')
    expect(lines).toContain('[gloski] ->')
    expect(lines).toContain('[gloski] <-')
    expect(lines).toContain('[redacted]')
    expect(lines).not.toContain('super-secret-token')
    expect(lines).not.toContain('super-secret-key')

    const [, init] = (globalThis.fetch as ReturnType<typeof mock>).mock.calls[0] as [string, RequestInit]
    const headers = init.headers as Headers
    expect(headers.get('X-Request-ID')).toBeTruthy()
  })
})
