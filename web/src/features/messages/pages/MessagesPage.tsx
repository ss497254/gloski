import { useState, useMemo } from 'react'
import { Button } from '@/ui/button'
import { Badge } from '@/ui/badge'
import { ScrollArea } from '@/ui/scroll-area'
import { PageLayout } from '@/layouts'
import { EmptyState } from '@/shared/components'
import { useMessagesStore, type Message, type MessageType } from '../stores/messages'
import { cn, formatRelativeTime } from '@/shared/lib/utils'
import {
  Mail,
  MailOpen,
  Star,
  Trash2,
  CheckCheck,
  Info,
  AlertTriangle,
  CheckCircle,
  XCircle,
  Filter,
  ArrowLeft,
} from 'lucide-react'

const typeIcons: Record<MessageType, React.ComponentType<{ className?: string }>> = {
  info: Info,
  warning: AlertTriangle,
  success: CheckCircle,
  error: XCircle,
}

const typeColors: Record<MessageType, string> = {
  info: 'text-blue-500',
  warning: 'text-yellow-500',
  success: 'text-green-500',
  error: 'text-red-500',
}

type FilterType = 'all' | 'unread' | 'starred'

function MessageItem({ message, isSelected, onClick }: { message: Message; isSelected: boolean; onClick: () => void }) {
  const Icon = typeIcons[message.type]

  return (
    <button
      onClick={onClick}
      className={cn(
        'w-full text-left p-4 border-b transition-colors',
        isSelected ? 'bg-accent' : 'hover:bg-muted/50',
        !message.read && 'bg-muted/30'
      )}
    >
      <div className="flex items-start gap-3">
        <Icon className={cn('h-5 w-5 mt-0.5 shrink-0', typeColors[message.type])} />
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2">
            <span className={cn('font-medium truncate', !message.read && 'font-semibold')}>{message.title}</span>
            {message.starred && <Star className="h-4 w-4 fill-yellow-400 text-yellow-400 shrink-0" />}
          </div>
          <p className="text-sm text-muted-foreground line-clamp-1 mt-0.5">{message.content}</p>
          <span className="text-xs text-muted-foreground mt-1 block">{formatRelativeTime(message.createdAt)}</span>
        </div>
        {!message.read && <div className="h-2 w-2 rounded-full bg-primary shrink-0 mt-2" />}
      </div>
    </button>
  )
}

export function MessagesPage() {
  const messages = useMessagesStore((s) => s.messages)
  const markAsRead = useMessagesStore((s) => s.markAsRead)
  const markAllAsRead = useMessagesStore((s) => s.markAllAsRead)
  const toggleStar = useMessagesStore((s) => s.toggleStar)
  const deleteMessage = useMessagesStore((s) => s.deleteMessage)

  const [filter, setFilter] = useState<FilterType>('all')
  const [selectedId, setSelectedId] = useState<string | null>(null)
  // Mobile view state: 'list' or 'detail'
  const [mobileView, setMobileView] = useState<'list' | 'detail'>('list')

  const unreadCount = messages.filter((m) => !m.read).length

  const filteredMessages = useMemo(() => {
    return messages.filter((m) => {
      if (filter === 'unread') return !m.read
      if (filter === 'starred') return m.starred
      return true
    })
  }, [messages, filter])

  const selectedMessage = selectedId ? messages.find((m) => m.id === selectedId) : null

  const handleSelect = (id: string) => {
    setSelectedId(id)
    markAsRead(id)
    setMobileView('detail')
  }

  const handleBackToList = () => {
    setMobileView('list')
  }

  const handleDelete = () => {
    if (selectedMessage) {
      deleteMessage(selectedMessage.id)
      setSelectedId(null)
      setMobileView('list')
    }
  }

  const Icon = selectedMessage ? typeIcons[selectedMessage.type] : null

  return (
    <PageLayout
      title="Messages"
      description={`${unreadCount} unread`}
      actions={
        unreadCount > 0 && (
          <Button variant="outline" size="sm" onClick={markAllAsRead}>
            <CheckCheck className="h-4 w-4 md:mr-2" />
            <span className="hidden md:inline">Mark all read</span>
          </Button>
        )
      }
      noPadding
    >
      <div className="flex h-full">
        {/* Message list (hidden on mobile when viewing detail) */}
        <div className={cn('w-full md:w-96 border-r flex flex-col', mobileView === 'detail' && 'hidden md:flex')}>
          {/* Filters */}
          <div className="p-3 border-b flex items-center gap-2">
            <Filter className="h-4 w-4 text-muted-foreground shrink-0" />
            <div className="flex gap-1 overflow-x-auto">
              {(['all', 'unread', 'starred'] as FilterType[]).map((f) => (
                <Button
                  key={f}
                  variant={filter === f ? 'secondary' : 'ghost'}
                  size="sm"
                  onClick={() => setFilter(f)}
                  className="capitalize shrink-0"
                >
                  {f}
                  {f === 'unread' && unreadCount > 0 && (
                    <Badge className="ml-1.5 h-5 px-1.5" variant="destructive">
                      {unreadCount}
                    </Badge>
                  )}
                </Button>
              ))}
            </div>
          </div>

          {/* Messages */}
          <ScrollArea className="flex-1">
            {filteredMessages.length > 0 ? (
              filteredMessages.map((message) => (
                <MessageItem
                  key={message.id}
                  message={message}
                  isSelected={selectedId === message.id}
                  onClick={() => handleSelect(message.id)}
                />
              ))
            ) : (
              <div className="p-8">
                <EmptyState
                  icon={Mail}
                  title="No messages"
                  description={filter !== 'all' ? `No ${filter} messages` : 'Your inbox is empty'}
                />
              </div>
            )}
          </ScrollArea>
        </div>

        {/* Message detail (hidden on mobile when viewing list) */}
        <div className={cn('flex-1 flex flex-col', mobileView === 'list' && 'hidden md:flex')}>
          {selectedMessage ? (
            <>
              <div className="p-3 md:p-4 border-b">
                <div className="flex items-start justify-between gap-2">
                  <div className="flex items-start gap-2 md:gap-3 min-w-0">
                    {/* Mobile back button */}
                    <Button variant="ghost" size="icon" onClick={handleBackToList} className="md:hidden shrink-0 -ml-1">
                      <ArrowLeft className="h-4 w-4" />
                    </Button>
                    {Icon && (
                      <Icon
                        className={cn(
                          'h-5 w-5 md:h-6 md:w-6 mt-0.5 md:mt-1 shrink-0',
                          typeColors[selectedMessage.type]
                        )}
                      />
                    )}
                    <div className="min-w-0">
                      <h2 className="text-base md:text-lg font-semibold truncate">{selectedMessage.title}</h2>
                      <span className="text-xs md:text-sm text-muted-foreground">
                        {formatRelativeTime(selectedMessage.createdAt)}
                      </span>
                    </div>
                  </div>
                  <div className="flex items-center gap-1 shrink-0">
                    <Button variant="ghost" size="icon" onClick={() => toggleStar(selectedMessage.id)}>
                      <Star className={cn('h-4 w-4', selectedMessage.starred && 'fill-yellow-400 text-yellow-400')} />
                    </Button>
                    <Button variant="ghost" size="icon" onClick={handleDelete}>
                      <Trash2 className="h-4 w-4 text-destructive" />
                    </Button>
                  </div>
                </div>
              </div>

              <ScrollArea className="flex-1 p-4 md:p-6">
                <p className="text-muted-foreground whitespace-pre-wrap text-sm md:text-base">
                  {selectedMessage.content}
                </p>
              </ScrollArea>
            </>
          ) : (
            <div className="flex-1 flex items-center justify-center">
              <EmptyState icon={MailOpen} title="No message selected" description="Select a message to read" />
            </div>
          )}
        </div>
      </div>
    </PageLayout>
  )
}
