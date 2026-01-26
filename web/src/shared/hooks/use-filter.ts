import { useState, useMemo, useCallback } from 'react'

interface UseFilterOptions<T> {
  items: T[]
  searchFields?: (keyof T)[]
  searchFn?: (item: T, search: string) => boolean
  filterFn?: (item: T, filter: string | null) => boolean
  sortFn?: (a: T, b: T) => number
}

interface UseFilterReturn<T> {
  // State
  search: string
  filter: string | null

  // Setters
  setSearch: (search: string) => void
  setFilter: (filter: string | null) => void

  // Results
  filteredItems: T[]

  // Helpers
  clearSearch: () => void
  clearFilter: () => void
  clearAll: () => void
}

export function useFilter<T>({
  items,
  searchFields,
  searchFn,
  filterFn,
  sortFn,
}: UseFilterOptions<T>): UseFilterReturn<T> {
  const [search, setSearch] = useState('')
  const [filter, setFilter] = useState<string | null>(null)

  const filteredItems = useMemo(() => {
    let result = [...items]

    // Apply search
    if (search) {
      const searchLower = search.toLowerCase()
      result = result.filter((item) => {
        if (searchFn) {
          return searchFn(item, searchLower)
        }
        if (searchFields) {
          return searchFields.some((field) => {
            const value = item[field]
            if (typeof value === 'string') {
              return value.toLowerCase().includes(searchLower)
            }
            if (Array.isArray(value)) {
              return value.some((v) => typeof v === 'string' && v.toLowerCase().includes(searchLower))
            }
            return false
          })
        }
        return true
      })
    }

    // Apply filter
    if (filter && filterFn) {
      result = result.filter((item) => filterFn(item, filter))
    }

    // Apply sort
    if (sortFn) {
      result.sort(sortFn)
    }

    return result
  }, [items, search, filter, searchFields, searchFn, filterFn, sortFn])

  const clearSearch = useCallback(() => setSearch(''), [])
  const clearFilter = useCallback(() => setFilter(null), [])
  const clearAll = useCallback(() => {
    setSearch('')
    setFilter(null)
  }, [])

  return {
    search,
    filter,
    setSearch,
    setFilter,
    filteredItems,
    clearSearch,
    clearFilter,
    clearAll,
  }
}
