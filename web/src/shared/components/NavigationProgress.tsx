import { useEffect, useState } from 'react'
import { useNavigation } from 'react-router-dom'
import { cn } from '@/shared/lib/utils'

export function NavigationProgress() {
  const navigation = useNavigation()
  const [progress, setProgress] = useState(0)

  const isNavigating = navigation.state === 'loading'
  const visible = isNavigating || progress > 0

  useEffect(() => {
    if (!isNavigating) return
    const timers = [
      setTimeout(() => setProgress(10), 0),
      setTimeout(() => setProgress(30), 100),
      setTimeout(() => setProgress(60), 300),
      setTimeout(() => setProgress(80), 600),
    ]
    return () => timers.forEach(clearTimeout)
  }, [isNavigating])

  useEffect(() => {
    if (isNavigating || progress === 0) return
    const complete = progress < 100 ? setTimeout(() => setProgress(100), 0) : undefined
    const reset = setTimeout(() => setProgress(0), 350)
    return () => {
      if (complete) clearTimeout(complete)
      clearTimeout(reset)
    }
  }, [isNavigating, progress])

  if (!visible) return null

  return (
    <div className="fixed top-0 left-0 right-0 z-50 h-0.25 bg-transparent">
      <div
        className={cn(
          'h-full bg-primary transition-all duration-300 ease-out shadow-sm',
          progress === 100 && 'transition-opacity duration-300 opacity-0'
        )}
        style={{ width: `${progress}%` }}
      />
    </div>
  )
}
