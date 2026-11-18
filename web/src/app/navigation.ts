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

export type DeviceTab = 'overview' | 'files' | 'terminal' | 'jobs'

export function devicePath(deviceId: string, tab: DeviceTab = 'overview'): string {
  const base = `/devices/${deviceId}`
  return tab === 'overview' ? base : `${base}/${tab}`
}

export const settingsNavItem: NavItem = {
  id: 'settings',
  name: 'Settings',
  icon: Settings,
  path: '/settings',
}

export const deviceNavItems: Array<{ id: DeviceTab; name: string; icon: LucideIcon }> = [
  { id: 'overview', name: 'Overview', icon: Radar },
  { id: 'files', name: 'Files', icon: FolderOpen },
  { id: 'terminal', name: 'Terminal', icon: Terminal },
  { id: 'jobs', name: 'Jobs', icon: ListTodo },
]
