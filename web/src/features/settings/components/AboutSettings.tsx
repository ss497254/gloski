import { Badge } from '@/ui/badge'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/ui/card'
import { Separator } from '@/ui/separator'
import { ExternalLink, Github } from 'lucide-react'

export function AboutSettings() {
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
