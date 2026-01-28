import { ConfirmDialog } from '@/shared/components'
import { Badge } from '@/ui/badge'
import { Button } from '@/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/ui/card'
import { Download, Trash2, Upload } from 'lucide-react'
import { useSettings } from '../context'

export function DataSettings() {
  const { dataCounts, handleExport, handleClearAll, showClearDialog, setShowClearDialog } = useSettings()

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
              <Badge variant="secondary">{dataCounts.servers}</Badge>
            </div>
            <div className="flex items-center justify-between py-2">
              <span className="text-sm">Bookmarks</span>
              <Badge variant="secondary">{dataCounts.bookmarks}</Badge>
            </div>
            <div className="flex items-center justify-between py-2">
              <span className="text-sm">Notes</span>
              <Badge variant="secondary">{dataCounts.notes}</Badge>
            </div>
            <div className="flex items-center justify-between py-2">
              <span className="text-sm">Snippets</span>
              <Badge variant="secondary">{dataCounts.snippets}</Badge>
            </div>
            <div className="flex items-center justify-between py-2">
              <span className="text-sm">Messages</span>
              <Badge variant="secondary">{dataCounts.messages}</Badge>
            </div>
            <div className="flex items-center justify-between py-2">
              <span className="text-sm">Activity Items</span>
              <Badge variant="secondary">{dataCounts.activity}</Badge>
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
          <Button variant="destructive" onClick={() => setShowClearDialog(true)}>
            <Trash2 className="h-4 w-4 mr-2" />
            Clear All Data
          </Button>
        </CardContent>
      </Card>

      <ConfirmDialog
        open={showClearDialog}
        onOpenChange={setShowClearDialog}
        title="Clear all data?"
        description="This will permanently delete all your servers, bookmarks, notes, snippets, messages, and activity history. This action cannot be undone."
        confirmLabel="Clear all data"
        variant="destructive"
        onConfirm={handleClearAll}
      />
    </>
  )
}
