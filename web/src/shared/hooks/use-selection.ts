import { useState, useCallback, useMemo } from 'react'

interface UseSelectionOptions<T> {
  items: T[]
  getId: (item: T) => string
  onSelect?: (item: T | null) => void
}

interface UseSelectionReturn<T> {
  selectedId: string | null
  selectedItem: T | null
  select: (id: string | null) => void
  selectItem: (item: T | null) => void
  clear: () => void
  isSelected: (id: string) => boolean
}

export function useSelection<T>({
  items,
  getId,
  onSelect,
}: UseSelectionOptions<T>): UseSelectionReturn<T> {
  const [selectedId, setSelectedId] = useState<string | null>(null)

  const selectedItem = useMemo(() => {
    if (!selectedId) return null
    return items.find((item) => getId(item) === selectedId) || null
  }, [items, selectedId, getId])

  const select = useCallback(
    (id: string | null) => {
      setSelectedId(id)
      const item = id ? items.find((i) => getId(i) === id) || null : null
      onSelect?.(item)
    },
    [items, getId, onSelect]
  )

  const selectItem = useCallback(
    (item: T | null) => {
      const id = item ? getId(item) : null
      setSelectedId(id)
      onSelect?.(item)
    },
    [getId, onSelect]
  )

  const clear = useCallback(() => {
    setSelectedId(null)
    onSelect?.(null)
  }, [onSelect])

  const isSelected = useCallback(
    (id: string) => selectedId === id,
    [selectedId]
  )

  return {
    selectedId,
    selectedItem,
    select,
    selectItem,
    clear,
    isSelected,
  }
}
