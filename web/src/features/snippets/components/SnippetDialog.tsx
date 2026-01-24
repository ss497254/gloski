/* eslint-disable react-hooks/set-state-in-effect */
import { useState, useEffect } from 'react'
import { Button } from '@/ui/button'
import { Input } from '@/ui/input'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from '@/ui/dialog'
import type { Snippet } from '../stores/snippets'

interface SnippetDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  snippet: Snippet | null
  onSubmit: (data: {
    title: string
    code: string
    language: string
    description?: string
    tags: string[]
  }) => void
}

export function SnippetDialog({
  open,
  onOpenChange,
  snippet,
  onSubmit,
}: SnippetDialogProps) {
  const [title, setTitle] = useState('')
  const [code, setCode] = useState('')
  const [language, setLanguage] = useState('javascript')
  const [description, setDescription] = useState('')
  const [tags, setTags] = useState('')

  const isEditing = !!snippet

  useEffect(() => {
    if (open && snippet) {
      setTitle(snippet.title)
      setCode(snippet.code)
      setLanguage(snippet.language)
      setDescription(snippet.description || '')
      setTags(snippet.tags.join(', '))
    } else if (open) {
      setTitle('')
      setCode('')
      setLanguage('javascript')
      setDescription('')
      setTags('')
    }
  }, [open, snippet])

  const handleSubmit = () => {
    const parsedTags = tags
      .split(',')
      .map((t) => t.trim())
      .filter(Boolean)

    onSubmit({
      title,
      code,
      language,
      description: description || undefined,
      tags: parsedTags,
    })
  }

  const isValid = title.trim() && code.trim()

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-2xl">
        <DialogHeader>
          <DialogTitle>{isEditing ? 'Edit Snippet' : 'New Snippet'}</DialogTitle>
        </DialogHeader>

        <div className="space-y-4 py-4">
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <label className="text-sm font-medium">Title</label>
              <Input
                value={title}
                onChange={(e) => setTitle(e.target.value)}
                placeholder="My Snippet"
              />
            </div>

            <div className="space-y-2">
              <label className="text-sm font-medium">Language</label>
              <Input
                value={language}
                onChange={(e) => setLanguage(e.target.value)}
                placeholder="javascript"
              />
            </div>
          </div>

          <div className="space-y-2">
            <label className="text-sm font-medium">Description</label>
            <Input
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder="Optional description"
            />
          </div>

          <div className="space-y-2">
            <label className="text-sm font-medium">Code</label>
            <textarea
              value={code}
              onChange={(e) => setCode(e.target.value)}
              placeholder="// Your code here"
              className="w-full h-48 p-3 font-mono text-sm bg-muted rounded-lg border-0 resize-none focus:outline-none focus:ring-2 focus:ring-ring"
            />
          </div>

          <div className="space-y-2">
            <label className="text-sm font-medium">Tags</label>
            <Input
              value={tags}
              onChange={(e) => setTags(e.target.value)}
              placeholder="react, hooks, utils"
            />
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button onClick={handleSubmit} disabled={!isValid}>
            {isEditing ? 'Save' : 'Create'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
