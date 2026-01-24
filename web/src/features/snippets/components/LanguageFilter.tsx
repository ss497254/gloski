import { Button } from '@/ui/button'
import { Star } from 'lucide-react'
import { cn } from '@/shared/lib/utils'

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

interface LanguageFilterProps {
  languages: string[]
  selected: string | null
  onSelect: (language: string | null) => void
  showFavorites: boolean
  onToggleFavorites: () => void
}

export function LanguageFilter({
  languages,
  selected,
  onSelect,
  showFavorites,
  onToggleFavorites,
}: LanguageFilterProps) {
  return (
    <div className="flex items-center gap-2">
      <Button
        variant={showFavorites ? 'secondary' : 'outline'}
        size="sm"
        onClick={onToggleFavorites}
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
        variant={!selected ? 'secondary' : 'outline'}
        size="sm"
        onClick={() => onSelect(null)}
      >
        All
      </Button>

      {languages.map((lang) => (
        <Button
          key={lang}
          variant={selected === lang ? 'secondary' : 'outline'}
          size="sm"
          onClick={() => onSelect(lang)}
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
  )
}
