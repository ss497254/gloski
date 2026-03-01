import { safe } from '../errors'
import type { HttpClient } from '../http'
import type { AddDownloadRequest, CreateShareOptions, Download, DownloadsResponse, Result, ShareLink } from '../types'

export class DownloadsResource {
  private http: HttpClient

  constructor(http: HttpClient) {
    this.http = http
  }

  /**
   * List all downloads
   */
  async list(): Promise<Result<Download[]>> {
    return safe(this.http.get<DownloadsResponse>('/downloads').then(r => r.downloads))
  }

  /**
   * Get a specific download by ID
   */
  async get(id: string): Promise<Result<Download>> {
    return safe(this.http.get<Download>(`/downloads/${id}`))
  }

  /**
   * Add a new download
   * @param url - URL to download from
   * @param destination - Destination directory on the server
   * @param filename - Optional filename (auto-detected if not provided)
   */
  async add(url: string, destination: string, filename?: string): Promise<Result<Download>> {
    const request: AddDownloadRequest = { url, destination }
    if (filename) {
      request.filename = filename
    }
    return safe(this.http.post<Download>('/downloads', request))
  }

  /**
   * Pause a download
   */
  async pause(id: string): Promise<Result<void>> {
    return safe(this.http.post<{ status: string }>(`/downloads/${id}/pause`, {}).then(() => {}))
  }

  /**
   * Resume a paused download
   */
  async resume(id: string): Promise<Result<void>> {
    return safe(this.http.post<{ status: string }>(`/downloads/${id}/resume`, {}).then(() => {}))
  }

  /**
   * Cancel a download
   */
  async cancel(id: string): Promise<Result<void>> {
    return safe(this.http.post<{ status: string }>(`/downloads/${id}/cancel`, {}).then(() => {}))
  }

  /**
   * Retry a failed download
   */
  async retry(id: string): Promise<Result<void>> {
    return safe(this.http.post<{ status: string }>(`/downloads/${id}/retry`, {}).then(() => {}))
  }

  /**
   * Delete a download
   * @param id - Download ID
   * @param deleteFile - Also delete the downloaded file (default: false)
   */
  async delete(id: string, deleteFile = false): Promise<Result<void>> {
    const params = deleteFile ? '?delete_file=true' : ''
    return safe(this.http.delete<{ status: string }>(`/downloads/${id}${params}`).then(() => {}))
  }

  /**
   * Get the download URL for a completed download (authenticated)
   */
  getDownloadUrl(id: string): string {
    return this.http.getFullUrl(`/downloads/${id}/file`)
  }

  /**
   * Create a share link for a completed download
   * @param id - Download ID
   * @param options - Share options (expiry time)
   */
  async createShareLink(id: string, options?: CreateShareOptions): Promise<Result<ShareLink>> {
    return safe(this.http.post<ShareLink>(`/downloads/${id}/share`, options ?? {}))
  }

  /**
   * Revoke a share link
   * @param id - Download ID
   * @param token - Share link token to revoke
   */
  async revokeShareLink(id: string, token: string): Promise<Result<void>> {
    return safe(this.http.delete<{ status: string }>(`/downloads/${id}/share/${token}`).then(() => {}))
  }
}
