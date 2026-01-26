import { Suspense } from 'react'
import { Outlet } from 'react-router-dom'
import { TooltipProvider } from '@/ui/tooltip'
import { Sidebar, MobileSidebar } from './Sidebar'
import { CommandPalette } from '@/shared/components/CommandPalette'
import { PageLoader } from '@/shared/components'
import { useSettingsStore } from '@/features/settings'
import { Layers, Menu } from 'lucide-react'

export function AppLayout() {
  const { toggleMobileSidebar } = useSettingsStore()

  return (
    <TooltipProvider delayDuration={0}>
      <div className="h-screen flex flex-col md:flex-row bg-background">
        {/* Mobile Header */}
        <header className="md:hidden flex items-center justify-between px-4 py-3 border-b bg-background/95 backdrop-blur supports-backdrop-filter:bg-background/60 shrink-0">
          <div className="flex items-center gap-2.5">
            <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-primary text-primary-foreground shadow-sm">
              <Layers className="h-4 w-4" />
            </div>
            <span className="font-semibold text-foreground">Gloski</span>
          </div>
          <button
            onClick={toggleMobileSidebar}
            className="p-2 rounded-md hover:bg-muted text-muted-foreground hover:text-foreground transition-colors"
            aria-label="Open menu"
          >
            <Menu className="h-5 w-5" />
          </button>
        </header>

        {/* Desktop Sidebar */}
        <Sidebar className="hidden md:flex" />

        {/* Mobile Sidebar Drawer */}
        <MobileSidebar />

        {/* Main Content */}
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
