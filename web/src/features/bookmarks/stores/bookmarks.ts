import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import { generateId } from '@/shared/lib/utils'

export interface Bookmark {
  id: string
  title: string
  url: string
  description?: string
  folder?: string
  tags: string[]
  favicon?: string
  createdAt: string
}

interface BookmarksState {
  bookmarks: Bookmark[]
  folders: string[]
  addBookmark: (bookmark: Omit<Bookmark, 'id' | 'createdAt'>) => void
  updateBookmark: (id: string, data: Partial<Bookmark>) => void
  deleteBookmark: (id: string) => void
  addFolder: (name: string) => void
  deleteFolder: (name: string) => void
}

// Mock data
const initialBookmarks: Bookmark[] = [
  {
    id: '1',
    title: 'GitHub',
    url: 'https://github.com',
    description: 'Where the world builds software',
    folder: 'Development',
    tags: ['code', 'git'],
    createdAt: '2024-01-15T10:30:00Z',
  },
  {
    id: '2',
    title: 'Stack Overflow',
    url: 'https://stackoverflow.com',
    description: 'Where developers learn & share',
    folder: 'Development',
    tags: ['code', 'help'],
    createdAt: '2024-01-14T09:00:00Z',
  },
  {
    id: '3',
    title: 'MDN Web Docs',
    url: 'https://developer.mozilla.org',
    description: 'Resources for developers, by developers',
    folder: 'Documentation',
    tags: ['docs', 'web'],
    createdAt: '2024-01-13T14:20:00Z',
  },
  {
    id: '4',
    title: 'Tailwind CSS',
    url: 'https://tailwindcss.com',
    description: 'A utility-first CSS framework',
    folder: 'Documentation',
    tags: ['css', 'styling'],
    createdAt: '2024-01-12T11:00:00Z',
  },
  {
    id: '5',
    title: 'React Documentation',
    url: 'https://react.dev',
    description: 'The library for web and native user interfaces',
    folder: 'Documentation',
    tags: ['react', 'docs'],
    createdAt: '2024-01-11T16:45:00Z',
  },
  {
    id: '6',
    title: 'Hacker News',
    url: 'https://news.ycombinator.com',
    description: 'Social news website focusing on computer science',
    folder: 'News',
    tags: ['news', 'tech'],
    createdAt: '2024-01-10T08:30:00Z',
  },
  {
    id: '7',
    title: 'TypeScript Handbook',
    url: 'https://www.typescriptlang.org/docs/handbook',
    description: 'The TypeScript Handbook',
    folder: 'Documentation',
    tags: ['typescript', 'docs'],
    createdAt: '2024-01-09T13:15:00Z',
  },
  {
    id: '8',
    title: 'Vercel',
    url: 'https://vercel.com',
    description: 'Develop. Preview. Ship.',
    folder: 'Tools',
    tags: ['hosting', 'deploy'],
    createdAt: '2024-01-08T10:00:00Z',
  },
  {
    id: '9',
    title: 'Linear',
    url: 'https://linear.app',
    description: 'The issue tracking tool you will enjoy using',
    folder: 'Tools',
    tags: ['project', 'management'],
    createdAt: '2024-01-07T15:30:00Z',
  },
  {
    id: '10',
    title: 'Figma',
    url: 'https://figma.com',
    description: 'The collaborative interface design tool',
    folder: 'Design',
    tags: ['design', 'ui'],
    createdAt: '2024-01-06T09:45:00Z',
  },
  {
    id: '11',
    title: 'Dribbble',
    url: 'https://dribbble.com',
    description: 'Discover the world\'s top designers & creatives',
    folder: 'Design',
    tags: ['design', 'inspiration'],
    createdAt: '2024-01-05T12:00:00Z',
  },
  {
    id: '12',
    title: 'Go Documentation',
    url: 'https://go.dev/doc',
    description: 'Official Go programming language documentation',
    folder: 'Documentation',
    tags: ['go', 'docs'],
    createdAt: '2024-01-04T14:30:00Z',
  },
]

const initialFolders = ['Development', 'Documentation', 'News', 'Tools', 'Design']

export const useBookmarksStore = create<BookmarksState>()(
  persist(
    (set) => ({
      bookmarks: initialBookmarks,
      folders: initialFolders,

      addBookmark: (bookmark) =>
        set((state) => ({
          bookmarks: [
            {
              ...bookmark,
              id: generateId(),
              createdAt: new Date().toISOString(),
            },
            ...state.bookmarks,
          ],
        })),

      updateBookmark: (id, data) =>
        set((state) => ({
          bookmarks: state.bookmarks.map((b) =>
            b.id === id ? { ...b, ...data } : b
          ),
        })),

      deleteBookmark: (id) =>
        set((state) => ({
          bookmarks: state.bookmarks.filter((b) => b.id !== id),
        })),

      addFolder: (name) =>
        set((state) => ({
          folders: state.folders.includes(name)
            ? state.folders
            : [...state.folders, name],
        })),

      deleteFolder: (name) =>
        set((state) => ({
          folders: state.folders.filter((f) => f !== name),
          bookmarks: state.bookmarks.map((b) =>
            b.folder === name ? { ...b, folder: undefined } : b
          ),
        })),
    }),
    {
      name: 'gloski-bookmarks',
    }
  )
)
