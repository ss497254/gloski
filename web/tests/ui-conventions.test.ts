import { describe, expect, test } from 'bun:test'
import { readFileSync, readdirSync } from 'node:fs'
import { join } from 'node:path'

const webRoot = join(import.meta.dir, '..')
const uiDir = join(webRoot, 'src/ui')
const uiFiles = readdirSync(uiDir).filter((name) => name.endsWith('.tsx'))
const readUi = (name: string) => readFileSync(join(uiDir, name), 'utf8')
const packageJson = JSON.parse(readFileSync(join(webRoot, 'package.json'), 'utf8'))

describe('shadcn conventions', () => {
  test('no UI file imports cn from the npm package', () => {
    for (const name of uiFiles) {
      expect(readUi(name)).not.toMatch(/from ['"]cn['"]/)
    }
  })

  test('package.json has no stray cn dependency', () => {
    expect(packageJson.dependencies?.cn).toBeUndefined()
  })

  test('Button keeps its default variant after a CLI overwrite', () => {
    expect(readUi('button.tsx')).toContain('bg-primary text-primary-foreground hover:bg-primary/90')
  })

  test('Badge keeps its status variants', () => {
    const badge = readUi('badge.tsx')
    expect(badge).toContain('success:')
    expect(badge).toContain('warning:')
    expect(badge).toContain('info:')
  })

  test('Card keeps the original surface shadow', () => {
    expect(readUi('card.tsx')).toContain('shadow-sm')
  })
})
