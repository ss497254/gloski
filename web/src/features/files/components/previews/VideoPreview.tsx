import { AlertCircle, Loader2 } from 'lucide-react'
import { useState } from 'react'

interface VideoPreviewProps {
  src: string
  filename: string
}

export function VideoPreview({ src, filename }: VideoPreviewProps) {
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(false)

  return (
    <div className="flex items-center justify-center h-full p-4 bg-muted/30">
      {loading && !error && (
        <div className="absolute inset-0 flex items-center justify-center">
          <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
        </div>
      )}

      {error ? (
        <div className="flex flex-col items-center gap-2 text-muted-foreground">
          <AlertCircle className="h-12 w-12" />
          <p>Failed to load video</p>
          <p className="text-sm">This format may not be supported by your browser</p>
        </div>
      ) : (
        <video
          src={src}
          controls
          className="max-w-full max-h-full"
          onLoadedData={() => setLoading(false)}
          onError={() => {
            setLoading(false)
            setError(true)
          }}
          style={{ display: loading ? 'none' : 'block' }}
        >
          <track kind="captions" label={filename} />
          Your browser does not support the video element.
        </video>
      )}
    </div>
  )
}
