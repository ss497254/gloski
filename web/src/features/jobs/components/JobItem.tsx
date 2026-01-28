import type { Job } from '@/shared/lib/types'
import { Button } from '@/ui/button'
import { Card, CardContent } from '@/ui/card'
import { AlertCircle, CheckCircle, Clock, Square, XCircle } from 'lucide-react'
import { useJobs } from '../context'

interface JobItemProps {
  job: Job
}

function getStatusIcon(status: string) {
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

function formatTime(dateStr: string) {
  return new Date(dateStr).toLocaleTimeString()
}

export function JobItem({ job }: JobItemProps) {
  const { selectedJob, fetchJobLogs, stopJob } = useJobs()

  return (
    <Card
      className={`cursor-pointer transition-colors ${selectedJob === job.id ? 'ring-2 ring-primary' : ''}`}
      onClick={() => fetchJobLogs(job.id)}
    >
      <CardContent className="p-4">
        <div className="flex items-center gap-3">
          {getStatusIcon(job.status)}
          <div className="flex-1 min-w-0">
            <div className="font-mono text-sm truncate">{job.command}</div>
            <div className="text-xs text-muted-foreground">
              {job.pid && `PID: ${job.pid} | `}Started: {job.started_at ? formatTime(job.started_at) : 'N/A'}
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
  )
}
