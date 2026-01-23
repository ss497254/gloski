import type React from 'react'
import { NavLink, useParams } from 'react-router-dom'
import { useServersStore } from '@/stores/servers'
import { useSettingsStore } from '@/stores/settings'
import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import {
  LayoutDashboard,
  FolderOpen,
  Terminal,
  ListTodo,
  Search,
  Settings,
  Sun,
  Moon,
  Monitor,
  ChevronLeft,
  ChevronRight,
  Layers,
} from 'lucide-react'
import { cn } from '@/lib/utils'
import { ServerList } from '@/components/servers'

interface LayoutProps {
  children: React.ReactNode
}

export function Layout({ children }: LayoutProps) {
  const { serverId } = useParams()
  const { getServer } = useServersStore()
  const { setTheme, sidebarCollapsed, toggleSidebar } = useSettingsStore()

  const currentServer = serverId ? getServer(serverId) : null

  // Server-scoped navigation items (only shown when viewing a server)
  const serverNavItems = currentServer
    ? [
        { to: `/servers/${serverId}`, icon: LayoutDashboard, label: 'Overview', end: true },
        { to: `/servers/${serverId}/files`, icon: FolderOpen, label: 'Files' },
        { to: `/servers/${serverId}/search`, icon: Search, label: 'Search' },
        { to: `/servers/${serverId}/terminal`, icon: Terminal, label: 'Terminal' },
        { to: `/servers/${serverId}/tasks`, icon: ListTodo, label: 'Tasks' },
      ]
    : []

  return (
    <TooltipProvider delayDuration={0}>
      <div className="h-screen flex bg-background">
        {/* Sidebar */}
        <aside
          className={cn(
            'relative flex flex-col border-r bg-card transition-all duration-300 ease-in-out',
            sidebarCollapsed ? 'w-[68px]' : 'w-64'
          )}
        >
          {/* Logo */}
          <div className="h-16 flex items-center gap-3 px-4 border-b">
            <NavLink to="/" className="flex items-center gap-3">
              <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-primary text-primary-foreground">
                <Layers className="h-5 w-5" />
              </div>
              {!sidebarCollapsed && (
                <div className="flex flex-col">
                  <span className="font-semibold text-lg leading-none">Gloski</span>
                  <span className="text-xs text-muted-foreground">Control Center</span>
                </div>
              )}
            </NavLink>
          </div>

          {/* Main Navigation */}
          <nav className="flex-1 overflow-y-auto py-4 px-3 space-y-6">
            {/* Dashboard Link */}
            {!sidebarCollapsed && (
              <div>
                <Tooltip>
                  <TooltipTrigger asChild>
                    <NavLink
                      to="/"
                      end
                      className={({ isActive }) =>
                        cn(
                          'flex items-center gap-3 rounded-lg px-3 py-2.5 text-sm font-medium transition-all',
                          'hover:bg-accent hover:text-accent-foreground',
                          isActive
                            ? 'bg-primary text-primary-foreground hover:bg-primary/90 hover:text-primary-foreground'
                            : 'text-muted-foreground'
                        )
                      }
                    >
                      <LayoutDashboard className="h-5 w-5 shrink-0" />
                      <span>Dashboard</span>
                    </NavLink>
                  </TooltipTrigger>
                </Tooltip>
              </div>
            )}

            {/* Servers List */}
            {!sidebarCollapsed && <ServerList />}

            {/* Server-scoped Navigation (when viewing a server) */}
            {currentServer && !sidebarCollapsed && (
              <div className="space-y-1">
                <div className="px-2 py-1">
                  <span className="text-xs font-medium text-muted-foreground uppercase tracking-wider">
                    {currentServer.name}
                  </span>
                </div>
                {serverNavItems.map((item) => (
                  <Tooltip key={item.to}>
                    <TooltipTrigger asChild>
                      <NavLink
                        to={item.to}
                        end={item.end}
                        className={({ isActive }) =>
                          cn(
                            'flex items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium transition-all',
                            'hover:bg-accent hover:text-accent-foreground',
                            isActive
                              ? 'bg-accent text-accent-foreground'
                              : 'text-muted-foreground'
                          )
                        }
                      >
                        <item.icon className="h-4 w-4 shrink-0" />
                        <span>{item.label}</span>
                      </NavLink>
                    </TooltipTrigger>
                  </Tooltip>
                ))}
              </div>
            )}

            {/* Collapsed sidebar - show icons only */}
            {sidebarCollapsed && (
              <div className="space-y-1">
                <Tooltip>
                  <TooltipTrigger asChild>
                    <NavLink
                      to="/"
                      end
                      className={({ isActive }) =>
                        cn(
                          'flex items-center justify-center rounded-lg p-2.5 transition-all',
                          'hover:bg-accent hover:text-accent-foreground',
                          isActive
                            ? 'bg-primary text-primary-foreground'
                            : 'text-muted-foreground'
                        )
                      }
                    >
                      <LayoutDashboard className="h-5 w-5" />
                    </NavLink>
                  </TooltipTrigger>
                  <TooltipContent side="right">Dashboard</TooltipContent>
                </Tooltip>

                {serverNavItems.map((item) => (
                  <Tooltip key={item.to}>
                    <TooltipTrigger asChild>
                      <NavLink
                        to={item.to}
                        end={item.end}
                        className={({ isActive }) =>
                          cn(
                            'flex items-center justify-center rounded-lg p-2.5 transition-all',
                            'hover:bg-accent hover:text-accent-foreground',
                            isActive
                              ? 'bg-accent text-accent-foreground'
                              : 'text-muted-foreground'
                          )
                        }
                      >
                        <item.icon className="h-5 w-5" />
                      </NavLink>
                    </TooltipTrigger>
                    <TooltipContent side="right">{item.label}</TooltipContent>
                  </Tooltip>
                ))}
              </div>
            )}
          </nav>

          {/* Footer */}
          <div className="border-t p-3 space-y-2">
            {/* Settings Dropdown */}
            <DropdownMenu>
              <Tooltip>
                <TooltipTrigger asChild>
                  <DropdownMenuTrigger asChild>
                    <Button
                      variant="ghost"
                      className={cn(
                        'w-full justify-start gap-3',
                        sidebarCollapsed && 'justify-center px-2'
                      )}
                    >
                      <Settings className="h-5 w-5" />
                      {!sidebarCollapsed && <span>Settings</span>}
                    </Button>
                  </DropdownMenuTrigger>
                </TooltipTrigger>
                {sidebarCollapsed && (
                  <TooltipContent side="right">Settings</TooltipContent>
                )}
              </Tooltip>
              <DropdownMenuContent
                align={sidebarCollapsed ? 'center' : 'start'}
                className="w-48"
              >
                <DropdownMenuItem onClick={() => setTheme('light')}>
                  <Sun className="h-4 w-4 mr-2" />
                  Light Mode
                </DropdownMenuItem>
                <DropdownMenuItem onClick={() => setTheme('dark')}>
                  <Moon className="h-4 w-4 mr-2" />
                  Dark Mode
                </DropdownMenuItem>
                <DropdownMenuItem onClick={() => setTheme('system')}>
                  <Monitor className="h-4 w-4 mr-2" />
                  System
                </DropdownMenuItem>
                <DropdownMenuSeparator />
                <DropdownMenuItem onClick={toggleSidebar}>
                  {sidebarCollapsed ? (
                    <>
                      <ChevronRight className="h-4 w-4 mr-2" />
                      Expand Sidebar
                    </>
                  ) : (
                    <>
                      <ChevronLeft className="h-4 w-4 mr-2" />
                      Collapse Sidebar
                    </>
                  )}
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          </div>

          {/* Collapse Toggle */}
          <button
            onClick={toggleSidebar}
            className={cn(
              'absolute -right-3 top-20 flex h-6 w-6 items-center justify-center',
              'rounded-full border bg-background shadow-sm',
              'hover:bg-accent transition-colors'
            )}
          >
            {sidebarCollapsed ? (
              <ChevronRight className="h-4 w-4" />
            ) : (
              <ChevronLeft className="h-4 w-4" />
            )}
          </button>
        </aside>

        {/* Main Content */}
        <main className="flex-1 overflow-hidden">{children}</main>
      </div>
    </TooltipProvider>
  )
}
