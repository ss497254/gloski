import type { HttpClient } from '../http'
import type { AuthStatus } from '../types'

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
  async status(): Promise<AuthStatus> {
    return this.http.request<AuthStatus>('/auth/status')
  }
}
