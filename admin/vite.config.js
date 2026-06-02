import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

export default defineConfig({
  plugins: [react(), tailwindcss()],
  base: '/admin/',
  server: {
    proxy: {
      '/api/admin': {
        target: 'http://localhost:8082',
        changeOrigin: true,
      },
    },
  },
})
