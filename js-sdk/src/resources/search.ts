import type { HttpClient } from '../http'
import type { SearchOptions, SearchResponse, SearchResult } from '../types'

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
   */
  async search(options: SearchOptions): Promise<SearchResult[]> {
    const params = new URLSearchParams({
      path: options.path,
      q: options.query,
      limit: String(options.limit ?? 100),
    })

    if (options.content) {
      params.set('content', 'true')
    }

    const response = await this.http.request<SearchResponse>(`/search?${params}`)
    return response.results
  }

  /**
   * Search for files by name
   * @param path - Starting path
   * @param query - Search query
   * @param limit - Max results
   */
  async byName(path: string, query: string, limit = 100): Promise<SearchResult[]> {
    return this.search({ path, query, content: false, limit })
  }

  /**
   * Search within file contents
   * @param path - Starting path
   * @param query - Search query
   * @param limit - Max results
   */
  async byContent(path: string, query: string, limit = 100): Promise<SearchResult[]> {
    return this.search({ path, query, content: true, limit })
  }
}
