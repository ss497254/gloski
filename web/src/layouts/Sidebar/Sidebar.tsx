import { mainNavItems } from '@/app/navigation'
import { useSettingsStore } from '@/features/dashboard/stores/settings'
import type { Server } from '@/shared/lib/server'
import { cn } from '@/shared/lib/utils'
import { useTower } from '@/shared/tower'
import { useServersStore } from '@/shared/store/servers'
import { ChevronLeft, ChevronRight, Layers, Moon, Plus, Sun, X } from 'lucide-react'
import { useEffect } from 'react'
import { NavLink, useLocation, useParams } from 'react-router-dom'
import { useSnapshot } from 'valtio'
import { NavItem } from './NavItem'
import { ServerNav } from './ServerNav'

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

function CurrentServerBadge({ server }: { server: Server }) {
  const snap = useSnapshot(server)
  return (
    <div className="flex items-center gap-1.5">
      <span className={cn('h-1.5 w-1.5 rounded-full', getStatusColor(snap.getStatus()))} />
      <span className="text-[11px] text-muted-foreground truncate max-w-20">{snap.name}</span>
    </div>
  )
}

function SidebarServerLink({ server, collapsed }: { server: Server; collapsed: boolean }) {
  const snap = useSnapshot(server)
  const status = snap.getStatus()
  const isManaged = snap.managedBy === 'tower'
  const to = isManaged ? `/fleet/${snap.id.replace(/^tower-/, '')}` : `/servers/${snap.id}`

  return (
    <NavLink to={to}>
      {({ isActive }) => (
        <div
          className={cn(
            'flex items-center gap-2 rounded-lg px-3 py-2 text-sm transition-colors',
            'hover:bg-accent hover:text-accent-foreground',
            collapsed && 'justify-center px-2',
            isActive ? 'bg-accent text-accent-foreground' : 'text-muted-foreground'
          )}
        >
          <span className={cn('h-2 w-2 rounded-full shrink-0', getStatusColor(status))} />
          {!collapsed && <span className="truncate">{snap.name}</span>}
        </div>
      )}
    </NavLink>
  )
}

function ThemeToggle({ collapsed }: { collapsed: boolean }) {
  const theme = useSettingsStore((s) => s.theme)
  const toggleTheme = useSettingsStore((s) => s.toggleTheme)
  const isDark = theme === 'dark' || (theme === 'system' && window.matchMedia('(prefers-color-scheme: dark)').matches)

  return (
    <button
      onClick={toggleTheme}
      className={cn(
        'flex items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium text-muted-foreground transition-colors hover:bg-accent hover:text-accent-foreground',
        collapsed && 'justify-center px-2'
      )}
      title={isDark ? 'Switch to light theme' : 'Switch to dark theme'}
    >
      {isDark ? <Sun className="h-4 w-4 shrink-0" /> : <Moon className="h-4 w-4 shrink-0" />}
      {!collapsed && <span>{isDark ? 'Light mode' : 'Dark mode'}</span>}
    </button>
  )
}

interface SidebarContentProps {
  collapsed: boolean
}

function SidebarContent({ collapsed }: SidebarContentProps) {
  const { serverId } = useParams()
  const { mode } = useTower()
  const servers = useServersStore((s) => s.servers)
  const currentServer = servers.find((s) => s.id === serverId)
  const tower = mode === 'tower'

  const navItems = tower
    ? mainNavItems.filter((item) => item.id === 'fleet')
    : mainNavItems.filter((item) => item.id === 'dashboard' || item.id === 'servers')
  const settingsItem = mainNavItems.find((item) => item.id === 'settings')!

  return (
    <>
      <nav className="flex-1 overflow-y-auto px-3 py-3">
        <div className="space-y-1">
          {navItems.map((item) => (
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

        {currentServer && (
          <div className="mt-6">
            {!collapsed && (
              <div className="px-3 mb-2 flex items-center justify-between">
                <span className="text-[11px] font-semibold text-muted-foreground/70 uppercase tracking-wider">
                  {currentServer.managedBy === 'tower' ? 'Device' : 'Server'}
                </span>
                <CurrentServerBadge server={currentServer} />
              </div>
            )}
            {collapsed && <div className="border-t my-3" />}
            <ServerNav collapsed={collapsed} />
          </div>
        )}

        {!currentServer && servers.length > 0 && (
          <div className="mt-6">
            {!collapsed && (
              <div className="px-3 mb-2 flex items-center justify-between">
                <span className="text-[11px] font-semibold text-muted-foreground/70 uppercase tracking-wider">
                  {tower ? 'Devices' : 'Servers'}
                </span>
                {!tower && (
                  <NavLink
                    to="/servers/add"
                    className="p-1 rounded hover:bg-muted text-muted-foreground hover:text-foreground transition-colors"
                  >
                    <Plus className="h-3.5 w-3.5" />
                  </NavLink>
                )}
              </div>
            )}
            {collapsed && <div className="border-t my-3" />}
            <div className="space-y-1">
              {servers.slice(0, 5).map((server) => (
                <SidebarServerLink key={server.id} server={server} collapsed={collapsed} />
              ))}
              {servers.length > 5 && !collapsed && (
                <NavLink
                  to={tower ? '/fleet' : '/servers'}
                  className="flex items-center gap-2 rounded-lg px-3 py-2 text-sm text-muted-foreground hover:text-foreground transition-colors"
                >
                  <span className="text-xs">+{servers.length - 5} more</span>
                </NavLink>
              )}
            </div>
          </div>
        )}
      </nav>

      <div className="border-t p-3 bg-background/30 space-y-1">
        <ThemeToggle collapsed={collapsed} />
        <NavItem to={settingsItem.path} icon={settingsItem.icon} label={settingsItem.name} collapsed={collapsed} />
      </div>
    </>
  )
}

interface SidebarProps {
  className?: string
}

export function Sidebar({ className }: SidebarProps) {
  const sidebarCollapsed = useSettingsStore((s) => s.sidebarCollapsed)
  const toggleSidebar = useSettingsStore((s) => s.toggleSidebar)
  const { mode } = useTower()

  return (
    <aside
      className={cn(
        'relative flex flex-col border-r border-border bg-sidebar text-sidebar-foreground transition-all duration-200',
        sidebarCollapsed ? 'w-16' : 'w-60',
        className
      )}
    >
      <div className="h-18 flex items-center justify-between px-3 border-b border-border">
        <NavLink to={mode === 'tower' ? '/fleet' : '/'} className="flex items-center gap-2.5">
          <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-primary text-primary-foreground">
            <Layers className="h-4 w-4" />
          </div>
          {!sidebarCollapsed && (
            <div className="flex flex-col">
              <span className="font-semibold leading-none">Gloski</span>
              <span className="text-[10px] text-muted-foreground leading-none mt-0.5">
                {mode === 'tower' ? 'Control tower' : mode === 'loading' ? 'Connecting…' : 'Server management'}
              </span>
            </div>
          )}
        </NavLink>
        {!sidebarCollapsed && (
          <button
            onClick={toggleSidebar}
            className="p-1.5 rounded-md hover:bg-accent text-muted-foreground hover:text-foreground transition-colors"
          >
            <ChevronLeft className="h-4 w-4" />
          </button>
        )}
      </div>

      <SidebarContent collapsed={sidebarCollapsed} />

      {sidebarCollapsed && (
        <button
          onClick={toggleSidebar}
          className="absolute -right-3 top-17 flex h-6 w-6 items-center justify-center rounded-full border border-border bg-background shadow-sm hover:bg-accent transition-colors"
        >
          <ChevronRight className="h-3.5 w-3.5" />
        </button>
      )}
    </aside>
  )
}

export function MobileSidebar() {
  const location = useLocation()
  const mobileSidebarOpen = useSettingsStore((s) => s.mobileSidebarOpen)
  const setMobileSidebarOpen = useSettingsStore((s) => s.setMobileSidebarOpen)
  const { mode } = useTower()

  useEffect(() => {
    setMobileSidebarOpen(false)
  }, [location.pathname, setMobileSidebarOpen])

  useEffect(() => {
    const handleEscape = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && mobileSidebarOpen) setMobileSidebarOpen(false)
    }
    document.addEventListener('keydown', handleEscape)
    return () => document.removeEventListener('keydown', handleEscape)
  }, [mobileSidebarOpen, setMobileSidebarOpen])

  useEffect(() => {
    document.body.style.overflow = mobileSidebarOpen ? 'hidden' : ''
    return () => {
      document.body.style.overflow = ''
    }
  }, [mobileSidebarOpen])

  if (!mobileSidebarOpen) return null

  return (
    <div className="md:hidden fixed inset-0 z-50">
      <div className="absolute inset-0 bg-background/80 backdrop-blur-sm" onClick={() => setMobileSidebarOpen(false)} />
      <aside className="absolute inset-y-0 left-0 w-72 max-w-[85vw] flex flex-col border-r border-border bg-sidebar text-sidebar-foreground shadow-xl animate-in slide-in-from-left duration-200">
        <div className="h-14 flex items-center justify-between px-4 border-b border-border">
          <NavLink to={mode === 'tower' ? '/fleet' : '/'} className="flex items-center gap-2.5">
            <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-primary text-primary-foreground">
              <Layers className="h-4 w-4" />
            </div>
            <div className="flex flex-col">
              <span className="font-semibold leading-none">Gloski</span>
              <span className="text-[10px] text-muted-foreground leading-none mt-0.5">Control tower</span>
            </div>
          </NavLink>
          <button
            onClick={() => setMobileSidebarOpen(false)}
            className="p-2 rounded-md hover:bg-accent text-muted-foreground hover:text-foreground transition-colors"
            aria-label="Close menu"
          >
            <X className="h-5 w-5" />
          </button>
        </div>
        <SidebarContent collapsed={false} />
      </aside>
    </div>
  )
}
