import { safe } from '../errors'
import type { HttpClient } from '../http'
import type { AuthStatus, Result } from '../types'

/**
 * Authentication resource
 */
export class AuthResource {
  private http: HttpClient

  constructor(http: HttpClient) {
    this.http = http
  }

  /**
   * Check authentication status
   */
  async status(): Promise<Result<AuthStatus>> {
    return safe(this.http.request<AuthStatus>('/auth/status'))
  }
}
