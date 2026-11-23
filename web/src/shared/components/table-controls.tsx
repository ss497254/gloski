import { Button } from '@/ui/button'
import {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuLabel,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/ui/dropdown-menu'
import { SlidersHorizontal } from 'lucide-react'

export interface TableOption {
  value: string
  label: string
}

interface TableControlsProps {
  sort: string
  sortOptions: TableOption[]
  onSortChange: (value: string) => void
  visibleFields: string[]
  fieldOptions: TableOption[]
  onToggleField: (value: string) => void
  pageSize: number
  pageSizes?: number[]
  onPageSizeChange: (size: number) => void
}

/** One dropdown for sort, visible columns and page size, using shadcn items. */
export function TableControls({
  sort,
  sortOptions,
  onSortChange,
  visibleFields,
  fieldOptions,
  onToggleField,
  pageSize,
  pageSizes = [10, 25, 50],
  onPageSizeChange,
}: TableControlsProps) {
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button variant="outline" size="sm">
          <SlidersHorizontal className="mr-2 size-3.5" />
          View
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-48">
        <DropdownMenuLabel>Sort by</DropdownMenuLabel>
        <DropdownMenuRadioGroup value={sort} onValueChange={onSortChange}>
          {sortOptions.map((option) => (
            <DropdownMenuRadioItem key={option.value} value={option.value}>
              {option.label}
            </DropdownMenuRadioItem>
          ))}
        </DropdownMenuRadioGroup>
        <DropdownMenuSeparator />
        <DropdownMenuLabel>Fields</DropdownMenuLabel>
        {fieldOptions.map((option) => (
          <DropdownMenuCheckboxItem
            key={option.value}
            checked={visibleFields.includes(option.value)}
            onCheckedChange={() => onToggleField(option.value)}
          >
            {option.label}
          </DropdownMenuCheckboxItem>
        ))}
        <DropdownMenuSeparator />
        <DropdownMenuLabel>Rows per page</DropdownMenuLabel>
        <DropdownMenuRadioGroup value={String(pageSize)} onValueChange={(value) => onPageSizeChange(Number(value))}>
          {pageSizes.map((size) => (
            <DropdownMenuRadioItem key={size} value={String(size)}>
              {size}
            </DropdownMenuRadioItem>
          ))}
        </DropdownMenuRadioGroup>
      </DropdownMenuContent>
    </DropdownMenu>
  )
}
