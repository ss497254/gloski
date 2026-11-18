import { useSettingsStore } from '@/features/dashboard/stores/settings'
import { NavigationProgress, PageLoader } from '@/shared/components'
import { useTower } from '@/shared/tower'
import { TooltipProvider } from '@/ui/tooltip'
import { Layers, Menu } from 'lucide-react'
import { Suspense } from 'react'
import { Outlet } from 'react-router-dom'
import { MobileSidebar, Sidebar } from './Sidebar'

export function AppLayout() {
  const { toggleMobileSidebar } = useSettingsStore()
  const { mode } = useTower()

  return (
    <TooltipProvider delayDuration={0}>
      <NavigationProgress />
      <div className="h-screen flex flex-col md:flex-row bg-background">
        {/* Mobile Header */}
        <header className="md:hidden flex items-center justify-between px-4 py-3 border-b bg-background shrink-0">
          <div className="flex items-center gap-2.5">
            <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-primary text-primary-foreground">
              <Layers className="h-4 w-4" />
            </div>
            <div className="flex flex-col">
              <span className="font-semibold text-foreground leading-none">Gloski</span>
              <span className="text-[10px] text-muted-foreground leading-none mt-0.5">
                {mode === 'tower' ? 'Control tower' : mode === 'loading' ? 'Connecting…' : 'Server management'}
              </span>
            </div>
          </div>
          <button
            onClick={toggleMobileSidebar}
            className="p-2 rounded-md hover:bg-muted text-muted-foreground hover:text-foreground transition-colors"
            aria-label="Open menu"
          >
            <Menu className="h-5 w-5" />
          </button>
        </header>

        <Sidebar className="hidden md:flex" />
        <MobileSidebar />

        <main className="flex-1 overflow-hidden">
          <Suspense fallback={<PageLoader />}>
            <Outlet />
          </Suspense>
        </main>
      </div>
    </TooltipProvider>
  )
}
