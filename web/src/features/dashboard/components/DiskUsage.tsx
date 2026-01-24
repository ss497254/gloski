import { HardDrive } from 'lucide-react'
import { ProgressBar } from '@/shared/components'
import { formatBytes } from '@/shared/lib/utils'
import type { DiskStats } from '@/shared/lib/types'

interface DiskUsageProps {
  disks: DiskStats[]
}

export function DiskUsage({ disks }: DiskUsageProps) {
  if (disks.length === 0) {
    return null
  }

  return (
    <div className="rounded-xl border bg-card p-6 shadow-sm">
      <div className="flex items-center gap-2 mb-6">
        <HardDrive className="h-5 w-5 text-muted-foreground" />
        <h2 className="text-lg font-semibold">Disk Usage</h2>
      </div>

      <div className="space-y-5">
        {disks.map((disk) => (
          <div key={disk.mount_point}>
            <div className="flex items-center justify-between mb-2">
              <div>
                <p className="text-sm font-medium">{disk.mount_point}</p>
                <p className="text-xs text-muted-foreground">
                  {disk.device} ({disk.fs_type})
                </p>
              </div>
              <div className="text-right">
                <p className="text-sm font-medium">
                  {formatBytes(disk.used)} / {formatBytes(disk.total)}
                </p>
                <p className="text-xs text-muted-foreground">
                  {formatBytes(disk.free)} free
                </p>
              </div>
            </div>
            <ProgressBar value={disk.used_percent} size="md" />
          </div>
        ))}
      </div>
    </div>
  )
}
