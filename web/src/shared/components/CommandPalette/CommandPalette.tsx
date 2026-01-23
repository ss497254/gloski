import { useState, useEffect, useMemo, useCallback } from 'react'
import { useNavigate } from 'react-router-dom'
import { Dialog, DialogContent } from '@/ui/dialog'
import { Input } from '@/ui/input'
import { ScrollArea } from '@/ui/scroll-area'
import {
  features,
  getMainFeatures,
  getWorkspaceFeatures,
} from '@/app/feature-registry'
import { useServersStore } from '@/features/servers'
import { cn } from '@/shared/lib/utils'
import { Search, ArrowRight, Plus, Moon, Sun, Server } from 'lucide-react'
import { useSettingsStore } from '@/features/settings'

interface CommandItem {
  id: string
  name: string
  icon: React.ComponentType<{ className?: string }>
  action: () => void
  category: string
  keywords?: string[]
}

export function CommandPalette() {
  const [open, setOpen] = useState(false)
  const [query, setQuery] = useState('')
  const [selectedIndex, setSelectedIndex] = useState(0)
  const navigate = useNavigate()
  const servers = useServersStore((s) => s.servers)
  const { theme, setTheme } = useSettingsStore()

  const commands = useMemo<CommandItem[]>(() => {
    const items: CommandItem[] = []

    // Navigation - Main features
    getMainFeatures().forEach((feature) => {
      items.push({
        id: `nav-${feature.id}`,
        name: `Go to ${feature.name}`,
        icon: feature.icon,
        action: () => navigate(feature.path),
        category: 'Navigation',
        keywords: [feature.name.toLowerCase()],
      })
    })

    // Navigation - Workspace features
    getWorkspaceFeatures().forEach((feature) => {
      items.push({
        id: `nav-${feature.id}`,
        name: `Go to ${feature.name}`,
        icon: feature.icon,
        action: () => navigate(feature.path),
        category: 'Navigation',
        keywords: [feature.name.toLowerCase()],
      })
    })

    // Settings navigation
    const settingsFeature = features.find((f) => f.id === 'settings')!
    items.push({
      id: 'nav-settings',
      name: 'Go to Settings',
      icon: settingsFeature.icon,
      action: () => navigate(settingsFeature.path),
      category: 'Navigation',
      keywords: ['settings', 'preferences', 'config'],
    })

    // Server commands
    servers.forEach((server) => {
      items.push({
        id: `server-${server.id}`,
        name: server.name,
        icon: Server,
        action: () => navigate(`/servers/${server.id}`),
        category: 'Servers',
        keywords: [server.name.toLowerCase(), server.url.toLowerCase()],
      })
    })

    // Actions
    items.push({
      id: 'action-add-server',
      name: 'Add new server',
      icon: Plus,
      action: () => navigate('/servers/add'),
      category: 'Actions',
      keywords: ['add', 'new', 'server', 'create'],
    })

    items.push({
      id: 'action-toggle-theme',
      name: `Switch to ${theme === 'dark' ? 'light' : 'dark'} mode`,
      icon: theme === 'dark' ? Sun : Moon,
      action: () => setTheme(theme === 'dark' ? 'light' : 'dark'),
      category: 'Actions',
      keywords: ['theme', 'dark', 'light', 'mode', 'toggle'],
    })

    return items
  }, [navigate, servers, theme, setTheme])

  const filtered = useMemo(() => {
    if (!query) return commands
    const q = query.toLowerCase()
    return commands.filter(
      (cmd) =>
        cmd.name.toLowerCase().includes(q) ||
        cmd.keywords?.some((k) => k.includes(q)) ||
        cmd.category.toLowerCase().includes(q)
    )
  }, [commands, query])

  // Keyboard shortcut to open
  useEffect(() => {
    const down = (e: KeyboardEvent) => {
      if (e.key === 'k' && (e.metaKey || e.ctrlKey)) {
        e.preventDefault()
        setOpen((o) => !o)
      }
    }
    document.addEventListener('keydown', down)
    return () => document.removeEventListener('keydown', down)
  }, [])

  // Reset on open
  useEffect(() => {
    if (open) {
      setQuery('')
      setSelectedIndex(0)
    }
  }, [open])

  // Reset selection when filtered results change
  useEffect(() => {
    setSelectedIndex(0)
  }, [filtered.length])

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      if (e.key === 'ArrowDown') {
        e.preventDefault()
        setSelectedIndex((i) => Math.min(i + 1, filtered.length - 1))
      } else if (e.key === 'ArrowUp') {
        e.preventDefault()
        setSelectedIndex((i) => Math.max(i - 1, 0))
      } else if (e.key === 'Enter') {
        e.preventDefault()
        const item = filtered[selectedIndex]
        if (item) {
          item.action()
          setOpen(false)
        }
      }
    },
    [filtered, selectedIndex]
  )

  // Group by category
  const grouped = useMemo(() => {
    const groups: Record<string, CommandItem[]> = {}
    filtered.forEach((item) => {
      if (!groups[item.category]) groups[item.category] = []
      groups[item.category].push(item)
    })
    return groups
  }, [filtered])

  // Calculate flat index for highlighting
  const getFlatIndex = (item: CommandItem) => {
    return filtered.indexOf(item)
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogContent className="p-0 gap-0 max-w-lg overflow-hidden">
        <div className="flex items-center border-b px-3">
          <Search className="h-4 w-4 text-muted-foreground shrink-0" />
          <Input
            placeholder="Type a command or search..."
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            onKeyDown={handleKeyDown}
            className="border-0 focus-visible:ring-0 h-12"
            autoFocus
          />
          <kbd className="hidden sm:inline-flex h-5 items-center gap-1 rounded border bg-muted px-1.5 font-mono text-[10px] font-medium text-muted-foreground">
            <span className="text-xs">⌘</span>K
          </kbd>
        </div>

        <ScrollArea className="max-h-80">
          {Object.entries(grouped).length > 0 ? (
            Object.entries(grouped).map(([category, items]) => (
              <div key={category} className="p-2">
                <div className="px-2 py-1.5 text-xs font-medium text-muted-foreground">
                  {category}
                </div>
                {items.map((item) => {
                  const flatIndex = getFlatIndex(item)
                  return (
                    <button
                      key={item.id}
                      onClick={() => {
                        item.action()
                        setOpen(false)
                      }}
                      onMouseEnter={() => setSelectedIndex(flatIndex)}
                      className={cn(
                        'w-full flex items-center gap-3 px-2 py-2 rounded-md text-sm transition-colors',
                        flatIndex === selectedIndex
                          ? 'bg-accent text-accent-foreground'
                          : 'hover:bg-accent/50'
                      )}
                    >
                      <item.icon className="h-4 w-4 text-muted-foreground shrink-0" />
                      <span className="flex-1 text-left truncate">{item.name}</span>
                      <ArrowRight className="h-3 w-3 text-muted-foreground shrink-0" />
                    </button>
                  )
                })}
              </div>
            ))
          ) : (
            <div className="p-8 text-center text-muted-foreground">
              <Search className="h-8 w-8 mx-auto mb-2 opacity-50" />
              <p className="text-sm">No results found</p>
            </div>
          )}
        </ScrollArea>

        <div className="border-t px-3 py-2 text-xs text-muted-foreground flex items-center gap-4">
          <span className="flex items-center gap-1">
            <kbd className="px-1 py-0.5 rounded bg-muted">↑↓</kbd>
            navigate
          </span>
          <span className="flex items-center gap-1">
            <kbd className="px-1 py-0.5 rounded bg-muted">↵</kbd>
            select
          </span>
          <span className="flex items-center gap-1">
            <kbd className="px-1 py-0.5 rounded bg-muted">esc</kbd>
            close
          </span>
        </div>
      </DialogContent>
    </Dialog>
  )
}
