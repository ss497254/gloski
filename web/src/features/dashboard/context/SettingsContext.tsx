/* eslint-disable react-refresh/only-export-components */
import { createContext, useContext, useMemo, useState, type ReactNode } from 'react'
import { useSettingsStore } from '../stores/settings'

export type SettingsTab = 'appearance' | 'about'

interface SettingsContextValue {
  activeTab: SettingsTab
  setActiveTab: (tab: SettingsTab) => void
  theme: 'light' | 'dark' | 'system'
  sidebarCollapsed: boolean
  setTheme: (theme: 'light' | 'dark' | 'system') => void
  toggleSidebar: () => void
}

const SettingsContext = createContext<SettingsContextValue | null>(null)

export function SettingsProvider({ children }: { children: ReactNode }) {
  const [activeTab, setActiveTab] = useState<SettingsTab>('appearance')
  const theme = useSettingsStore((state) => state.theme)
  const setTheme = useSettingsStore((state) => state.setTheme)
  const sidebarCollapsed = useSettingsStore((state) => state.sidebarCollapsed)
  const toggleSidebar = useSettingsStore((state) => state.toggleSidebar)

  const value = useMemo<SettingsContextValue>(
    () => ({ activeTab, setActiveTab, theme, sidebarCollapsed, setTheme, toggleSidebar }),
    [activeTab, theme, sidebarCollapsed, setTheme, toggleSidebar]
  )

  return <SettingsContext.Provider value={value}>{children}</SettingsContext.Provider>
}

export function useSettings(): SettingsContextValue {
  const context = useContext(SettingsContext)
  if (!context) throw new Error('useSettings must be used within a SettingsProvider')
  return context
}
