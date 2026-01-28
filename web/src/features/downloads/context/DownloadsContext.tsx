/* eslint-disable react-refresh/only-export-components */
import { useServer } from '@/features/servers/context'
import type { Download, ShareLink } from '@/shared/lib/types'
import { createContext, useCallback, useContext, useEffect, useMemo, useRef, useState, type ReactNode } from 'react'
import { toast } from 'sonner'

// ─────────────────────────────────────────────────────────────────────────────
// Types
// ─────────────────────────────────────────────────────────────────────────────

export type DownloadFilter = 'all' | 'active' | 'completed' | 'failed'

interface DownloadsContextValue {
  // Data
  downloads: Download[]
  sortedDownloads: Download[]
  loading: boolean
  filter: DownloadFilter

  // Filter
  setFilter: (filter: DownloadFilter) => void

  // Actions
  handleAdd: (url: string, destination: string, filename?: string) => Promise<void>
  handlePause: (id: string) => Promise<void>
  handleResume: (id: string) => Promise<void>
  handleCancel: (id: string) => Promise<void>
  handleRetry: (id: string) => Promise<void>
  handleDelete: (id: string, deleteFile: boolean) => Promise<void>
  handleDownload: (download: Download) => void
  handleCreateShareLink: (id: string, expiresIn?: number) => Promise<ShareLink>
  handleRevokeShareLink: (downloadId: string, token: string) => Promise<void>

  // Dialogs
  showAddDialog: boolean
  setShowAddDialog: (show: boolean) => void
  shareDownload: Download | null
  setShareDownload: (download: Download | null) => void

  // Refresh
  refresh: () => Promise<void>
}

const DownloadsContext = createContext<DownloadsContextValue | null>(null)

// ─────────────────────────────────────────────────────────────────────────────
// Provider
// ─────────────────────────────────────────────────────────────────────────────

interface DownloadsProviderProps {
  children: ReactNode
}

// Polling intervals
const POLLING_INTERVAL_ACTIVE = 1000
const POLLING_INTERVAL_IDLE = 10000

export function DownloadsProvider({ children }: DownloadsProviderProps) {
  const { server, serverId } = useServer()
  const [downloads, setDownloads] = useState<Download[]>([])
  const [loading, setLoading] = useState(true)
  const [showAddDialog, setShowAddDialog] = useState(false)
  const [shareDownload, setShareDownload] = useState<Download | null>(null)
  const [filter, setFilter] = useState<DownloadFilter>('all')

  // Track if there are active downloads for polling interval
  const hasActiveRef = useRef(false)

  const fetchDownloads = useCallback(async () => {
    try {
      const response = await server.getClient().downloads.list()
      setDownloads(response.downloads || [])
    } catch {
      // Silently fail for polling
    }
  }, [server])

  const refresh = useCallback(async () => {
    setLoading(true)
    try {
      await fetchDownloads()
    } finally {
      setLoading(false)
    }
  }, [fetchDownloads])

  // Update active status ref when downloads change
  useEffect(() => {
    hasActiveRef.current = downloads.some((d) => d.status === 'downloading' || d.status === 'pending')
  }, [downloads])

  // Initial load with cleanup for unmount
  useEffect(() => {
    let isMounted = true
    const load = async () => {
      if (isMounted) {
        await refresh()
      }
    }
    load()
    return () => {
      isMounted = false
    }
  }, [serverId, refresh])

  // Polling with dynamic interval based on active downloads
  useEffect(() => {
    const poll = () => {
      fetchDownloads()
    }
    // Check active status and set appropriate interval
    const getInterval = () => (hasActiveRef.current ? POLLING_INTERVAL_ACTIVE : POLLING_INTERVAL_IDLE)

    let interval = setInterval(poll, getInterval())

    // Re-check interval periodically to adjust polling speed
    const intervalChecker = setInterval(() => {
      clearInterval(interval)
      interval = setInterval(poll, getInterval())
    }, POLLING_INTERVAL_IDLE)

    return () => {
      clearInterval(interval)
      clearInterval(intervalChecker)
    }
  }, [fetchDownloads])

  // ─────────────────────────────────────────────────────────────────────────────
  // Actions
  // ─────────────────────────────────────────────────────────────────────────────

  const handleAdd = useCallback(
    async (url: string, destination: string, filename?: string) => {
      try {
        await server.getClient().downloads.add(url, destination, filename)
        toast.success('Download added')
        fetchDownloads()
        setShowAddDialog(false)
      } catch (err) {
        toast.error(err instanceof Error ? err.message : 'Failed to add download')
      }
    },
    [server, fetchDownloads]
  )

  const handlePause = useCallback(
    async (id: string) => {
      try {
        await server.getClient().downloads.pause(id)
        toast.success('Download paused')
        fetchDownloads()
      } catch (err) {
        toast.error(err instanceof Error ? err.message : 'Failed to pause download')
      }
    },
    [server, fetchDownloads]
  )

  const handleResume = useCallback(
    async (id: string) => {
      try {
        await server.getClient().downloads.resume(id)
        toast.success('Download resumed')
        fetchDownloads()
      } catch (err) {
        toast.error(err instanceof Error ? err.message : 'Failed to resume download')
      }
    },
    [server, fetchDownloads]
  )

  const handleCancel = useCallback(
    async (id: string) => {
      try {
        await server.getClient().downloads.cancel(id)
        toast.success('Download cancelled')
        fetchDownloads()
      } catch (err) {
        toast.error(err instanceof Error ? err.message : 'Failed to cancel download')
      }
    },
    [server, fetchDownloads]
  )

  const handleRetry = useCallback(
    async (id: string) => {
      try {
        await server.getClient().downloads.retry(id)
        toast.success('Retrying download')
        fetchDownloads()
      } catch (err) {
        toast.error(err instanceof Error ? err.message : 'Failed to retry download')
      }
    },
    [server, fetchDownloads]
  )

  const handleDelete = useCallback(
    async (id: string, deleteFile: boolean) => {
      try {
        await server.getClient().downloads.delete(id, deleteFile)
        toast.success('Download deleted')
        fetchDownloads()
      } catch (err) {
        toast.error(err instanceof Error ? err.message : 'Failed to delete download')
      }
    },
    [server, fetchDownloads]
  )

  const handleDownload = useCallback(
    (download: Download) => {
      const url = server.getClient().downloads.getDownloadUrl(download.id)
      window.open(url, '_blank')
    },
    [server]
  )

  const handleCreateShareLink = useCallback(
    async (id: string, expiresIn?: number) => {
      try {
        const link = await server
          .getClient()
          .downloads.createShareLink(id, expiresIn ? { expires_in: expiresIn } : undefined)
        await navigator.clipboard.writeText(link.url)
        toast.success('Share link copied to clipboard')
        fetchDownloads()
        return link
      } catch (err) {
        toast.error(err instanceof Error ? err.message : 'Failed to create share link')
        throw err
      }
    },
    [server, fetchDownloads]
  )

  const handleRevokeShareLink = useCallback(
    async (downloadId: string, token: string) => {
      try {
        await server.getClient().downloads.revokeShareLink(downloadId, token)
        toast.success('Share link revoked')
        fetchDownloads()
      } catch (err) {
        toast.error(err instanceof Error ? err.message : 'Failed to revoke share link')
      }
    },
    [server, fetchDownloads]
  )

  // ─────────────────────────────────────────────────────────────────────────────
  // Computed
  // ─────────────────────────────────────────────────────────────────────────────

  const sortedDownloads = useMemo(() => {
    const filtered = downloads.filter((d) => {
      switch (filter) {
        case 'active':
          return d.status === 'downloading' || d.status === 'pending' || d.status === 'paused'
        case 'completed':
          return d.status === 'completed'
        case 'failed':
          return d.status === 'failed' || d.status === 'cancelled'
        default:
          return true
      }
    })
    // Sort by created_at (newest first)
    return [...filtered].sort((a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime())
  }, [downloads, filter])

  // ─────────────────────────────────────────────────────────────────────────────
  // Context value
  // ─────────────────────────────────────────────────────────────────────────────

  const value: DownloadsContextValue = useMemo(
    () => ({
      downloads,
      sortedDownloads,
      loading,
      filter,
      setFilter,
      handleAdd,
      handlePause,
      handleResume,
      handleCancel,
      handleRetry,
      handleDelete,
      handleDownload,
      handleCreateShareLink,
      handleRevokeShareLink,
      showAddDialog,
      setShowAddDialog,
      shareDownload,
      setShareDownload,
      refresh,
    }),
    [
      downloads,
      sortedDownloads,
      loading,
      filter,
      handleAdd,
      handlePause,
      handleResume,
      handleCancel,
      handleRetry,
      handleDelete,
      handleDownload,
      handleCreateShareLink,
      handleRevokeShareLink,
      showAddDialog,
      shareDownload,
      refresh,
    ]
  )

  return <DownloadsContext.Provider value={value}>{children}</DownloadsContext.Provider>
}

// ─────────────────────────────────────────────────────────────────────────────
// Hook
// ─────────────────────────────────────────────────────────────────────────────

export function useDownloads(): DownloadsContextValue {
  const context = useContext(DownloadsContext)

  if (!context) {
    throw new Error('useDownloads must be used within a DownloadsProvider.')
  }

  return context
}
