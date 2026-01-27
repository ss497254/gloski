/**
 * Map snippet language names to file extensions for CodeMirror syntax highlighting
 */
const languageExtensions: Record<string, string> = {
  // JavaScript/TypeScript
  javascript: 'js',
  typescript: 'ts',
  jsx: 'jsx',
  tsx: 'tsx',

  // Web
  html: 'html',
  css: 'css',
  scss: 'scss',
  less: 'less',

  // Data formats
  json: 'json',
  xml: 'xml',
  yaml: 'yaml',
  yml: 'yaml',

  // Markdown
  markdown: 'md',
  md: 'md',

  // Python
  python: 'py',
  py: 'py',

  // Go
  go: 'go',
  golang: 'go',

  // Rust
  rust: 'rs',
  rs: 'rs',

  // Java/Kotlin
  java: 'java',
  kotlin: 'kt',
  kt: 'kt',

  // C/C++
  c: 'c',
  cpp: 'cpp',
  'c++': 'cpp',

  // SQL
  sql: 'sql',

  // Shell
  bash: 'sh',
  shell: 'sh',
  sh: 'sh',
  zsh: 'sh',
}

/**
 * Convert a language name to a filename with appropriate extension
 * This allows CodeMirror to apply the correct syntax highlighting
 */
export function languageToFilename(language: string): string {
  const lang = language.toLowerCase().trim()
  const ext = languageExtensions[lang] || lang
  return `snippet.${ext}`
}
