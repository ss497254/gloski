import { cn } from '@/shared/lib/utils'
import { Button } from '@/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/ui/card'
import { Monitor, Moon, PanelLeft, PanelLeftClose, Sun } from 'lucide-react'
import { useSettings } from '../context'

const themes = [
  { id: 'light', label: 'Light', icon: Sun },
  { id: 'dark', label: 'Dark', icon: Moon },
  { id: 'system', label: 'System', icon: Monitor },
] as const

export function AppearanceSettings() {
  const { theme, setTheme, sidebarCollapsed, toggleSidebar } = useSettings()

  return (
    <>
      <Card>
        <CardHeader>
          <CardTitle>Theme</CardTitle>
          <CardDescription>Select your preferred color theme</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="flex gap-3">
            {themes.map((t) => (
              <button
                key={t.id}
                onClick={() => setTheme(t.id)}
                className={cn(
                  'flex flex-col items-center gap-2 p-4 rounded-lg border-2 transition-colors flex-1',
                  theme === t.id ? 'border-primary bg-accent' : 'border-transparent bg-muted hover:bg-accent/50'
                )}
              >
                <t.icon className="h-6 w-6" />
                <span className="text-sm font-medium">{t.label}</span>
              </button>
            ))}
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Sidebar</CardTitle>
          <CardDescription>Customize the sidebar appearance</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-3">
              {sidebarCollapsed ? (
                <PanelLeftClose className="h-5 w-5 text-muted-foreground" />
              ) : (
                <PanelLeft className="h-5 w-5 text-muted-foreground" />
              )}
              <div>
                <p className="font-medium">Collapsed Sidebar</p>
                <p className="text-sm text-muted-foreground">Show icons only in the sidebar</p>
              </div>
            </div>
            <Button variant={sidebarCollapsed ? 'default' : 'outline'} onClick={toggleSidebar}>
              {sidebarCollapsed ? 'Expanded' : 'Collapse'}
            </Button>
          </div>
        </CardContent>
      </Card>
    </>
  )
}
