import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/ui/card'
import { Label } from '@/ui/label'
import { Switch } from '@/ui/switch'
import { ToggleGroup, ToggleGroupItem } from '@/ui/toggle-group'
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
          <ToggleGroup
            type="single"
            variant="outline"
            value={theme}
            onValueChange={(value) => {
              if (value) setTheme(value as typeof theme)
            }}
            className="w-full gap-3"
          >
            {themes.map((t) => (
              <ToggleGroupItem
                key={t.id}
                value={t.id}
                aria-label={`${t.label} theme`}
                className="flex h-auto flex-1 flex-col gap-2 rounded-lg px-4 py-4 data-[state=on]:border-primary data-[state=on]:bg-accent data-[state=on]:text-accent-foreground"
              >
                <t.icon className="h-6 w-6" />
                <span className="text-sm font-medium">{t.label}</span>
              </ToggleGroupItem>
            ))}
          </ToggleGroup>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Sidebar</CardTitle>
          <CardDescription>Customize the sidebar appearance</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="flex items-center justify-between gap-4">
            <div className="flex items-center gap-3">
              {sidebarCollapsed ? (
                <PanelLeftClose className="h-5 w-5 text-muted-foreground" />
              ) : (
                <PanelLeft className="h-5 w-5 text-muted-foreground" />
              )}
              <div className="space-y-0.5">
                <Label htmlFor="sidebar-collapsed" className="font-medium">
                  Collapsed Sidebar
                </Label>
                <p className="text-sm text-muted-foreground">Show icons only in the sidebar</p>
              </div>
            </div>
            <Switch id="sidebar-collapsed" checked={sidebarCollapsed} onCheckedChange={() => toggleSidebar()} />
          </div>
        </CardContent>
      </Card>
    </>
  )
}
