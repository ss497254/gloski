import { describe, test, expect, mock, beforeEach, afterEach } from 'bun:test'
import { GloskiClient } from '../client'

function mockResponse(body: unknown, status = 200): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  })
}

describe('GloskiClient', () => {
  const originalFetch = globalThis.fetch

  beforeEach(() => {
    globalThis.fetch = mock(() => Promise.resolve(mockResponse({ success: true, data: {} })))
  })

  afterEach(() => {
    globalThis.fetch = originalFetch
  })

  test('initializes all resource properties', () => {
    const client = new GloskiClient({ url: 'http://localhost:3000', apiKey: 'key' })

    expect(client.auth).toBeDefined()
    expect(client.system).toBeDefined()
    expect(client.files).toBeDefined()
    expect(client.jobs).toBeDefined()
    expect(client.search).toBeDefined()
    expect(client.terminal).toBeDefined()
    expect(client.packages).toBeDefined()
    expect(client.cron).toBeDefined()
    expect(client.downloads).toBeDefined()
  })

  test('normalizes trailing slashes in URL', async () => {
    const client = new GloskiClient({ url: 'http://localhost:3000///', apiKey: 'key' })

    await client.system.getStats()

    const [url] = (globalThis.fetch as ReturnType<typeof mock>).mock.calls[0] as [string, RequestInit]
    expect(url).toBe('http://localhost:3000/api/system/stats')
  })

  test('dispose() does not throw', () => {
    const client = new GloskiClient({ url: 'http://localhost:3000', apiKey: 'key' })
    expect(() => client.dispose()).not.toThrow()
  })

  test('passes callbacks to HttpClient', async () => {
    const onUnauthorized = mock(() => {})
    globalThis.fetch = mock(() => Promise.resolve(mockResponse({ error: 'Unauthorized' }, 401)))

    const client = new GloskiClient({
      url: 'http://localhost:3000',
      apiKey: 'bad',
      onUnauthorized,
    })

    const result = await client.auth.status()
    expect(result.error).not.toBeNull()

    expect(onUnauthorized).toHaveBeenCalledTimes(1)
  })
})

describe('GloskiClient resources', () => {
  const originalFetch = globalThis.fetch

  afterEach(() => {
    globalThis.fetch = originalFetch
  })

  function setupClient() {
    const client = new GloskiClient({ url: 'http://localhost:3000', apiKey: 'key' })
    return client
  }

  function getLastFetchCall(): [string, RequestInit] {
    const calls = (globalThis.fetch as ReturnType<typeof mock>).mock.calls
    return calls[calls.length - 1] as [string, RequestInit]
  }

  // ─── Auth ─────────────────────────────────────────────────────────────

  test('auth.status() calls /auth/status', async () => {
    globalThis.fetch = mock(() =>
      Promise.resolve(mockResponse({ success: true, data: { authenticated: true } }))
    )
    const client = setupClient()

    const result = await client.auth.status()

    const [url] = getLastFetchCall()
    expect(url).toContain('/api/auth/status')
    expect(result.data).toEqual({ authenticated: true })
    expect(result.error).toBeNull()
  })

  // ─── System ───────────────────────────────────────────────────────────

  test('system.getStats() calls /system/stats', async () => {
    globalThis.fetch = mock(() =>
      Promise.resolve(mockResponse({ success: true, data: { hostname: 'test' } }))
    )
    const client = setupClient()

    const result = await client.system.getStats()

    const [url] = getLastFetchCall()
    expect(url).toContain('/api/system/stats')
    expect(result.data!.hostname).toBe('test')
    expect(result.error).toBeNull()
  })

  test('system.getHistory() includes duration param', async () => {
    globalThis.fetch = mock(() =>
      Promise.resolve(mockResponse({ success: true, data: { samples: [], count: 0, duration: '5m' } }))
    )
    const client = setupClient()

    const result = await client.system.getHistory('5m')

    const [url] = getLastFetchCall()
    expect(url).toContain('duration=5m')
    expect(result.error).toBeNull()
  })

  test('system.getProcesses() calls /system/processes with limit', async () => {
    globalThis.fetch = mock(() =>
      Promise.resolve(mockResponse({ success: true, data: { processes: [{ pid: 1 }] } }))
    )
    const client = setupClient()

    const result = await client.system.getProcesses(50)

    const [url] = getLastFetchCall()
    expect(url).toContain('limit=50')
    expect(result.data).toEqual([{ pid: 1 }])
    expect(result.error).toBeNull()
  })

  // ─── Jobs ─────────────────────────────────────────────────────────────

  test('jobs.list() calls /jobs and unwraps .jobs', async () => {
    globalThis.fetch = mock(() =>
      Promise.resolve(mockResponse({ success: true, data: { jobs: [{ id: '1', command: 'ls' }] } }))
    )
    const client = setupClient()

    const result = await client.jobs.list()

    const [url] = getLastFetchCall()
    expect(url).toContain('/api/jobs')
    expect(result.data).toEqual([{ id: '1', command: 'ls' }])
    expect(result.error).toBeNull()
  })

  test('jobs.start() sends POST with command', async () => {
    globalThis.fetch = mock(() =>
      Promise.resolve(mockResponse({ success: true, data: { id: '2', command: 'echo hi' } }))
    )
    const client = setupClient()

    const result = await client.jobs.start('echo hi', '/tmp')

    const [url, options] = getLastFetchCall()
    expect(url).toContain('/api/jobs')
    expect(options.method).toBe('POST')
    expect(JSON.parse(options.body as string)).toEqual({ command: 'echo hi', cwd: '/tmp' })
    expect(result.error).toBeNull()
  })

  test('jobs.stop() sends POST to /jobs/:id/stop', async () => {
    globalThis.fetch = mock(() =>
      Promise.resolve(mockResponse({ success: true, data: {} }))
    )
    const client = setupClient()

    const result = await client.jobs.stop('abc')

    const [url, options] = getLastFetchCall()
    expect(url).toContain('/api/jobs/abc/stop')
    expect(options.method).toBe('POST')
    expect(result.error).toBeNull()
  })

  test('jobs.delete() sends DELETE to /jobs/:id', async () => {
    globalThis.fetch = mock(() =>
      Promise.resolve(mockResponse({ success: true, data: {} }))
    )
    const client = setupClient()

    const result = await client.jobs.delete('abc')

    const [url, options] = getLastFetchCall()
    expect(url).toContain('/api/jobs/abc')
    expect(options.method).toBe('DELETE')
    expect(result.error).toBeNull()
  })

  // ─── Cron ─────────────────────────────────────────────────────────────

  test('cron.list() calls /cron/jobs and returns full response', async () => {
    globalThis.fetch = mock(() =>
      Promise.resolve(mockResponse({ success: true, data: { jobs: [{ schedule: '* * * * *', command: 'echo hi', enabled: true, source: 'user' }], count: 1 } }))
    )
    const client = setupClient()

    const result = await client.cron.list()

    expect(result.data!.jobs).toHaveLength(1)
    expect(result.data!.count).toBe(1)
    expect(result.error).toBeNull()
  })

  test('cron.list() passes scope param', async () => {
    globalThis.fetch = mock(() =>
      Promise.resolve(mockResponse({ success: true, data: { jobs: [], count: 0 } }))
    )
    const client = setupClient()

    await client.cron.list('user')

    const [url] = getLastFetchCall()
    expect(url).toContain('scope=user')
  })

  test('cron.add() sends POST with schedule and command', async () => {
    globalThis.fetch = mock(() =>
      Promise.resolve(mockResponse({ success: true }))
    )
    const client = setupClient()

    const result = await client.cron.add({ schedule: '0 * * * *', command: '/usr/bin/backup' })

    const [, options] = getLastFetchCall()
    expect(options.method).toBe('POST')
    expect(JSON.parse(options.body as string)).toEqual({ schedule: '0 * * * *', command: '/usr/bin/backup' })
    expect(result.error).toBeNull()
  })

  test('cron.remove() sends DELETE with schedule and command in body', async () => {
    globalThis.fetch = mock(() =>
      Promise.resolve(mockResponse({ success: true }))
    )
    const client = setupClient()

    const result = await client.cron.remove('0 * * * *', '/usr/bin/backup')

    const [, options] = getLastFetchCall()
    expect(options.method).toBe('DELETE')
    expect(JSON.parse(options.body as string)).toEqual({ schedule: '0 * * * *', command: '/usr/bin/backup' })
    expect(result.error).toBeNull()
  })

  // ─── Downloads ────────────────────────────────────────────────────────

  test('downloads.list() calls /downloads and unwraps .downloads', async () => {
    globalThis.fetch = mock(() =>
      Promise.resolve(mockResponse({ success: true, data: { downloads: [{ id: 'd1', url: 'http://example.com' }] } }))
    )
    const client = setupClient()

    const result = await client.downloads.list()

    expect(result.data).toEqual([{ id: 'd1', url: 'http://example.com' }])
    expect(result.error).toBeNull()
  })

  test('downloads.add() sends POST with url and destination', async () => {
    globalThis.fetch = mock(() =>
      Promise.resolve(mockResponse({ success: true, data: { id: 'd1', url: 'http://example.com/file' } }))
    )
    const client = setupClient()

    const result = await client.downloads.add('http://example.com/file', '/tmp')

    const [, options] = getLastFetchCall()
    expect(options.method).toBe('POST')
    expect(JSON.parse(options.body as string)).toEqual({
      url: 'http://example.com/file',
      destination: '/tmp',
    })
    expect(result.data!.id).toBe('d1')
    expect(result.error).toBeNull()
  })

  test('downloads.pause() sends POST to /downloads/:id/pause', async () => {
    globalThis.fetch = mock(() =>
      Promise.resolve(mockResponse({ success: true, data: { status: 'paused' } }))
    )
    const client = setupClient()

    const result = await client.downloads.pause('d1')

    const [url, options] = getLastFetchCall()
    expect(url).toContain('/api/downloads/d1/pause')
    expect(options.method).toBe('POST')
    expect(result.error).toBeNull()
  })

  test('downloads.delete() with deleteFile appends query param', async () => {
    globalThis.fetch = mock(() =>
      Promise.resolve(mockResponse({ success: true, data: { status: 'deleted' } }))
    )
    const client = setupClient()

    const result = await client.downloads.delete('d1', true)

    const [url, options] = getLastFetchCall()
    expect(url).toContain('delete_file=true')
    expect(options.method).toBe('DELETE')
    expect(result.error).toBeNull()
  })

  // ─── Search ───────────────────────────────────────────────────────────

  test('search.byName() sends correct params and returns full response', async () => {
    globalThis.fetch = mock(() =>
      Promise.resolve(mockResponse({ success: true, data: { results: [{ name: 'test.txt' }], count: 1 } }))
    )
    const client = setupClient()

    const result = await client.search.byName('/home', 'test')

    const [url] = getLastFetchCall()
    expect(url).toContain('path=%2Fhome')
    expect(url).toContain('q=test')
    expect(url).not.toContain('content=true')
    expect(result.data!.results).toEqual([{ name: 'test.txt' }])
    expect(result.data!.count).toBe(1)
    expect(result.error).toBeNull()
  })

  test('search.byContent() includes content=true', async () => {
    globalThis.fetch = mock(() =>
      Promise.resolve(mockResponse({ success: true, data: { results: [], count: 0 } }))
    )
    const client = setupClient()

    const result = await client.search.byContent('/home', 'TODO')

    const [url] = getLastFetchCall()
    expect(url).toContain('content=true')
    expect(result.data!.count).toBe(0)
    expect(result.error).toBeNull()
  })

  // ─── Packages ─────────────────────────────────────────────────────────

  test('packages.search() encodes query and returns full response', async () => {
    globalThis.fetch = mock(() =>
      Promise.resolve(mockResponse({ success: true, data: { packages: [{ name: 'nginx', version: '1.0' }], count: 1 } }))
    )
    const client = setupClient()

    const result = await client.packages.search('nginx server')

    const [url] = getLastFetchCall()
    expect(url).toContain('q=nginx%20server')
    expect(result.data!.packages).toHaveLength(1)
    expect(result.data!.count).toBe(1)
    expect(result.error).toBeNull()
  })

  test('packages.listInstalled() returns full response with manager and count', async () => {
    globalThis.fetch = mock(() =>
      Promise.resolve(mockResponse({ success: true, data: { manager: 'apt', packages: [{ name: 'curl', version: '7.68.0' }], count: 1 } }))
    )
    const client = setupClient()

    const result = await client.packages.listInstalled()

    expect(result.data!.manager).toBe('apt')
    expect(result.data!.packages).toHaveLength(1)
    expect(result.data!.count).toBe(1)
    expect(result.error).toBeNull()
  })

  test('packages.checkUpgrades() returns upgrade info', async () => {
    globalThis.fetch = mock(() =>
      Promise.resolve(mockResponse({ success: true, data: { upgradable_count: 5, security_count: 2, packages: [] } }))
    )
    const client = setupClient()

    const result = await client.packages.checkUpgrades()

    expect(result.data!.upgradable_count).toBe(5)
    expect(result.data!.security_count).toBe(2)
    expect(result.error).toBeNull()
  })

  // ─── Error handling ───────────────────────────────────────────────────

  test('resource methods return error on API failure', async () => {
    globalThis.fetch = mock(() =>
      Promise.resolve(mockResponse({ error: 'Not found' }, 404))
    )
    const client = setupClient()

    const result = await client.jobs.get('nonexistent')

    expect(result.data).toBeNull()
    expect(result.error).not.toBeNull()
    expect(result.error!.status).toBe(404)
  })
})
