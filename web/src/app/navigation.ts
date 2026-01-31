import type { LucideIcon } from 'lucide-react'
import {
  Download,
  FolderOpen,
  LayoutDashboard,
  ListTodo,
  Search,
  Server,
  Settings,
  Terminal,
} from 'lucide-react'

export type NavSection = 'main' | 'server'

/**
 * Navigation item metadata for sidebar, command palette, and UI.
 * Actual route definitions are in App.tsx to handle complex nesting.
 */
export interface NavItem {
  id: string
  name: string
  icon: LucideIcon
  path: string
  section: NavSection
  serverScoped?: boolean
  shortcut?: string
}

export const mainNavItems: NavItem[] = [
  {
    id: 'dashboard',
    name: 'Dashboard',
    icon: LayoutDashboard,
    path: '/',
    section: 'main',
    shortcut: 'g d',
  },
  {
    id: 'servers',
    name: 'Servers',
    icon: Server,
    path: '/servers',
    section: 'main',
    shortcut: 'g s',
  },
  {
    id: 'settings',
    name: 'Settings',
    icon: Settings,
    path: '/settings',
    section: 'main',
    shortcut: 'g ,',
  },
]

// Server-scoped navigation items (shown when a server is selected)
export const serverNavItems: NavItem[] = [
  {
    id: 'server-overview',
    name: 'Overview',
    icon: LayoutDashboard,
    path: '/servers/:serverId',
    section: 'server',
    serverScoped: true,
  },
  {
    id: 'files',
    name: 'Files',
    icon: FolderOpen,
    path: '/servers/:serverId/files',
    section: 'server',
    serverScoped: true,
  },
  {
    id: 'search',
    name: 'Search',
    icon: Search,
    path: '/servers/:serverId/search',
    section: 'server',
    serverScoped: true,
  },
  {
    id: 'terminal',
    name: 'Terminal',
    icon: Terminal,
    path: '/servers/:serverId/terminal',
    section: 'server',
    serverScoped: true,
  },
  {
    id: 'jobs',
    name: 'Jobs',
    icon: ListTodo,
    path: '/servers/:serverId/jobs',
    section: 'server',
    serverScoped: true,
  },
  {
    id: 'downloads',
    name: 'Downloads',
    icon: Download,
    path: '/servers/:serverId/downloads',
    section: 'server',
    serverScoped: true,
  },
]

// Helper functions
export const getNavItemsBySection = (section: NavSection) => mainNavItems.filter((item) => item.section === section)

export const getMainNavItems = () => mainNavItems.filter((item) => item.id !== 'settings')

export const getSettingsNavItem = () => mainNavItems.find((item) => item.id === 'settings')!

export const getAllNavItems = () => [...mainNavItems, ...serverNavItems]

export const getNavItemById = (id: string) => getAllNavItems().find((item) => item.id === id)
