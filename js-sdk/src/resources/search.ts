import { safe } from '../errors'
import type { HttpClient, RequestOptions } from '../http'
import type { Result, SearchOptions, SearchResponse } from '../types'

/**
 * File search resource
 */
export class SearchResource {
  private http: HttpClient

  constructor(http: HttpClient) {
    this.http = http
  }

  /**
   * Search for files
   * @param options - Search options
   * @param requestOptions - Request options (signal, timeout, etc.)
   */
  async search(options: SearchOptions, requestOptions?: RequestOptions): Promise<Result<SearchResponse>> {
    const params = new URLSearchParams({
      path: options.path,
      q: options.query,
      limit: String(options.limit ?? 100),
    })

    if (options.content) {
      params.set('content', 'true')
    }

    return safe(this.http.request<SearchResponse>(`/search?${params}`, requestOptions))
  }

  /**
   * Search for files by name
   * @param path - Starting path
   * @param query - Search query
   * @param limit - Max results
   */
  async byName(path: string, query: string, limit = 100, options?: RequestOptions): Promise<Result<SearchResponse>> {
    return this.search({ path, query, content: false, limit }, options)
  }

  /**
   * Search within file contents
   * @param path - Starting path
   * @param query - Search query
   * @param limit - Max results
   */
  async byContent(path: string, query: string, limit = 100, options?: RequestOptions): Promise<Result<SearchResponse>> {
    return this.search({ path, query, content: true, limit }, options)
  }
}
