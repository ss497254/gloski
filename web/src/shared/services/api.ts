import { GloskiClient } from '@gloski/sdk'

/**
 * Check server health (static, no auth required)
 */
export async function checkServerHealth(
  serverUrl: string
): Promise<{ status: string }> {
  return GloskiClient.checkHealth(serverUrl, 5000)
}

// Re-export SDK types and classes
export { GloskiClient, GloskiError } from '@gloski/sdk'
export type { TerminalConnection } from '@gloski/sdk'
