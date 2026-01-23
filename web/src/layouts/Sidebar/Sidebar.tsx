import { NavLink, useParams } from 'react-router-dom'
import { NavItem } from './NavItem'
import { ServerNav } from './ServerNav'
import {
  getMainFeatures,
  getWorkspaceFeatures,
  getSettingsFeature,
} from '@/app/feature-registry'
import { useSettingsStore } from '@/features/settings'
import { useMessagesStore } from '@/features/messages'
import { useServersStore } from '@/features/servers'
import { cn } from '@/shared/lib/utils'
import { Layers, ChevronLeft, ChevronRight, Plus, Command } from 'lucide-react'

export function Sidebar() {
  const { serverId } = useParams()
  const { sidebarCollapsed, toggleSidebar } = useSettingsStore()
  const unreadCount = useMessagesStore((s) => s.messages.filter((m) => !m.read).length)
  const servers = useServersStore((s) => s.servers)
  const currentServer = servers.find((s) => s.id === serverId)

  const mainFeatures = getMainFeatures()
  const workspaceFeatures = getWorkspaceFeatures()
  const settingsFeature = getSettingsFeature()

  return (
    <aside
      className={cn(
        'relative flex flex-col border-r bg-muted/30 transition-all duration-200',
        sidebarCollapsed ? 'w-16' : 'w-60'
      )}
    >
      {/* Logo */}
      <div className="h-18 flex items-center justify-between px-3 border-b bg-background/50">
        <NavLink to="/" className="flex items-center gap-2.5">
          <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-primary text-primary-foreground shadow-sm">
            <Layers className="h-4 w-4" />
          </div>
          {!sidebarCollapsed && (
            <span className="font-semibold text-foreground">Gloski</span>
          )}
        </NavLink>
        {!sidebarCollapsed && (
          <button
            onClick={toggleSidebar}
            className="p-1.5 rounded-md hover:bg-muted text-muted-foreground hover:text-foreground transition-colors"
          >
            <ChevronLeft className="h-4 w-4" />
          </button>
        )}
      </div>

      {/* Command Palette Hint */}
      {!sidebarCollapsed && (
        <div className="px-3 py-2">
          <button
            onClick={() => {
              // Trigger command palette
              const event = new KeyboardEvent('keydown', {
                key: 'k',
                metaKey: true,
                bubbles: true,
              })
              document.dispatchEvent(event)
            }}
            className="w-full flex items-center gap-2 px-3 py-2 rounded-lg border border-dashed border-muted-foreground/25 text-muted-foreground hover:border-muted-foreground/50 hover:text-foreground transition-colors text-sm"
          >
            <Command className="h-3.5 w-3.5" />
            <span className="flex-1 text-left">Search...</span>
            <kbd className="text-[10px] bg-muted px-1.5 py-0.5 rounded font-mono">
              ⌘K
            </kbd>
          </button>
        </div>
      )}

      {/* Navigation */}
      <nav className="flex-1 overflow-y-auto px-3 py-2">
        {/* Main Navigation */}
        <div className="space-y-1">
          {mainFeatures.map((feature) => (
            <NavItem
              key={feature.id}
              to={feature.path}
              icon={feature.icon}
              label={feature.name}
              end={feature.path === '/'}
              collapsed={sidebarCollapsed}
            />
          ))}
        </div>

        {/* Workspace Section */}
        <div className="mt-6">
          {!sidebarCollapsed && (
            <div className="px-3 mb-2">
              <span className="text-[11px] font-semibold text-muted-foreground/70 uppercase tracking-wider">
                Workspace
              </span>
            </div>
          )}
          {sidebarCollapsed && <div className="border-t my-3" />}
          <div className="space-y-1">
            {workspaceFeatures.map((feature) => (
              <NavItem
                key={feature.id}
                to={feature.path}
                icon={feature.icon}
                label={feature.name}
                collapsed={sidebarCollapsed}
                badge={feature.id === 'messages' ? unreadCount : undefined}
              />
            ))}
          </div>
        </div>

        {/* Server Section */}
        {currentServer && (
          <div className="mt-6">
            {!sidebarCollapsed && (
              <div className="px-3 mb-2 flex items-center justify-between">
                <span className="text-[11px] font-semibold text-muted-foreground/70 uppercase tracking-wider">
                  Server
                </span>
                <div className="flex items-center gap-1.5">
                  <span
                    className={cn(
                      'h-1.5 w-1.5 rounded-full',
                      currentServer.status === 'online' && 'bg-green-500',
                      currentServer.status === 'connecting' && 'bg-yellow-500',
                      currentServer.status === 'offline' && 'bg-gray-400',
                      currentServer.status === 'unauthorized' && 'bg-red-500'
                    )}
                  />
                  <span className="text-[11px] text-muted-foreground truncate max-w-20">
                    {currentServer.name}
                  </span>
                </div>
              </div>
            )}
            {sidebarCollapsed && <div className="border-t my-3" />}
            <ServerNav collapsed={sidebarCollapsed} />
          </div>
        )}

        {/* Quick Server Access (when no server selected) */}
        {!currentServer && servers.length > 0 && (
          <div className="mt-6">
            {!sidebarCollapsed && (
              <div className="px-3 mb-2 flex items-center justify-between">
                <span className="text-[11px] font-semibold text-muted-foreground/70 uppercase tracking-wider">
                  Servers
                </span>
                <NavLink
                  to="/servers/add"
                  className="p-1 rounded hover:bg-muted text-muted-foreground hover:text-foreground transition-colors"
                >
                  <Plus className="h-3.5 w-3.5" />
                </NavLink>
              </div>
            )}
            {sidebarCollapsed && <div className="border-t my-3" />}
            <div className="space-y-1">
              {servers.slice(0, 4).map((server) => (
                <NavLink
                  key={server.id}
                  to={`/servers/${server.id}`}
                >
                  {({ isActive }) => (
                    <div className={cn(
                      'flex items-center gap-2 rounded-lg px-3 py-2 text-sm transition-colors',
                      'hover:bg-accent hover:text-accent-foreground',
                      sidebarCollapsed && 'justify-center px-2',
                      isActive
                        ? 'bg-accent text-accent-foreground'
                        : 'text-muted-foreground'
                    )}>
                      <span
                        className={cn(
                          'h-2 w-2 rounded-full shrink-0',
                          server.status === 'online' && 'bg-green-500',
                          server.status === 'connecting' && 'bg-yellow-500 animate-pulse',
                          server.status === 'offline' && 'bg-gray-400',
                          server.status === 'unauthorized' && 'bg-red-500'
                        )}
                      />
                      {!sidebarCollapsed && (
                        <span className="truncate">{server.name}</span>
                      )}
                    </div>
                  )}
                </NavLink>
              ))}
              {servers.length > 4 && !sidebarCollapsed && (
                <NavLink
                  to="/servers"
                  className="flex items-center gap-2 rounded-lg px-3 py-2 text-sm text-muted-foreground hover:text-foreground transition-colors"
                >
                  <span className="text-xs">+{servers.length - 4} more</span>
                </NavLink>
              )}
            </div>
          </div>
        )}
      </nav>

      {/* Footer */}
      <div className="border-t p-3 bg-background/30">
        <NavItem
          to={settingsFeature.path}
          icon={settingsFeature.icon}
          label={settingsFeature.name}
          collapsed={sidebarCollapsed}
        />
      </div>

      {/* Collapsed expand button */}
      {sidebarCollapsed && (
        <button
          onClick={toggleSidebar}
          className="absolute -right-3 top-17 flex h-6 w-6 items-center justify-center rounded-full border bg-background shadow-sm hover:bg-accent transition-colors"
        >
          <ChevronRight className="h-3.5 w-3.5" />
        </button>
      )}
    </aside>
  )
}
