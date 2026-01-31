import { useParams } from 'react-router-dom'
import { MetricsHistory } from '../components'
import { PageLayout } from '@/layouts'

/**
 * Dedicated page for viewing server metrics history with charts
 */
export function ServerMetricsPage() {
  const { serverId } = useParams<{ serverId: string }>()

  if (!serverId) {
    return (
      <PageLayout title="Metrics">
        <div className="p-6">
          <p className="text-muted-foreground">No server selected</p>
        </div>
      </PageLayout>
    )
  }

  return (
    <PageLayout title="Metrics" description="System performance over time">
      <div className="p-6">
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          <MetricsHistory enabled />
        </div>
      </div>
    </PageLayout>
  )
}
