import { useState, useMemo } from 'react'
import { Button } from '@/ui/button'
import { Input } from '@/ui/input'
import { Card, CardContent, CardHeader, CardTitle } from '@/ui/card'
import { Badge } from '@/ui/badge'
import { ScrollArea } from '@/ui/scroll-area'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from '@/ui/dialog'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/ui/dropdown-menu'
import { PageLayout } from '@/layouts'
import { useSnippetsStore, type Snippet } from '../stores/snippets'
import { cn } from '@/shared/lib/utils'
import {
  Plus,
  Search,
  Code2,
  Star,
  Copy,
  Pencil,
  Trash2,
  MoreVertical,
  Check,
} from 'lucide-react'
import { toast } from 'sonner'

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

function SnippetCard({
  snippet,
  onEdit,
  onDelete,
  onToggleFavorite,
}: {
  snippet: Snippet
  onEdit: (s: Snippet) => void
  onDelete: (id: string) => void
  onToggleFavorite: (id: string) => void
}) {
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
            <div
              className={cn(
                'h-3 w-3 rounded-full',
                languageColors[snippet.language] || 'bg-gray-400'
              )}
            />
            <CardTitle className="text-base">{snippet.title}</CardTitle>
          </div>
          <div className="flex items-center gap-1">
            <Button
              variant="ghost"
              size="icon"
              className="h-8 w-8"
              onClick={() => onToggleFavorite(snippet.id)}
            >
              <Star
                className={cn(
                  'h-4 w-4',
                  snippet.favorite && 'fill-yellow-400 text-yellow-400'
                )}
              />
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
                <DropdownMenuItem onClick={() => onEdit(snippet)}>
                  <Pencil className="h-4 w-4 mr-2" />
                  Edit
                </DropdownMenuItem>
                <DropdownMenuSeparator />
                <DropdownMenuItem
                  onClick={() => onDelete(snippet.id)}
                  className="text-destructive"
                >
                  <Trash2 className="h-4 w-4 mr-2" />
                  Delete
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          </div>
        </div>
        {snippet.description && (
          <p className="text-sm text-muted-foreground">{snippet.description}</p>
        )}
      </CardHeader>
      <CardContent>
        <div className="relative">
          <ScrollArea className="h-40">
            <pre className="text-xs font-mono bg-muted p-3 rounded-lg overflow-x-auto">
              <code>{snippet.code}</code>
            </pre>
          </ScrollArea>
          <Button
            variant="secondary"
            size="sm"
            className="absolute top-2 right-2 opacity-0 group-hover:opacity-100 transition-opacity"
            onClick={handleCopy}
          >
            {copied ? (
              <Check className="h-3 w-3 mr-1" />
            ) : (
              <Copy className="h-3 w-3 mr-1" />
            )}
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

export function SnippetsPage() {
  const { snippets, addSnippet, updateSnippet, deleteSnippet, toggleFavorite } =
    useSnippetsStore()

  const [search, setSearch] = useState('')
  const [selectedLanguage, setSelectedLanguage] = useState<string | null>(null)
  const [showFavorites, setShowFavorites] = useState(false)
  const [dialogOpen, setDialogOpen] = useState(false)
  const [editingSnippet, setEditingSnippet] = useState<Snippet | null>(null)

  // Form state
  const [formTitle, setFormTitle] = useState('')
  const [formCode, setFormCode] = useState('')
  const [formLanguage, setFormLanguage] = useState('javascript')
  const [formDescription, setFormDescription] = useState('')
  const [formTags, setFormTags] = useState('')

  const languages = useMemo(() => {
    const langs = new Set(snippets.map((s) => s.language))
    return Array.from(langs).sort()
  }, [snippets])

  const filteredSnippets = useMemo(() => {
    return snippets.filter((s) => {
      const matchesSearch =
        !search ||
        s.title.toLowerCase().includes(search.toLowerCase()) ||
        s.code.toLowerCase().includes(search.toLowerCase()) ||
        s.tags.some((t) => t.toLowerCase().includes(search.toLowerCase()))

      const matchesLanguage = !selectedLanguage || s.language === selectedLanguage
      const matchesFavorite = !showFavorites || s.favorite

      return matchesSearch && matchesLanguage && matchesFavorite
    })
  }, [snippets, search, selectedLanguage, showFavorites])

  const openDialog = (snippet?: Snippet) => {
    if (snippet) {
      setEditingSnippet(snippet)
      setFormTitle(snippet.title)
      setFormCode(snippet.code)
      setFormLanguage(snippet.language)
      setFormDescription(snippet.description || '')
      setFormTags(snippet.tags.join(', '))
    } else {
      setEditingSnippet(null)
      setFormTitle('')
      setFormCode('')
      setFormLanguage('javascript')
      setFormDescription('')
      setFormTags('')
    }
    setDialogOpen(true)
  }

  const handleSubmit = () => {
    const tags = formTags
      .split(',')
      .map((t) => t.trim())
      .filter(Boolean)

    if (editingSnippet) {
      updateSnippet(editingSnippet.id, {
        title: formTitle,
        code: formCode,
        language: formLanguage,
        description: formDescription || undefined,
        tags,
      })
    } else {
      addSnippet({
        title: formTitle,
        code: formCode,
        language: formLanguage,
        description: formDescription || undefined,
        tags,
      })
    }

    setDialogOpen(false)
  }

  return (
    <PageLayout
      title="Snippets"
      description="Save and organize code snippets"
      actions={
        <Button onClick={() => openDialog()}>
          <Plus className="h-4 w-4 mr-2" />
          New Snippet
        </Button>
      }
    >
      <div className="space-y-4">
        {/* Filters */}
        <div className="flex items-center gap-4">
          <div className="relative flex-1 max-w-md">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
            <Input
              placeholder="Search snippets..."
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              className="pl-9"
            />
          </div>

          <div className="flex items-center gap-2">
            <Button
              variant={showFavorites ? 'secondary' : 'outline'}
              size="sm"
              onClick={() => setShowFavorites(!showFavorites)}
            >
              <Star
                className={cn(
                  'h-4 w-4 mr-2',
                  showFavorites && 'fill-yellow-400 text-yellow-400'
                )}
              />
              Favorites
            </Button>

            <Button
              variant={!selectedLanguage ? 'secondary' : 'outline'}
              size="sm"
              onClick={() => setSelectedLanguage(null)}
            >
              All
            </Button>

            {languages.map((lang) => (
              <Button
                key={lang}
                variant={selectedLanguage === lang ? 'secondary' : 'outline'}
                size="sm"
                onClick={() => setSelectedLanguage(lang)}
                className="capitalize"
              >
                <div
                  className={cn(
                    'h-2 w-2 rounded-full mr-2',
                    languageColors[lang] || 'bg-gray-400'
                  )}
                />
                {lang}
              </Button>
            ))}
          </div>
        </div>

        {/* Snippets grid */}
        {filteredSnippets.length > 0 ? (
          <div className="grid gap-4 md:grid-cols-2">
            {filteredSnippets.map((snippet) => (
              <SnippetCard
                key={snippet.id}
                snippet={snippet}
                onEdit={openDialog}
                onDelete={deleteSnippet}
                onToggleFavorite={toggleFavorite}
              />
            ))}
          </div>
        ) : (
          <div className="text-center py-12 text-muted-foreground">
            <Code2 className="h-12 w-12 mx-auto mb-4 opacity-50" />
            <p>No snippets found</p>
          </div>
        )}
      </div>

      {/* Add/Edit Dialog */}
      <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
        <DialogContent className="max-w-2xl">
          <DialogHeader>
            <DialogTitle>
              {editingSnippet ? 'Edit Snippet' : 'New Snippet'}
            </DialogTitle>
          </DialogHeader>

          <div className="space-y-4 py-4">
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <label className="text-sm font-medium">Title</label>
                <Input
                  value={formTitle}
                  onChange={(e) => setFormTitle(e.target.value)}
                  placeholder="My Snippet"
                />
              </div>

              <div className="space-y-2">
                <label className="text-sm font-medium">Language</label>
                <Input
                  value={formLanguage}
                  onChange={(e) => setFormLanguage(e.target.value)}
                  placeholder="javascript"
                />
              </div>
            </div>

            <div className="space-y-2">
              <label className="text-sm font-medium">Description</label>
              <Input
                value={formDescription}
                onChange={(e) => setFormDescription(e.target.value)}
                placeholder="Optional description"
              />
            </div>

            <div className="space-y-2">
              <label className="text-sm font-medium">Code</label>
              <textarea
                value={formCode}
                onChange={(e) => setFormCode(e.target.value)}
                placeholder="// Your code here"
                className="w-full h-48 p-3 font-mono text-sm bg-muted rounded-lg border-0 resize-none focus:outline-none focus:ring-2 focus:ring-ring"
              />
            </div>

            <div className="space-y-2">
              <label className="text-sm font-medium">Tags</label>
              <Input
                value={formTags}
                onChange={(e) => setFormTags(e.target.value)}
                placeholder="react, hooks, utils"
              />
            </div>
          </div>

          <DialogFooter>
            <Button variant="outline" onClick={() => setDialogOpen(false)}>
              Cancel
            </Button>
            <Button onClick={handleSubmit} disabled={!formTitle || !formCode}>
              {editingSnippet ? 'Save' : 'Create'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </PageLayout>
  )
}
