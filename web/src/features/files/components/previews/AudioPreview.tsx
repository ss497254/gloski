import { useState } from 'react'
import { Loader2, AlertCircle, Music } from 'lucide-react'

interface AudioPreviewProps {
  src: string
  filename: string
}

export function AudioPreview({ src, filename }: AudioPreviewProps) {
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(false)

  return (
    <div className="flex flex-col items-center justify-center h-full p-8 bg-muted/30 gap-6">
      {loading && !error && (
        <div className="absolute inset-0 flex items-center justify-center">
          <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
        </div>
      )}

      {error ? (
        <div className="flex flex-col items-center gap-2 text-muted-foreground">
          <AlertCircle className="h-12 w-12" />
          <p>Failed to load audio</p>
          <p className="text-sm">This format may not be supported by your browser</p>
        </div>
      ) : (
        <>
          <div
            className="flex flex-col items-center gap-3"
            style={{ display: loading ? 'none' : 'flex' }}
          >
            <div className="w-32 h-32 rounded-full bg-muted flex items-center justify-center">
              <Music className="h-16 w-16 text-muted-foreground" />
            </div>
            <p className="text-sm font-medium text-center max-w-xs truncate">{filename}</p>
          </div>
          <audio
            src={src}
            controls
            className="w-full max-w-md"
            onLoadedData={() => setLoading(false)}
            onError={() => {
              setLoading(false)
              setError(true)
            }}
            style={{ display: loading ? 'none' : 'block' }}
          >
            Your browser does not support the audio element.
          </audio>
        </>
      )}
    </div>
  )
}
