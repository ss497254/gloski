import { useServer } from '@/features/servers'
import { cn } from '@/shared/lib/utils'
import { Button } from '@/ui/button'
import type { TerminalConnection } from '@gloski/sdk'
import { FitAddon } from '@xterm/addon-fit'
import { WebLinksAddon } from '@xterm/addon-web-links'
import { Terminal } from '@xterm/xterm'
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
      const term = new Terminal({
        cursorBlink: true,
        fontSize: 14,
        fontFamily: 'NerdFontsSymbols Nerd Font, Menlo, Monaco, "Courier New", monospace',
        theme: {
          background: '#0d1117',
          foreground: '#c9d1d9',
          cursor: '#58a6ff',
          cursorAccent: '#0d1117',
          selectionBackground: '#264f78',
          black: '#484f58',
          red: '#ff7b72',
          green: '#3fb950',
          yellow: '#d29922',
          blue: '#58a6ff',
          magenta: '#bc8cff',
          cyan: '#39c5cf',
          white: '#b1bac4',
          brightBlack: '#6e7681',
          brightRed: '#ffa198',
          brightGreen: '#56d364',
          brightYellow: '#e3b341',
          brightBlue: '#79c0ff',
          brightMagenta: '#d2a8ff',
          brightCyan: '#56d4dd',
          brightWhite: '#f0f6fc',
        },
      })

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
    <div className="h-full flex flex-col bg-[#0d1117]">
      {/* Tab bar */}
      <div className="flex items-center border-b border-[#30363d] bg-[#161b22]">
        <style href="https://www.nerdfonts.com/assets/css/webfont.css" />
        <div className="flex-1 flex items-center overflow-x-auto">
          {tabs.map((tab) => (
            <button
              key={tab.id}
              onClick={() => setActiveTab(tab.id)}
              className={cn(
                'flex items-center gap-2 px-4 py-2 text-sm border-r border-[#30363d] transition-colors',
                activeTab === tab.id
                  ? 'bg-[#0d1117] text-white'
                  : 'text-[#8b949e] hover:text-white hover:bg-[#0d1117]/50'
              )}
            >
              <TerminalIcon className="h-4 w-4" />
              <span>{tab.title}</span>
              {tabs.length > 1 && (
                <button
                  onClick={(e) => closeTab(tab.id, e)}
                  className="ml-1 p-1.5 -m-0.5 rounded hover:bg-[#30363d] transition-colors"
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
          className="m-1 text-[#8b949e] hover:text-white hover:bg-[#30363d]"
        >
          <Plus className="h-4 w-4" />
        </Button>
      </div>

      {/* Terminal containers */}
      <div className="flex-1 relative">
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
