import { useJobs } from '../context'
import { JobItem } from './JobItem'
import { JobListSkeleton } from '@/shared/components'

export function JobList() {
  const { jobs, loading } = useJobs()

  if (loading && jobs.length === 0) {
    return <JobListSkeleton />
  }

  if (jobs.length === 0) {
    return <div className="text-muted-foreground text-center py-8">No jobs</div>
  }

  return (
    <div className="space-y-2">
      {jobs.map((job) => (
        <JobItem key={job.id} job={job} />
      ))}
    </div>
  )
}
