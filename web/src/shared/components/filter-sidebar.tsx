import type { LucideIcon } from 'lucide-react'
import { cn } from '@/shared/lib/utils'

export interface FilterItem {
  id: string
  label: string
  icon?: LucideIcon
  count?: number
}

export interface FilterSection {
  title?: string
  items: FilterItem[]
}

interface FilterSidebarProps {
  sections: FilterSection[]
  selected: string | null
  onSelect: (id: string | null) => void
  allItem?: {
    label: string
    icon?: LucideIcon
    count?: number
  }
  className?: string
}

export function FilterSidebar({
  sections,
  selected,
  onSelect,
  allItem,
  className,
}: FilterSidebarProps) {
  return (
    <div className={cn('w-48 shrink-0 space-y-1', className)}>
      {/* All items button */}
      {allItem && (
        <button
          onClick={() => onSelect(null)}
          className={cn(
            'w-full flex items-center gap-2 px-3 py-2 rounded-lg text-sm font-medium transition-colors',
            selected === null
              ? 'bg-accent text-accent-foreground'
              : 'text-muted-foreground hover:bg-accent/50'
          )}
        >
          {allItem.icon && <allItem.icon className="h-4 w-4" />}
          {allItem.label}
          {allItem.count !== undefined && (
            <span className="ml-auto text-xs">{allItem.count}</span>
          )}
        </button>
      )}

      {/* Sections */}
      {sections.map((section, sectionIndex) => (
        <div key={sectionIndex}>
          {section.title && (
            <div className="pt-4 pb-2 px-3">
              <span className="text-xs font-medium text-muted-foreground uppercase">
                {section.title}
              </span>
            </div>
          )}
          {section.items.map((item) => (
            <button
              key={item.id}
              onClick={() => onSelect(item.id)}
              className={cn(
                'w-full flex items-center gap-2 px-3 py-2 rounded-lg text-sm font-medium transition-colors',
                selected === item.id
                  ? 'bg-accent text-accent-foreground'
                  : 'text-muted-foreground hover:bg-accent/50'
              )}
            >
              {item.icon && <item.icon className="h-4 w-4" />}
              {item.label}
              {item.count !== undefined && (
                <span className="ml-auto text-xs">{item.count}</span>
              )}
            </button>
          ))}
        </div>
      ))}
    </div>
  )
}
