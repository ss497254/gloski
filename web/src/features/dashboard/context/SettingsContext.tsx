/* eslint-disable react-refresh/only-export-components */
import { createContext, useCallback, useContext, useMemo, useState, type ReactNode } from 'react'
import { toast } from 'sonner'
import { useSettingsStore } from '../stores/settings'

// ─────────────────────────────────────────────────────────────────────────────
// Types
// ─────────────────────────────────────────────────────────────────────────────

export type SettingsTab = 'appearance' | 'shortcuts' | 'data' | 'about'

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
  handleClearAll: () => void

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
      handleClearAll,
      showClearDialog,
      setShowClearDialog,
    }),
    [activeTab, theme, sidebarCollapsed, setTheme, toggleSidebar, handleClearAll, showClearDialog]
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
