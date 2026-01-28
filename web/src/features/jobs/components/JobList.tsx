import { useJobs } from '../context'
import { JobItem } from './JobItem'

export function JobList() {
  const { jobs } = useJobs()

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
