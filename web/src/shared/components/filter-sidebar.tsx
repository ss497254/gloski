import { cn } from '@/shared/lib/utils'
import { Button } from '@/ui/button'
import type { LucideIcon } from 'lucide-react'
import { ChevronDown, Filter } from 'lucide-react'
import { useState } from 'react'

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

export function FilterSidebar({ sections, selected, onSelect, allItem, className }: FilterSidebarProps) {
  const [mobileOpen, setMobileOpen] = useState(false)

  // Get selected label for mobile button
  const selectedLabel = selected
    ? sections.flatMap((s) => s.items).find((i) => i.id === selected)?.label
    : allItem?.label || 'All'

  const content = (
    <>
      {/* All items button */}
      {allItem && (
        <button
          onClick={() => {
            onSelect(null)
            setMobileOpen(false)
          }}
          className={cn(
            'w-full flex items-center gap-2 px-3 py-2 rounded-lg text-sm font-medium transition-colors',
            selected === null ? 'bg-accent text-accent-foreground' : 'text-muted-foreground hover:bg-accent/50'
          )}
        >
          {allItem.icon && <allItem.icon className="h-4 w-4" />}
          {allItem.label}
          {allItem.count !== undefined && <span className="ml-auto text-xs">{allItem.count}</span>}
        </button>
      )}

      {/* Sections */}
      {sections.map((section, sectionIndex) => (
        <div key={sectionIndex}>
          {section.title && (
            <div className="pt-4 pb-2 px-3">
              <span className="text-xs font-medium text-muted-foreground uppercase">{section.title}</span>
            </div>
          )}
          {section.items.map((item) => (
            <button
              key={item.id}
              onClick={() => {
                onSelect(item.id)
                setMobileOpen(false)
              }}
              className={cn(
                'w-full flex items-center gap-2 px-3 py-2 rounded-lg text-sm font-medium transition-colors',
                selected === item.id ? 'bg-accent text-accent-foreground' : 'text-muted-foreground hover:bg-accent/50'
              )}
            >
              {item.icon && <item.icon className="h-4 w-4" />}
              {item.label}
              {item.count !== undefined && <span className="ml-auto text-xs">{item.count}</span>}
            </button>
          ))}
        </div>
      ))}
    </>
  )

  return (
    <>
      {/* Mobile toggle button */}
      <div className="md:hidden mb-4">
        <Button
          variant="outline"
          size="sm"
          onClick={() => setMobileOpen(!mobileOpen)}
          className="w-full justify-between"
        >
          <span className="flex items-center gap-2">
            <Filter className="h-4 w-4" />
            {selectedLabel}
          </span>
          <ChevronDown className={cn('h-4 w-4 transition-transform', mobileOpen && 'rotate-180')} />
        </Button>

        {/* Mobile dropdown */}
        {mobileOpen && <div className="mt-2 p-2 rounded-lg border bg-background shadow-lg space-y-1">{content}</div>}
      </div>

      {/* Desktop sidebar */}
      <div className={cn('hidden md:block shrink-0 space-y-1 min-w-52', className)}>{content}</div>
    </>
  )
}
