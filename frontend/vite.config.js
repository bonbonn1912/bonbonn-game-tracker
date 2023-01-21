import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [react()],
  base:"./",
  build: {
    outDir: "../build/frontend/",
  },
  server: {
    port: 33159,
    strictPort: true,
    proxy: {
      "/test": {
        target: "http://localhost:3000",
        changeOrigin: true,
        secure: false,
      },
    },
  },
})
