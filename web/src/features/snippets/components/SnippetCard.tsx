import { CodeEditor } from '@/shared/components/code-editor'
import { cn } from '@/shared/lib/utils'
import { Badge } from '@/ui/badge'
import { Button } from '@/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/ui/card'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/ui/dropdown-menu'
import { Check, Copy, MoreVertical, Pencil, Star, Trash2 } from 'lucide-react'
import { useState } from 'react'
import { toast } from 'sonner'
import { languageToFilename } from '../lib/language-utils'
import type { Snippet } from '../stores/snippets'

const languageColors: Record<string, string> = {
  typescript: 'bg-blue-500',
  javascript: 'bg-yellow-500',
  python: 'bg-green-500',
  go: 'bg-cyan-500',
  rust: 'bg-orange-500',
  sql: 'bg-purple-500',
  bash: 'bg-gray-500',
  css: 'bg-pink-500',
  yaml: 'bg-red-500',
}

interface SnippetCardProps {
  snippet: Snippet
  onEdit: () => void
  onDelete: () => void
  onToggleFavorite: () => void
}

export function SnippetCard({ snippet, onEdit, onDelete, onToggleFavorite }: SnippetCardProps) {
  const [copied, setCopied] = useState(false)

  const handleCopy = async () => {
    await navigator.clipboard.writeText(snippet.code)
    setCopied(true)
    toast.success('Copied to clipboard')
    setTimeout(() => setCopied(false), 2000)
  }

  return (
    <Card className="group">
      <CardHeader className="pb-2">
        <div className="flex items-start justify-between">
          <div className="flex items-center gap-2">
            <div className={cn('h-3 w-3 rounded-full', languageColors[snippet.language] || 'bg-gray-400')} />
            <CardTitle className="text-base">{snippet.title}</CardTitle>
          </div>
          <div className="flex items-center gap-1">
            <Button variant="ghost" size="icon" className="h-8 w-8" onClick={onToggleFavorite}>
              <Star className={cn('h-4 w-4', snippet.favorite && 'fill-yellow-400 text-yellow-400')} />
            </Button>
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button variant="ghost" size="icon" className="h-8 w-8">
                  <MoreVertical className="h-4 w-4" />
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end">
                <DropdownMenuItem onClick={handleCopy}>
                  <Copy className="h-4 w-4 mr-2" />
                  Copy
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
        </div>
        {snippet.description && <p className="text-sm text-muted-foreground">{snippet.description}</p>}
      </CardHeader>
      <CardContent>
        <div className="relative">
          <div className="h-40 overflow-hidden rounded-lg">
            <CodeEditor
              value={snippet.code}
              filename={languageToFilename(snippet.language)}
              readOnly
              lineNumbers={false}
              className="h-full"
            />
          </div>
          <Button
            variant="secondary"
            size="sm"
            className="absolute top-2 right-2 opacity-0 group-hover:opacity-100 transition-opacity"
            onClick={handleCopy}
          >
            {copied ? <Check className="h-3 w-3 mr-1" /> : <Copy className="h-3 w-3 mr-1" />}
            {copied ? 'Copied' : 'Copy'}
          </Button>
        </div>
        <div className="flex items-center gap-2 mt-3">
          <Badge variant="outline">{snippet.language}</Badge>
          {snippet.tags.map((tag) => (
            <Badge key={tag} variant="secondary" className="text-xs">
              {tag}
            </Badge>
          ))}
        </div>
      </CardContent>
    </Card>
  )
}
