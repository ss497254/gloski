import { safe } from '../errors'
import type { HttpClient, RequestOptions } from '../http'
import type { CronJobsResponse, CronScope, NewCronJob, Result } from '../types'

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
  async list(scope?: CronScope, options?: RequestOptions): Promise<Result<CronJobsResponse>> {
    const params = scope ? `?scope=${scope}` : ''
    return safe(this.http.request<CronJobsResponse>(`/cron/jobs${params}`, options))
  }

  /**
   * Add a new cron job
   * @param job - Cron job configuration
   */
  async add(job: NewCronJob, options?: RequestOptions): Promise<Result<void>> {
    return safe(
      this.http
        .request('/cron/jobs', {
          ...options,
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
  async remove(schedule: string, command: string, options?: RequestOptions): Promise<Result<void>> {
    return safe(
      this.http
        .request('/cron/jobs', {
          ...options,
          method: 'DELETE',
          body: { schedule, command },
        })
        .then(() => {})
    )
  }
}
