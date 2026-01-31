import { Button } from '@/ui/button'
import { AlertCircle, Loader2, ZoomIn, ZoomOut, RotateCcw } from 'lucide-react'
import { useState } from 'react'

interface ImagePreviewProps {
  src: string
  alt: string
}

const MAX_ZOOM = 200
const MIN_ZOOM = 25

export function ImagePreview({ src, alt }: ImagePreviewProps) {
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(false)
  const [zoom, setZoom] = useState(100)

  const handleZoomIn = () => {
    setZoom((prev) => Math.min(prev + 25, MAX_ZOOM))
  }

  const handleZoomOut = () => {
    setZoom((prev) => Math.max(prev - 25, MIN_ZOOM))
  }

  const handleZoomReset = () => {
    setZoom(100)
  }

  return (
    <div className="flex flex-col h-full">
      {/* Controls */}
      <div className="flex items-center justify-center gap-2 p-3 border-b bg-background/50">
        <Button variant="outline" size="sm" onClick={handleZoomOut} disabled={zoom <= MIN_ZOOM}>
          <ZoomOut className="h-4 w-4" />
        </Button>
        <span className="text-sm font-medium min-w-16 text-center">{zoom}%</span>
        <Button variant="outline" size="sm" onClick={handleZoomIn} disabled={zoom >= MAX_ZOOM}>
          <ZoomIn className="h-4 w-4" />
        </Button>
        <Button variant="outline" size="sm" onClick={handleZoomReset} title="Reset">
          <RotateCcw className="h-4 w-4" />
        </Button>
      </div>

      {/* Image Container */}
      <div className="flex-1 flex items-center justify-center overflow-auto bg-muted/30 relative">
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
            className="select-none w-full"
            onLoad={() => setLoading(false)}
            onError={() => {
              setLoading(false)
              setError(true)
            }}
            style={{
              display: loading ? 'none' : 'block',
              scale: zoom / 100,
            }}
            draggable={false}
          />
        )}
      </div>
    </div>
  )
}
