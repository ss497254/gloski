import { GloskiError, safe } from '../errors'
import type { HttpClient } from '../http'
import type {
  ChunkedUploadChunkResponse,
  ChunkedUploadCompleteRequest,
  ChunkedUploadCompleteResponse,
  ChunkedUploadInfo,
  ChunkedUploadInit,
  ListResponse,
  PinnedFolder,
  PinnedFoldersResponse,
  ReadResponse,
  Result,
  UploadResponse,
} from '../types'

/**
 * Pinned folders sub-resource (accessed via files.pinned)
 */
export class PinnedSubResource {
  private http: HttpClient

  constructor(http: HttpClient) {
    this.http = http
  }

  /**
   * List all pinned folders and get home directory
   */
  async list(): Promise<Result<PinnedFoldersResponse>> {
    return safe(this.http.request<PinnedFoldersResponse>('/files/pinned'))
  }

  /**
   * Pin a folder
   * @param path - Folder path to pin
   * @param name - Display name for the pinned folder
   */
  async pin(path: string, name: string): Promise<Result<PinnedFolder>> {
    return safe(
      this.http.request<PinnedFolder>('/files/pinned', {
        method: 'POST',
        body: { path, name },
      })
    )
  }

  /**
   * Unpin a folder by ID
   * @param id - Pinned folder ID
   */
  async unpin(id: string): Promise<Result<void>> {
    return safe(
      this.http
        .request(`/files/pinned/${id}`, {
          method: 'DELETE',
        })
        .then(() => {})
    )
  }
}

/**
 * Progress callback for file operations
 */
export type ProgressCallback = (loaded: number, total: number | null) => void

/**
 * File operations resource
 */
export class FilesResource {
  private http: HttpClient

  /** Pinned folders sub-resource */
  readonly pinned: PinnedSubResource

  constructor(http: HttpClient) {
    this.http = http
    this.pinned = new PinnedSubResource(http)
  }

  /**
   * List directory contents
   * @param path - Directory path (default: "/")
   */
  async list(path = '/'): Promise<Result<ListResponse>> {
    return safe(this.http.request<ListResponse>(`/files?path=${encodeURIComponent(path)}`))
  }

  /**
   * Read file content
   * @param path - File path
   */
  async read(path: string): Promise<Result<ReadResponse>> {
    return safe(this.http.request<ReadResponse>(`/files/read?path=${encodeURIComponent(path)}`))
  }

  /**
   * Read file with progress callback (for large files)
   * @param path - File path
   * @param onProgress - Progress callback
   */
  async readWithProgress(path: string, onProgress?: ProgressCallback): Promise<Result<ReadResponse>> {
    return safe(
      (async () => {
        const url = this.http.buildAuthUrl(`/files/read?path=${encodeURIComponent(path)}`)

        const response = await fetch(url)
        if (!response.ok) {
          const data = await response.json().catch(() => ({ error: 'Unknown error' }))
          throw new GloskiError(response.status, data.error || `HTTP ${response.status}`)
        }

        const contentLength = response.headers.get('Content-Length')
        const total = contentLength ? parseInt(contentLength, 10) : null

        const reader = response.body?.getReader()
        if (!reader) {
          // Fallback if streaming not supported
          const json = await response.json()
          return json.data ?? json
        }

        const chunks: Uint8Array[] = []
        let loaded = 0

        while (true) {
          const { done, value } = await reader.read()
          if (done) break

          chunks.push(value)
          loaded += value.length
          onProgress?.(loaded, total)
        }

        const allChunks = new Uint8Array(loaded)
        let position = 0
        for (const chunk of chunks) {
          allChunks.set(chunk, position)
          position += chunk.length
        }

        const text = new TextDecoder().decode(allChunks)
        const json = JSON.parse(text)
        return json.data ?? json
      })()
    )
  }

  /**
   * Read file as a stream (async generator)
   * Yields string chunks as they arrive
   * @param path - File path
   */
  async *readStream(path: string): AsyncGenerator<string, void, unknown> {
    const url = this.http.buildAuthUrl(`/files/read?path=${encodeURIComponent(path)}`)

    const response = await fetch(url)
    if (!response.ok) {
      const data = await response.json().catch(() => ({ error: 'Unknown error' }))
      throw new GloskiError(response.status, data.error || `HTTP ${response.status}`)
    }

    const reader = response.body?.getReader()
    if (!reader) {
      throw new GloskiError(0, 'Streaming not supported')
    }

    const decoder = new TextDecoder()

    try {
      while (true) {
        const { done, value } = await reader.read()
        if (done) break
        yield decoder.decode(value, { stream: true })
      }
    } finally {
      reader.releaseLock()
    }
  }

  /**
   * Write content to a file
   * @param path - File path
   * @param content - File content
   */
  async write(path: string, content: string): Promise<Result<void>> {
    return safe(
      this.http
        .request('/files/write', {
          method: 'POST',
          body: { path, content },
        })
        .then(() => {})
    )
  }

  /**
   * Create a directory
   * @param path - Directory path
   */
  async mkdir(path: string): Promise<Result<void>> {
    return safe(
      this.http
        .request('/files/mkdir', {
          method: 'POST',
          body: { path },
        })
        .then(() => {})
    )
  }

  /**
   * Delete a file or directory
   * @param path - Path to delete
   */
  async delete(path: string): Promise<Result<void>> {
    return safe(
      this.http
        .request(`/files?path=${encodeURIComponent(path)}`, {
          method: 'DELETE',
        })
        .then(() => {})
    )
  }

  /**
   * Rename or move a file/directory
   * @param oldPath - Current path
   * @param newPath - New path
   */
  async rename(oldPath: string, newPath: string): Promise<Result<void>> {
    return safe(
      this.http
        .request('/files/rename', {
          method: 'POST',
          body: { old_path: oldPath, new_path: newPath },
        })
        .then(() => {})
    )
  }

  /**
   * Upload a file
   * @param destPath - Destination directory path
   * @param file - File or Blob to upload
   * @param filename - Optional filename (uses file.name if not provided)
   */
  async upload(destPath: string, file: File | Blob, filename?: string): Promise<Result<UploadResponse>> {
    const formData = new FormData()
    formData.append('path', destPath)

    if (file instanceof File) {
      formData.append('file', file, filename || file.name)
    } else {
      formData.append('file', file, filename || 'upload')
    }

    return safe(this.http.upload<UploadResponse>('/files/upload', formData))
  }

  /**
   * Get download URL for a file (authenticated)
   * @param path - File path
   */
  getDownloadUrl(path: string): string {
    return this.http.buildAuthUrl('/files/download', {
      path,
    })
  }

  /**
   * Initialize a chunked upload session (for large files)
   * @param init - Chunked upload parameters
   */
  async initChunkedUpload(init: ChunkedUploadInit): Promise<Result<ChunkedUploadInfo>> {
    return safe(
      this.http.request<ChunkedUploadInfo>('/files/upload/init', {
        method: 'POST',
        body: init,
      })
    )
  }

  /**
   * Upload a single chunk
   * @param uploadId - Upload session ID from initChunkedUpload
   * @param destination - Destination directory
   * @param filename - Original filename
   * @param chunkIndex - 0-based chunk index
   * @param chunk - Chunk data (File or Blob)
   */
  async uploadChunk(
    uploadId: string,
    destination: string,
    filename: string,
    chunkIndex: number,
    chunk: File | Blob
  ): Promise<Result<ChunkedUploadChunkResponse>> {
    const formData = new FormData()
    formData.append('upload_id', uploadId)
    formData.append('destination', destination)
    formData.append('filename', filename)
    formData.append('chunk_index', String(chunkIndex))
    formData.append('chunk', chunk)

    return safe(this.http.upload<ChunkedUploadChunkResponse>('/files/upload/chunk', formData))
  }

  /**
   * Complete a chunked upload by assembling all chunks
   * @param request - Completion parameters
   */
  async completeChunkedUpload(request: ChunkedUploadCompleteRequest): Promise<Result<ChunkedUploadCompleteResponse>> {
    return safe(
      this.http.request<ChunkedUploadCompleteResponse>('/files/upload/complete', {
        method: 'POST',
        body: request,
      })
    )
  }

  /**
   * Abort a chunked upload and clean up temporary chunks
   * @param uploadId - Upload session ID
   * @param destination - Destination directory
   */
  async abortChunkedUpload(uploadId: string, destination: string): Promise<Result<void>> {
    return safe(
      this.http
        .request('/files/upload/abort', {
          method: 'POST',
          body: { upload_id: uploadId, destination },
        })
        .then(() => {})
    )
  }
}
