import { fileURLToPath, URL } from 'node:url'

import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

// https://vite.dev/config/
export default defineConfig({
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
    },
  },
  build: {
    rollupOptions: {
      input: {
        index: fileURLToPath(new URL('./index.html', import.meta.url)),
        app: fileURLToPath(new URL('./app.html', import.meta.url)),
      },
    },
  },
  server: {
    port: 5173,
    proxy: {
      // 当本地开发遇到 /v1 开头的请求时
      '/v1': {
        target: 'http://localhost:8080', // 转发给本地起起来的 Go 后端
        changeOrigin: true,
        rewrite: (path) => path.replace(/^\/api/, ''), // 把 /api 去掉，变成 /v1/... 传给后端
      },
    },
  },
})