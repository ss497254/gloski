import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'
import path from 'path'

export default defineConfig({
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  server: {
    allowedHosts: true,
    port: 3000,
    proxy: {
      '/api/terminal': {
        target: 'ws://localhost:8080',
        rewriteWsOrigin: true,
        ws: true,
      },
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
  build: {
    cssCodeSplit: false,
    rollupOptions: {
      output: {
        manualChunks(id) {
          if (id.includes('@codemirror')) {
            return 'vendor.code-editor'
          } else if (id.includes('@xterm')) {
            return 'vendor.xterm'
          } else if (id.includes('@lezer')) {
            return 'vendor.lezer'
          } else if (id.includes('node_modules')) {
            return 'vendor'
          } else if (id.includes('src/features/')) {
            const featureName = id.split('src/features/')[1].split('/')[0] ?? '[fix-me]'
            return 'feature_' + featureName
          }
        },
      },
    },
  },
})
