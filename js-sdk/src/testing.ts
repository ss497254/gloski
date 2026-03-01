import { GloskiError } from './errors'
import type { GloskiClient } from './client'
import type { Result } from './types'

/**
 * Create a successful Result
 */
export function mockResult<T>(data: T): Result<T> {
  return { data, error: null }
}

/**
 * Create a failed Result
 */
export function mockError<T>(status: number, message: string): Result<T> {
  return { data: null, error: new GloskiError(status, message) }
}

/** Deeply partial type: methods can be overridden, sub-resources are also partial */
type MockOverrides = {
  [K in keyof GloskiClient]?: GloskiClient[K] extends object
    ? Partial<GloskiClient[K]>
    : GloskiClient[K]
}

function unmockedResult(): Result<never> {
  return { data: null, error: new GloskiError(0, 'Not mocked') }
}

function createResourceProxy(overrides?: Record<string, unknown>): unknown {
  return new Proxy(overrides ?? {}, {
    get(target, prop) {
      if (typeof prop === 'symbol') return undefined
      if (prop in target) return (target as Record<string, unknown>)[prop]

      // Return a function that returns a fresh not-mocked result
      return () => Promise.resolve(unmockedResult())
    },
  })
}

/**
 * Create a mock GloskiClient for testing.
 *
 * Any method not overridden returns `{ data: null, error: GloskiError(0, 'Not mocked') }`.
 *
 * @example
 * ```ts
 * import { createMockClient, mockResult } from '@gloski/sdk/testing'
 *
 * const client = createMockClient({
 *   system: {
 *     getStats: () => mockResult({ hostname: 'test', ... }),
 *   },
 *   files: {
 *     list: () => mockResult({ path: '/', entries: [] }),
 *   },
 * })
 *
 * const stats = await client.system.getStats()
 * // stats.data.hostname === 'test'
 * ```
 */
export function createMockClient(overrides?: MockOverrides): GloskiClient {
  const resources = [
    'auth', 'system', 'files', 'jobs', 'search',
    'terminal', 'packages', 'cron', 'downloads',
  ]

  const client: Record<string, unknown> = {
    use: () => client,
    dispose: () => {},
    ping: () => Promise.resolve(unmockedResult()),
  }

  for (const name of resources) {
    const resourceOverrides = overrides?.[name as keyof MockOverrides]
    client[name] = createResourceProxy(resourceOverrides as Record<string, unknown> | undefined)
  }

  // Apply any top-level overrides (use, dispose, ping)
  if (overrides) {
    for (const key of Object.keys(overrides)) {
      if (!resources.includes(key)) {
        client[key] = (overrides as Record<string, unknown>)[key]
      }
    }
  }

  return client as unknown as GloskiClient
}
