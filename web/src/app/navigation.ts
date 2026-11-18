import type { LucideIcon } from 'lucide-react'
import { FolderOpen, ListTodo, Radar, Settings, Terminal } from 'lucide-react'

export type NavSection = 'main' | 'server'

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
    id: 'fleet',
    name: 'Fleet',
    icon: Radar,
    path: '/fleet',
    section: 'main',
    shortcut: 'g f',
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

export const serverNavItems: NavItem[] = [
  {
    id: 'server-overview',
    name: 'Overview',
    icon: Radar,
    path: '/fleet/:serverId',
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
]

export const getMainNavItems = () => mainNavItems.filter((item) => item.id !== 'settings')
export const getSettingsNavItem = () => mainNavItems.find((item) => item.id === 'settings')!
export const getAllNavItems = () => [...mainNavItems, ...serverNavItems]
export const getNavItemById = (id: string) => getAllNavItems().find((item) => item.id === id)
