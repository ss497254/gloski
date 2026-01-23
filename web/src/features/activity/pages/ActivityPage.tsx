import { useState, useMemo } from 'react'
import { Button } from '@/ui/button'
import { Badge } from '@/ui/badge'
import { ScrollArea } from '@/ui/scroll-area'
import { PageLayout } from '@/layouts'
import { FilterSidebar, EmptyState } from '@/shared/components'
import { useActivityStore, type ActivityAction, type ActivityItem } from '../stores/activity'
import { cn, formatDate, formatRelativeTime } from '@/shared/lib/utils'
import {
  Activity,
  Server,
  File,
  FileText,
  FilePlus,
  FileX,
  Upload,
  Download,
  Terminal,
  Play,
  CheckCircle,
  Bookmark,
  Code2,
  Settings,
  Trash2,
} from 'lucide-react'

const actionIcons: Record<ActivityAction, React.ComponentType<{ className?: string }>> = {
  server_added: Server,
  server_removed: Server,
  server_connected: Server,
  server_disconnected: Server,
  file_created: FilePlus,
  file_deleted: FileX,
  file_modified: FileText,
  file_uploaded: Upload,
  file_downloaded: Download,
  terminal_opened: Terminal,
  task_started: Play,
  task_completed: CheckCircle,
  note_created: FileText,
  note_updated: FileText,
  bookmark_added: Bookmark,
  snippet_created: Code2,
  settings_changed: Settings,
}

const actionColors: Record<ActivityAction, string> = {
  server_added: 'text-green-500',
  server_removed: 'text-red-500',
  server_connected: 'text-green-500',
  server_disconnected: 'text-yellow-500',
  file_created: 'text-green-500',
  file_deleted: 'text-red-500',
  file_modified: 'text-blue-500',
  file_uploaded: 'text-cyan-500',
  file_downloaded: 'text-purple-500',
  terminal_opened: 'text-yellow-500',
  task_started: 'text-blue-500',
  task_completed: 'text-green-500',
  note_created: 'text-green-500',
  note_updated: 'text-blue-500',
  bookmark_added: 'text-yellow-500',
  snippet_created: 'text-purple-500',
  settings_changed: 'text-gray-500',
}

type FilterCategory = 'all' | 'servers' | 'files' | 'tasks' | 'other'

const categoryFilters: Record<FilterCategory, ActivityAction[]> = {
  all: [],
  servers: ['server_added', 'server_removed', 'server_connected', 'server_disconnected'],
  files: ['file_created', 'file_deleted', 'file_modified', 'file_uploaded', 'file_downloaded'],
  tasks: ['task_started', 'task_completed', 'terminal_opened'],
  other: ['note_created', 'note_updated', 'bookmark_added', 'snippet_created', 'settings_changed'],
}

function ActivityItemComponent({ item }: { item: ActivityItem }) {
  const Icon = actionIcons[item.action] || Activity

  return (
    <div className="flex items-start gap-4 py-4">
      <div
        className={cn(
          'h-8 w-8 rounded-full bg-muted flex items-center justify-center shrink-0',
          actionColors[item.action]
        )}
      >
        <Icon className="h-4 w-4" />
      </div>

      <div className="flex-1 min-w-0">
        <p className="text-sm">{item.description}</p>
        <div className="flex items-center gap-2 mt-1">
          <span className="text-xs text-muted-foreground">
            {formatRelativeTime(item.timestamp)}
          </span>
          {item.serverName && (
            <Badge variant="secondary" className="text-xs">
              <Server className="h-3 w-3 mr-1" />
              {item.serverName}
            </Badge>
          )}
        </div>
      </div>
    </div>
  )
}

function groupByDate(items: ActivityItem[]): Record<string, ActivityItem[]> {
  const groups: Record<string, ActivityItem[]> = {}

  items.forEach((item) => {
    const date = new Date(item.timestamp)
    const today = new Date()
    const yesterday = new Date(today)
    yesterday.setDate(yesterday.getDate() - 1)

    let key: string
    if (date.toDateString() === today.toDateString()) {
      key = 'Today'
    } else if (date.toDateString() === yesterday.toDateString()) {
      key = 'Yesterday'
    } else {
      key = formatDate(date)
    }

    if (!groups[key]) groups[key] = []
    groups[key].push(item)
  })

  return groups
}

export function ActivityPage() {
  const { items, clearAll, clearOlderThan } = useActivityStore()
  const [filter, setFilter] = useState<FilterCategory>('all')

  const filteredItems = useMemo(() => {
    if (filter === 'all') return items
    return items.filter((item) => categoryFilters[filter].includes(item.action))
  }, [items, filter])

  const groupedItems = useMemo(() => groupByDate(filteredItems), [filteredItems])

  return (
    <PageLayout
      title="Activity"
      description="Recent actions across all features"
      actions={
        <div className="flex items-center gap-2">
          <Button
            variant="outline"
            size="sm"
            onClick={() => clearOlderThan(7)}
          >
            Clear older than 7 days
          </Button>
          <Button
            variant="outline"
            size="sm"
            onClick={() => {
              if (confirm('Clear all activity?')) clearAll()
            }}
          >
            <Trash2 className="h-4 w-4 mr-2" />
            Clear all
          </Button>
        </div>
      }
    >
      <div className="flex gap-6">
        {/* Filters */}
        <FilterSidebar
          selected={filter === 'all' ? null : filter}
          onSelect={(id) => setFilter((id as FilterCategory) || 'all')}
          allItem={{
            label: 'All',
            icon: Activity,
          }}
          sections={[
            {
              items: [
                { id: 'servers', label: 'Servers', icon: Server },
                { id: 'files', label: 'Files', icon: File },
                { id: 'tasks', label: 'Tasks', icon: Terminal },
                { id: 'other', label: 'Other', icon: Bookmark },
              ],
            },
          ]}
        />

        {/* Timeline */}
        <ScrollArea className="flex-1">
          {Object.keys(groupedItems).length > 0 ? (
            <div className="space-y-6">
              {Object.entries(groupedItems).map(([date, dateItems]) => (
                <div key={date}>
                  <h3 className="text-sm font-medium text-muted-foreground mb-2 sticky top-0 bg-background py-2">
                    {date}
                  </h3>
                  <div className="divide-y">
                    {dateItems.map((item) => (
                      <ActivityItemComponent key={item.id} item={item} />
                    ))}
                  </div>
                </div>
              ))}
            </div>
          ) : (
            <EmptyState
              icon={Activity}
              title="No activity to show"
              description={filter !== 'all' ? 'Try selecting a different category' : 'Your activity will appear here'}
            />
          )}
        </ScrollArea>
      </div>
    </PageLayout>
  )
}
