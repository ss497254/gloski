import { MemoryStick, Database } from 'lucide-react'
import { ProgressBar } from '@/shared/components'
import { formatBytes } from '@/shared/lib/utils'
import type { MemoryStats, SwapStats } from '@/shared/lib/types'

interface MemoryWidgetProps {
  memory: MemoryStats
  swap: SwapStats
}

export function MemoryWidget({ memory, swap }: MemoryWidgetProps) {
  return (
    <div className="rounded-xl border bg-card p-6 shadow-sm">
      <div className="flex items-center gap-2 mb-6">
        <MemoryStick className="h-5 w-5 text-muted-foreground" />
        <h2 className="text-lg font-semibold">Memory</h2>
      </div>

      {/* Memory breakdown */}
      <div className="space-y-4">
        {/* RAM */}
        <div>
          <div className="flex items-center justify-between mb-2">
            <span className="text-sm">RAM Usage</span>
            <span className="text-sm font-medium">
              {formatBytes(memory.used)} / {formatBytes(memory.total)}
            </span>
          </div>
          <ProgressBar value={memory.used_percent} size="md" />
          <div className="flex justify-between text-xs text-muted-foreground mt-2">
            <span>Available: {formatBytes(memory.available)}</span>
            <span>{memory.used_percent.toFixed(1)}%</span>
          </div>
        </div>

        {/* Memory details */}
        <div className="grid grid-cols-2 gap-3 pt-2">
          <div className="p-3 rounded-lg bg-muted/50">
            <p className="text-xs text-muted-foreground">Cached</p>
            <p className="text-sm font-medium">{formatBytes(memory.cached)}</p>
          </div>
          <div className="p-3 rounded-lg bg-muted/50">
            <p className="text-xs text-muted-foreground">Buffers</p>
            <p className="text-sm font-medium">{formatBytes(memory.buffers)}</p>
          </div>
        </div>

        {/* Swap */}
        {swap.total > 0 && (
          <div className="pt-4 border-t">
            <div className="flex items-center gap-2 mb-3">
              <Database className="h-4 w-4 text-muted-foreground" />
              <span className="text-sm font-medium">Swap</span>
            </div>
            <div className="flex items-center justify-between mb-2">
              <span className="text-xs text-muted-foreground">
                {formatBytes(swap.used)} / {formatBytes(swap.total)}
              </span>
              <span className="text-xs">{swap.used_percent.toFixed(1)}%</span>
            </div>
            <ProgressBar value={swap.used_percent} size="sm" />
          </div>
        )}
      </div>
    </div>
  )
}
