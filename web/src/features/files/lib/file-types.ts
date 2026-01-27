export type FileType = 'image' | 'video' | 'audio' | 'code' | 'pdf' | 'text' | 'binary'

const IMAGE_EXTENSIONS = new Set(['png', 'jpg', 'jpeg', 'gif', 'svg', 'webp', 'ico', 'bmp', 'avif'])

const VIDEO_EXTENSIONS = new Set(['mp4', 'webm', 'ogv', 'mov', 'avi', 'mkv', 'm4v'])

const AUDIO_EXTENSIONS = new Set(['mp3', 'wav', 'ogg', 'flac', 'aac', 'm4a', 'wma', 'opus'])

const CODE_EXTENSIONS = new Set([
  // JavaScript/TypeScript
  'js',
  'jsx',
  'ts',
  'tsx',
  'mjs',
  'cjs',
  // Web
  'html',
  'htm',
  'css',
  'scss',
  'sass',
  'less',
  // Data formats
  'json',
  'xml',
  'yaml',
  'yml',
  'toml',
  // Markdown
  'md',
  'mdx',
  // Python
  'py',
  'pyw',
  'pyi',
  // Go
  'go',
  'mod',
  'sum',
  // Rust
  'rs',
  // Java/Kotlin
  'java',
  'kt',
  'kts',
  // C/C++
  'c',
  'h',
  'cpp',
  'hpp',
  'cc',
  'cxx',
  // C#
  'cs',
  // Ruby
  'rb',
  'erb',
  // PHP
  'php',
  // Shell
  'sh',
  'bash',
  'zsh',
  'fish',
  // SQL
  'sql',
  // Lua
  'lua',
  // Perl
  'pl',
  'pm',
  // Swift
  'swift',
  // Makefile
  'makefile',
  'mk',
  // Docker
  'dockerfile',
])

const TEXT_EXTENSIONS = new Set([
  'txt',
  'log',
  'conf',
  'cfg',
  'ini',
  'env',
  'gitignore',
  'gitattributes',
  'dockerignore',
  'editorconfig',
  'prettierrc',
  'eslintrc',
  'nvmrc',
  'npmrc',
  'yarnrc',
])

const PDF_EXTENSIONS = new Set(['pdf'])

// Large file threshold (1MB)
export const LARGE_FILE_THRESHOLD = 1024 * 1024

/**
 * Get the file extension from a filename
 */
export function getExtension(filename: string): string {
  // Handle dotfiles like .gitignore
  const name = filename.toLowerCase()

  // Special cases for known dotfiles
  if (name === '.gitignore' || name === '.dockerignore') return 'gitignore'
  if (name === '.env' || name.startsWith('.env.')) return 'env'
  if (name === 'dockerfile') return 'dockerfile'
  if (name === 'makefile') return 'makefile'

  const lastDot = name.lastIndexOf('.')
  if (lastDot === -1 || lastDot === 0) return ''
  return name.slice(lastDot + 1)
}

/**
 * Determine the file type based on extension
 */
export function getFileType(filename: string): FileType {
  const ext = getExtension(filename)

  if (!ext) return 'text' // Files without extension are likely text
  if (IMAGE_EXTENSIONS.has(ext)) return 'image'
  if (VIDEO_EXTENSIONS.has(ext)) return 'video'
  if (AUDIO_EXTENSIONS.has(ext)) return 'audio'
  if (PDF_EXTENSIONS.has(ext)) return 'pdf'
  if (CODE_EXTENSIONS.has(ext)) return 'code'
  if (TEXT_EXTENSIONS.has(ext)) return 'text'

  return 'binary'
}

/**
 * Check if a file can be previewed as text (code or plain text)
 */
export function isTextFile(filename: string): boolean {
  const type = getFileType(filename)
  return type === 'code' || type === 'text'
}

/**
 * Check if a file can be previewed at all
 */
export function isPreviewable(filename: string): boolean {
  const type = getFileType(filename)
  return type !== 'binary'
}

/**
 * Check if a file should show a large file warning
 */
export function shouldWarnLargeFile(size: number): boolean {
  return size > LARGE_FILE_THRESHOLD
}

/**
 * Format file size for display
 */
export function formatFileSize(bytes: number): string {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return `${parseFloat((bytes / Math.pow(k, i)).toFixed(1))} ${sizes[i]}`
}
