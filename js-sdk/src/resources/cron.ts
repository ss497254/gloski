import { safe } from '../errors'
import type { HttpClient } from '../http'
import type { CronJobInput, CronJobsResponse, CronScope, Result } from '../types'

/**
 * Cron job management resource
 */
export class CronResource {
  private http: HttpClient

  constructor(http: HttpClient) {
    this.http = http
  }

  /**
   * List cron jobs
   * @param scope - Which jobs to list: "user", "system", or "all" (default: all)
   */
  async list(scope?: CronScope): Promise<Result<CronJobsResponse>> {
    const params = scope ? `?scope=${scope}` : ''
    return safe(this.http.request<CronJobsResponse>(`/cron/jobs${params}`))
  }

  /**
   * Add a new cron job
   * @param job - Cron job configuration
   */
  async add(job: CronJobInput): Promise<Result<void>> {
    return safe(
      this.http
        .request('/cron/jobs', {
          method: 'POST',
          body: job,
        })
        .then(() => {})
    )
  }

  /**
   * Remove a cron job by its schedule and command
   * @param schedule - Cron schedule expression
   * @param command - Command string
   */
  async remove(schedule: string, command: string): Promise<Result<void>> {
    return safe(
      this.http
        .request('/cron/jobs', {
          method: 'DELETE',
          body: { schedule, command },
        })
        .then(() => {})
    )
  }
}
