import { describe, test, expect } from 'bun:test'
import { createMockClient, mockResult, mockError } from '../testing'
import { GloskiError } from '../errors'

describe('mockResult', () => {
  test('creates a success result', () => {
    const result = mockResult({ id: 1 })
    expect(result.data).toEqual({ id: 1 })
    expect(result.error).toBeNull()
  })
})

describe('mockError', () => {
  test('creates an error result', () => {
    const result = mockError(404, 'Not found')
    expect(result.data).toBeNull()
    expect(result.error).toBeInstanceOf(GloskiError)
    expect(result.error!.status).toBe(404)
    expect(result.error!.message).toBe('Not found')
  })
})

describe('createMockClient', () => {
  test('returns not-mocked error for unoverridden methods', async () => {
    const client = createMockClient()
    const result = await client.system.getStats()
    expect(result.data).toBeNull()
    expect(result.error).toBeInstanceOf(GloskiError)
    expect(result.error!.message).toBe('Not mocked')
  })

  test('uses overridden methods', async () => {
    const client = createMockClient({
      auth: {
        status: () => Promise.resolve(mockResult({ authenticated: true })),
      },
    })

    const result = await client.auth.status()
    expect(result.data).toEqual({ authenticated: true })
    expect(result.error).toBeNull()
  })

  test('handles nested sub-resources', async () => {
    const client = createMockClient({
      files: {
        list: () => Promise.resolve(mockResult({ path: '/', entries: [] })),
      },
    })

    const result = await client.files.list()
    expect(result.data).toEqual({ path: '/', entries: [] })

    // Non-overridden method on same resource returns not-mocked
    const readResult = await client.files.read('/test')
    expect(readResult.error).toBeInstanceOf(GloskiError)
  })

  test('use() returns the client for chaining', () => {
    const client = createMockClient()
    const returned = client.use(async (req, next) => next(req))
    expect(returned).toBe(client)
  })

  test('dispose() does not throw', () => {
    const client = createMockClient()
    expect(() => client.dispose()).not.toThrow()
  })

  test('ping() returns not-mocked by default', async () => {
    const client = createMockClient()
    const result = await client.ping()
    expect(result.error).toBeInstanceOf(GloskiError)
    expect(result.error!.message).toBe('Not mocked')
  })

  test('multiple resources can be overridden', async () => {
    const client = createMockClient({
      jobs: {
        list: () => Promise.resolve(mockResult([])),
      },
      downloads: {
        list: () => Promise.resolve(mockResult([])),
      },
    })

    const jobs = await client.jobs.list()
    expect(jobs.data).toEqual([])

    const downloads = await client.downloads.list()
    expect(downloads.data).toEqual([])
  })
})
