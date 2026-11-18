import { describe, test, expect, mock } from 'bun:test'
import { EventEmitter } from '../events'

interface TestEvents {
  message: [text: string]
  count: [n: number]
  empty: []
  multi: [a: string, b: number]
}

class TestEmitter extends EventEmitter<TestEvents> {
  // Expose emit for testing
  public emit<K extends keyof TestEvents>(event: K, ...args: TestEvents[K]): boolean {
    return super.emit(event, ...args)
  }
}

describe('EventEmitter', () => {
  test('on() registers a listener and emit() calls it', () => {
    const emitter = new TestEmitter()
    const fn = mock(() => {})

    emitter.on('message', fn)
    emitter.emit('message', 'hello')

    expect(fn).toHaveBeenCalledTimes(1)
    expect(fn).toHaveBeenCalledWith('hello')
  })

  test('on() supports multiple listeners for the same event', () => {
    const emitter = new TestEmitter()
    const fn1 = mock(() => {})
    const fn2 = mock(() => {})

    emitter.on('message', fn1)
    emitter.on('message', fn2)
    emitter.emit('message', 'hi')

    expect(fn1).toHaveBeenCalledTimes(1)
    expect(fn2).toHaveBeenCalledTimes(1)
  })

  test('on() returns this for chaining', () => {
    const emitter = new TestEmitter()
    const result = emitter.on('message', () => {})
    expect(result).toBe(emitter)
  })

  test('off() removes a specific listener', () => {
    const emitter = new TestEmitter()
    const fn = mock(() => {})

    emitter.on('message', fn)
    emitter.off('message', fn)
    emitter.emit('message', 'hello')

    expect(fn).not.toHaveBeenCalled()
  })

  test('off() returns this for chaining', () => {
    const emitter = new TestEmitter()
    const result = emitter.off('message', () => {})
    expect(result).toBe(emitter)
  })

  test('once() fires listener only once', () => {
    const emitter = new TestEmitter()
    const fn = mock(() => {})

    emitter.once('message', fn)
    emitter.emit('message', 'first')
    emitter.emit('message', 'second')

    expect(fn).toHaveBeenCalledTimes(1)
    expect(fn).toHaveBeenCalledWith('first')
  })

  test('emit() returns false when no listeners', () => {
    const emitter = new TestEmitter()
    expect(emitter.emit('message', 'hello')).toBe(false)
  })

  test('emit() returns true when listeners exist', () => {
    const emitter = new TestEmitter()
    emitter.on('message', () => {})
    expect(emitter.emit('message', 'hello')).toBe(true)
  })

  test('emit() passes multiple arguments', () => {
    const emitter = new TestEmitter()
    const fn = mock(() => {})

    emitter.on('multi', fn)
    emitter.emit('multi', 'test', 42)

    expect(fn).toHaveBeenCalledWith('test', 42)
  })

  test('emit() handles zero-arg events', () => {
    const emitter = new TestEmitter()
    const fn = mock(() => {})

    emitter.on('empty', fn)
    emitter.emit('empty')

    expect(fn).toHaveBeenCalledTimes(1)
    expect(fn).toHaveBeenCalledWith()
  })

  test('emit() isolates listener errors', () => {
    const emitter = new TestEmitter()
    const errorFn = mock(() => {
      throw new Error('boom')
    })
    const okFn = mock(() => {})

    // Suppress console.error for this test
    const origError = console.error
    console.error = () => {}

    emitter.on('message', errorFn)
    emitter.on('message', okFn)
    emitter.emit('message', 'test')

    console.error = origError

    expect(errorFn).toHaveBeenCalledTimes(1)
    expect(okFn).toHaveBeenCalledTimes(1)
  })

  test('removeAllListeners() clears all listeners for a specific event', () => {
    const emitter = new TestEmitter()
    const fn1 = mock(() => {})
    const fn2 = mock(() => {})

    emitter.on('message', fn1)
    emitter.on('count', fn2)
    emitter.removeAllListeners('message')

    emitter.emit('message', 'hello')
    emitter.emit('count', 5)

    expect(fn1).not.toHaveBeenCalled()
    expect(fn2).toHaveBeenCalledTimes(1)
  })

  test('removeAllListeners() with no arg clears everything', () => {
    const emitter = new TestEmitter()
    const fn1 = mock(() => {})
    const fn2 = mock(() => {})

    emitter.on('message', fn1)
    emitter.on('count', fn2)
    emitter.removeAllListeners()

    emitter.emit('message', 'hello')
    emitter.emit('count', 5)

    expect(fn1).not.toHaveBeenCalled()
    expect(fn2).not.toHaveBeenCalled()
  })

  test('listenerCount() returns correct count', () => {
    const emitter = new TestEmitter()
    expect(emitter.listenerCount('message')).toBe(0)

    emitter.on('message', () => {})
    expect(emitter.listenerCount('message')).toBe(1)

    emitter.on('message', () => {})
    expect(emitter.listenerCount('message')).toBe(2)
  })

  test('listenerCount() returns 0 for unregistered events', () => {
    const emitter = new TestEmitter()
    expect(emitter.listenerCount('count')).toBe(0)
  })
})
