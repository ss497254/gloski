import { useState, useEffect, useCallback } from 'react'
import { Navigate } from 'react-router-dom'
import { Button } from '@/ui/button'
import { Input } from '@/ui/input'
import { Card, CardContent, CardHeader, CardTitle } from '@/ui/card'
import { useServer } from '@/features/servers/hooks/use-server'
import type { Job } from '@/shared/lib/types'
import { Play, Square, Trash2, RefreshCw, Terminal, Clock, CheckCircle, XCircle, AlertCircle } from 'lucide-react'

export function JobsPage() {
  const { server, serverId } = useServer()

  const [jobs, setJobs] = useState<Job[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  const [newCommand, setNewCommand] = useState('')
  const [newCwd, setNewCwd] = useState('')

  const [selectedJob, setSelectedJob] = useState<string | null>(null)
  const [jobLogs, setJobLogs] = useState<string[]>([])

  const fetchJobs = useCallback(async () => {
    if (!server) return
    try {
      const result = await server.getClient().jobs.list()
      setJobs(result || [])
    } catch {
      setError('Failed to fetch jobs')
    }
  }, [server])

  const loadData = useCallback(async () => {
    setLoading(true)
    setError('')
    await fetchJobs()
    setLoading(false)
  }, [fetchJobs])

  const startJob = useCallback(async () => {
    if (!server || !newCommand.trim()) return

    try {
      await server.getClient().jobs.start(newCommand, newCwd || undefined)
      setNewCommand('')
      setNewCwd('')
      fetchJobs()
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to start job')
    }
  }, [server, newCommand, newCwd, fetchJobs])

  const stopJob = useCallback(
    async (id: string) => {
      if (!server) return
      try {
        await server.getClient().jobs.stop(id)
        fetchJobs()
      } catch {
        setError('Failed to stop job')
      }
    },
    [server, fetchJobs]
  )

  const fetchJobLogs = useCallback(
    async (id: string) => {
      if (!server) return
      try {
        const logs = await server.getClient().jobs.getLogs(id)
        setJobLogs(logs || [])
        setSelectedJob(id)
      } catch {
        setError('Failed to fetch logs')
      }
    },
    [server]
  )

  // Initial load and polling
  useEffect(() => {
    if (!server) return

    loadData()
    const interval = setInterval(fetchJobs, 5000)
    return () => clearInterval(interval)
  }, [serverId, server, loadData, fetchJobs])

  // Redirect if no server
  if (!server) {
    return <Navigate to="/" replace />
  }

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'running':
        return <Clock className="h-4 w-4 text-blue-500 animate-pulse" />
      case 'finished':
        return <CheckCircle className="h-4 w-4 text-green-500" />
      case 'failed':
        return <XCircle className="h-4 w-4 text-red-500" />
      case 'stopped':
        return <AlertCircle className="h-4 w-4 text-yellow-500" />
      case 'pending':
        return <Clock className="h-4 w-4 text-gray-400" />
      default:
        return null
    }
  }

  const formatTime = (dateStr: string) => {
    return new Date(dateStr).toLocaleTimeString()
  }

  return (
    <div className="h-screen flex flex-col">
      {/* Header */}
      <div className="border-b p-4 flex items-center gap-4">
        <Terminal className="h-5 w-5" />
        <h1 className="text-lg font-semibold">Jobs</h1>

        <div className="flex-1" />

        <Button variant="ghost" size="icon" onClick={loadData}>
          <RefreshCw className={`h-4 w-4 ${loading ? 'animate-spin' : ''}`} />
        </Button>
      </div>

      {error && <div className="bg-destructive/10 text-destructive px-4 py-2 text-sm">{error}</div>}

      <div className="flex-1 flex overflow-hidden">
        {/* Main content */}
        <div className="flex-1 overflow-auto p-4">
          <div className="space-y-4">
            {/* New job form */}
            <Card>
              <CardHeader className="pb-3">
                <CardTitle className="text-sm">Start New Job</CardTitle>
              </CardHeader>
              <CardContent className="space-y-3">
                <Input
                  placeholder="Command (e.g., bun run dev)"
                  value={newCommand}
                  onChange={(e) => setNewCommand(e.target.value)}
                  onKeyDown={(e) => e.key === 'Enter' && startJob()}
                />
                <div className="flex gap-2">
                  <Input
                    placeholder="Working directory (optional)"
                    value={newCwd}
                    onChange={(e) => setNewCwd(e.target.value)}
                    className="flex-1"
                  />
                  <Button onClick={startJob} disabled={!newCommand.trim()}>
                    <Play className="h-4 w-4 mr-2" />
                    Start
                  </Button>
                </div>
              </CardContent>
            </Card>

            {/* Job list */}
            <div className="space-y-2">
              {jobs.length === 0 ? (
                <div className="text-muted-foreground text-center py-8">No jobs</div>
              ) : (
                jobs.map((job) => (
                  <Card
                    key={job.id}
                    className={`cursor-pointer transition-colors ${
                      selectedJob === job.id ? 'ring-2 ring-primary' : ''
                    }`}
                    onClick={() => fetchJobLogs(job.id)}
                  >
                    <CardContent className="p-4">
                      <div className="flex items-center gap-3">
                        {getStatusIcon(job.status)}
                        <div className="flex-1 min-w-0">
                          <div className="font-mono text-sm truncate">{job.command}</div>
                          <div className="text-xs text-muted-foreground">
                            {job.pid && `PID: ${job.pid} | `}Started:{' '}
                            {job.started_at ? formatTime(job.started_at) : 'N/A'}
                            {job.cwd && ` | ${job.cwd}`}
                          </div>
                        </div>
                        {job.status === 'running' && (
                          <Button
                            variant="ghost"
                            size="icon"
                            onClick={(e) => {
                              e.stopPropagation()
                              stopJob(job.id)
                            }}
                          >
                            <Square className="h-4 w-4" />
                          </Button>
                        )}
                      </div>
                    </CardContent>
                  </Card>
                ))
              )}
            </div>
          </div>
        </div>

        {/* Logs panel */}
        {selectedJob && (
          <div className="w-1/2 border-l flex flex-col">
            <div className="p-3 border-b flex items-center justify-between">
              <span className="font-medium text-sm">Job Logs</span>
              <Button variant="ghost" size="icon" onClick={() => setSelectedJob(null)}>
                <Trash2 className="h-4 w-4" />
              </Button>
            </div>
            <div className="flex-1 overflow-auto p-4 bg-muted/30">
              <pre className="text-xs font-mono whitespace-pre-wrap">
                {jobLogs.length > 0 ? jobLogs.join('\n') : 'No output yet'}
              </pre>
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
