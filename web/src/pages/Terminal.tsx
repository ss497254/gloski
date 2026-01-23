import { useEffect, useRef, useCallback, useState } from 'react'
import { useSearchParams, Navigate } from 'react-router-dom'
import { Terminal } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import { WebLinksAddon } from '@xterm/addon-web-links'
import { useServer } from '@/hooks'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'
import { Plus, X, Terminal as TerminalIcon } from 'lucide-react'
import '@xterm/xterm/css/xterm.css'

interface TerminalTab {
  id: string
  title: string
  cwd?: string
}

interface TerminalInstance {
  terminal: Terminal
  fitAddon: FitAddon
  ws: WebSocket | null
}

export function TerminalPage() {
  const { server, api } = useServer()
  const [searchParams] = useSearchParams()
  const initialCwd = searchParams.get('cwd') || ''

  const [tabs, setTabs] = useState<TerminalTab[]>([
    { id: '1', title: 'Terminal 1', cwd: initialCwd },
  ])
  const [activeTab, setActiveTab] = useState('1')

  const terminalRefs = useRef<Map<string, HTMLDivElement>>(new Map())
  const instancesRef = useRef<Map<string, TerminalInstance>>(new Map())

  // Redirect if no server
  if (!server || !api) {
    return <Navigate to="/" replace />
  }

  const sendResize = useCallback(
    (ws: WebSocket | null, cols: number, rows: number) => {
      if (ws?.readyState === WebSocket.OPEN) {
        const data = new Uint8Array(5)
        data[0] = 1
        data[1] = (cols >> 8) & 0xff
        data[2] = cols & 0xff
        data[3] = (rows >> 8) & 0xff
        data[4] = rows & 0xff
        ws.send(data)
      }
    },
    []
  )

  const createTerminalInstance = useCallback(
    (tabId: string, container: HTMLDivElement, cwd?: string) => {
      if (!api) return

      const term = new Terminal({
        cursorBlink: true,
        fontSize: 14,
        fontFamily: 'Menlo, Monaco, "Courier New", monospace',
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

      // Connect WebSocket using the API helper
      const wsUrl = api.getTerminalUrl(cwd)
      const ws = new WebSocket(wsUrl)
      ws.binaryType = 'arraybuffer'

      ws.onopen = () => {
        sendResize(ws, term.cols, term.rows)
      }

      ws.onmessage = (event) => {
        if (event.data instanceof ArrayBuffer) {
          term.write(new TextDecoder().decode(event.data))
        } else {
          term.write(event.data)
        }
      }

      ws.onclose = () => {
        term.writeln('')
        term.writeln('\x1b[31mConnection closed\x1b[0m')
      }

      ws.onerror = () => {
        term.writeln('\x1b[31mConnection error\x1b[0m')
      }

      term.onData((data) => {
        if (ws.readyState === WebSocket.OPEN) {
          ws.send(new TextEncoder().encode(data))
        }
      })

      term.onResize(({ cols, rows }) => {
        sendResize(ws, cols, rows)
      })

      instancesRef.current.set(tabId, { terminal: term, fitAddon, ws })
    },
    [api, sendResize]
  )

  const destroyTerminalInstance = useCallback((tabId: string) => {
    const instance = instancesRef.current.get(tabId)
    if (instance) {
      instance.ws?.close()
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
        sendResize(instance.ws, instance.terminal.cols, instance.terminal.rows)
      }
    }

    window.addEventListener('resize', handleResize)
    return () => window.removeEventListener('resize', handleResize)
  }, [activeTab, sendResize])

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      instancesRef.current.forEach((_, tabId) => destroyTerminalInstance(tabId))
    }
  }, [destroyTerminalInstance])

  const addTab = () => {
    const newId = String(Date.now())
    setTabs((prev) => [...prev, { id: newId, title: `Terminal ${prev.length + 1}` }])
    setActiveTab(newId)
  }

  const closeTab = (tabId: string, e: React.MouseEvent) => {
    e.stopPropagation()

    if (tabs.length === 1) return // Don't close last tab

    destroyTerminalInstance(tabId)
    terminalRefs.current.delete(tabId)

    const newTabs = tabs.filter((t) => t.id !== tabId)
    setTabs(newTabs)

    if (activeTab === tabId) {
      setActiveTab(newTabs[newTabs.length - 1].id)
    }
  }

  return (
    <div className="h-full flex flex-col bg-[#0d1117]">
      {/* Tab bar */}
      <div className="flex items-center border-b border-[#30363d] bg-[#161b22]">
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
                  className="ml-1 p-0.5 rounded hover:bg-[#30363d] transition-colors"
                >
                  <X className="h-3 w-3" />
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
            className={cn(
              'absolute inset-0 p-1',
              activeTab === tab.id ? 'visible' : 'invisible'
            )}
          />
        ))}
      </div>
    </div>
  )
}
