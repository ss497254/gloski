import { safe } from '../errors'
import type { HttpClient } from '../http'
import type {
  InstalledPackagesResponse,
  Package,
  PackageManagerInfo,
  PackageSearchResponse,
  Result,
  UpgradeInfo,
} from '../types'

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
   * @param limit - Maximum number of packages to return (default: 100)
   */
  async listInstalled(limit = 100): Promise<Result<InstalledPackagesResponse>> {
    return safe(this.http.request<InstalledPackagesResponse>(`/packages/installed?limit=${limit}`))
  }

  /**
   * Check for available upgrades
   */
  async checkUpgrades(): Promise<Result<UpgradeInfo>> {
    return safe(this.http.request<UpgradeInfo>('/packages/upgrades'))
  }

  /**
   * Search for packages
   * @param query - Search query
   * @param limit - Maximum number of results (default: 50)
   */
  async search(query: string, limit = 50): Promise<Result<PackageSearchResponse>> {
    return safe(
      this.http.request<PackageSearchResponse>(`/packages/search?q=${encodeURIComponent(query)}&limit=${limit}`)
    )
  }

  /**
   * Get package details
   * @param name - Package name
   */
  async get(name: string): Promise<Result<Package>> {
    return safe(this.http.request<Package>(`/packages/${encodeURIComponent(name)}`))
  }
}
