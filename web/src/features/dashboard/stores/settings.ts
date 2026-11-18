import { create } from 'zustand'
import { persist } from 'zustand/middleware'

type Theme = 'light' | 'dark' | 'system'

interface SettingsState {
  theme: Theme
  setTheme: (theme: Theme) => void
  toggleTheme: () => void
  sidebarCollapsed: boolean
  toggleSidebar: () => void
  mobileSidebarOpen: boolean
  setMobileSidebarOpen: (open: boolean) => void
  toggleMobileSidebar: () => void
}

export const useSettingsStore = create<SettingsState>()(
  persist(
    (set, get) => ({
      theme: 'dark',
      setTheme: (theme) => {
        set({ theme })
        applyTheme(theme)
      },
      toggleTheme: () => {
        const next = isDark(get().theme) ? 'light' : 'dark'
        set({ theme: next })
        applyTheme(next)
      },
      sidebarCollapsed: false,
      toggleSidebar: () => set((state) => ({ sidebarCollapsed: !state.sidebarCollapsed })),
      mobileSidebarOpen: false,
      setMobileSidebarOpen: (open) => set({ mobileSidebarOpen: open }),
      toggleMobileSidebar: () => set((state) => ({ mobileSidebarOpen: !state.mobileSidebarOpen })),
    }),
    {
      name: 'gloski-settings',
    }
  )
)

function isDark(theme: Theme): boolean {
  if (theme === 'system') {
    return window.matchMedia('(prefers-color-scheme: dark)').matches
  }
  return theme === 'dark'
}

function applyTheme(theme: Theme) {
  document.documentElement.classList.toggle('dark', isDark(theme))
}

// Initialize theme on load
if (typeof window !== 'undefined') {
  const stored = localStorage.getItem('gloski-settings')
  try {
    const state = stored ? JSON.parse(stored)?.state : null
    applyTheme(state?.theme || 'dark')
  } catch {
    applyTheme('dark')
  }

  const media = window.matchMedia('(prefers-color-scheme: dark)')
  media.addEventListener('change', () => {
    const current = useSettingsStore.getState().theme
    if (current === 'system') applyTheme('system')
  })
}
