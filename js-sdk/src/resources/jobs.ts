import type { HttpClient } from '../http'
import type { Job, JobsResponse, JobLogsResponse } from '../types'

/**
 * Job management resource
 *
 * Jobs are ad-hoc command executions that run on the server.
 * Output is logged to files and persisted in the database.
 */
export class JobsResource {
  private http: HttpClient

  constructor(http: HttpClient) {
    this.http = http
  }

  /**
   * List all jobs
   */
  async list(): Promise<Job[]> {
    const response = await this.http.request<JobsResponse>('/jobs')
    return response.jobs
  }

  /**
   * Start a new job
   * @param command - Command to execute
   * @param cwd - Working directory (optional)
   */
  async start(command: string, cwd?: string): Promise<Job> {
    return this.http.request<Job>('/jobs', {
      method: 'POST',
      body: { command, cwd },
    })
  }

  /**
   * Get job by ID
   * @param id - Job ID
   */
  async get(id: string): Promise<Job> {
    return this.http.request<Job>(`/jobs/${id}`)
  }

  /**
   * Get job logs
   * @param id - Job ID
   */
  async getLogs(id: string): Promise<string[]> {
    const response = await this.http.request<JobLogsResponse>(`/jobs/${id}/logs`)
    return response.logs
  }

  /**
   * Stop/kill a running job
   * @param id - Job ID
   */
  async stop(id: string): Promise<void> {
    await this.http.request(`/jobs/${id}/stop`, {
      method: 'POST',
    })
  }

  /**
   * Delete a job and its logs
   * @param id - Job ID
   */
  async delete(id: string): Promise<void> {
    await this.http.request(`/jobs/${id}`, {
      method: 'DELETE',
    })
  }
}
