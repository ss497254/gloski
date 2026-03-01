import { EventEmitter } from '../events'
import type { BaseWebSocketOptions, BaseWebSocketState } from '../types'

/**
 * Base event shape that all WebSocket connections must include
 */
export interface BaseWebSocketEvents {
  open: []
  close: [event: CloseEvent]
  error: [error: Event | Error]
  reconnecting: [attempt: number]
  reconnected: []
}

const DEFAULT_MAX_RECONNECT_ATTEMPTS = 5
const DEFAULT_RECONNECT_DELAY = 1000
const DEFAULT_MAX_RECONNECT_DELAY = 30000

/**
 * Abstract base class for WebSocket connections with auto-reconnect.
 * Subclasses implement `handleMessage()` for protocol-specific parsing.
 */
export abstract class BaseWebSocketConnection<
  Events extends BaseWebSocketEvents & Record<keyof Events, unknown[]>,
> extends EventEmitter<Events> {
  protected ws: WebSocket | null = null
  private _state: BaseWebSocketState = 'connecting'
  private reconnectAttempts = 0
  private reconnectTimer?: ReturnType<typeof setTimeout>
  private manualClose = false
  private wasReconnect = false
  protected readonly url: string

  protected readonly wsOptions: {
    autoReconnect: boolean
    maxReconnectAttempts: number
    reconnectDelay: number
    maxReconnectDelay: number
  }

  constructor(url: string, options: BaseWebSocketOptions = {}) {
    super()
    this.url = url
    this.wsOptions = {
      autoReconnect: options.autoReconnect ?? true,
      maxReconnectAttempts: options.maxReconnectAttempts ?? DEFAULT_MAX_RECONNECT_ATTEMPTS,
      reconnectDelay: options.reconnectDelay ?? DEFAULT_RECONNECT_DELAY,
      maxReconnectDelay: options.maxReconnectDelay ?? DEFAULT_MAX_RECONNECT_DELAY,
    }
    this.setupWebSocket()
  }

  /** Current connection state */
  get state(): BaseWebSocketState {
    return this._state
  }

  /** Check if the connection is open */
  get isOpen(): boolean {
    return this._state === 'open'
  }

  /** Check if the connection is connecting or reconnecting */
  get isConnecting(): boolean {
    return this._state === 'connecting' || this._state === 'reconnecting'
  }

  /** Close the connection (disables auto-reconnect) */
  close(): void {
    this.manualClose = true
    this.clearReconnectTimer()
    if (this.ws) {
      this.ws.close()
      this.ws = null
    }
    this._state = 'closed'
  }

  /** Manually trigger a reconnect */
  reconnect(): void {
    if (this._state === 'closed' || this._state === 'closing') {
      this.manualClose = false
      this.reconnectAttempts = 0
      this.wasReconnect = true
      this.setupWebSocket()
    }
  }

  /** Called to configure the WebSocket before connection (e.g., binaryType) */
  protected configureWebSocket(_ws: WebSocket): void {
    // Override in subclass if needed
  }

  /** Called when a WebSocket message is received */
  protected abstract handleMessage(event: MessageEvent): void

  /** Called after connection opens successfully */
  protected onOpen(): void {
    // Override in subclass if needed
  }

  /** Emit a base event with proper typing */
  private emitBase<K extends keyof BaseWebSocketEvents>(event: K, ...args: BaseWebSocketEvents[K]): void {
    // Safe: subclass Events always extends BaseWebSocketEvents
    this.emit(event as keyof Events, ...(args as unknown as Events[keyof Events]))
  }

  private setupWebSocket(): void {
    this._state = this.wasReconnect ? 'reconnecting' : 'connecting'

    try {
      this.ws = new WebSocket(this.url)
      this.configureWebSocket(this.ws)

      this.ws.onopen = () => {
        this._state = 'open'
        this.reconnectAttempts = 0

        if (this.wasReconnect) {
          this.emitBase('reconnected')
          this.wasReconnect = false
        } else {
          this.emitBase('open')
        }

        this.onOpen()
      }

      this.ws.onclose = (event) => {
        this.emitBase('close', event)

        if (this.wsOptions.autoReconnect && !this.manualClose) {
          this.scheduleReconnect()
        } else {
          this._state = 'closed'
        }
      }

      this.ws.onerror = (error) => {
        this.emitBase('error', error)
      }

      this.ws.onmessage = (event) => this.handleMessage(event)
    } catch (error) {
      this._state = 'closed'
      this.emitBase('error', error as Event)
    }
  }

  private scheduleReconnect(): void {
    if (this.reconnectAttempts >= this.wsOptions.maxReconnectAttempts) {
      this._state = 'closed'
      return
    }

    this._state = 'reconnecting'
    this.reconnectAttempts++

    const delay = Math.min(
      this.wsOptions.reconnectDelay * Math.pow(2, this.reconnectAttempts - 1),
      this.wsOptions.maxReconnectDelay
    )

    this.emitBase('reconnecting', this.reconnectAttempts)

    this.reconnectTimer = setTimeout(() => {
      this.wasReconnect = true
      this.setupWebSocket()
    }, delay)
  }

  private clearReconnectTimer(): void {
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer)
      this.reconnectTimer = undefined
    }
  }
}
