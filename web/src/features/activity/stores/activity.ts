import { create } from 'zustand'
import { generateId } from '@/shared/lib/utils'
import { persist } from 'zustand/middleware'

export type ActivityAction =
  | 'server_added'
  | 'server_removed'
  | 'server_connected'
  | 'server_disconnected'
  | 'file_created'
  | 'file_deleted'
  | 'file_modified'
  | 'file_uploaded'
  | 'file_downloaded'
  | 'terminal_opened'
  | 'task_started'
  | 'task_completed'
  | 'note_created'
  | 'note_updated'
  | 'bookmark_added'
  | 'snippet_created'
  | 'settings_changed'

export interface ActivityItem {
  id: string
  action: ActivityAction
  target: string
  description: string
  serverId?: string
  serverName?: string
  timestamp: string
  metadata?: Record<string, unknown>
}

interface ActivityState {
  items: ActivityItem[]
  addActivity: (activity: Omit<ActivityItem, 'id' | 'timestamp'>) => void
  clearAll: () => void
  clearOlderThan: (days: number) => void
}

// Generate mock activity data spanning the last 7 days
function generateMockActivity(): ActivityItem[] {
  const actions: Array<{
    action: ActivityAction
    target: string
    description: string
    serverName?: string
  }> = [
    {
      action: 'server_added',
      target: 'production-1',
      description: 'Added new server "production-1"',
      serverName: 'production-1',
    },
    {
      action: 'server_connected',
      target: 'production-1',
      description: 'Connected to server "production-1"',
      serverName: 'production-1',
    },
    {
      action: 'file_uploaded',
      target: '/var/www/app/config.json',
      description: 'Uploaded config.json',
      serverName: 'production-1',
    },
    {
      action: 'file_modified',
      target: '/etc/nginx/nginx.conf',
      description: 'Modified nginx.conf',
      serverName: 'production-1',
    },
    {
      action: 'terminal_opened',
      target: 'production-1',
      description: 'Opened terminal session',
      serverName: 'production-1',
    },
    {
      action: 'task_started',
      target: 'npm run build',
      description: 'Started task: npm run build',
      serverName: 'production-1',
    },
    {
      action: 'task_completed',
      target: 'npm run build',
      description: 'Task completed: npm run build',
      serverName: 'production-1',
    },
    {
      action: 'server_added',
      target: 'staging-2',
      description: 'Added new server "staging-2"',
      serverName: 'staging-2',
    },
    {
      action: 'server_connected',
      target: 'staging-2',
      description: 'Connected to server "staging-2"',
      serverName: 'staging-2',
    },
    {
      action: 'file_created',
      target: '/home/user/scripts/deploy.sh',
      description: 'Created deploy.sh',
      serverName: 'staging-2',
    },
    {
      action: 'file_downloaded',
      target: '/var/log/app.log',
      description: 'Downloaded app.log',
      serverName: 'staging-2',
    },
    {
      action: 'note_created',
      target: 'Server Setup Guide',
      description: 'Created note: Server Setup Guide',
    },
    {
      action: 'note_updated',
      target: 'API Endpoints Reference',
      description: 'Updated note: API Endpoints Reference',
    },
    { action: 'bookmark_added', target: 'GitHub', description: 'Added bookmark: GitHub' },
    {
      action: 'snippet_created',
      target: 'React useDebounce Hook',
      description: 'Created snippet: React useDebounce Hook',
    },
    { action: 'settings_changed', target: 'theme', description: 'Changed theme to dark mode' },
    {
      action: 'server_disconnected',
      target: 'dev-server',
      description: 'Disconnected from "dev-server"',
      serverName: 'dev-server',
    },
    {
      action: 'file_deleted',
      target: '/tmp/old-backup.tar.gz',
      description: 'Deleted old-backup.tar.gz',
      serverName: 'production-1',
    },
    {
      action: 'task_started',
      target: 'docker-compose up',
      description: 'Started task: docker-compose up',
      serverName: 'staging-2',
    },
    {
      action: 'terminal_opened',
      target: 'staging-2',
      description: 'Opened terminal session',
      serverName: 'staging-2',
    },
    {
      action: 'file_modified',
      target: '/app/src/index.ts',
      description: 'Modified index.ts',
      serverName: 'staging-2',
    },
    {
      action: 'bookmark_added',
      target: 'Stack Overflow',
      description: 'Added bookmark: Stack Overflow',
    },
    {
      action: 'note_created',
      target: 'Meeting Notes - Jan 20',
      description: 'Created note: Meeting Notes - Jan 20',
    },
    {
      action: 'snippet_created',
      target: 'Python FastAPI Endpoint',
      description: 'Created snippet: Python FastAPI Endpoint',
    },
    {
      action: 'file_uploaded',
      target: '/var/www/assets/logo.png',
      description: 'Uploaded logo.png',
      serverName: 'production-1',
    },
  ]

  const now = Date.now()
  const dayMs = 24 * 60 * 60 * 1000

  return actions
    .map((item, index) => ({
      id: String(index + 1),
      ...item,
      timestamp: new Date(now - Math.random() * 7 * dayMs).toISOString(),
    }))
    .sort((a, b) => new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime())
}

export const useActivityStore = create<ActivityState>()(
  persist(
    (set) => ({
      items: generateMockActivity(),

      addActivity: (activity) =>
        set((state) => ({
          items: [
            {
              ...activity,
              id: generateId(),
              timestamp: new Date().toISOString(),
            },
            ...state.items,
          ].slice(0, 100), // Keep only last 100 items
        })),

      clearAll: () => set({ items: [] }),

      clearOlderThan: (days) =>
        set((state) => {
          const cutoff = Date.now() - days * 24 * 60 * 60 * 1000
          return {
            items: state.items.filter((item) => new Date(item.timestamp).getTime() > cutoff),
          }
        }),
    }),
    {
      name: 'gloski-activity',
    }
  )
)
