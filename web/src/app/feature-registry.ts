import { type ComponentType, type LazyExoticComponent, lazy } from 'react'
import type { LucideIcon } from 'lucide-react'
import {
  Activity,
  Bookmark,
  Code2,
  Download,
  FileText,
  FolderOpen,
  LayoutDashboard,
  ListTodo,
  MessageSquare,
  Search,
  Server,
  Settings,
  Terminal,
} from 'lucide-react'

export type FeatureSection = 'main' | 'workspace' | 'server'

export interface FeatureDefinition {
  id: string
  name: string
  icon: LucideIcon
  path: string
  component: LazyExoticComponent<ComponentType>
  section: FeatureSection
  serverScoped?: boolean
  shortcut?: string
}

export const features: FeatureDefinition[] = [
  // Main section
  {
    id: 'dashboard',
    name: 'Dashboard',
    icon: LayoutDashboard,
    path: '/',
    component: lazy(() => import('@/features/dashboard')),
    section: 'main',
    shortcut: 'g d',
  },
  {
    id: 'servers',
    name: 'Servers',
    icon: Server,
    path: '/servers',
    component: lazy(() => import('@/features/servers')),
    section: 'main',
    shortcut: 'g s',
  },

  // Workspace section
  {
    id: 'bookmarks',
    name: 'Bookmarks',
    icon: Bookmark,
    path: '/bookmarks',
    component: lazy(() => import('@/features/bookmarks')),
    section: 'workspace',
    shortcut: 'g b',
  },
  {
    id: 'notes',
    name: 'Notes',
    icon: FileText,
    path: '/notes',
    component: lazy(() => import('@/features/notes')),
    section: 'workspace',
    shortcut: 'g n',
  },
  {
    id: 'messages',
    name: 'Messages',
    icon: MessageSquare,
    path: '/messages',
    component: lazy(() => import('@/features/messages')),
    section: 'workspace',
    shortcut: 'g m',
  },
  {
    id: 'snippets',
    name: 'Snippets',
    icon: Code2,
    path: '/snippets',
    component: lazy(() => import('@/features/snippets')),
    section: 'workspace',
    shortcut: 'g c',
  },
  {
    id: 'activity',
    name: 'Activity',
    icon: Activity,
    path: '/activity',
    component: lazy(() => import('@/features/activity')),
    section: 'workspace',
    shortcut: 'g a',
  },

  // Settings
  {
    id: 'settings',
    name: 'Settings',
    icon: Settings,
    path: '/settings',
    component: lazy(() => import('@/features/settings')),
    section: 'main',
    shortcut: 'g ,',
  },
]

// Server-scoped features (shown when a server is selected)
export const serverFeatures: FeatureDefinition[] = [
  {
    id: 'server-overview',
    name: 'Overview',
    icon: LayoutDashboard,
    path: '/servers/:serverId',
    component: lazy(() =>
      import('@/features/servers').then((m) => ({
        default: m.ServerDetailPage,
      }))
    ),
    section: 'server',
    serverScoped: true,
  },
  {
    id: 'files',
    name: 'Files',
    icon: FolderOpen,
    path: '/servers/:serverId/files',
    component: lazy(() => import('@/features/files')),
    section: 'server',
    serverScoped: true,
  },
  {
    id: 'search',
    name: 'Search',
    icon: Search,
    path: '/servers/:serverId/search',
    component: lazy(() => import('@/features/search')),
    section: 'server',
    serverScoped: true,
  },
  {
    id: 'terminal',
    name: 'Terminal',
    icon: Terminal,
    path: '/servers/:serverId/terminal',
    component: lazy(() => import('@/features/terminal')),
    section: 'server',
    serverScoped: true,
  },
  {
    id: 'jobs',
    name: 'Jobs',
    icon: ListTodo,
    path: '/servers/:serverId/jobs',
    component: lazy(() => import('@/features/jobs')),
    section: 'server',
    serverScoped: true,
  },
  {
    id: 'downloads',
    name: 'Downloads',
    icon: Download,
    path: '/servers/:serverId/downloads',
    component: lazy(() => import('@/features/downloads')),
    section: 'server',
    serverScoped: true,
  },
]

// Helper functions
export const getFeaturesBySection = (section: FeatureSection) => features.filter((f) => f.section === section)

export const getMainFeatures = () => features.filter((f) => f.section === 'main' && f.id !== 'settings')

export const getWorkspaceFeatures = () => features.filter((f) => f.section === 'workspace')

export const getSettingsFeature = () => features.find((f) => f.id === 'settings')!

export const getAllFeatures = () => [...features, ...serverFeatures]

export const getFeatureById = (id: string) => getAllFeatures().find((f) => f.id === id)
