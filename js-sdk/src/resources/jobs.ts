import { safe } from '../errors'
import type { HttpClient } from '../http'
import type { Job, JobLogsResponse, JobsResponse, Result } from '../types'

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
  async list(): Promise<Result<Job[]>> {
    return safe(this.http.request<JobsResponse>('/jobs').then(r => r.jobs))
  }

  /**
   * Start a new job
   * @param command - Command to execute
   * @param cwd - Working directory (optional)
   */
  async start(command: string, cwd?: string): Promise<Result<Job>> {
    return safe(this.http.request<Job>('/jobs', {
      method: 'POST',
      body: { command, cwd },
    }))
  }

  /**
   * Get job by ID
   * @param id - Job ID
   */
  async get(id: string): Promise<Result<Job>> {
    return safe(this.http.request<Job>(`/jobs/${id}`))
  }

  /**
   * Get job logs
   * @param id - Job ID
   */
  async getLogs(id: string): Promise<Result<string[]>> {
    return safe(this.http.request<JobLogsResponse>(`/jobs/${id}/logs`).then(r => r.logs))
  }

  /**
   * Stop/kill a running job
   * @param id - Job ID
   */
  async stop(id: string): Promise<Result<void>> {
    return safe(this.http.request(`/jobs/${id}/stop`, {
      method: 'POST',
    }).then(() => {}))
  }

  /**
   * Delete a job and its logs
   * @param id - Job ID
   */
  async delete(id: string): Promise<Result<void>> {
    return safe(this.http.request(`/jobs/${id}`, {
      method: 'DELETE',
    }).then(() => {}))
  }
}
