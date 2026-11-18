import { File, FileArchive, FileCode, FileCog, FileImage, FileJson, FileText, Folder } from 'lucide-react'

/**
 * File extension to icon mapping
 */
const fileIcons: Record<string, typeof File> = {
  // Code files
  ts: FileCode,
  tsx: FileCode,
  js: FileCode,
  jsx: FileCode,
  py: FileCode,
  go: FileCode,
  rs: FileCode,
  java: FileCode,
  cpp: FileCode,
  c: FileCode,
  h: FileCode,
  css: FileCode,
  scss: FileCode,
  html: FileCode,
  vue: FileCode,
  svelte: FileCode,
  sh: FileCode,
  bash: FileCode,
  // Config/Data
  json: FileJson,
  yaml: FileCog,
  yml: FileCog,
  toml: FileCog,
  xml: FileCode,
  env: FileCog,
  // Images
  png: FileImage,
  jpg: FileImage,
  jpeg: FileImage,
  gif: FileImage,
  svg: FileImage,
  webp: FileImage,
  ico: FileImage,
  // Archives
  zip: FileArchive,
  tar: FileArchive,
  gz: FileArchive,
  rar: FileArchive,
  '7z': FileArchive,
  // Text
  md: FileText,
  txt: FileText,
  log: FileText,
}

/**
 * File extension to color mapping
 */
const fileColors: Record<string, string> = {
  ts: 'text-info',
  tsx: 'text-info',
  js: 'text-warning',
  jsx: 'text-warning',
  py: 'text-success',
  go: 'text-info',
  json: 'text-warning',
  md: 'text-muted-foreground',
  txt: 'text-muted-foreground',
  png: 'text-pink-400',
  jpg: 'text-pink-400',
  jpeg: 'text-pink-400',
  gif: 'text-pink-400',
  svg: 'text-pink-400',
}

/**
 * Get the appropriate icon component for a file
 */
export function getFileIcon(name: string, type: string) {
  if (type === 'directory') return Folder
  const ext = name.split('.').pop()?.toLowerCase() || ''
  return fileIcons[ext] || File
}

/**
 * Get the appropriate color class for a file
 */
export function getFileColor(name: string, type: string): string {
  if (type === 'directory') return 'text-info'
  const ext = name.split('.').pop()?.toLowerCase() || ''
  return fileColors[ext] || 'text-muted-foreground'
}
