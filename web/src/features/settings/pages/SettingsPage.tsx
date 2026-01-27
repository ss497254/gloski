import { useState } from 'react'
import { Button } from '@/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/ui/card'
import { Badge } from '@/ui/badge'
import { Separator } from '@/ui/separator'
import { PageLayout } from '@/layouts'
import { useSettingsStore } from '../stores/settings'
import { useBookmarksStore } from '@/features/bookmarks'
import { useMessagesStore } from '@/features/messages'
import { useNotesStore } from '@/features/notes'
import { useSnippetsStore } from '@/features/snippets'
import { useActivityStore } from '@/features/activity'
import { useServersStore } from '@/features/servers'
import { cn } from '@/shared/lib/utils'
import {
  Sun,
  Moon,
  Monitor,
  Palette,
  Keyboard,
  Database,
  Info,
  Download,
  Upload,
  Trash2,
  Github,
  ExternalLink,
  PanelLeftClose,
  PanelLeft,
} from 'lucide-react'
import { toast } from 'sonner'

type SettingsTab = 'appearance' | 'shortcuts' | 'data' | 'about'

export function SettingsPage() {
  const [activeTab, setActiveTab] = useState<SettingsTab>('appearance')
  const { theme, setTheme, sidebarCollapsed, toggleSidebar } = useSettingsStore()

  const tabs: {
    id: SettingsTab
    label: string
    icon: React.ComponentType<{ className?: string }>
  }[] = [
    { id: 'appearance', label: 'Appearance', icon: Palette },
    { id: 'shortcuts', label: 'Keyboard Shortcuts', icon: Keyboard },
    { id: 'data', label: 'Data', icon: Database },
    { id: 'about', label: 'About', icon: Info },
  ]

  return (
    <PageLayout title="Settings" description="Manage your preferences">
      <div className="flex gap-8 max-w-4xl">
        {/* Sidebar */}
        <div className="w-48 shrink-0 space-y-1">
          {tabs.map((tab) => (
            <button
              key={tab.id}
              onClick={() => setActiveTab(tab.id)}
              className={cn(
                'w-full flex items-center gap-3 px-3 py-2 rounded-lg text-sm font-medium transition-colors',
                activeTab === tab.id ? 'bg-accent text-accent-foreground' : 'text-muted-foreground hover:bg-accent/50'
              )}
            >
              <tab.icon className="h-4 w-4" />
              {tab.label}
            </button>
          ))}
        </div>

        {/* Content */}
        <div className="flex-1 space-y-6">
          {activeTab === 'appearance' && (
            <AppearanceSettings
              theme={theme}
              setTheme={setTheme}
              sidebarCollapsed={sidebarCollapsed}
              toggleSidebar={toggleSidebar}
            />
          )}
          {activeTab === 'shortcuts' && <ShortcutsSettings />}
          {activeTab === 'data' && <DataSettings />}
          {activeTab === 'about' && <AboutSettings />}
        </div>
      </div>
    </PageLayout>
  )
}

function AppearanceSettings({
  theme,
  setTheme,
  sidebarCollapsed,
  toggleSidebar,
}: {
  theme: string
  setTheme: (t: 'light' | 'dark' | 'system') => void
  sidebarCollapsed: boolean
  toggleSidebar: () => void
}) {
  const themes = [
    { id: 'light', label: 'Light', icon: Sun },
    { id: 'dark', label: 'Dark', icon: Moon },
    { id: 'system', label: 'System', icon: Monitor },
  ] as const

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

function ShortcutsSettings() {
  const shortcuts = [
    { keys: ['⌘', 'K'], description: 'Open command palette' },
    { keys: ['⌘', 'B'], description: 'Toggle sidebar' },
    { keys: ['G', 'D'], description: 'Go to Dashboard' },
    { keys: ['G', 'S'], description: 'Go to Servers' },
    { keys: ['G', 'B'], description: 'Go to Bookmarks' },
    { keys: ['G', 'N'], description: 'Go to Notes' },
    { keys: ['G', 'M'], description: 'Go to Messages' },
    { keys: ['G', 'C'], description: 'Go to Snippets' },
    { keys: ['G', 'A'], description: 'Go to Activity' },
    { keys: ['G', ','], description: 'Go to Settings' },
  ]

  return (
    <Card>
      <CardHeader>
        <CardTitle>Keyboard Shortcuts</CardTitle>
        <CardDescription>Quick access keys for common actions</CardDescription>
      </CardHeader>
      <CardContent>
        <div className="space-y-3">
          {shortcuts.map((shortcut, i) => (
            <div key={i} className="flex items-center justify-between py-2 border-b last:border-0">
              <span className="text-sm">{shortcut.description}</span>
              <div className="flex items-center gap-1">
                {shortcut.keys.map((key, j) => (
                  <kbd key={j} className="px-2 py-1 text-xs font-mono bg-muted rounded border">
                    {key}
                  </kbd>
                ))}
              </div>
            </div>
          ))}
        </div>
      </CardContent>
    </Card>
  )
}

function DataSettings() {
  const bookmarks = useBookmarksStore((s) => s.bookmarks)
  const messages = useMessagesStore((s) => s.messages)
  const notes = useNotesStore((s) => s.notes)
  const snippets = useSnippetsStore((s) => s.snippets)
  const activity = useActivityStore((s) => s.items)
  const servers = useServersStore((s) => s.servers)

  const handleExport = () => {
    const data = {
      exportedAt: new Date().toISOString(),
      version: '1.0',
      bookmarks,
      messages,
      notes,
      snippets,
      activity,
      servers: servers.map((s) => ({ ...s, apiKey: null, token: null })), // Don't export secrets
    }

    const blob = new Blob([JSON.stringify(data, null, 2)], { type: 'application/json' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `gloski-export-${new Date().toISOString().split('T')[0]}.json`
    a.click()
    URL.revokeObjectURL(url)
    toast.success('Data exported successfully')
  }

  const handleClearAll = () => {
    if (confirm('This will clear all local data. Are you sure?')) {
      localStorage.clear()
      toast.success('All data cleared. Refreshing...')
      setTimeout(() => window.location.reload(), 1000)
    }
  }

  return (
    <>
      <Card>
        <CardHeader>
          <CardTitle>Local Storage</CardTitle>
          <CardDescription>Data stored locally in your browser</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="space-y-3">
            <div className="flex items-center justify-between py-2">
              <span className="text-sm">Servers</span>
              <Badge variant="secondary">{servers.length}</Badge>
            </div>
            <div className="flex items-center justify-between py-2">
              <span className="text-sm">Bookmarks</span>
              <Badge variant="secondary">{bookmarks.length}</Badge>
            </div>
            <div className="flex items-center justify-between py-2">
              <span className="text-sm">Notes</span>
              <Badge variant="secondary">{notes.length}</Badge>
            </div>
            <div className="flex items-center justify-between py-2">
              <span className="text-sm">Snippets</span>
              <Badge variant="secondary">{snippets.length}</Badge>
            </div>
            <div className="flex items-center justify-between py-2">
              <span className="text-sm">Messages</span>
              <Badge variant="secondary">{messages.length}</Badge>
            </div>
            <div className="flex items-center justify-between py-2">
              <span className="text-sm">Activity Items</span>
              <Badge variant="secondary">{activity.length}</Badge>
            </div>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Export & Import</CardTitle>
          <CardDescription>Backup or restore your data</CardDescription>
        </CardHeader>
        <CardContent className="flex gap-3">
          <Button variant="outline" onClick={handleExport}>
            <Download className="h-4 w-4 mr-2" />
            Export Data
          </Button>
          <Button variant="outline" disabled>
            <Upload className="h-4 w-4 mr-2" />
            Import Data
          </Button>
        </CardContent>
      </Card>

      <Card className="border-destructive/50">
        <CardHeader>
          <CardTitle className="text-destructive">Danger Zone</CardTitle>
          <CardDescription>Irreversible actions</CardDescription>
        </CardHeader>
        <CardContent>
          <Button variant="destructive" onClick={handleClearAll}>
            <Trash2 className="h-4 w-4 mr-2" />
            Clear All Data
          </Button>
        </CardContent>
      </Card>
    </>
  )
}

function AboutSettings() {
  return (
    <>
      <Card>
        <CardHeader>
          <CardTitle>Gloski Control Center</CardTitle>
          <CardDescription>A modern control center for managing Linux servers</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex items-center justify-between">
            <span className="text-sm text-muted-foreground">Version</span>
            <Badge>1.0.0</Badge>
          </div>
          <Separator />
          <div className="space-y-2">
            <p className="text-sm text-muted-foreground">
              Built with React, TypeScript, Tailwind CSS, and shadcn/ui. Backend powered by Go.
            </p>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Links</CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          <a
            href="https://github.com"
            target="_blank"
            rel="noopener noreferrer"
            className="flex items-center gap-3 p-3 rounded-lg hover:bg-muted transition-colors"
          >
            <Github className="h-5 w-5" />
            <div className="flex-1">
              <p className="font-medium">GitHub Repository</p>
              <p className="text-sm text-muted-foreground">View source code and contribute</p>
            </div>
            <ExternalLink className="h-4 w-4 text-muted-foreground" />
          </a>
        </CardContent>
      </Card>
    </>
  )
}
