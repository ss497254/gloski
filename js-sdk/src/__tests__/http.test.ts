import { describe, test, expect, mock, beforeEach, afterEach } from 'bun:test'
import { HttpClient } from '../http'
import { GloskiError } from '../errors'

// Helper to create a mock Response
function mockResponse(body: unknown, status = 200, statusText = 'OK'): Response {
  return new Response(JSON.stringify(body), {
    status,
    statusText,
    headers: { 'Content-Type': 'application/json' },
  })
}

describe('HttpClient', () => {
  const originalFetch = globalThis.fetch

  beforeEach(() => {
    globalThis.fetch = mock(() => Promise.resolve(mockResponse({ success: true, data: 'ok' })))
  })

  afterEach(() => {
    globalThis.fetch = originalFetch
  })

  test('sends GET request with correct URL and auth header (apiKey)', async () => {
    const client = new HttpClient({ url: 'http://localhost:3000', apiKey: 'test-key' })

    await client.request('/system/stats')

    expect(globalThis.fetch).toHaveBeenCalledTimes(1)
    const [url, options] = (globalThis.fetch as ReturnType<typeof mock>).mock.calls[0] as [string, RequestInit]
    expect(url).toBe('http://localhost:3000/api/system/stats')
    expect(options.headers).toEqual(
      expect.objectContaining({ 'X-API-Key': 'test-key' })
    )
  })

  test('sends auth via Bearer token when token is set', async () => {
    const client = new HttpClient({ url: 'http://localhost:3000', token: 'jwt-token' })

    await client.request('/auth/status')

    const [, options] = (globalThis.fetch as ReturnType<typeof mock>).mock.calls[0] as [string, RequestInit]
    expect((options.headers as Record<string, string>)['Authorization']).toBe('Bearer jwt-token')
  })

  test('prefers apiKey over token when both are set', async () => {
    const client = new HttpClient({ url: 'http://localhost:3000', apiKey: 'key', token: 'token' })

    await client.request('/test')

    const [, options] = (globalThis.fetch as ReturnType<typeof mock>).mock.calls[0] as [string, RequestInit]
    const headers = options.headers as Record<string, string>
    expect(headers['X-API-Key']).toBe('key')
    expect(headers['Authorization']).toBeUndefined()
  })

  test('unwraps { success, data } envelope', async () => {
    globalThis.fetch = mock(() => Promise.resolve(mockResponse({ success: true, data: { id: 1 } })))

    const client = new HttpClient({ url: 'http://localhost:3000', apiKey: 'key' })
    const result = await client.get<{ id: number }>('/test')

    expect(result).toEqual({ id: 1 })
  })

  test('returns raw body when no envelope', async () => {
    globalThis.fetch = mock(() => Promise.resolve(mockResponse({ status: 'ok' })))

    const client = new HttpClient({ url: 'http://localhost:3000', apiKey: 'key' })
    const result = await client.get<{ status: string }>('/test')

    expect(result).toEqual({ status: 'ok' })
  })

  test('POST sends JSON body', async () => {
    const client = new HttpClient({ url: 'http://localhost:3000', apiKey: 'key' })

    await client.post('/jobs', { command: 'ls', cwd: '/tmp' })

    const [, options] = (globalThis.fetch as ReturnType<typeof mock>).mock.calls[0] as [string, RequestInit]
    expect(options.method).toBe('POST')
    expect(options.body).toBe(JSON.stringify({ command: 'ls', cwd: '/tmp' }))
  })

  test('DELETE request', async () => {
    const client = new HttpClient({ url: 'http://localhost:3000', apiKey: 'key' })

    await client.delete('/jobs/123')

    const [url, options] = (globalThis.fetch as ReturnType<typeof mock>).mock.calls[0] as [string, RequestInit]
    expect(url).toBe('http://localhost:3000/api/jobs/123')
    expect(options.method).toBe('DELETE')
  })

  test('deduplicates identical GET requests', async () => {
    let resolvePromise: (v: Response) => void
    globalThis.fetch = mock(
      () => new Promise<Response>((resolve) => { resolvePromise = resolve })
    )

    const client = new HttpClient({ url: 'http://localhost:3000', apiKey: 'key' })

    const p1 = client.get('/test')
    const p2 = client.get('/test')

    // Only one fetch call should have been made
    expect(globalThis.fetch).toHaveBeenCalledTimes(1)

    resolvePromise!(mockResponse({ success: true, data: 'result' }))

    const [r1, r2] = await Promise.all([p1, p2])
    expect(r1).toBe(r2)
  })

  test('does NOT deduplicate POST requests', async () => {
    globalThis.fetch = mock(() => Promise.resolve(mockResponse({ success: true, data: 'ok' })))

    const client = new HttpClient({ url: 'http://localhost:3000', apiKey: 'key' })

    await client.post('/jobs', { command: 'ls' })
    await client.post('/jobs', { command: 'ls' })

    expect(globalThis.fetch).toHaveBeenCalledTimes(2)
  })

  test('throws GloskiError on 401 and calls onUnauthorized', async () => {
    globalThis.fetch = mock(() =>
      Promise.resolve(mockResponse({ error: 'Invalid key' }, 401))
    )

    const onUnauthorized = mock(() => {})
    const client = new HttpClient(
      { url: 'http://localhost:3000', apiKey: 'bad-key' },
      { onUnauthorized }
    )

    try {
      await client.get('/test')
      expect(true).toBe(false) // should not reach
    } catch (e) {
      expect(e).toBeInstanceOf(GloskiError)
      expect((e as GloskiError).status).toBe(401)
    }

    expect(onUnauthorized).toHaveBeenCalledTimes(1)
  })

  test('throws GloskiError with server error message on non-ok response', async () => {
    globalThis.fetch = mock(() =>
      Promise.resolve(mockResponse({ error: 'File not found' }, 404))
    )

    const client = new HttpClient({ url: 'http://localhost:3000', apiKey: 'key' })

    try {
      await client.get('/files/nonexistent')
      expect(true).toBe(false)
    } catch (e) {
      expect(e).toBeInstanceOf(GloskiError)
      expect((e as GloskiError).status).toBe(404)
      expect((e as GloskiError).message).toBe('File not found')
    }
  })

  test('throws GloskiError on network failure and calls onOffline', async () => {
    globalThis.fetch = mock(() => Promise.reject(new TypeError('Failed to fetch')))

    const onOffline = mock(() => {})
    const client = new HttpClient(
      { url: 'http://localhost:3000', apiKey: 'key' },
      { onOffline }
    )

    try {
      await client.get('/test')
      expect(true).toBe(false)
    } catch (e) {
      expect(e).toBeInstanceOf(GloskiError)
      expect((e as GloskiError).status).toBe(0)
      expect((e as GloskiError).isNetworkError).toBe(true)
    }

    expect(onOffline).toHaveBeenCalledTimes(1)
  })

  test('calls onOnline when server recovers', async () => {
    const onOffline = mock(() => {})
    const onOnline = mock(() => {})
    const client = new HttpClient(
      { url: 'http://localhost:3000', apiKey: 'key' },
      { onOffline, onOnline }
    )

    // First request fails
    globalThis.fetch = mock(() => Promise.reject(new TypeError('Failed to fetch')))
    try { await client.get('/test') } catch {}
    expect(onOffline).toHaveBeenCalledTimes(1)

    // Second request succeeds
    globalThis.fetch = mock(() => Promise.resolve(mockResponse({ success: true, data: 'ok' })))
    await client.get('/test')
    expect(onOnline).toHaveBeenCalledTimes(1)
  })

  test('custom apiPrefix', async () => {
    const client = new HttpClient({ url: 'http://localhost:3000', apiKey: 'key', apiPrefix: '/v2' })

    await client.get('/system/stats')

    const [url] = (globalThis.fetch as ReturnType<typeof mock>).mock.calls[0] as [string, RequestInit]
    expect(url).toBe('http://localhost:3000/v2/system/stats')
  })

  test('does not duplicate apiPrefix if already in path', async () => {
    const client = new HttpClient({ url: 'http://localhost:3000', apiKey: 'key' })

    await client.get('/api/system/stats')

    const [url] = (globalThis.fetch as ReturnType<typeof mock>).mock.calls[0] as [string, RequestInit]
    expect(url).toBe('http://localhost:3000/api/system/stats')
  })

  test('buildAuthUrl constructs URL with api_key', () => {
    const client = new HttpClient({ url: 'http://localhost:3000', apiKey: 'my-key' })
    const url = client.buildAuthUrl('/downloads/123/file')

    expect(url).toContain('api_key=my-key')
    expect(url).toContain('/api/downloads/123/file')
  })

  test('buildAuthUrl with extra params', () => {
    const client = new HttpClient({ url: 'http://localhost:3000', apiKey: 'key' })
    const url = client.buildAuthUrl('/test', { format: 'json' })

    expect(url).toContain('format=json')
    expect(url).toContain('api_key=key')
  })

  test('buildWebSocketUrl converts http to ws', () => {
    const client = new HttpClient({ url: 'http://localhost:3000', apiKey: 'key' })
    const url = client.buildWebSocketUrl('/ws/stats')

    expect(url).toMatch(/^ws:/)
    expect(url).toContain('/api/ws/stats')
  })

  test('buildWebSocketUrl converts https to wss', () => {
    const client = new HttpClient({ url: 'https://server.example.com', apiKey: 'key' })
    const url = client.buildWebSocketUrl('/ws/stats')

    expect(url).toMatch(/^wss:/)
  })

  test('dispose clears pending requests', async () => {
    let resolvePromise: (v: Response) => void
    globalThis.fetch = mock(
      () => new Promise<Response>((resolve) => { resolvePromise = resolve })
    )

    const client = new HttpClient({ url: 'http://localhost:3000', apiKey: 'key' })

    // Start a request that won't resolve
    const promise = client.get('/slow')
    client.dispose()

    // Starting a new identical request should make a new fetch (not deduplicated)
    globalThis.fetch = mock(() => Promise.resolve(mockResponse({ success: true, data: 'fresh' })))
    const result = await client.get('/slow')
    expect(result).toBe('fresh')

    // Clean up dangling promise
    resolvePromise!(mockResponse({ success: true, data: 'stale' }))
    await promise.catch(() => {})
  })

  test('requestNoAuth does not include auth headers', async () => {
    const client = new HttpClient({ url: 'http://localhost:3000', apiKey: 'key' })

    await client.requestNoAuth('/health')

    const [, options] = (globalThis.fetch as ReturnType<typeof mock>).mock.calls[0] as [string, RequestInit]
    const headers = options.headers as Record<string, string>
    expect(headers['X-API-Key']).toBeUndefined()
    expect(headers['Authorization']).toBeUndefined()
  })
})
