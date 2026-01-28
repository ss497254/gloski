import { useServer } from '@/features/servers'
import { cn } from '@/shared/lib/utils'
import { Button } from '@/ui/button'
import type { TerminalConnection } from '@gloski/sdk'
import { FitAddon } from '@xterm/addon-fit'
import { WebLinksAddon } from '@xterm/addon-web-links'
import { Terminal, type ITerminalInitOnlyOptions, type ITerminalOptions } from '@xterm/xterm'
import '@xterm/xterm/css/xterm.css'
import { Plus, Terminal as TerminalIcon, X } from 'lucide-react'
import { useCallback, useEffect, useRef, useState } from 'react'
import { useSearchParams } from 'react-router-dom'

interface TerminalTab {
  id: string
  title: string
  cwd?: string
}

interface TerminalInstance {
  terminal: Terminal
  fitAddon: FitAddon
  connection: TerminalConnection | null
}

function getTerminalTheme(isDarkMode: boolean) {
  if (isDarkMode) {
    // Dark theme - optimized for readability
    return {
      background: '#0a0a0a',
      foreground: '#e0e0e0',
      cursor: '#3b82f6',
      cursorAccent: '#1a1a1a',
      selectionBackground: '#3b82f6' + '40', // 25% opacity
      black: '#4b5563',
      red: '#ff6b6b',
      green: '#51cf66',
      yellow: '#ffd43b',
      blue: '#4dabf7',
      magenta: '#da77f2',
      cyan: '#22d3ee',
      white: '#d0d0d0',
      brightBlack: '#808080',
      brightRed: '#ff8787',
      brightGreen: '#69db7c',
      brightYellow: '#ffec99',
      brightBlue: '#74c0fc',
      brightMagenta: '#e599f7',
      brightCyan: '#4dd0e1',
      brightWhite: '#f8f8f8',
    }
  } else {
    // Light theme - optimized for readability on light background
    return {
      background: '#ffffff',
      foreground: '#1f2937',
      cursor: '#2563eb',
      cursorAccent: '#ffffff',
      selectionBackground: '#2563eb' + '30', // 20% opacity
      black: '#1f2937',
      red: '#dc2626',
      green: '#16a34a',
      yellow: '#d97706',
      blue: '#2563eb',
      magenta: '#9333ea',
      cyan: '#0891b2',
      white: '#e5e7eb',
      brightBlack: '#6b7280',
      brightRed: '#ef4444',
      brightGreen: '#22c55e',
      brightYellow: '#eab308',
      brightBlue: '#3b82f6',
      brightMagenta: '#a855f7',
      brightCyan: '#06b6d4',
      brightWhite: '#f3f4f6',
    }
  }
}

function getTerminalConfig(): ITerminalOptions & ITerminalInitOnlyOptions {
  return {
    cursorBlink: true,
    fontSize: 14,
    fontWeight: '500',
    fontFamily: 'NerdFontsSymbols Nerd Font, Menlo, Monaco, "Courier New", monospace',
    theme: getTerminalTheme(document.documentElement.classList.contains('dark')),
    allowProposedApi: true,
  }
}

export function TerminalPage() {
  const { server } = useServer()
  const [searchParams] = useSearchParams()
  const initialCwd = searchParams.get('cwd') || ''

  const [tabs, setTabs] = useState<TerminalTab[]>([{ id: '1', title: 'Terminal 1', cwd: initialCwd }])
  const [activeTab, setActiveTab] = useState('1')

  const terminalRefs = useRef<Map<string, HTMLDivElement>>(new Map())
  const instancesRef = useRef<Map<string, TerminalInstance>>(new Map())

  const createTerminalInstance = useCallback(
    (tabId: string, container: HTMLDivElement, cwd?: string) => {
      const term = new Terminal(getTerminalConfig())

      const fitAddon = new FitAddon()
      const webLinksAddon = new WebLinksAddon()

      term.loadAddon(fitAddon)
      term.loadAddon(webLinksAddon)
      term.open(container)
      fitAddon.fit()

      // Connect using SDK's TerminalConnection with auto-reconnect
      const connection = server.getClient().terminal.connect({
        cwd,
        autoReconnect: true,
        maxReconnectAttempts: 5,
        reconnectDelay: 1000,
      })

      connection.on('open', () => {
        connection.resize(term.cols, term.rows)
      })

      connection.on('data', (data: string) => {
        term.write(data)
      })

      connection.on('close', () => {
        term.writeln('')
        term.writeln('\x1b[31mConnection closed\x1b[0m')
      })

      connection.on('error', () => {
        term.writeln('\x1b[31mConnection error\x1b[0m')
      })

      connection.on('reconnecting', (attempt: number) => {
        term.writeln(`\x1b[33mReconnecting (attempt ${attempt})...\x1b[0m`)
      })

      connection.on('reconnected', () => {
        term.writeln('\x1b[32mReconnected!\x1b[0m')
      })

      term.onData((data) => {
        connection.write(data)
      })

      term.onResize(({ cols, rows }) => {
        connection.resize(cols, rows)
      })

      instancesRef.current.set(tabId, { terminal: term, fitAddon, connection })
    },
    [server]
  )

  const destroyTerminalInstance = useCallback((tabId: string) => {
    const instance = instancesRef.current.get(tabId)
    if (instance) {
      instance.connection?.close()
      instance.terminal.dispose()
      instancesRef.current.delete(tabId)
    }
  }, [])

  // Handle tab changes and terminal creation
  useEffect(() => {
    const container = terminalRefs.current.get(activeTab)
    if (!container) return

    // Check if instance already exists
    if (instancesRef.current.has(activeTab)) {
      // Just fit the existing terminal
      const instance = instancesRef.current.get(activeTab)
      instance?.fitAddon.fit()
      instance?.terminal.focus()
      return
    }

    // Create new terminal instance
    const tab = tabs.find((t) => t.id === activeTab)
    createTerminalInstance(activeTab, container, tab?.cwd)

    return () => {
      // Don't destroy on unmount if just switching tabs
    }
  }, [activeTab, tabs, createTerminalInstance])

  // Handle window resize
  useEffect(() => {
    const handleResize = () => {
      const instance = instancesRef.current.get(activeTab)
      if (instance) {
        instance.fitAddon.fit()
        instance.connection?.resize(instance.terminal.cols, instance.terminal.rows)
      }
    }

    window.addEventListener('resize', handleResize)
    return () => window.removeEventListener('resize', handleResize)
  }, [activeTab])

  // Cleanup on unmount
  useEffect(() => {
    const instances = instancesRef.current
    return () => {
      instances.forEach((_, tabId) => destroyTerminalInstance(tabId))
    }
  }, [destroyTerminalInstance])

  // Handle theme changes - CSS handles the visual theme changes
  useEffect(() => {
    const handleThemeChange = () => {
      // Theme changes are handled by CSS variables
      // The terminal styling will automatically update via CSS transitions
    }

    // Listen for theme changes on the document element
    const observer = new MutationObserver((mutations) => {
      mutations.forEach((mutation) => {
        if (mutation.attributeName === 'class') {
          handleThemeChange()
        }
      })
    })

    observer.observe(document.documentElement, {
      attributes: true,
      attributeFilter: ['class'],
    })

    return () => observer.disconnect()
  }, [])

  const addTab = useCallback(() => {
    const newId = String(Date.now())
    setTabs((prev) => [...prev, { id: newId, title: `Terminal ${prev.length + 1}` }])
    setActiveTab(newId)
  }, [])

  const closeTab = useCallback(
    (tabId: string, e: React.MouseEvent) => {
      e.stopPropagation()

      setTabs((prevTabs) => {
        if (prevTabs.length === 1) return prevTabs // Don't close last tab

        destroyTerminalInstance(tabId)
        terminalRefs.current.delete(tabId)

        const newTabs = prevTabs.filter((t) => t.id !== tabId)

        setActiveTab((prevActive) => {
          if (prevActive === tabId) {
            return newTabs[newTabs.length - 1].id
          }
          return prevActive
        })

        return newTabs
      })
    },
    [destroyTerminalInstance]
  )

  return (
    <div className="h-full flex flex-col bg-background">
      {/* Tab bar */}
      <div className="terminal-tab-bar flex items-center bg-muted border-b">
        <style href="https://www.nerdfonts.com/assets/css/webfont.css" rel="stylesheet" />
        <div className="flex-1 flex items-center h-full overflow-x-auto">
          {tabs.map((tab) => (
            <button
              key={tab.id}
              onClick={() => setActiveTab(tab.id)}
              className={cn(
                'flex items-center gap-2 px-4 h-full text-sm border-r border-border transition-colors',
                activeTab === tab.id
                  ? 'bg-background text-foreground font-medium'
                  : 'text-muted-foreground hover:text-foreground hover:bg-accent/30'
              )}
            >
              <TerminalIcon className="h-4 w-4" />
              <span className="truncate max-w-32">{tab.title}</span>
              {tabs.length > 1 && (
                <button
                  onClick={(e) => closeTab(tab.id, e)}
                  className="ml-1 p-1 -m-0.5 rounded transition-colors hover:bg-muted-foreground/20 hover:text-foreground"
                  aria-label={`Close ${tab.title}`}
                >
                  <X className="h-3.5 w-3.5" />
                </button>
              )}
            </button>
          ))}
        </div>
        <Button
          variant="ghost"
          size="sm"
          onClick={addTab}
          className="m-1 transition-colors"
        >
          <Plus className="h-4 w-4" />
        </Button>
      </div>

      {/* Terminal containers */}
      <div className="terminal-viewport flex-1 relative overflow-hidden">
        {tabs.map((tab) => (
          <div
            key={tab.id}
            ref={(el) => {
              if (el) terminalRefs.current.set(tab.id, el)
            }}
            className={cn('absolute inset-0 p-1', activeTab === tab.id ? 'visible' : 'invisible')}
          />
        ))}
      </div>
    </div>
  )
}
