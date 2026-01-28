import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/ui/card'

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

export function ShortcutsSettings() {
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
