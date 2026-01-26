import type { HttpClient } from '../http'
import type { TerminalOptions, TerminalState, TerminalEvents } from '../types'
import { EventEmitter } from '../events'

const DEFAULT_MAX_RECONNECT_ATTEMPTS = 5
const DEFAULT_RECONNECT_DELAY = 1000

/**
 * Terminal WebSocket connection with auto-reconnect
 */
export class TerminalConnection extends EventEmitter<TerminalEvents> {
  private ws: WebSocket | null = null
  private _state: TerminalState = 'connecting'
  private reconnectAttempts = 0
  private reconnectTimer?: ReturnType<typeof setTimeout>
  private manualClose = false
  private wasReconnect = false
  private lastSize: { cols: number; rows: number } | null = null
  private readonly encoder = new TextEncoder()
  private readonly decoder = new TextDecoder()
  private readonly url: string

  private readonly options: {
    cwd?: string
    autoReconnect: boolean
    maxReconnectAttempts: number
    reconnectDelay: number
  }

  constructor(url: string, options: TerminalOptions = {}) {
    super()
    this.url = url

    this.options = {
      cwd: options.cwd,
      autoReconnect: options.autoReconnect ?? true,
      maxReconnectAttempts: options.maxReconnectAttempts ?? DEFAULT_MAX_RECONNECT_ATTEMPTS,
      reconnectDelay: options.reconnectDelay ?? DEFAULT_RECONNECT_DELAY,
    }

    this.setupWebSocket()
  }

  /**
   * Current connection state
   */
  get state(): TerminalState {
    return this._state
  }

  /**
   * Send data to the terminal (user input)
   * @param data - String data to send
   */
  write(data: string): void {
    if (this.ws?.readyState === WebSocket.OPEN) {
      this.ws.send(this.encoder.encode(data))
    }
  }

  /**
   * Send resize event to the terminal
   * @param cols - Number of columns
   * @param rows - Number of rows
   */
  resize(cols: number, rows: number): void {
    this.lastSize = { cols, rows }

    if (this.ws?.readyState === WebSocket.OPEN) {
      // Binary protocol: [0x01, cols_high, cols_low, rows_high, rows_low]
      const data = new Uint8Array(5)
      data[0] = 0x01 // resize command
      data[1] = (cols >> 8) & 0xff
      data[2] = cols & 0xff
      data[3] = (rows >> 8) & 0xff
      data[4] = rows & 0xff
      this.ws.send(data)
    }
  }

  /**
   * Close the connection (disables auto-reconnect)
   */
  close(): void {
    this.manualClose = true
    this.clearReconnectTimer()
    
    if (this.ws) {
      this.ws.close()
      this.ws = null
    }
    
    this._state = 'closed'
  }

  /**
   * Manually trigger a reconnect
   */
  reconnect(): void {
    if (this._state === 'closed' || this._state === 'closing') {
      this.manualClose = false
      this.reconnectAttempts = 0
      this.wasReconnect = true
      this.setupWebSocket()
    }
  }

  /**
   * Check if the connection is open
   */
  get isOpen(): boolean {
    return this._state === 'open'
  }

  /**
   * Check if the connection is connecting or reconnecting
   */
  get isConnecting(): boolean {
    return this._state === 'connecting' || this._state === 'reconnecting'
  }

  private setupWebSocket(): void {
    this._state = this.wasReconnect ? 'reconnecting' : 'connecting'

    try {
      this.ws = new WebSocket(this.url)
      this.ws.binaryType = 'arraybuffer'

      this.ws.onopen = () => {
        this._state = 'open'
        this.reconnectAttempts = 0

        if (this.wasReconnect) {
          this.emit('reconnected')
          this.wasReconnect = false
        } else {
          this.emit('open')
        }

        // Restore terminal size if we have it
        if (this.lastSize) {
          this.resize(this.lastSize.cols, this.lastSize.rows)
        }
      }

      this.ws.onclose = (event) => {
        this.emit('close', event)

        // Auto-reconnect if enabled and not manually closed
        if (this.options.autoReconnect && !this.manualClose) {
          this.scheduleReconnect()
        } else {
          this._state = 'closed'
        }
      }

      this.ws.onerror = (error) => {
        this.emit('error', error)
      }

      this.ws.onmessage = (event) => {
        let data: string
        if (event.data instanceof ArrayBuffer) {
          data = this.decoder.decode(event.data)
        } else {
          data = event.data as string
        }
        this.emit('data', data)
      }
    } catch (error) {
      this._state = 'closed'
      this.emit('error', error as Event)
    }
  }

  private scheduleReconnect(): void {
    if (this.reconnectAttempts >= this.options.maxReconnectAttempts) {
      this._state = 'closed'
      return
    }

    this._state = 'reconnecting'
    this.reconnectAttempts++

    // Exponential backoff: 1s, 2s, 4s, 8s, 16s...
    const delay = this.options.reconnectDelay * Math.pow(2, this.reconnectAttempts - 1)

    this.emit('reconnecting', this.reconnectAttempts)

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

/**
 * Terminal resource
 */
export class TerminalResource {
  private http: HttpClient

  constructor(http: HttpClient) {
    this.http = http
  }

  /**
   * Get WebSocket URL for terminal connection
   * @param cwd - Initial working directory (optional)
   */
  getWebSocketUrl(cwd?: string): string {
    const params: Record<string, string> = {}
    if (cwd) {
      params.cwd = cwd
    }
    return this.http.buildWebSocketUrl('/terminal', params)
  }

  /**
   * Create a managed terminal connection with auto-reconnect
   * @param options - Terminal options
   */
  connect(options: TerminalOptions = {}): TerminalConnection {
    const url = this.getWebSocketUrl(options.cwd)
    return new TerminalConnection(url, options)
  }
}
