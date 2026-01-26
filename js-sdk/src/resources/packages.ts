import type { HttpClient } from '../http'
import type { PackageManagerInfo, Package, PackageDetails } from '../types'

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
  async info(): Promise<PackageManagerInfo> {
    return this.http.request<PackageManagerInfo>('/packages/info')
  }

  /**
   * List installed packages
   */
  async listInstalled(): Promise<Package[]> {
    const response = await this.http.request<{ packages: Package[] }>('/packages/installed')
    return response.packages
  }

  /**
   * Check for available upgrades
   */
  async checkUpgrades(): Promise<Package[]> {
    const response = await this.http.request<{ packages: Package[] }>('/packages/upgrades')
    return response.packages
  }

  /**
   * Search for packages
   * @param query - Search query
   */
  async search(query: string): Promise<Package[]> {
    const response = await this.http.request<{ packages: Package[] }>(
      `/packages/search?q=${encodeURIComponent(query)}`
    )
    return response.packages
  }

  /**
   * Get package details
   * @param name - Package name
   */
  async get(name: string): Promise<PackageDetails> {
    return this.http.request<PackageDetails>(`/packages/${encodeURIComponent(name)}`)
  }
}
