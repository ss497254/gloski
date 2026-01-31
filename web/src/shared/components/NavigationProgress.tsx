import { useEffect, useState } from 'react'
import { useNavigation } from 'react-router-dom'
import { cn } from '@/shared/lib/utils'

export function NavigationProgress() {
  const navigation = useNavigation()
  const [progress, setProgress] = useState(0)
  const [visible, setVisible] = useState(false)

  const isNavigating = navigation.state === 'loading'

  useEffect(() => {
    if (isNavigating) {
      setVisible(true)
      setProgress(0)

      // Simulate progress
      const timer1 = setTimeout(() => setProgress(30), 100)
      const timer2 = setTimeout(() => setProgress(60), 300)
      const timer3 = setTimeout(() => setProgress(80), 600)

      return () => {
        clearTimeout(timer1)
        clearTimeout(timer2)
        clearTimeout(timer3)
      }
    } else if (visible) {
      // Complete the progress
      setProgress(100)

      // Hide after animation completes
      const timer = setTimeout(() => {
        setVisible(false)
        setProgress(0)
      }, 300)

      return () => clearTimeout(timer)
    }
  }, [isNavigating, visible])

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
