import { useState, useRef, useEffect, useCallback } from 'react'
import { cn } from '@/shared/lib/utils'

interface ResizablePreviewProps {
  children: React.ReactNode
  defaultWidth?: number
  minWidth?: number
  maxWidth?: number
  onWidthChange?: (width: number) => void
}

export function ResizablePreview({
  children,
  defaultWidth = 50,
  minWidth = 30,
  maxWidth = 80,
  onWidthChange,
}: ResizablePreviewProps) {
  const [width, setWidth] = useState(defaultWidth)
  const [isDragging, setIsDragging] = useState(false)
  const containerRef = useRef<HTMLDivElement>(null)

  const handlePointerDown = (e: React.PointerEvent<HTMLDivElement>) => {
    e.preventDefault()
    e.currentTarget.setPointerCapture(e.pointerId)
    setIsDragging(true)
  }

  const handlePointerMove = useCallback(
    (e: React.PointerEvent<HTMLDivElement>) => {
      if (!isDragging || !containerRef.current) return

      const container = containerRef.current.parentElement
      if (!container) return

      const containerRect = container.getBoundingClientRect()
      const newWidth = ((containerRect.right - e.clientX) / containerRect.width) * 100

      const clampedWidth = Math.min(Math.max(newWidth, minWidth), maxWidth)
      setWidth(clampedWidth)
      onWidthChange?.(clampedWidth)
    },
    [isDragging, minWidth, maxWidth, onWidthChange]
  )

  const handleMouseUp = useCallback(() => {
    setIsDragging(false)
  }, [])

  const handleKeyDown = (e: React.KeyboardEvent<HTMLDivElement>) => {
    let nextWidth = width
    if (e.key === 'ArrowLeft') nextWidth += e.shiftKey ? 5 : 1
    else if (e.key === 'ArrowRight') nextWidth -= e.shiftKey ? 5 : 1
    else if (e.key === 'Home') nextWidth = maxWidth
    else if (e.key === 'End') nextWidth = minWidth
    else return

    e.preventDefault()
    const clampedWidth = Math.min(Math.max(nextWidth, minWidth), maxWidth)
    setWidth(clampedWidth)
    onWidthChange?.(clampedWidth)
  }

  useEffect(() => {
    if (!isDragging) return
    const previousCursor = document.body.style.cursor
    const previousUserSelect = document.body.style.userSelect
    document.body.style.cursor = 'ew-resize'
    document.body.style.userSelect = 'none'
    return () => {
      document.body.style.cursor = previousCursor
      document.body.style.userSelect = previousUserSelect
    }
  }, [isDragging])

  return (
    <>
      {/* Resize Handle */}
      <div
        role="separator"
        aria-label="Resize file preview"
        aria-orientation="vertical"
        aria-valuemin={minWidth}
        aria-valuemax={maxWidth}
        aria-valuenow={Math.round(width)}
        tabIndex={0}
        className={cn(
          'w-1 bg-border hover:bg-primary hover:w-1.5 cursor-ew-resize transition-all shrink-0 group relative',
          isDragging && 'bg-primary w-1.5'
        )}
        onPointerDown={handlePointerDown}
        onPointerMove={handlePointerMove}
        onPointerUp={handleMouseUp}
        onPointerCancel={handleMouseUp}
        onKeyDown={handleKeyDown}
      >
        {/* Drag indicator */}
        <div className="absolute inset-y-0 left-1/2 -translate-x-1/2 w-1 opacity-0 group-hover:opacity-100 transition-opacity">
          <div className="h-full w-full flex items-center justify-center">
            <div className="w-0.5 h-8 bg-primary/50 rounded-full" />
          </div>
        </div>
      </div>

      {/* Preview Content */}
      <div ref={containerRef} style={{ width: `${width}%` }} className="flex flex-col bg-muted/20">
        {children}
      </div>
    </>
  )
}
