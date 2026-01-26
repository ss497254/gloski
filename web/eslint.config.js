import js from '@eslint/js'
import globals from 'globals'
import reactHooks from 'eslint-plugin-react-hooks'
import reactRefresh from 'eslint-plugin-react-refresh'
import tseslint from 'typescript-eslint'
import boundaries from 'eslint-plugin-boundaries'
import { defineConfig, globalIgnores } from 'eslint/config'

export default defineConfig([
  globalIgnores(['dist']),
  {
    files: ['**/*.{ts,tsx}'],
    extends: [
      js.configs.recommended,
      tseslint.configs.recommended,
      reactHooks.configs.flat.recommended,
      reactRefresh.configs.vite,
    ],
    plugins: {
      boundaries,
    },
    languageOptions: {
      ecmaVersion: 2020,
      globals: globals.browser,
    },
    settings: {
      'boundaries/include': ['src/**/*'],
      'boundaries/elements': [
        { type: 'ui', pattern: ['src/ui/*'], mode: 'folder' },
        { type: 'shared', pattern: ['src/shared/*'], mode: 'folder' },
        { type: 'feature', pattern: ['src/features/*'], mode: 'folder' },
        { type: 'layout', pattern: ['src/layouts/*'], mode: 'folder' },
        { type: 'app', pattern: ['src/app/*'], mode: 'folder' },
      ],
    },
    rules: {
      'boundaries/element-types': [
        'warn',
        {
          default: 'disallow',
          rules: [
            { from: 'ui', allow: ['ui'] },
            { from: 'shared', allow: ['ui', 'shared'] },
            { from: 'feature', allow: ['ui', 'shared'] },
            { from: 'layout', allow: ['ui', 'shared', 'feature'] },
            { from: 'app', allow: ['ui', 'shared', 'feature', 'layout', 'app'] },
          ],
        },
      ],
      'react-hooks/set-state-in-effect': 'warn',
    },
  },
])
