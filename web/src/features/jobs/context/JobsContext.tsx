/* eslint-disable react-refresh/only-export-components */
import { useDevice } from '@/shared/context'
import { jobId as toJobId, unwrap, type Job } from '@/shared/api'
import { createContext, useCallback, useContext, useEffect, useMemo, useRef, useState, type ReactNode } from 'react'

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
  const mountedRef = useRef(false)
  const fetchingRef = useRef(false)
  const deviceIdRef = useRef(deviceId)
  deviceIdRef.current = deviceId

  const fetchJobs = useCallback(async () => {
    if (fetchingRef.current) return
    fetchingRef.current = true
    const requestedDevice = deviceId
    try {
      const result = unwrap(await device.getClient().jobs.list())
      if (mountedRef.current && deviceIdRef.current === requestedDevice) {
        setJobs(result || [])
        setError('')
      }
    } catch {
      if (mountedRef.current && deviceIdRef.current === requestedDevice) {
        setError('Failed to fetch jobs')
      }
    } finally {
      fetchingRef.current = false
    }
  }, [device, deviceId])

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

  // Initial load and poll lifecycle follow the current device.
  useEffect(() => {
    void refresh()
  }, [deviceId, refresh])

  useEffect(() => {
    mountedRef.current = true
    return () => {
      mountedRef.current = false
    }
  }, [])

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
