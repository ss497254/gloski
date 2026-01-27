import { cn } from '@/shared/lib/utils'
import { Upload } from 'lucide-react'
import { type DragEvent, type ReactNode, useCallback, useRef, useState } from 'react'

interface DropZoneProps {
  onDrop: (files: File[]) => void
  children: ReactNode
  disabled?: boolean
  className?: string
}

export function DropZone({ onDrop, children, disabled = false, className }: DropZoneProps) {
  const [isDragging, setIsDragging] = useState(false)
  const dragCounter = useRef(0)

  const handleDragEnter = useCallback(
    (e: DragEvent) => {
      e.preventDefault()
      e.stopPropagation()
      if (disabled) return

      dragCounter.current++
      if (e.dataTransfer.items && e.dataTransfer.items.length > 0) {
        setIsDragging(true)
      }
    },
    [disabled]
  )

  const handleDragLeave = useCallback(
    (e: DragEvent) => {
      e.preventDefault()
      e.stopPropagation()
      if (disabled) return

      dragCounter.current--
      if (dragCounter.current === 0) {
        setIsDragging(false)
      }
    },
    [disabled]
  )

  const handleDragOver = useCallback(
    (e: DragEvent) => {
      e.preventDefault()
      e.stopPropagation()
      if (disabled) return
      e.dataTransfer.dropEffect = 'copy'
    },
    [disabled]
  )

  const handleDrop = useCallback(
    (e: DragEvent) => {
      e.preventDefault()
      e.stopPropagation()
      if (disabled) return

      setIsDragging(false)
      dragCounter.current = 0

      const files = Array.from(e.dataTransfer.files)
      if (files.length > 0) {
        onDrop(files)
      }
    },
    [onDrop, disabled]
  )

  return (
    <div
      onDragEnter={handleDragEnter}
      onDragLeave={handleDragLeave}
      onDragOver={handleDragOver}
      onDrop={handleDrop}
      className={cn('relative', className)}
    >
      {children}

      {/* Drop overlay */}
      {isDragging && (
        <div className="absolute inset-0 z-50 bg-primary/10 border-2 border-dashed border-primary rounded-lg flex items-center justify-center backdrop-blur-sm">
          <div className="flex flex-col items-center gap-2 text-primary">
            <Upload className="h-12 w-12" />
            <span className="text-lg font-medium">Drop files to upload</span>
          </div>
        </div>
      )}
    </div>
  )
}
