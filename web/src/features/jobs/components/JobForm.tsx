import { Button } from '@/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/ui/card'
import { Input } from '@/ui/input'
import { Play } from 'lucide-react'
import { useJobs } from '../context'

export function JobForm() {
  const { newCommand, setNewCommand, newCwd, setNewCwd, startJob } = useJobs()

  return (
    <Card>
      <CardHeader className="pb-3">
        <CardTitle className="text-sm">Start New Job</CardTitle>
      </CardHeader>
      <CardContent className="space-y-3">
        <Input
          placeholder="Command (e.g., bun run dev)"
          value={newCommand}
          onChange={(e) => setNewCommand(e.target.value)}
          onKeyDown={(e) => e.key === 'Enter' && startJob()}
        />
        <div className="flex gap-2">
          <Input
            placeholder="Working directory (optional)"
            value={newCwd}
            onChange={(e) => setNewCwd(e.target.value)}
            className="flex-1"
          />
          <Button onClick={startJob} disabled={!newCommand.trim()}>
            <Play className="h-4 w-4 mr-2" />
            Start
          </Button>
        </div>
      </CardContent>
    </Card>
  )
}
