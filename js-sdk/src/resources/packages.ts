import { safe } from '../errors'
import type { HttpClient, RequestOptions } from '../http'
import type {
  InstalledPackagesResponse,
  PackageManagerInfo,
  PackageSearchResponse,
  Result,
  SystemPackage,
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
  async info(options?: RequestOptions): Promise<Result<PackageManagerInfo>> {
    return safe(this.http.request<PackageManagerInfo>('/packages/info', options))
  }

  /**
   * List installed packages
   * @param limit - Maximum number of packages to return (default: 100)
   */
  async listInstalled(limit = 100, options?: RequestOptions): Promise<Result<InstalledPackagesResponse>> {
    return safe(this.http.request<InstalledPackagesResponse>(`/packages/installed?limit=${limit}`, options))
  }

  /**
   * Check for available upgrades
   */
  async checkUpgrades(options?: RequestOptions): Promise<Result<UpgradeInfo>> {
    return safe(this.http.request<UpgradeInfo>('/packages/upgrades', options))
  }

  /**
   * Search for packages
   * @param query - Search query
   * @param limit - Maximum number of results (default: 50)
   */
  async search(query: string, limit = 50, options?: RequestOptions): Promise<Result<PackageSearchResponse>> {
    return safe(
      this.http.request<PackageSearchResponse>(
        `/packages/search?q=${encodeURIComponent(query)}&limit=${limit}`,
        options
      )
    )
  }

  /**
   * Get package details
   * @param name - Package name
   */
  async get(name: string, options?: RequestOptions): Promise<Result<SystemPackage>> {
    return safe(this.http.request<SystemPackage>(`/packages/${encodeURIComponent(name)}`, options))
  }
}
