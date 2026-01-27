import { CodeEditor } from '@/shared/components/code-editor'

interface TextPreviewProps {
  content: string
  onChange?: (content: string) => void
  readOnly?: boolean
  filename?: string
}

export function TextPreview({ content, onChange, readOnly = true, filename }: TextPreviewProps) {
  return (
    <CodeEditor
      value={content}
      onChange={onChange}
      readOnly={readOnly}
      filename={filename}
      className="h-full border-0 rounded-none"
    />
  )
}
