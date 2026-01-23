import { create } from 'zustand'
import { generateId } from '@/shared/lib/utils'
import { persist } from 'zustand/middleware'

export type MessageType = 'info' | 'warning' | 'success' | 'error'

export interface Message {
  id: string
  title: string
  content: string
  read: boolean
  starred: boolean
  type: MessageType
  createdAt: string
}

interface MessagesState {
  messages: Message[]
  addMessage: (message: Omit<Message, 'id' | 'createdAt' | 'read' | 'starred'>) => void
  markAsRead: (id: string) => void
  markAllAsRead: () => void
  toggleStar: (id: string) => void
  deleteMessage: (id: string) => void
  clearAll: () => void
}

// Mock data
const initialMessages: Message[] = [
  {
    id: '1',
    title: 'Welcome to Gloski',
    content: 'Thank you for using Gloski Control Center. This is your central hub for managing servers, files, and more. Get started by adding your first server!',
    read: false,
    starred: true,
    type: 'info',
    createdAt: new Date().toISOString(),
  },
  {
    id: '2',
    title: 'Server "production-1" is online',
    content: 'Your server production-1 has come online and is ready for connections. All health checks passed successfully.',
    read: false,
    starred: false,
    type: 'success',
    createdAt: new Date(Date.now() - 1000 * 60 * 30).toISOString(), // 30 min ago
  },
  {
    id: '3',
    title: 'High CPU usage detected',
    content: 'Server "staging-2" is experiencing high CPU usage (95%). Consider scaling up or investigating running processes.',
    read: false,
    starred: false,
    type: 'warning',
    createdAt: new Date(Date.now() - 1000 * 60 * 60 * 2).toISOString(), // 2 hours ago
  },
  {
    id: '4',
    title: 'Backup completed successfully',
    content: 'Daily backup for all servers completed successfully. 3 servers backed up, total size: 2.4 GB.',
    read: true,
    starred: false,
    type: 'success',
    createdAt: new Date(Date.now() - 1000 * 60 * 60 * 5).toISOString(), // 5 hours ago
  },
  {
    id: '5',
    title: 'Connection failed to "dev-server"',
    content: 'Unable to establish connection to dev-server. Please check the server status and network configuration.',
    read: true,
    starred: false,
    type: 'error',
    createdAt: new Date(Date.now() - 1000 * 60 * 60 * 24).toISOString(), // 1 day ago
  },
  {
    id: '6',
    title: 'New version available',
    content: 'Gloski v2.1.0 is now available with new features including improved terminal performance and better file search.',
    read: true,
    starred: true,
    type: 'info',
    createdAt: new Date(Date.now() - 1000 * 60 * 60 * 24 * 2).toISOString(), // 2 days ago
  },
  {
    id: '7',
    title: 'Disk space warning',
    content: 'Server "production-2" has less than 10% disk space remaining. Consider cleaning up old files or expanding storage.',
    read: true,
    starred: false,
    type: 'warning',
    createdAt: new Date(Date.now() - 1000 * 60 * 60 * 24 * 3).toISOString(), // 3 days ago
  },
  {
    id: '8',
    title: 'Security scan completed',
    content: 'Weekly security scan completed. No vulnerabilities found across all managed servers.',
    read: true,
    starred: false,
    type: 'success',
    createdAt: new Date(Date.now() - 1000 * 60 * 60 * 24 * 5).toISOString(), // 5 days ago
  },
]


export const useMessagesStore = create<MessagesState>()(
  persist(
    (set) => ({
      messages: initialMessages,

      addMessage: (message) =>
        set((state) => ({
          messages: [
            {
              ...message,
              id: generateId(),
              read: false,
              starred: false,
              createdAt: new Date().toISOString(),
            },
            ...state.messages,
          ],
        })),

      markAsRead: (id) =>
        set((state) => ({
          messages: state.messages.map((m) =>
            m.id === id ? { ...m, read: true } : m
          ),
        })),

      markAllAsRead: () =>
        set((state) => ({
          messages: state.messages.map((m) => ({ ...m, read: true })),
        })),

      toggleStar: (id) =>
        set((state) => ({
          messages: state.messages.map((m) =>
            m.id === id ? { ...m, starred: !m.starred } : m
          ),
        })),

      deleteMessage: (id) =>
        set((state) => ({
          messages: state.messages.filter((m) => m.id !== id),
        })),

      clearAll: () => set({ messages: [] }),
    }),
    {
      name: 'gloski-messages',
    }
  )
)
