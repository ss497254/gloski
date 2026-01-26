import type { HttpClient } from '../http'
import type { CronJob, CronJobInput } from '../types'

/**
 * Cron job management resource
 */
export class CronResource {
  private http: HttpClient

  constructor(http: HttpClient) {
    this.http = http
  }

  /**
   * List all cron jobs
   */
  async list(): Promise<CronJob[]> {
    const response = await this.http.request<{ jobs: CronJob[] }>('/cron/jobs')
    return response.jobs
  }

  /**
   * Add a new cron job
   * @param job - Cron job configuration
   */
  async add(job: CronJobInput): Promise<void> {
    await this.http.request('/cron/jobs', {
      method: 'POST',
      body: job,
    })
  }

  /**
   * Remove a cron job
   * @param id - Job ID or line identifier
   */
  async remove(id: string): Promise<void> {
    await this.http.request('/cron/jobs', {
      method: 'DELETE',
      body: { id },
    })
  }
}
