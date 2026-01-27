import { useRef, useEffect, useCallback, type KeyboardEvent } from 'react'
import { cn } from '@/shared/lib/utils'

interface CodeEditorProps {
  value: string
  onChange?: (value: string) => void
  readOnly?: boolean
  className?: string
}

export function CodeEditor({ value, onChange, readOnly = false, className }: CodeEditorProps) {
  const textareaRef = useRef<HTMLTextAreaElement>(null)
  const lineNumbersRef = useRef<HTMLDivElement>(null)

  const lines = value.split('\n')
  const lineCount = lines.length

  // Sync scroll between line numbers and textarea
  const handleScroll = useCallback(() => {
    if (textareaRef.current && lineNumbersRef.current) {
      lineNumbersRef.current.scrollTop = textareaRef.current.scrollTop
    }
  }, [])

  // Handle tab key to insert spaces
  const handleKeyDown = useCallback(
    (e: KeyboardEvent<HTMLTextAreaElement>) => {
      if (readOnly) return

      if (e.key === 'Tab') {
        e.preventDefault()
        const textarea = e.currentTarget
        const start = textarea.selectionStart
        const end = textarea.selectionEnd

        // Insert 2 spaces at cursor position
        const newValue = value.substring(0, start) + '  ' + value.substring(end)
        onChange?.(newValue)

        // Move cursor after the inserted spaces
        requestAnimationFrame(() => {
          textarea.selectionStart = textarea.selectionEnd = start + 2
        })
      }
    },
    [value, onChange, readOnly]
  )

  // Auto-resize textarea to content
  useEffect(() => {
    if (textareaRef.current) {
      textareaRef.current.style.height = 'auto'
      textareaRef.current.style.height = `${textareaRef.current.scrollHeight}px`
    }
  }, [value])

  return (
    <div className={cn('flex font-mono text-sm border rounded-md overflow-hidden', className)}>
      {/* Line numbers */}
      <div
        ref={lineNumbersRef}
        className="select-none text-right py-3 px-2 bg-muted/50 text-muted-foreground border-r overflow-hidden"
        style={{ minWidth: `${Math.max(3, String(lineCount).length + 1)}ch` }}
      >
        {Array.from({ length: lineCount }, (_, i) => (
          <div key={i + 1} className="leading-6 h-6">
            {i + 1}
          </div>
        ))}
      </div>

      {/* Editor */}
      <textarea
        ref={textareaRef}
        value={value}
        onChange={(e) => onChange?.(e.target.value)}
        onScroll={handleScroll}
        onKeyDown={handleKeyDown}
        readOnly={readOnly}
        spellCheck={false}
        className={cn(
          'flex-1 py-3 px-3 bg-background resize-none outline-none leading-6',
          'min-h-50',
          readOnly && 'cursor-default'
        )}
        style={{
          tabSize: 2,
        }}
      />
    </div>
  )
}
