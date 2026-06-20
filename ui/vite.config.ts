// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import path from 'path'
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

export default defineConfig({
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
