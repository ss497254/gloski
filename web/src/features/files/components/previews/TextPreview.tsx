import { CodeEditor } from '../CodeEditor'

interface TextPreviewProps {
  content: string
  onChange?: (content: string) => void
  readOnly?: boolean
}

export function TextPreview({ content, onChange, readOnly = true }: TextPreviewProps) {
  return (
    <div className="h-full overflow-auto p-4">
      <CodeEditor value={content} onChange={onChange} readOnly={readOnly} className="min-h-full" />
    </div>
  )
}
