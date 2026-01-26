import type { HttpClient } from '../http'
import type { SystemdUnit, SystemdResponse, SystemdLogsResponse, SystemdAction } from '../types'

/**
 * Systemd service management resource
 */
export class SystemdResource {
  private http: HttpClient

  constructor(http: HttpClient) {
    this.http = http
  }

  /**
   * List systemd units
   * @param userMode - Use user mode (default: true)
   */
  async list(userMode = true): Promise<SystemdUnit[]> {
    const response = await this.http.request<SystemdResponse>(
      `/systemd?user=${userMode}`
    )
    return response.units
  }

  /**
   * Perform action on a systemd unit
   * @param unit - Unit name
   * @param action - Action to perform
   * @param userMode - Use user mode (default: true)
   */
  async action(unit: string, action: SystemdAction, userMode = true): Promise<void> {
    await this.http.request(`/systemd/${unit}/${action}?user=${userMode}`, {
      method: 'POST',
    })
  }

  /**
   * Start a systemd unit
   */
  async start(unit: string, userMode = true): Promise<void> {
    return this.action(unit, 'start', userMode)
  }

  /**
   * Stop a systemd unit
   */
  async stop(unit: string, userMode = true): Promise<void> {
    return this.action(unit, 'stop', userMode)
  }

  /**
   * Restart a systemd unit
   */
  async restart(unit: string, userMode = true): Promise<void> {
    return this.action(unit, 'restart', userMode)
  }

  /**
   * Enable a systemd unit
   */
  async enable(unit: string, userMode = true): Promise<void> {
    return this.action(unit, 'enable', userMode)
  }

  /**
   * Disable a systemd unit
   */
  async disable(unit: string, userMode = true): Promise<void> {
    return this.action(unit, 'disable', userMode)
  }

  /**
   * Get logs for a systemd unit
   * @param unit - Unit name
   * @param userMode - Use user mode (default: true)
   * @param lines - Number of lines to return (default: 100)
   */
  async getLogs(unit: string, userMode = true, lines = 100): Promise<string> {
    const response = await this.http.request<SystemdLogsResponse>(
      `/systemd/${unit}/logs?user=${userMode}&lines=${lines}`
    )
    return response.logs
  }
}
