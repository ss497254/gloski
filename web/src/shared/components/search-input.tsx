import { Search, X } from 'lucide-react'
import {
  InputGroup,
  InputGroupAddon,
  InputGroupButton,
  InputGroupInput,
} from '@/ui/input-group'
import { cn } from '@/shared/lib/utils'

interface SearchInputProps {
  value: string
  onChange: (value: string) => void
  placeholder?: string
  className?: string
  showClear?: boolean
}

export function SearchInput({
  value,
  onChange,
  placeholder = 'Search...',
  className,
  showClear = true,
}: SearchInputProps) {
  return (
    <InputGroup className={cn('max-w-sm', className)}>
      <InputGroupAddon>
        <Search />
      </InputGroupAddon>
      <InputGroupInput placeholder={placeholder} value={value} onChange={(e) => onChange(e.target.value)} />
      {showClear && value && (
        <InputGroupAddon align="inline-end">
          <InputGroupButton onClick={() => onChange('')} aria-label="Clear search">
            <X />
          </InputGroupButton>
        </InputGroupAddon>
      )}
    </InputGroup>
  )
}
