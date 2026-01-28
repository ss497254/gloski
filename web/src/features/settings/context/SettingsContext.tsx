/* eslint-disable react-refresh/only-export-components */
import { useActivityStore } from '@/features/activity'
import { useBookmarksStore } from '@/features/bookmarks'
import { useMessagesStore } from '@/features/messages'
import { useNotesStore } from '@/features/notes'
import { useServersStore } from '@/features/servers'
import { useSnippetsStore } from '@/features/snippets'
import { createContext, useCallback, useContext, useMemo, useState, type ReactNode } from 'react'
import { toast } from 'sonner'
import { useSettingsStore } from '../stores/settings'

// ─────────────────────────────────────────────────────────────────────────────
// Types
// ─────────────────────────────────────────────────────────────────────────────

export type SettingsTab = 'appearance' | 'shortcuts' | 'data' | 'about'

interface DataCounts {
  servers: number
  bookmarks: number
  notes: number
  snippets: number
  messages: number
  activity: number
}

interface SettingsContextValue {
  // Tab navigation
  activeTab: SettingsTab
  setActiveTab: (tab: SettingsTab) => void

  // Settings values (from store)
  theme: 'light' | 'dark' | 'system'
  sidebarCollapsed: boolean

  // Settings actions
  setTheme: (theme: 'light' | 'dark' | 'system') => void
  toggleSidebar: () => void

  // Data actions
  handleExport: () => void
  handleClearAll: () => void

  // Data counts (for DataSettings display)
  dataCounts: DataCounts

  // Clear all dialog
  showClearDialog: boolean
  setShowClearDialog: (show: boolean) => void
}

const SettingsContext = createContext<SettingsContextValue | null>(null)

// ─────────────────────────────────────────────────────────────────────────────
// Provider
// ─────────────────────────────────────────────────────────────────────────────

interface SettingsProviderProps {
  children: ReactNode
}

export function SettingsProvider({ children }: SettingsProviderProps) {
  const [activeTab, setActiveTab] = useState<SettingsTab>('appearance')
  const [showClearDialog, setShowClearDialog] = useState(false)

  // Settings store
  const theme = useSettingsStore((s) => s.theme)
  const setTheme = useSettingsStore((s) => s.setTheme)
  const sidebarCollapsed = useSettingsStore((s) => s.sidebarCollapsed)
  const toggleSidebar = useSettingsStore((s) => s.toggleSidebar)

  // Data stores for counts
  const bookmarks = useBookmarksStore((s) => s.bookmarks)
  const messages = useMessagesStore((s) => s.messages)
  const notes = useNotesStore((s) => s.notes)
  const snippets = useSnippetsStore((s) => s.snippets)
  const activity = useActivityStore((s) => s.items)
  const servers = useServersStore((s) => s.servers)

  const dataCounts: DataCounts = useMemo(
    () => ({
      servers: servers.length,
      bookmarks: bookmarks.length,
      notes: notes.length,
      snippets: snippets.length,
      messages: messages.length,
      activity: activity.length,
    }),
    [servers.length, bookmarks.length, notes.length, snippets.length, messages.length, activity.length]
  )

  const handleExport = useCallback(() => {
    const data = {
      exportedAt: new Date().toISOString(),
      version: '1.0',
      bookmarks,
      messages,
      notes,
      snippets,
      activity,
      servers: servers.map((s) => ({ ...s, apiKey: null, token: null })), // Don't export secrets
    }

    const blob = new Blob([JSON.stringify(data, null, 2)], { type: 'application/json' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `gloski-export-${new Date().toISOString().split('T')[0]}.json`
    a.click()
    URL.revokeObjectURL(url)
    toast.success('Data exported successfully')
  }, [bookmarks, messages, notes, snippets, activity, servers])

  const handleClearAll = useCallback(() => {
    localStorage.clear()
    toast.success('All data cleared. Refreshing...')
    setTimeout(() => window.location.reload(), 1000)
  }, [])

  const value: SettingsContextValue = useMemo(
    () => ({
      activeTab,
      setActiveTab,
      theme,
      sidebarCollapsed,
      setTheme,
      toggleSidebar,
      handleExport,
      handleClearAll,
      dataCounts,
      showClearDialog,
      setShowClearDialog,
    }),
    [activeTab, theme, sidebarCollapsed, setTheme, toggleSidebar, handleExport, handleClearAll, dataCounts, showClearDialog]
  )

  return <SettingsContext.Provider value={value}>{children}</SettingsContext.Provider>
}

// ─────────────────────────────────────────────────────────────────────────────
// Hook
// ─────────────────────────────────────────────────────────────────────────────

export function useSettings(): SettingsContextValue {
  const context = useContext(SettingsContext)

  if (!context) {
    throw new Error('useSettings must be used within a SettingsProvider.')
  }

  return context
}
