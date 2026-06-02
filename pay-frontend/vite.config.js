import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

export default defineConfig({
  plugins: [
    react(),
    tailwindcss(),
    {
      name: 'redirect-to-pay',
      configureServer(server) {
        server.middlewares.use((req, res, next) => {
          if (req.url === '/' || req.url === '/pay') {
            res.writeHead(302, { Location: '/pay/' })
            res.end()
            return
          }
          next()
        })
      },
    },
  ],
  base: '/pay/',
  server: {
    open: '/pay/',
  },
})
