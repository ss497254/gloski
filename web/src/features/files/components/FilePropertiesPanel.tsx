import { Badge } from '@/ui/badge'
import { Button } from '@/ui/button'
import { ScrollArea } from '@/ui/scroll-area'
import { Separator } from '@/ui/separator'
import { Input } from '@/ui/input'
import type { FileEntry } from '@/shared/lib/types'
import { cn } from '@/shared/lib/utils'
import {
  Calendar,
  Clock,
  Copy,
  File,
  Folder,
  HardDrive,
  X,
} from 'lucide-react'
import { toast } from 'sonner'
import { formatDistanceToNow } from 'date-fns'

interface FilePropertiesPanelProps {
  entry: FileEntry
  onClose: () => void
}

function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return `${(bytes / Math.pow(k, i)).toFixed(2)} ${sizes[i]}`
}

export function FilePropertiesPanel({ entry, onClose }: FilePropertiesPanelProps) {

  const handleCopyPath = () => {
    navigator.clipboard.writeText(entry.path)
    toast.success('Path copied to clipboard')
  }

  return (
    <div className="w-80 border-l bg-background flex flex-col">
      {/* Header */}
      <div className="p-4 border-b flex items-center justify-between">
        <h3 className="font-semibold text-sm">Properties</h3>
        <Button variant="ghost" size="icon" onClick={onClose} className="h-8 w-8">
          <X className="h-4 w-4" />
        </Button>
      </div>

      <ScrollArea className="flex-1">
        <div className="p-4 space-y-6">
          {/* Icon and Name */}
          <div className="flex flex-col items-center gap-3 py-4">
            <div
              className={cn(
                'h-16 w-16 rounded-xl flex items-center justify-center',
                entry.type === 'directory' ? 'bg-primary/10' : 'bg-muted'
              )}
            >
              {entry.type === 'directory' ? (
                <Folder className="h-8 w-8 text-primary" />
              ) : (
                <File className="h-8 w-8 text-muted-foreground" />
              )}
            </div>
            <div className="text-center">
              <p className="font-semibold text-sm break-all">{entry.name}</p>
              <Badge variant="outline" className="mt-1 text-xs">
                {entry.type === 'directory' ? 'Folder' : 'File'}
              </Badge>
            </div>
          </div>

          <Separator />

          {/* General Info */}
          <div className="space-y-3">
            <h4 className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">
              General
            </h4>

            <div className="space-y-2">
              <div className="flex items-center gap-2 text-sm">
                <HardDrive className="h-4 w-4 text-muted-foreground shrink-0" />
                <span className="text-muted-foreground">Size:</span>
                <span className="font-medium ml-auto">{formatBytes(entry.size)}</span>
              </div>

              <div className="flex items-center gap-2 text-sm">
                <Calendar className="h-4 w-4 text-muted-foreground shrink-0" />
                <span className="text-muted-foreground">Modified:</span>
                <span className="font-medium ml-auto text-xs">
                  {formatDistanceToNow(new Date(entry.modified), { addSuffix: true })}
                </span>
              </div>

              <div className="flex items-center gap-2 text-sm">
                <Clock className="h-4 w-4 text-muted-foreground shrink-0" />
                <span className="text-muted-foreground">Created:</span>
                <span className="font-medium ml-auto text-xs">
                  {formatDistanceToNow(new Date(entry.modified), { addSuffix: true })}
                </span>
              </div>
            </div>
          </div>

          <Separator />

          {/* Path */}
          <div className="space-y-2">
            <h4 className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">
              Path
            </h4>
            <div className="flex gap-2">
              <Input
                value={entry.path}
                readOnly
                className="text-xs font-mono"
              />
              <Button variant="outline" size="icon" onClick={handleCopyPath} className="shrink-0">
                <Copy className="h-4 w-4" />
              </Button>
            </div>
          </div>
        </div>
      </ScrollArea>
    </div>
  )
}
