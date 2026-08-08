import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [
    react(),
    {
      name: 'preserve-embed-placeholder',
      generateBundle() {
        this.emitFile({ type: 'asset', fileName: '.gitkeep', source: '' })
      },
    },
  ],
  server: {
    port: 5173,
    proxy: {
      '/api': 'http://127.0.0.1:41739',
    },
  },
  build: {
    outDir: '../internal/server/web',
    emptyOutDir: true,
    sourcemap: false,
  },
})
