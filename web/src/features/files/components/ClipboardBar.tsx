import { Button } from '@/ui/button'
import { Badge } from '@/ui/badge'
import { Copy, Scissors, X, Files } from 'lucide-react'
import type { ClipboardItem } from '../stores/files'

interface ClipboardBarProps {
  items: ClipboardItem[]
  operation: 'copy' | 'cut' | null
  onPaste: () => void
  onClear: () => void
  disabled?: boolean
}

export function ClipboardBar({ items, operation, onPaste, onClear, disabled }: ClipboardBarProps) {
  if (items.length === 0 || !operation) return null

  const Icon = operation === 'copy' ? Copy : Scissors

  return (
    <div className="flex items-center gap-3 px-4 py-2 bg-primary/10 border-b">
      <div className="flex items-center gap-2">
        <Icon className="h-4 w-4 text-primary" />
        <span className="text-sm font-medium">
          {items.length} {items.length === 1 ? 'item' : 'items'} {operation === 'copy' ? 'copied' : 'cut'}
        </span>
      </div>

      <div className="flex-1 flex items-center gap-1 overflow-x-auto scrollbar-none">
        {items.slice(0, 3).map((item, idx) => (
          <Badge key={idx} variant="secondary" className="text-xs shrink-0">
            {item.isDirectory ? <Files className="h-3 w-3 mr-1" /> : null}
            {item.name}
          </Badge>
        ))}
        {items.length > 3 && (
          <Badge variant="secondary" className="text-xs shrink-0">
            +{items.length - 3} more
          </Badge>
        )}
      </div>

      <div className="flex items-center gap-1 shrink-0">
        <Button variant="default" size="sm" onClick={onPaste} disabled={disabled}>
          Paste Here
        </Button>
        <Button variant="ghost" size="sm" onClick={onClear}>
          <X className="h-4 w-4" />
        </Button>
      </div>
    </div>
  )
}
