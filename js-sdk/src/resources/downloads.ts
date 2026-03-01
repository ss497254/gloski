import { safe } from '../errors'
import { EventEmitter } from '../events'
import type { HttpClient, RequestOptions } from '../http'
import type {
  NewDownloadRequest,
  CreateShareOptions,
  Download,
  DownloadId,
  DownloadWatcherEvents,
  DownloadWatcherOptions,
  DownloadsResponse,
  Result,
  ShareLink,
} from '../types'

export class DownloadsResource {
  private http: HttpClient

  constructor(http: HttpClient) {
    this.http = http
  }

  /**
   * List all downloads
   */
  async list(options?: RequestOptions): Promise<Result<Download[]>> {
    return safe(this.http.get<DownloadsResponse>('/downloads', options).then((response) => response.downloads))
  }

  /**
   * Get a specific download by ID
   */
  async get(id: DownloadId, options?: RequestOptions): Promise<Result<Download>> {
    return safe(this.http.get<Download>(`/downloads/${id}`, options))
  }

  /**
   * Add a new download
   * @param url - URL to download from
   * @param destination - Destination directory on the server
   * @param filename - Optional filename (auto-detected if not provided)
   */
  async add(url: string, destination: string, filename?: string, options?: RequestOptions): Promise<Result<Download>> {
    const request: NewDownloadRequest = { url, destination }
    if (filename) {
      request.filename = filename
    }
    return safe(this.http.post<Download>('/downloads', request, options))
  }

  /**
   * Pause a download
   */
  async pause(id: DownloadId, options?: RequestOptions): Promise<Result<void>> {
    return safe(this.http.post<{ status: string }>(`/downloads/${id}/pause`, {}, options).then(() => {}))
  }

  /**
   * Resume a paused download
   */
  async resume(id: DownloadId, options?: RequestOptions): Promise<Result<void>> {
    return safe(this.http.post<{ status: string }>(`/downloads/${id}/resume`, {}, options).then(() => {}))
  }

  /**
   * Cancel a download
   */
  async cancel(id: DownloadId, options?: RequestOptions): Promise<Result<void>> {
    return safe(this.http.post<{ status: string }>(`/downloads/${id}/cancel`, {}, options).then(() => {}))
  }

  /**
   * Retry a failed download
   */
  async retry(id: DownloadId, options?: RequestOptions): Promise<Result<void>> {
    return safe(this.http.post<{ status: string }>(`/downloads/${id}/retry`, {}, options).then(() => {}))
  }

  /**
   * Delete a download
   * @param id - Download ID
   * @param deleteFile - Also delete the downloaded file (default: false)
   */
  async delete(id: DownloadId, deleteFile = false, options?: RequestOptions): Promise<Result<void>> {
    const params = deleteFile ? '?delete_file=true' : ''
    return safe(this.http.delete<{ status: string }>(`/downloads/${id}${params}`, options).then(() => {}))
  }

  /**
   * Cancel multiple downloads in parallel
   */
  async cancelAll(ids: DownloadId[], options?: RequestOptions): Promise<Result<void>[]> {
    return Promise.all(ids.map((id) => this.cancel(id, options)))
  }

  /**
   * Pause multiple downloads in parallel
   */
  async pauseAll(ids: DownloadId[], options?: RequestOptions): Promise<Result<void>[]> {
    return Promise.all(ids.map((id) => this.pause(id, options)))
  }

  /**
   * Resume multiple downloads in parallel
   */
  async resumeAll(ids: DownloadId[], options?: RequestOptions): Promise<Result<void>[]> {
    return Promise.all(ids.map((id) => this.resume(id, options)))
  }

  /**
   * Get the download URL for a completed download (authenticated)
   */
  getDownloadUrl(id: DownloadId): string {
    return this.http.getFullUrl(`/downloads/${id}/file`)
  }

  /**
   * Create a share link for a completed download
   * @param id - Download ID
   * @param shareOptions - Share options (expiry time)
   */
  async createShareLink(
    id: DownloadId,
    shareOptions?: CreateShareOptions,
    options?: RequestOptions
  ): Promise<Result<ShareLink>> {
    return safe(this.http.post<ShareLink>(`/downloads/${id}/share`, shareOptions ?? {}, options))
  }

  /**
   * Revoke a share link
   * @param id - Download ID
   * @param token - Share link token to revoke
   */
  async revokeShareLink(id: DownloadId, token: string, options?: RequestOptions): Promise<Result<void>> {
    return safe(this.http.delete<{ status: string }>(`/downloads/${id}/share/${token}`, options).then(() => {}))
  }

  /**
   * Create a download watcher that polls for updates.
   * @param options - Watcher configuration
   */
  watch(options?: DownloadWatcherOptions): DownloadWatcher {
    return new DownloadWatcher(this, options)
  }
}

/**
 * Polls the downloads list and emits events on state changes.
 */
export class DownloadWatcher extends EventEmitter<DownloadWatcherEvents> {
  private resource: DownloadsResource
  private interval: number
  private timerId: ReturnType<typeof setInterval> | null = null
  private polling = false
  private previousDownloads = new Map<DownloadId, Download>()

  constructor(resource: DownloadsResource, options?: DownloadWatcherOptions) {
    super()
    this.resource = resource
    this.interval = options?.interval ?? 2000
  }

  /** Start polling */
  start(): void {
    if (this.timerId !== null) return
    this.poll()
    this.timerId = setInterval(() => this.poll(), this.interval)
  }

  /** Stop polling */
  stop(): void {
    if (this.timerId !== null) {
      clearInterval(this.timerId)
      this.timerId = null
    }
    this.polling = false
  }

  private async poll(): Promise<void> {
    if (this.polling) return
    this.polling = true

    try {
      const result = await this.resource.list()
      if (result.error) {
        this.emit('error', result.error)
        return
      }

      const downloads = result.data
      this.emit('update', downloads)

      for (const download of downloads) {
        const previousState = this.previousDownloads.get(download.id)

        if (previousState) {
          if (download.progress !== previousState.progress || download.status !== previousState.status) {
            this.emit('progress', download)
          }
          if (download.status === 'completed' && previousState.status !== 'completed') {
            this.emit('completed', download)
          }
          if (download.status === 'failed' && previousState.status !== 'failed') {
            this.emit('failed', download)
          }
        }

        this.previousDownloads.set(download.id, download)
      }
    } catch (err) {
      this.emit('error', err instanceof Error ? err : new Error(String(err)))
    } finally {
      this.polling = false
    }
  }
}
