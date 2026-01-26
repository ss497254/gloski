import { FileQuestion, Download } from 'lucide-react'
import { Button } from '@/ui/button'
import { formatFileSize } from '../../lib/file-types'

interface BinaryPreviewProps {
  filename: string
  size: number
  onDownload: () => void
}

export function BinaryPreview({ filename, size, onDownload }: BinaryPreviewProps) {
  return (
    <div className="h-full flex flex-col items-center justify-center gap-4 p-8 text-center">
      <FileQuestion className="h-16 w-16 text-muted-foreground" />

      <div className="space-y-1">
        <h3 className="font-medium">Cannot preview this file</h3>
        <p className="text-sm text-muted-foreground">Binary files cannot be displayed in the browser.</p>
      </div>

      <div className="text-sm text-muted-foreground">
        <span className="font-medium">{filename}</span>
        <span className="mx-2">-</span>
        <span>{formatFileSize(size)}</span>
      </div>

      <Button onClick={onDownload} className="mt-2">
        <Download className="h-4 w-4 mr-2" />
        Download File
      </Button>
    </div>
  )
}
