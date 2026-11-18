import type { LucideIcon } from 'lucide-react'
import { FolderOpen, LayoutDashboard, ListTodo, Radar, Settings, Terminal } from 'lucide-react'

export interface NavItem {
  id: string
  name: string
  icon: LucideIcon
  path: string
}

export const mainNavItems: NavItem[] = [
  {
    id: 'dashboard',
    name: 'Dashboard',
    icon: LayoutDashboard,
    path: '/',
  },
  {
    id: 'devices',
    name: 'Devices',
    icon: Radar,
    path: '/devices',
  },
]

export const settingsNavItem: NavItem = {
  id: 'settings',
  name: 'Settings',
  icon: Settings,
  path: '/settings',
}

export const deviceNavItems: NavItem[] = [
  {
    id: 'overview',
    name: 'Overview',
    icon: Radar,
    path: '/devices/:deviceId',
  },
  {
    id: 'files',
    name: 'Files',
    icon: FolderOpen,
    path: '/devices/:deviceId/files',
  },
  {
    id: 'terminal',
    name: 'Terminal',
    icon: Terminal,
    path: '/devices/:deviceId/terminal',
  },
  {
    id: 'jobs',
    name: 'Jobs',
    icon: ListTodo,
    path: '/devices/:deviceId/jobs',
  },
]
