import { Suspense } from 'react'
import { Outlet } from 'react-router-dom'
import { TooltipProvider } from '@/ui/tooltip'
import { Sidebar } from './Sidebar'
import { CommandPalette } from '@/shared/components/CommandPalette'
import { PageLoader } from '@/shared/components'

export function AppLayout() {
  return (
    <TooltipProvider delayDuration={0}>
      <div className="h-screen flex bg-background">
        <Sidebar />
        <main className="flex-1 overflow-hidden">
          <Suspense fallback={<PageLoader />}>
            <Outlet />
          </Suspense>
        </main>
        <CommandPalette />
      </div>
    </TooltipProvider>
  )
}
