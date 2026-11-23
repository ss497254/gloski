export type StatusTone = 'success' | 'warning' | 'info' | 'destructive' | 'muted'

/** Map the status strings used across the API to one of five visual tones. */
export function statusTone(status: string): StatusTone {
  switch (status.toLowerCase()) {
    case 'online':
    case 'ok':
    case 'healthy':
    case 'success':
    case 'finished':
    case 'connected':
      return 'success'
    case 'connecting':
    case 'degraded':
    case 'warning':
    case 'queued':
    case 'pending':
      return 'warning'
    case 'running':
    case 'active':
    case 'info':
      return 'info'
    case 'unauthorized':
    case 'failed':
    case 'error':
    case 'critical':
    case 'destructive':
      return 'destructive'
    default:
      // offline, stopped, skipped, unknown and idle are all neutral.
      return 'muted'
  }
}

/** Tinted icon container used by stat cards and summary tiles. */
export const toneSurface: Record<StatusTone, string> = {
  success: 'bg-success/12 text-success',
  warning: 'bg-warning/15 text-warning',
  info: 'bg-info/12 text-info',
  destructive: 'bg-destructive/12 text-destructive',
  muted: 'bg-muted text-muted-foreground',
}
