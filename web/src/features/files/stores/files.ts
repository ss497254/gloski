import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import type { PinnedFolder } from '@gloski/sdk'

type ViewMode = 'list' | 'grid'
type SortBy = 'name' | 'size' | 'modified'
type SortOrder = 'asc' | 'desc'

// Re-export PinnedFolder type for components
export type { PinnedFolder }

interface FilesState {
  // Pinned folders (synced with server)
  pinnedFolders: PinnedFolder[]
  setPinnedFolders: (folders: PinnedFolder[]) => void
  addPinnedFolder: (folder: PinnedFolder) => void
  removePinnedFolder: (id: string) => void
  isPinned: (path: string) => boolean
  getPinnedFolderByPath: (path: string) => PinnedFolder | undefined

  // Selection state (not persisted)
  selectedPaths: Set<string>
  lastSelectedPath: string | null
  selectPath: (path: string) => void
  deselectPath: (path: string) => void
  toggleSelection: (path: string) => void
  selectAll: (paths: string[]) => void
  clearSelection: () => void
  selectRange: (allPaths: string[], targetPath: string) => void
  isSelected: (path: string) => boolean

  // View preferences (persisted locally)
  viewMode: ViewMode
  sortBy: SortBy
  sortOrder: SortOrder
  setViewMode: (mode: ViewMode) => void
  setSortBy: (by: SortBy) => void
  setSortOrder: (order: SortOrder) => void
  toggleSortOrder: () => void

  // Sidebar state
  sidebarOpen: boolean
  setSidebarOpen: (open: boolean) => void
  toggleSidebar: () => void
}

export const useFilesStore = create<FilesState>()(
  persist(
    (set, get) => ({
      // Pinned folders (synced with server, not persisted locally)
      pinnedFolders: [],

      setPinnedFolders: (folders) => {
        set({ pinnedFolders: folders })
      },

      addPinnedFolder: (folder) => {
        set((state) => ({
          pinnedFolders: [...state.pinnedFolders, folder],
        }))
      },

      removePinnedFolder: (id) => {
        set((state) => ({
          pinnedFolders: state.pinnedFolders.filter((f) => f.id !== id),
        }))
      },

      isPinned: (path) => {
        return get().pinnedFolders.some((f) => f.path === path)
      },

      getPinnedFolderByPath: (path) => {
        return get().pinnedFolders.find((f) => f.path === path)
      },

      // Selection state
      selectedPaths: new Set(),
      lastSelectedPath: null,

      selectPath: (path) => {
        set((state) => {
          const newSet = new Set(state.selectedPaths)
          newSet.add(path)
          return { selectedPaths: newSet, lastSelectedPath: path }
        })
      },

      deselectPath: (path) => {
        set((state) => {
          const newSet = new Set(state.selectedPaths)
          newSet.delete(path)
          return { selectedPaths: newSet }
        })
      },

      toggleSelection: (path) => {
        set((state) => {
          const newSet = new Set(state.selectedPaths)
          if (newSet.has(path)) {
            newSet.delete(path)
          } else {
            newSet.add(path)
          }
          return { selectedPaths: newSet, lastSelectedPath: path }
        })
      },

      selectAll: (paths) => {
        set({ selectedPaths: new Set(paths), lastSelectedPath: paths[paths.length - 1] || null })
      },

      clearSelection: () => {
        set({ selectedPaths: new Set(), lastSelectedPath: null })
      },

      selectRange: (allPaths, targetPath) => {
        const { lastSelectedPath, selectedPaths } = get()
        if (!lastSelectedPath) {
          set({ selectedPaths: new Set([targetPath]), lastSelectedPath: targetPath })
          return
        }

        const startIdx = allPaths.indexOf(lastSelectedPath)
        const endIdx = allPaths.indexOf(targetPath)

        if (startIdx === -1 || endIdx === -1) {
          set({ selectedPaths: new Set([targetPath]), lastSelectedPath: targetPath })
          return
        }

        const [from, to] = startIdx < endIdx ? [startIdx, endIdx] : [endIdx, startIdx]
        const rangePaths = allPaths.slice(from, to + 1)

        const newSet = new Set(selectedPaths)
        rangePaths.forEach((p) => newSet.add(p))

        set({ selectedPaths: newSet, lastSelectedPath: targetPath })
      },

      isSelected: (path) => {
        return get().selectedPaths.has(path)
      },

      // View preferences
      viewMode: 'list',
      sortBy: 'name',
      sortOrder: 'asc',

      setViewMode: (mode) => set({ viewMode: mode }),
      setSortBy: (by) => set({ sortBy: by }),
      setSortOrder: (order) => set({ sortOrder: order }),
      toggleSortOrder: () => set((state) => ({ sortOrder: state.sortOrder === 'asc' ? 'desc' : 'asc' })),

      // Sidebar state
      sidebarOpen: true,
      setSidebarOpen: (open) => set({ sidebarOpen: open }),
      toggleSidebar: () => set((state) => ({ sidebarOpen: !state.sidebarOpen })),
    }),
    {
      name: 'gloski-files',
      // Only persist view preferences locally, not pinned folders (those come from server)
      partialize: (state) => ({
        viewMode: state.viewMode,
        sortBy: state.sortBy,
        sortOrder: state.sortOrder,
        sidebarOpen: state.sidebarOpen,
      }),
    }
  )
)
