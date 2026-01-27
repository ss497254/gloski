import { defaultKeymap, history, historyKeymap, indentWithTab } from '@codemirror/commands'
import { bracketMatching, defaultHighlightStyle, syntaxHighlighting } from '@codemirror/language'
import { EditorState, type Extension } from '@codemirror/state'
import { oneDark } from '@codemirror/theme-one-dark'
import { EditorView, highlightActiveLine, highlightActiveLineGutter, keymap, lineNumbers } from '@codemirror/view'
import { useEffect, useMemo, useRef } from 'react'

// Language imports
import { cpp } from '@codemirror/lang-cpp'
import { css } from '@codemirror/lang-css'
import { go } from '@codemirror/lang-go'
import { html } from '@codemirror/lang-html'
import { java } from '@codemirror/lang-java'
import { javascript } from '@codemirror/lang-javascript'
import { json } from '@codemirror/lang-json'
import { markdown } from '@codemirror/lang-markdown'
import { python } from '@codemirror/lang-python'
import { rust } from '@codemirror/lang-rust'
import { sql } from '@codemirror/lang-sql'
import { xml } from '@codemirror/lang-xml'
import { yaml } from '@codemirror/lang-yaml'

import { cn } from '@/shared/lib/utils'

// Map file extensions to language support
function getLanguageExtension(filename?: string): Extension | null {
  if (!filename) return null

  const ext = filename.split('.').pop()?.toLowerCase()
  if (!ext) return null

  switch (ext) {
    // JavaScript/TypeScript
    case 'js':
    case 'mjs':
    case 'cjs':
      return javascript()
    case 'jsx':
      return javascript({ jsx: true })
    case 'ts':
      return javascript({ typescript: true })
    case 'tsx':
      return javascript({ jsx: true, typescript: true })

    // Web
    case 'html':
    case 'htm':
      return html()
    case 'css':
    case 'scss':
    case 'less':
      return css()

    // Data formats
    case 'json':
      return json()
    case 'xml':
    case 'svg':
      return xml()
    case 'yaml':
    case 'yml':
      return yaml()

    // Markdown
    case 'md':
    case 'mdx':
      return markdown()

    // Python
    case 'py':
    case 'pyw':
    case 'pyi':
      return python()

    // Go
    case 'go':
      return go()

    // Rust
    case 'rs':
      return rust()

    // Java/Kotlin
    case 'java':
    case 'kt':
    case 'kts':
      return java()

    // C/C++
    case 'c':
    case 'h':
    case 'cpp':
    case 'hpp':
    case 'cc':
    case 'cxx':
      return cpp()

    // SQL
    case 'sql':
      return sql()

    default:
      return null
  }
}

export interface CodeEditorProps {
  value: string
  onChange?: (value: string) => void
  filename?: string
  readOnly?: boolean
  className?: string
  lineNumbers?: boolean
  darkMode?: boolean
  placeholder?: string
}

export function CodeEditor({
  value,
  onChange,
  filename,
  readOnly = false,
  className,
  darkMode = true,
  placeholder,
}: CodeEditorProps) {
  const containerRef = useRef<HTMLDivElement>(null)
  const viewRef = useRef<EditorView | null>(null)
  const onChangeRef = useRef(onChange)

  // Keep onChange ref updated
  onChangeRef.current = onChange

  // Build extensions
  const extensions = useMemo(() => {
    const exts: Extension[] = [
      // Basic setup
      history(),
      bracketMatching(),
      syntaxHighlighting(defaultHighlightStyle, { fallback: true }),
      highlightActiveLine(),
      highlightActiveLineGutter(),

      // Keymaps
      keymap.of([...defaultKeymap, ...historyKeymap, indentWithTab]),

      // Theme
      darkMode ? oneDark : [],

      // Editor styling
      EditorView.theme({
        '&': {
          height: '100%',
          fontSize: '14px',
        },
        '.cm-scroller': {
          fontFamily: 'ui-monospace, SFMono-Regular, "SF Mono", Menlo, Monaco, Consolas, monospace',
          overflow: 'auto',
        },
        '.cm-content': {
          padding: '12px 0',
        },
        '.cm-gutters': {
          backgroundColor: 'transparent',
          border: 'none',
        },
        '.cm-lineNumbers .cm-gutterElement': {
          padding: '0 12px 0 16px',
          minWidth: '40px',
        },
        '&.cm-focused': {
          outline: 'none',
        },
      }),

      // Update listener
      EditorView.updateListener.of((update) => {
        if (update.docChanged && onChangeRef.current) {
          onChangeRef.current(update.state.doc.toString())
        }
      }),
    ]

    // Line numbers
    exts.push(lineNumbers())

    // Read-only mode
    if (readOnly) {
      exts.push(EditorState.readOnly.of(true))
      exts.push(EditorView.editable.of(false))
    }

    // Language support
    const langExt = getLanguageExtension(filename)
    if (langExt) {
      exts.push(langExt)
    }

    // Placeholder
    if (placeholder) {
      exts.push(
        EditorView.theme({
          '.cm-placeholder': {
            color: 'var(--muted-foreground)',
            fontStyle: 'italic',
          },
        })
      )
      exts.push(
        EditorView.contentAttributes.of({
          'aria-placeholder': placeholder,
        })
      )
    }

    return exts
  }, [readOnly, filename, darkMode, placeholder])

  // Initialize editor
  useEffect(() => {
    if (!containerRef.current) return

    const state = EditorState.create({
      doc: value,
      extensions,
    })

    const view = new EditorView({
      state,
      parent: containerRef.current,
    })

    viewRef.current = view

    return () => {
      view.destroy()
      viewRef.current = null
    }
    // Only recreate when extensions change, not value
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [extensions])

  // Update content when value changes externally
  useEffect(() => {
    const view = viewRef.current
    if (!view) return

    const currentContent = view.state.doc.toString()
    if (currentContent !== value) {
      view.dispatch({
        changes: {
          from: 0,
          to: currentContent.length,
          insert: value,
        },
      })
    }
  }, [value])

  return (
    <div
      ref={containerRef}
      className={cn('h-full overflow-hidden rounded-md border bg-background', darkMode && 'bg-[#282c34]', className)}
    />
  )
}
