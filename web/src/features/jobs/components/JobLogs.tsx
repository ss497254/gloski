import { Button } from '@/ui/button'
import { X } from 'lucide-react'
import { useJobs } from '../context'

export function JobLogs() {
  const { selectedJob, jobLogs, clearSelectedJob } = useJobs()

  if (!selectedJob) return null

  return (
    <div className="w-1/2 border-l flex flex-col">
      <div className="p-3 border-b flex items-center justify-between">
        <span className="font-medium text-sm">Job Logs</span>
        <Button variant="ghost" size="icon" onClick={clearSelectedJob}>
          <X className="h-4 w-4" />
        </Button>
      </div>
      <div className="flex-1 overflow-auto p-4 bg-muted/30">
        <pre className="text-xs font-mono whitespace-pre-wrap">
          {jobLogs.length > 0 ? jobLogs.join('\n') : 'No output yet'}
        </pre>
      </div>
    </div>
  )
}
