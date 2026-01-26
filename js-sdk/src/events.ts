/**
 * Type-safe EventEmitter implementation
 * Zero dependencies, works in browser and Node.js
 */

// eslint-disable-next-line @typescript-eslint/no-explicit-any
export type EventMap = Record<string, any[]>
type EventListener<Args extends unknown[]> = (...args: Args) => void

export class EventEmitter<Events extends EventMap> {
  private listeners = new Map<keyof Events, Set<EventListener<Events[keyof Events]>>>()

  /**
   * Register an event listener
   */
  on<K extends keyof Events>(event: K, listener: EventListener<Events[K]>): this {
    if (!this.listeners.has(event)) {
      this.listeners.set(event, new Set())
    }
    this.listeners.get(event)!.add(listener as EventListener<Events[keyof Events]>)
    return this
  }

  /**
   * Remove an event listener
   */
  off<K extends keyof Events>(event: K, listener: EventListener<Events[K]>): this {
    this.listeners.get(event)?.delete(listener as EventListener<Events[keyof Events]>)
    return this
  }

  /**
   * Register a one-time event listener
   */
  once<K extends keyof Events>(event: K, listener: EventListener<Events[K]>): this {
    const wrapper: EventListener<Events[K]> = (...args) => {
      this.off(event, wrapper)
      listener(...args)
    }
    return this.on(event, wrapper)
  }

  /**
   * Emit an event to all registered listeners
   */
  protected emit<K extends keyof Events>(event: K, ...args: Events[K]): boolean {
    const eventListeners = this.listeners.get(event)
    if (!eventListeners?.size) return false

    for (const listener of eventListeners) {
      try {
        listener(...args)
      } catch (error) {
        // Don't let one listener's error break others
        console.error(`Error in event listener for "${String(event)}":`, error)
      }
    }

    return true
  }

  /**
   * Remove all listeners for an event, or all listeners if no event specified
   */
  removeAllListeners(event?: keyof Events): this {
    if (event !== undefined) {
      this.listeners.delete(event)
    } else {
      this.listeners.clear()
    }
    return this
  }

  /**
   * Get the number of listeners for an event
   */
  listenerCount(event: keyof Events): number {
    return this.listeners.get(event)?.size ?? 0
  }
}
