import { useState, useMemo } from 'react'
import { Button } from '@/ui/button'
import { PageLayout } from '@/layouts'
import { SearchInput, EmptyState } from '@/shared/components'
import { useSnippetsStore, type Snippet } from '../stores/snippets'
import { SnippetCard, SnippetDialog, LanguageFilter } from '../components'
import { Plus, Code2 } from 'lucide-react'

export function SnippetsPage() {
  const { snippets, addSnippet, updateSnippet, deleteSnippet, toggleFavorite } = useSnippetsStore()

  const [search, setSearch] = useState('')
  const [selectedLanguage, setSelectedLanguage] = useState<string | null>(null)
  const [showFavorites, setShowFavorites] = useState(false)
  const [dialogOpen, setDialogOpen] = useState(false)
  const [editingSnippet, setEditingSnippet] = useState<Snippet | null>(null)

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
    setEditingSnippet(snippet || null)
    setDialogOpen(true)
  }

  const handleSubmit = (formData: {
    title: string
    code: string
    language: string
    description?: string
    tags: string[]
  }) => {
    if (editingSnippet) {
      updateSnippet(editingSnippet.id, formData)
    } else {
      addSnippet(formData)
    }
    setDialogOpen(false)
    setEditingSnippet(null)
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
          <SearchInput
            value={search}
            onChange={setSearch}
            placeholder="Search snippets..."
            className="flex-1 max-w-md"
          />

          <LanguageFilter
            languages={languages}
            selected={selectedLanguage}
            onSelect={setSelectedLanguage}
            showFavorites={showFavorites}
            onToggleFavorites={() => setShowFavorites(!showFavorites)}
          />
        </div>

        {/* Snippets grid */}
        {filteredSnippets.length > 0 ? (
          <div className="grid gap-4 md:grid-cols-2">
            {filteredSnippets.map((snippet) => (
              <SnippetCard
                key={snippet.id}
                snippet={snippet}
                onEdit={() => openDialog(snippet)}
                onDelete={() => deleteSnippet(snippet.id)}
                onToggleFavorite={() => toggleFavorite(snippet.id)}
              />
            ))}
          </div>
        ) : (
          <EmptyState
            icon={Code2}
            title="No snippets found"
            description={search ? 'Try a different search term' : 'Create your first snippet to get started'}
          />
        )}
      </div>

      {/* Dialog */}
      <SnippetDialog
        open={dialogOpen}
        onOpenChange={(open) => {
          setDialogOpen(open)
          if (!open) setEditingSnippet(null)
        }}
        snippet={editingSnippet}
        onSubmit={handleSubmit}
      />
    </PageLayout>
  )
}
