import { safe } from '../errors'
import type { HttpClient, RequestOptions } from '../http'
import type { Job, JobId, JobLogsResponse, JobStatus, JobsResponse, Result } from '../types'

/**
 * Job management resource
 *
 * Jobs are ad-hoc command executions that run on the device.
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
  async list(options?: RequestOptions): Promise<Result<Job[]>> {
    return safe(this.http.request<JobsResponse>('/jobs', options).then((response) => response.jobs))
  }

  /**
   * Start a new job
   * @param command - Command to execute
   * @param cwd - Working directory (optional)
   */
  async start(command: string, cwd?: string, options?: RequestOptions): Promise<Result<Job>> {
    return safe(
      this.http.request<Job>('/jobs', {
        ...options,
        method: 'POST',
        body: { command, cwd },
      })
    )
  }

  /**
   * Get job by ID
   * @param id - Job ID
   */
  async get(id: JobId, options?: RequestOptions): Promise<Result<Job>> {
    return safe(this.http.request<Job>(`/jobs/${id}`, options))
  }

  /**
   * Get job logs
   * @param id - Job ID
   */
  async getLogs(id: JobId, options?: RequestOptions): Promise<Result<string[]>> {
    return safe(this.http.request<JobLogsResponse>(`/jobs/${id}/logs`, options).then((response) => response.logs))
  }

  /**
   * Stop/kill a running job
   * @param id - Job ID
   */
  async stop(id: JobId, options?: RequestOptions): Promise<Result<void>> {
    return safe(
      this.http
        .request(`/jobs/${id}/stop`, {
          ...options,
          method: 'POST',
        })
        .then(() => {})
    )
  }

  /**
   * Delete a job and its logs
   * @param id - Job ID
   */
  async delete(id: JobId, options?: RequestOptions): Promise<Result<void>> {
    return safe(
      this.http
        .request(`/jobs/${id}`, {
          ...options,
          method: 'DELETE',
        })
        .then(() => {})
    )
  }

  /**
   * Stream job logs as they are produced (polling-based).
   * Yields arrays of new log lines since the last poll.
   * Stops automatically when the job reaches a terminal status.
   *
   * @param id - Job ID
   * @param options - Polling interval (default: 1000ms) and AbortSignal
   *
   * @example
   * ```ts
   * for await (const lines of client.jobs.streamLogs(job.id)) {
   *   lines.forEach(line => console.log(line))
   * }
   * ```
   */
  async *streamLogs(
    id: JobId,
    options?: { interval?: number; signal?: AbortSignal }
  ): AsyncGenerator<string[], void, unknown> {
    const interval = options?.interval ?? 1000
    const terminalStatuses: JobStatus[] = ['finished', 'failed', 'stopped']
    let lastLogCount = 0

    while (!options?.signal?.aborted) {
      const result = await this.getLogs(id)
      if (result.error) throw result.error

      const logs = result.data
      if (logs.length > lastLogCount) {
        yield logs.slice(lastLogCount)
        lastLogCount = logs.length
      }

      // Check if job is finished
      const job = await this.get(id)
      if (job.error) throw job.error
      if (terminalStatuses.includes(job.data.status)) {
        // One final check for remaining logs
        const final = await this.getLogs(id)
        if (!final.error && final.data.length > lastLogCount) {
          yield final.data.slice(lastLogCount)
        }
        return
      }

      await new Promise((r) => setTimeout(r, interval))
    }
  }
}
