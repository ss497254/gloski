import { safe } from '../errors'
import type { HttpClient } from '../http'
import type { Package, PackageDetails, PackageManagerInfo, Result } from '../types'

/**
 * Package manager resource
 */
export class PackagesResource {
  private http: HttpClient

  constructor(http: HttpClient) {
    this.http = http
  }

  /**
   * Get package manager info
   */
  async info(): Promise<Result<PackageManagerInfo>> {
    return safe(this.http.request<PackageManagerInfo>('/packages/info'))
  }

  /**
   * List installed packages
   */
  async listInstalled(): Promise<Result<Package[]>> {
    return safe(this.http.request<{ packages: Package[] }>('/packages/installed').then(r => r.packages))
  }

  /**
   * Check for available upgrades
   */
  async checkUpgrades(): Promise<Result<Package[]>> {
    return safe(this.http.request<{ packages: Package[] }>('/packages/upgrades').then(r => r.packages))
  }

  /**
   * Search for packages
   * @param query - Search query
   */
  async search(query: string): Promise<Result<Package[]>> {
    return safe(this.http.request<{ packages: Package[] }>(`/packages/search?q=${encodeURIComponent(query)}`).then(r => r.packages))
  }

  /**
   * Get package details
   * @param name - Package name
   */
  async get(name: string): Promise<Result<PackageDetails>> {
    return safe(this.http.request<PackageDetails>(`/packages/${encodeURIComponent(name)}`))
  }
}
