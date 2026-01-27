import { useCallback, useState } from 'react'

interface UseDialogOptions<T> {
  onSubmit?: (data: T, isEditing: boolean) => void
  initialValues?: Partial<T>
}

interface UseDialogReturn<T> {
  // State
  isOpen: boolean
  isEditing: boolean
  editingItem: T | null

  // Actions
  open: (item?: T) => void
  close: () => void
  submit: (data: T) => void
}

export function useDialog<T extends { id: string }>({ onSubmit }: UseDialogOptions<T> = {}): UseDialogReturn<T> {
  const [isOpen, setIsOpen] = useState(false)
  const [editingItem, setEditingItem] = useState<T | null>(null)

  const open = useCallback((item?: T) => {
    setEditingItem(item || null)
    setIsOpen(true)
  }, [])

  const close = useCallback(() => {
    setIsOpen(false)
    setEditingItem(null)
  }, [])

  const submit = useCallback(
    (data: T) => {
      onSubmit?.(data, !!editingItem)
      close()
    },
    [editingItem, onSubmit, close]
  )

  return {
    isOpen,
    isEditing: !!editingItem,
    editingItem,
    open,
    close,
    submit,
  }
}
