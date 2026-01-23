import { Network, ArrowDownToLine, ArrowUpFromLine, Wifi, WifiOff } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { NetworkStats } from '@/lib/types'

interface NetworkStatsWidgetProps {
  network: NetworkStats
}

export function NetworkStatsWidget({ network }: NetworkStatsWidgetProps) {
  return (
    <div className="rounded-xl border bg-card p-6 shadow-sm">
      <div className="flex items-center gap-2 mb-6">
        <Network className="h-5 w-5 text-muted-foreground" />
        <h2 className="text-lg font-semibold">Network</h2>
      </div>

      {/* Total traffic */}
      <div className="grid grid-cols-2 gap-4 mb-6">
        <div className="flex items-center gap-3 p-3 rounded-lg bg-green-500/10">
          <ArrowDownToLine className="h-5 w-5 text-green-500" />
          <div>
            <p className="text-xs text-muted-foreground">Download</p>
            <p className="text-lg font-semibold">{formatBytes(network.total_rx)}</p>
          </div>
        </div>
        <div className="flex items-center gap-3 p-3 rounded-lg bg-blue-500/10">
          <ArrowUpFromLine className="h-5 w-5 text-blue-500" />
          <div>
            <p className="text-xs text-muted-foreground">Upload</p>
            <p className="text-lg font-semibold">{formatBytes(network.total_tx)}</p>
          </div>
        </div>
      </div>

      {/* Interfaces */}
      <div className="space-y-3">
        <p className="text-sm font-medium text-muted-foreground">Interfaces</p>
        {network.interfaces.map((iface) => (
          <div
            key={iface.name}
            className="flex items-center justify-between py-2 border-b last:border-0"
          >
            <div className="flex items-center gap-2">
              {iface.up ? (
                <Wifi className="h-4 w-4 text-green-500" />
              ) : (
                <WifiOff className="h-4 w-4 text-muted-foreground" />
              )}
              <span className={cn('text-sm', !iface.up && 'text-muted-foreground')}>
                {iface.name}
              </span>
            </div>
            <div className="flex items-center gap-4 text-xs">
              <span className="text-green-600">
                <ArrowDownToLine className="inline h-3 w-3 mr-1" />
                {formatBytes(iface.rx_bytes)}
              </span>
              <span className="text-blue-600">
                <ArrowUpFromLine className="inline h-3 w-3 mr-1" />
                {formatBytes(iface.tx_bytes)}
              </span>
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}

function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i]
}
