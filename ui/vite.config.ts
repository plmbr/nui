// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { readFileSync } from 'fs'
import path from 'path'
import { defineConfig } from 'vitest/config'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

const appVersion = readFileSync(path.resolve(__dirname, '../VERSION'), 'utf-8').trim()

export default defineConfig({
  define: {
    __NUI_VERSION__: JSON.stringify(appVersion),
  },
  plugins: [react(), tailwindcss()],
  base: '/',
  build: {
    outDir: 'dist',
    chunkSizeWarningLimit: 3200,
  },
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  test: {
    environment: 'jsdom',
    globals: true,
    setupFiles: ['./src/test/setup.ts'],
    include: ['src/**/*.test.{ts,tsx}'],
  },
  server: {
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
      '/mcp-resource': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
      '/mcp-call-tool': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
})
