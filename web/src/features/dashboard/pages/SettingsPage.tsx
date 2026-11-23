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
        value={activeTab}
        onValueChange={(value) => setActiveTab(value as SettingsTab)}
        className="max-w-4xl gap-6"
      >
        <TabsList className="w-full justify-start sm:w-auto">
          {tabs.map((tab) => (
            <TabsTrigger key={tab.id} value={tab.id} className="gap-2 px-4">
              <tab.icon className="h-4 w-4" />
              {tab.label}
            </TabsTrigger>
          ))}
        </TabsList>

        <div className="min-w-0 space-y-6">
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
