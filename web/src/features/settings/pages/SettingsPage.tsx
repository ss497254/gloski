import { PageLayout } from '@/layouts'
import { cn } from '@/shared/lib/utils'
import { Database, Info, Keyboard, Palette } from 'lucide-react'
import { AboutSettings, AppearanceSettings, DataSettings, ShortcutsSettings } from '../components'
import { SettingsProvider, useSettings, type SettingsTab } from '../context'

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

function SettingsContent() {
  const { activeTab, setActiveTab } = useSettings()

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
          {activeTab === 'appearance' && <AppearanceSettings />}
          {activeTab === 'shortcuts' && <ShortcutsSettings />}
          {activeTab === 'data' && <DataSettings />}
          {activeTab === 'about' && <AboutSettings />}
        </div>
      </div>
    </PageLayout>
  )
}

export function SettingsPage() {
  return (
    <SettingsProvider>
      <SettingsContent />
    </SettingsProvider>
  )
}
