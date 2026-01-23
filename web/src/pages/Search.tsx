import { useState, useEffect, useRef } from 'react'
import { Link, Navigate, useParams } from 'react-router-dom'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import { ScrollArea } from '@/components/ui/scroll-area'
import { useServer } from '@/hooks'
import type { SearchResult } from '@/lib/types'
import { toast } from 'sonner'
import { cn } from '@/lib/utils'
import {
  Search,
  Folder,
  File,
  FileText,
  FileCode,
  RefreshCw,
  FolderOpen,
  X,
  ToggleLeft,
  ToggleRight,
  ExternalLink,
} from 'lucide-react'

export function SearchPage() {
  const { serverId } = useParams()
  const { server, api } = useServer()

  const [searchPath, setSearchPath] = useState('/')
  const [query, setQuery] = useState('')
  const [searchContent, setSearchContent] = useState(false)
  const [results, setResults] = useState<SearchResult[]>([])
  const [loading, setLoading] = useState(false)
  const [searched, setSearched] = useState(false)

  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  // Redirect if no server
  if (!server || !api) {
    return <Navigate to="/" replace />
  }

  const performSearch = async (q: string, path: string, content: boolean) => {
    if (!api || !q.trim()) {
      setResults([])
      setSearched(false)
      return
    }

    setLoading(true)
    setSearched(true)

    try {
      const data = await api.search(path, q.trim(), content)
      setResults(data.results || [])
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Search failed')
      setResults([])
    } finally {
      setLoading(false)
    }
  }

  const handleQueryChange = (value: string) => {
    setQuery(value)

    // Debounce the search
    if (debounceRef.current) {
      clearTimeout(debounceRef.current)
    }
    debounceRef.current = setTimeout(() => {
      performSearch(value, searchPath, searchContent)
    }, 300)
  }

  // Cleanup timeout on unmount
  useEffect(() => {
    return () => {
      if (debounceRef.current) {
        clearTimeout(debounceRef.current)
      }
    }
  }, [])

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    performSearch(query, searchPath, searchContent)
  }

  const toggleContentSearch = () => {
    const newValue = !searchContent
    setSearchContent(newValue)
    if (query.trim()) {
      performSearch(query, searchPath, newValue)
    }
  }

  const getIcon = (result: SearchResult) => {
    if (result.type === 'directory') return Folder
    const ext = result.name.split('.').pop()?.toLowerCase() || ''
    if (
      ['ts', 'tsx', 'js', 'jsx', 'py', 'go', 'rs', 'java', 'c', 'cpp', 'h'].includes(ext)
    ) {
      return FileCode
    }
    if (['md', 'txt', 'log'].includes(ext)) {
      return FileText
    }
    return File
  }

  const getIconColor = (result: SearchResult) => {
    if (result.type === 'directory') return 'text-blue-500'
    const ext = result.name.split('.').pop()?.toLowerCase() || ''
    if (['ts', 'tsx'].includes(ext)) return 'text-blue-400'
    if (['js', 'jsx'].includes(ext)) return 'text-yellow-400'
    if (['py'].includes(ext)) return 'text-green-400'
    if (['go'].includes(ext)) return 'text-cyan-400'
    return 'text-muted-foreground'
  }

  const highlightMatch = (text: string, q: string) => {
    if (!q) return text
    const regex = new RegExp(`(${q.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')})`, 'gi')
    const parts = text.split(regex)
    return parts.map((part, i) =>
      regex.test(part) ? (
        <mark key={i} className="bg-yellow-300/30 text-yellow-200">
          {part}
        </mark>
      ) : (
        part
      )
    )
  }

  return (
    <div className="h-full flex flex-col">
      {/* Header */}
      <div className="border-b p-4">
        <form onSubmit={handleSubmit} className="space-y-3">
          <div className="flex gap-2">
            <div className="relative flex-1">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-5 w-5 text-muted-foreground" />
              <Input
                placeholder="Search for files..."
                value={query}
                onChange={(e) => handleQueryChange(e.target.value)}
                className="pl-10 h-11 text-lg"
                autoFocus
              />
              {query && (
                <Button
                  type="button"
                  variant="ghost"
                  size="icon"
                  className="absolute right-2 top-1/2 -translate-y-1/2 h-7 w-7"
                  onClick={() => {
                    setQuery('')
                    setResults([])
                    setSearched(false)
                  }}
                >
                  <X className="h-4 w-4" />
                </Button>
              )}
            </div>
            <Button type="submit" disabled={loading || !query.trim()}>
              {loading ? <RefreshCw className="h-4 w-4 animate-spin" /> : 'Search'}
            </Button>
          </div>

          <div className="flex items-center gap-4">
            <div className="flex items-center gap-2">
              <FolderOpen className="h-4 w-4 text-muted-foreground" />
              <Input
                placeholder="Search in path..."
                value={searchPath}
                onChange={(e) => setSearchPath(e.target.value)}
                className="h-8 w-64"
              />
            </div>

            <Button
              type="button"
              variant="ghost"
              size="sm"
              onClick={toggleContentSearch}
              className={cn('gap-2', searchContent && 'bg-primary/10 text-primary')}
            >
              {searchContent ? (
                <ToggleRight className="h-4 w-4" />
              ) : (
                <ToggleLeft className="h-4 w-4" />
              )}
              Search file contents
            </Button>
          </div>
        </form>
      </div>

      {/* Results */}
      <ScrollArea className="flex-1">
        {loading ? (
          <div className="p-8 text-center text-muted-foreground">
            <RefreshCw className="h-8 w-8 mx-auto mb-4 animate-spin" />
            <p>Searching...</p>
          </div>
        ) : results.length > 0 ? (
          <div className="divide-y">
            <div className="px-4 py-2 bg-muted/30 text-sm text-muted-foreground sticky top-0">
              Found {results.length} result{results.length !== 1 ? 's' : ''}
            </div>
            {results.map((result, index) => {
              const Icon = getIcon(result)
              const color = getIconColor(result)
              const parentPath = result.path.split('/').slice(0, -1).join('/') || '/'

              return (
                <div
                  key={`${result.path}-${result.line_num || index}`}
                  className="p-4 hover:bg-muted/50 transition-colors"
                >
                  <div className="flex items-start gap-3">
                    <Icon className={cn('h-5 w-5 mt-0.5 shrink-0', color)} />
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center gap-2 mb-1">
                        <span className="font-medium truncate">
                          {highlightMatch(result.name, query)}
                        </span>
                        {result.line_num && (
                          <Badge variant="secondary" className="shrink-0">
                            Line {result.line_num}
                          </Badge>
                        )}
                      </div>
                      <p className="text-sm text-muted-foreground truncate mb-2">
                        {result.path}
                      </p>
                      {result.match && (
                        <div className="bg-muted rounded p-2 text-sm font-mono overflow-x-auto">
                          {highlightMatch(result.match, query)}
                        </div>
                      )}
                      <div className="flex gap-2 mt-2">
                        <Button variant="ghost" size="sm" asChild>
                          <Link
                            to={`/servers/${serverId}/files?path=${encodeURIComponent(parentPath)}`}
                          >
                            <Folder className="h-3 w-3 mr-1" />
                            Open Folder
                          </Link>
                        </Button>
                        {result.type === 'file' && (
                          <Button variant="ghost" size="sm" asChild>
                            <Link
                              to={`/servers/${serverId}/files?path=${encodeURIComponent(parentPath)}&file=${encodeURIComponent(result.name)}`}
                            >
                              <ExternalLink className="h-3 w-3 mr-1" />
                              View File
                            </Link>
                          </Button>
                        )}
                      </div>
                    </div>
                  </div>
                </div>
              )
            })}
          </div>
        ) : searched ? (
          <div className="p-8 text-center text-muted-foreground">
            <Search className="h-12 w-12 mx-auto mb-4 opacity-50" />
            <p className="font-medium mb-1">No results found</p>
            <p className="text-sm">Try a different search term or path</p>
          </div>
        ) : (
          <div className="p-8 text-center text-muted-foreground">
            <Search className="h-16 w-16 mx-auto mb-4 opacity-30" />
            <p className="text-lg font-medium mb-2">Search Files</p>
            <p className="text-sm max-w-md mx-auto">
              Search for files by name. Enable "Search file contents" to search
              inside files too.
            </p>
          </div>
        )}
      </ScrollArea>
    </div>
  )
}
