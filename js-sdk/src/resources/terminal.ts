import type { HttpClient } from '../http'
import type { TerminalEvents, TerminalOptions } from '../types'
import { WebSocketConnection } from './base-ws'

/**
 * Terminal WebSocket connection with auto-reconnect
 */
export class TerminalConnection extends WebSocketConnection<TerminalEvents> {
  private lastSize: { cols: number; rows: number } | null = null
  private readonly decoder = new TextDecoder()

  constructor(url: string, options: TerminalOptions = {}) {
    super(url, {
      autoReconnect: options.autoReconnect,
      maxReconnectAttempts: options.maxReconnectAttempts,
      reconnectDelay: options.reconnectDelay,
    })
  }

  /**
   * Send data to the terminal (user input)
   * @param data - String data to send
   */
  write(data: string): void {
    if (this.ws?.readyState === WebSocket.OPEN) {
      this.ws.send(data)
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

  protected configureWebSocket(ws: WebSocket): void {
    ws.binaryType = 'arraybuffer'
  }

  protected onOpen(): void {
    if (this.lastSize) {
      this.resize(this.lastSize.cols, this.lastSize.rows)
    }
  }

  protected handleMessage(event: MessageEvent): void {
    let data: string
    if (event.data instanceof ArrayBuffer) {
      data = this.decoder.decode(event.data)
    } else {
      data = event.data as string
    }
    this.emit('data', data)
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
