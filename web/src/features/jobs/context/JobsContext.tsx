/* eslint-disable react-refresh/only-export-components */
import { useDevice } from '@/shared/context'
import { jobId as toJobId, unwrap, type Job } from '@/shared/api'
import { createContext, useCallback, useContext, useEffect, useMemo, useState, type ReactNode } from 'react'

// ─────────────────────────────────────────────────────────────────────────────
// Types
// ─────────────────────────────────────────────────────────────────────────────

interface JobsContextValue {
  // Data
  jobs: Job[]
  loading: boolean
  error: string

  // Selected job
  selectedJob: string | null
  jobLogs: string[]

  // New job form
  newCommand: string
  setNewCommand: (cmd: string) => void
  newCwd: string
  setNewCwd: (cwd: string) => void

  // Actions
  startJob: () => Promise<void>
  stopJob: (id: string) => Promise<void>
  fetchJobLogs: (id: string) => Promise<void>
  clearSelectedJob: () => void
  refresh: () => Promise<void>
}

const JobsContext = createContext<JobsContextValue | null>(null)

// ─────────────────────────────────────────────────────────────────────────────
// Constants
// ─────────────────────────────────────────────────────────────────────────────

const POLLING_INTERVAL = 5000

// ─────────────────────────────────────────────────────────────────────────────
// Provider
// ─────────────────────────────────────────────────────────────────────────────

interface JobsProviderProps {
  children: ReactNode
}

export function JobsProvider({ children }: JobsProviderProps) {
  const { device, deviceId } = useDevice()
  const [jobs, setJobs] = useState<Job[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  const [newCommand, setNewCommand] = useState('')
  const [newCwd, setNewCwd] = useState('')

  const [selectedJob, setSelectedJob] = useState<string | null>(null)
  const [jobLogs, setJobLogs] = useState<string[]>([])

  const fetchJobs = useCallback(async () => {
    try {
      const result = unwrap(await device.getClient().jobs.list())
      setJobs(result || [])
      setError('') // Clear error on successful fetch
    } catch {
      setError('Failed to fetch jobs')
    }
  }, [device])

  const refresh = useCallback(async () => {
    setLoading(true)
    setError('')
    try {
      await fetchJobs()
    } finally {
      setLoading(false)
    }
  }, [fetchJobs])

  const startJob = useCallback(async () => {
    if (!newCommand.trim()) return

    try {
      await device.getClient().jobs.start(newCommand, newCwd || undefined)
      setNewCommand('')
      setNewCwd('')
      setError('') // Clear error on success
      fetchJobs()
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to start job')
    }
  }, [device, newCommand, newCwd, fetchJobs])

  const stopJob = useCallback(
    async (id: string) => {
      try {
        await device.getClient().jobs.stop(toJobId(id))
        setError('') // Clear error on success
        fetchJobs()
      } catch {
        setError('Failed to stop job')
      }
    },
    [device, fetchJobs]
  )

  const fetchJobLogs = useCallback(
    async (id: string) => {
      try {
        const logs = unwrap(await device.getClient().jobs.getLogs(toJobId(id)))
        setJobLogs(logs || [])
        setSelectedJob(id)
        setError('') // Clear error on success
      } catch {
        setError('Failed to fetch logs')
      }
    },
    [device]
  )

  const clearSelectedJob = useCallback(() => {
    setSelectedJob(null)
    setJobLogs([])
  }, [])

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
  }, [deviceId, refresh])

  // Polling for job updates
  useEffect(() => {
    const interval = setInterval(fetchJobs, POLLING_INTERVAL)
    return () => clearInterval(interval)
  }, [fetchJobs])

  const value: JobsContextValue = useMemo(
    () => ({
      jobs,
      loading,
      error,
      selectedJob,
      jobLogs,
      newCommand,
      setNewCommand,
      newCwd,
      setNewCwd,
      startJob,
      stopJob,
      fetchJobLogs,
      clearSelectedJob,
      refresh,
    }),
    [
      jobs,
      loading,
      error,
      selectedJob,
      jobLogs,
      newCommand,
      newCwd,
      startJob,
      stopJob,
      fetchJobLogs,
      clearSelectedJob,
      refresh,
    ]
  )

  return <JobsContext.Provider value={value}>{children}</JobsContext.Provider>
}

// ─────────────────────────────────────────────────────────────────────────────
// Hook
// ─────────────────────────────────────────────────────────────────────────────

export function useJobs(): JobsContextValue {
  const context = useContext(JobsContext)

  if (!context) {
    throw new Error('useJobs must be used within a JobsProvider.')
  }

  return context
}
