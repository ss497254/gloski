import { PageLayout } from '@/layouts'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/ui/tabs'
import { Activity, Info, Palette } from 'lucide-react'
import { AboutSettings, AppearanceSettings, MasterHealthSettings } from '../components'
import { SettingsProvider, useSettings, type SettingsTab } from '../context'

const tabs: {
  id: SettingsTab
  label: string
  icon: React.ComponentType<{ className?: string }>
}[] = [
  { id: 'appearance', label: 'Appearance', icon: Palette },
  { id: 'health', label: 'Master health', icon: Activity },
  { id: 'about', label: 'About', icon: Info },
]

function SettingsContent() {
  const { activeTab, setActiveTab } = useSettings()

  return (
    <PageLayout title="Settings" description="Manage your preferences">
      <Tabs
        orientation="vertical"
        value={activeTab}
        onValueChange={(value) => setActiveTab(value as SettingsTab)}
        className="flex gap-8 max-w-4xl md:flex-row"
      >
        <TabsList className="flex h-auto w-full shrink-0 flex-row justify-start gap-1 bg-transparent p-0 md:w-48 md:flex-col">
          {tabs.map((tab) => (
            <TabsTrigger
              key={tab.id}
              value={tab.id}
              className="h-auto w-full justify-start gap-3 rounded-lg px-3 py-2 text-sm font-medium text-muted-foreground data-[state=active]:bg-accent data-[state=active]:text-accent-foreground data-[state=active]:shadow-none"
            >
              <tab.icon className="h-4 w-4" />
              {tab.label}
            </TabsTrigger>
          ))}
        </TabsList>

        <div className="min-w-0 flex-1 space-y-6">
          <TabsContent value="appearance">
            <AppearanceSettings />
          </TabsContent>
          <TabsContent value="health">
            <MasterHealthSettings />
          </TabsContent>
          <TabsContent value="about">
            <AboutSettings />
          </TabsContent>
        </div>
      </Tabs>
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
