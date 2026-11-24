import { devicePath, mainNavItems, settingsNavItem } from '@/app/navigation'
import { isDarkTheme, useSettingsStore } from '@/shared/store/settings'
import { cn } from '@/shared/lib/utils'
import { getSelfTest, useTower, type SelfTestResult } from '@/shared/tower'
import type { FleetDevice } from '@/shared/tower/types'
import { Activity, ChevronLeft, ChevronRight, Layers, Moon, Sun, X } from 'lucide-react'
import { useEffect, useState, type ReactNode } from 'react'
import { NavLink, useLocation, useParams } from 'react-router-dom'
import { Button } from '@/ui/button'
import { Dialog, DialogContent, DialogTitle } from '@/ui/dialog'
import { NavItem } from './NavItem'
import { DeviceNav } from './DeviceNav'
import { masterHealthBadge } from './sidebar-utils'

function getStatusColor(status: string) {
  switch (status) {
    case 'online':
      return 'bg-success'
    case 'connecting':
      return 'bg-warning animate-pulse'
    case 'offline':
      return 'bg-muted-foreground/40'
    case 'unauthorized':
      return 'bg-destructive'
    default:
      return 'bg-muted-foreground/40'
  }
}

/** Small-caps label with a hairline, the way SMDWin groups its navigation. */
function SectionLabel({ children, collapsed }: { children: ReactNode; collapsed: boolean }) {
  if (collapsed) return <div className="my-3 border-t border-border/70" />
  return (
    <div className="mb-1.5 mt-1 flex items-center gap-2 px-3">
      <span className="text-[10px] font-semibold uppercase tracking-[0.14em] text-muted-foreground/70">
        {children}
      </span>
      <span className="h-px flex-1 bg-border/70" />
    </div>
  )
}

/** One shared result for the desktop and mobile sidebar instances. */
let cachedHealth: SelfTestResult | null = null
let pendingHealth: Promise<SelfTestResult | null> | null = null

function fetchHealth(): Promise<SelfTestResult | null> {
  if (cachedHealth) return Promise.resolve(cachedHealth)
  if (!pendingHealth) {
    pendingHealth = getSelfTest()
      .then((result) => {
        cachedHealth = result
        return result
      })
      .catch(() => null)
      .finally(() => {
        pendingHealth = null
      })
  }
  return pendingHealth
}

function useMasterHealthBadge() {
  const [result, setResult] = useState<SelfTestResult | null>(cachedHealth)
  useEffect(() => {
    let active = true
    void fetchHealth().then((value) => {
      if (active && value) setResult(value)
    })
    return () => {
      active = false
    }
  }, [])
  return masterHealthBadge(result)
}

function SidebarDeviceLink({ device, collapsed }: { device: FleetDevice; collapsed: boolean }) {
  return (
    <NavLink to={devicePath(device.id)}>
      {({ isActive }) => (
        <div
          className={cn(
            'flex items-center gap-2 rounded-lg px-3 py-2 text-sm transition-colors',
            'hover:bg-accent hover:text-accent-foreground',
            collapsed && 'justify-center px-2',
            isActive ? 'bg-accent text-accent-foreground' : 'text-muted-foreground'
          )}
        >
          <span className={cn('h-2 w-2 rounded-full shrink-0', getStatusColor(device.status))} />
          {!collapsed && <span className="truncate">{device.name}</span>}
        </div>
      )}
    </NavLink>
  )
}

function ThemeToggle({ collapsed }: { collapsed: boolean }) {
  const theme = useSettingsStore((s) => s.theme)
  const toggleTheme = useSettingsStore((s) => s.toggleTheme)
  const isDark = isDarkTheme(theme)

  return (
    <Button
      variant="ghost"
      onClick={toggleTheme}
      className={cn(
        'h-auto w-full justify-start gap-3 px-3 py-2 text-sm font-medium text-muted-foreground hover:text-accent-foreground',
        collapsed && 'justify-center px-2'
      )}
      title={isDark ? 'Switch to light theme' : 'Switch to dark theme'}
    >
      {isDark ? <Sun className="h-4 w-4 shrink-0" /> : <Moon className="h-4 w-4 shrink-0" />}
      {!collapsed && <span>{isDark ? 'Light mode' : 'Dark mode'}</span>}
    </Button>
  )
}

function SidebarContent({ collapsed }: { collapsed: boolean }) {
  const { deviceId } = useParams()
  const { devices } = useTower()
  const currentDevice = devices.find((device) => device.id === deviceId)
  const healthBadge = useMasterHealthBadge()

  return (
    <>
      <nav className="flex-1 overflow-y-auto px-3 py-3">
        <SectionLabel collapsed={collapsed}>Fleet</SectionLabel>
        <div className="space-y-1">
          {mainNavItems.map((item) => (
            <NavItem
              key={item.id}
              to={item.path}
              icon={item.icon}
              label={item.name}
              end={item.path === '/'}
              collapsed={collapsed}
            />
          ))}
        </div>

        {currentDevice && (
          <div className="mt-5">
            <SectionLabel collapsed={collapsed}>This device</SectionLabel>
            {!collapsed && (
              <p className="mb-1 truncate px-3 text-[11px] text-muted-foreground/80">{currentDevice.name}</p>
            )}
            <DeviceNav collapsed={collapsed} />
          </div>
        )}

        {!currentDevice && devices.length > 0 && (
          <div className="mt-5">
            <SectionLabel collapsed={collapsed}>Devices</SectionLabel>
            <div className="space-y-1">
              {devices.slice(0, 5).map((device) => (
                <SidebarDeviceLink key={device.id} device={device} collapsed={collapsed} />
              ))}
              {devices.length > 5 && !collapsed && (
                <NavLink
                  to="/devices"
                  className="flex items-center gap-2 rounded-lg px-3 py-2 text-sm text-muted-foreground hover:text-foreground transition-colors"
                >
                  <span className="text-xs">+{devices.length - 5} more</span>
                </NavLink>
              )}
            </div>
          </div>
        )}
      </nav>

      <div className="border-t p-3 bg-background/30">
        <SectionLabel collapsed={collapsed}>Master</SectionLabel>
        <div className="space-y-1">
          <NavItem
            to="/settings?tab=health"
            icon={Activity}
            label="Master health"
            collapsed={collapsed}
            badge={healthBadge?.count ?? null}
            badgeTone={healthBadge?.tone}
          />
          <ThemeToggle collapsed={collapsed} />
          <NavItem
            to={settingsNavItem.path}
            icon={settingsNavItem.icon}
            label={settingsNavItem.name}
            collapsed={collapsed}
          />
        </div>
      </div>
    </>
  )
}

function Brand({ collapsed }: { collapsed: boolean }) {
  const { mode } = useTower()
  return (
    <NavLink to="/" className="flex items-center gap-2.5">
      <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-primary text-primary-foreground">
        <Layers className="h-4 w-4" />
      </div>
      {!collapsed && (
        <div className="flex flex-col">
          <span className="font-semibold leading-none">Gloski</span>
          <span className="text-[10px] text-muted-foreground leading-none mt-0.5">
            {mode === 'loading' ? 'Connecting…' : 'Control tower'}
          </span>
        </div>
      )}
    </NavLink>
  )
}

export function Sidebar({ className }: { className?: string }) {
  const sidebarCollapsed = useSettingsStore((s) => s.sidebarCollapsed)
  const toggleSidebar = useSettingsStore((s) => s.toggleSidebar)

  return (
    <aside
      className={cn(
        'relative flex flex-col border-r border-border bg-sidebar text-sidebar-foreground transition-all duration-200',
        sidebarCollapsed ? 'w-16' : 'w-60',
        className
      )}
    >
      <div className="h-18 flex items-center justify-between px-3 border-b border-border">
        <Brand collapsed={sidebarCollapsed} />
        {!sidebarCollapsed && (
          <Button variant="ghost" size="icon-sm" onClick={toggleSidebar} aria-label="Collapse sidebar">
            <ChevronLeft className="h-4 w-4" />
          </Button>
        )}
      </div>

      <SidebarContent collapsed={sidebarCollapsed} />

      {sidebarCollapsed && (
        <Button
          variant="outline"
          size="icon-sm"
          onClick={toggleSidebar}
          aria-label="Expand sidebar"
          className="absolute -right-3 top-6 size-6 rounded-full bg-background shadow-sm"
        >
          <ChevronRight className="h-3.5 w-3.5" />
        </Button>
      )}
    </aside>
  )
}

export function MobileSidebar() {
  const location = useLocation()
  const mobileSidebarOpen = useSettingsStore((s) => s.mobileSidebarOpen)
  const setMobileSidebarOpen = useSettingsStore((s) => s.setMobileSidebarOpen)

  useEffect(() => {
    setMobileSidebarOpen(false)
  }, [location.pathname, setMobileSidebarOpen])

  useEffect(() => {
    document.body.style.overflow = mobileSidebarOpen ? 'hidden' : ''
    return () => {
      document.body.style.overflow = ''
    }
  }, [mobileSidebarOpen])

  return (
    <Dialog open={mobileSidebarOpen} onOpenChange={setMobileSidebarOpen}>
      <DialogContent
        showCloseButton={false}
        aria-describedby={undefined}
        onCloseAutoFocus={(event) => {
          event.preventDefault()
          document.getElementById('app-mobile-menu-trigger')?.focus()
        }}
        className="fixed inset-y-0 left-0 top-0 flex h-dvh w-72 max-w-[85vw] translate-x-0 translate-y-0 flex-col gap-0 rounded-none border-y-0 border-l-0 bg-sidebar p-0 text-sidebar-foreground shadow-xl md:hidden"
      >
        <div className="flex h-14 items-center justify-between border-b border-border px-4">
          <DialogTitle className="sr-only">Main navigation</DialogTitle>
          <Brand collapsed={false} />
          <Button variant="ghost" size="icon-sm" onClick={() => setMobileSidebarOpen(false)} aria-label="Close menu">
            <X className="h-5 w-5" />
          </Button>
        </div>
        <SidebarContent collapsed={false} />
      </DialogContent>
    </Dialog>
  )
}
