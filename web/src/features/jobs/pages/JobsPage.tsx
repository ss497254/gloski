import { Button } from '@/ui/button'
import { RefreshCw, Terminal } from 'lucide-react'
import { JobForm, JobList, JobLogs } from '../components'
import { JobsProvider, useJobs } from '../context'

function JobsContent() {
  const { loading, error, refresh } = useJobs()

  return (
    <div className="h-screen flex flex-col">
      {/* Header */}
      <div className="border-b p-4 flex items-center gap-4">
        <Terminal className="h-5 w-5" />
        <h1 className="text-lg font-semibold">Jobs</h1>

        <div className="flex-1" />

        <Button variant="ghost" size="icon" onClick={refresh}>
          <RefreshCw className={`h-4 w-4 ${loading ? 'animate-spin' : ''}`} />
        </Button>
      </div>

      {error && <div className="bg-destructive/10 text-destructive px-4 py-2 text-sm">{error}</div>}

      <div className="flex-1 flex overflow-hidden">
        {/* Main content */}
        <div className="flex-1 overflow-auto p-4">
          <div className="space-y-4">
            <JobForm />
            <JobList />
          </div>
        </div>

        {/* Logs panel */}
        <JobLogs />
      </div>
    </div>
  )
}

export function JobsPage() {
  return (
    <JobsProvider>
      <JobsContent />
    </JobsProvider>
  )
}
