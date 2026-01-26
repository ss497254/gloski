interface PdfPreviewProps {
  src: string
  filename: string
}

export function PdfPreview({ src, filename }: PdfPreviewProps) {
  return (
    <div className="h-full w-full">
      <iframe src={src} title={filename} className="w-full h-full border-0" />
    </div>
  )
}
