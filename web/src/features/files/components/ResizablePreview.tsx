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

  const handleMouseDown = (e: React.MouseEvent) => {
    e.preventDefault()
    setIsDragging(true)
  }

  const handleMouseMove = useCallback(
    (e: MouseEvent) => {
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

  useEffect(() => {
    if (isDragging) {
      document.addEventListener('mousemove', handleMouseMove)
      document.addEventListener('mouseup', handleMouseUp)
      document.body.style.cursor = 'ew-resize'
      document.body.style.userSelect = 'none'

      return () => {
        document.removeEventListener('mousemove', handleMouseMove)
        document.removeEventListener('mouseup', handleMouseUp)
        document.body.style.cursor = ''
        document.body.style.userSelect = ''
      }
    }
  }, [isDragging, handleMouseMove, handleMouseUp])

  return (
    <>
      {/* Resize Handle */}
      <div
        className={cn(
          'w-1 bg-border hover:bg-primary hover:w-1.5 cursor-ew-resize transition-all shrink-0 group relative',
          isDragging && 'bg-primary w-1.5'
        )}
        onMouseDown={handleMouseDown}
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
