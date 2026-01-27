import { Button } from '@/ui/button'
import { Card, CardContent } from '@/ui/card'
import { Badge } from '@/ui/badge'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/ui/dropdown-menu'
import { ExternalLink, Folder, MoreVertical, Pencil, Trash2, Globe } from 'lucide-react'
import type { Bookmark } from '../stores/bookmarks'

interface BookmarkCardProps {
  bookmark: Bookmark
  onEdit: () => void
  onDelete: () => void
}

export function BookmarkCard({ bookmark, onEdit, onDelete }: BookmarkCardProps) {
  return (
    <Card className="group hover:border-primary/50 transition-colors">
      <CardContent className="p-4">
        <div className="flex items-start gap-3">
          <div className="h-10 w-10 rounded-lg bg-muted flex items-center justify-center shrink-0">
            <Globe className="h-5 w-5 text-muted-foreground" />
          </div>

          <div className="flex-1 min-w-0">
            <div className="flex items-start justify-between gap-2">
              <a
                href={bookmark.url}
                target="_blank"
                rel="noopener noreferrer"
                className="font-medium hover:text-primary truncate block"
              >
                {bookmark.title}
              </a>
              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <Button
                    variant="ghost"
                    size="icon"
                    className="h-8 w-8 opacity-0 group-hover:opacity-100 transition-opacity shrink-0"
                  >
                    <MoreVertical className="h-4 w-4" />
                  </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="end">
                  <DropdownMenuItem onClick={() => window.open(bookmark.url, '_blank')}>
                    <ExternalLink className="h-4 w-4 mr-2" />
                    Open
                  </DropdownMenuItem>
                  <DropdownMenuItem onClick={onEdit}>
                    <Pencil className="h-4 w-4 mr-2" />
                    Edit
                  </DropdownMenuItem>
                  <DropdownMenuSeparator />
                  <DropdownMenuItem onClick={onDelete} className="text-destructive">
                    <Trash2 className="h-4 w-4 mr-2" />
                    Delete
                  </DropdownMenuItem>
                </DropdownMenuContent>
              </DropdownMenu>
            </div>

            {bookmark.description && (
              <p className="text-sm text-muted-foreground line-clamp-2 mt-1">{bookmark.description}</p>
            )}

            <div className="flex items-center gap-2 mt-2 flex-wrap">
              {bookmark.folder && (
                <Badge variant="secondary" className="text-xs">
                  <Folder className="h-3 w-3 mr-1" />
                  {bookmark.folder}
                </Badge>
              )}
              {bookmark.tags.slice(0, 3).map((tag) => (
                <Badge key={tag} variant="outline" className="text-xs">
                  {tag}
                </Badge>
              ))}
            </div>
          </div>
        </div>
      </CardContent>
    </Card>
  )
}
