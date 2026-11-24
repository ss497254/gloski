import { Button } from '@/ui/button'
import { Card, CardContent } from '@/ui/card'
import type { Job } from '@/shared/api'
import { AlertCircle, CheckCircle, Clock, Square, XCircle } from 'lucide-react'
import { useJobs } from '../context'

interface JobItemProps {
  job: Job
}

function getStatusIcon(status: string) {
  switch (status) {
    case 'running':
      return <Clock className="h-4 w-4 text-info animate-pulse" />
    case 'finished':
      return <CheckCircle className="h-4 w-4 text-success" />
    case 'failed':
      return <XCircle className="h-4 w-4 text-destructive" />
    case 'stopped':
    case 'abandoned':
      return <AlertCircle className="h-4 w-4 text-warning" />
    case 'pending':
      return <Clock className="h-4 w-4 text-muted-foreground" />
    default:
      return null
  }
}

function formatTime(dateStr: string) {
  return new Date(dateStr).toLocaleTimeString()
}

export function JobItem({ job }: JobItemProps) {
  const { selectedJob, fetchJobLogs, stopJob } = useJobs()

  return (
    <Card className={`transition-colors ${selectedJob === job.id ? 'ring-2 ring-primary' : ''}`}>
      <CardContent className="flex items-center gap-3 p-4">
        <button
          type="button"
          className="flex min-w-0 flex-1 items-center gap-3 text-left"
          aria-pressed={selectedJob === job.id}
          onClick={() => void fetchJobLogs(job.id)}
        >
          {getStatusIcon(job.status)}
          <span className="min-w-0 flex-1">
            <span className="block truncate font-mono text-sm">{job.command}</span>
            <span className="block text-xs text-muted-foreground">
              {job.pid && `PID: ${job.pid} | `}Started: {job.started_at ? formatTime(job.started_at) : 'N/A'}
              {job.cwd && ` | ${job.cwd}`}
            </span>
          </span>
        </button>
        {job.status === 'running' && (
          <Button
            variant="ghost"
            size="icon"
            aria-label={`Stop job ${job.command}`}
            onClick={() => void stopJob(job.id)}
          >
              <Square className="h-4 w-4" />
          </Button>
        )}
      </CardContent>
    </Card>
  )
}
