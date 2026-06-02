import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

export default defineConfig({
  plugins: [
    react(),
    tailwindcss(),
    {
      name: 'redirect-to-portal',
      configureServer(server) {
        server.middlewares.use((req, res, next) => {
          if (req.url === '/' || req.url === '/portal') {
            res.writeHead(302, { Location: '/portal/' })
            res.end()
            return
          }
          next()
        })
      },
    },
  ],
  base: '/portal/',
  server: {
    open: '/portal/',
  },
})
