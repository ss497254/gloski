import { AlertCircle, Loader2 } from 'lucide-react'
import { useState } from 'react'

interface ImagePreviewProps {
  src: string
  alt: string
}

export function ImagePreview({ src, alt }: ImagePreviewProps) {
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
          <p>Failed to load image</p>
        </div>
      ) : (
        <img
          src={src}
          alt={alt}
          className="max-w-full max-h-full object-contain"
          onLoad={() => setLoading(false)}
          onError={() => {
            setLoading(false)
            setError(true)
          }}
          style={{ display: loading ? 'none' : 'block' }}
        />
      )}
    </div>
  )
}
