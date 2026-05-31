import { fileURLToPath, URL } from 'node:url'

import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import vueDevTools from 'vite-plugin-vue-devtools'
import AutoImport from 'unplugin-auto-import/vite'
import Components from 'unplugin-vue-components/vite'
import { ElementPlusResolver } from 'unplugin-vue-components/resolvers'

// https://vite.dev/config/
export default defineConfig({
  plugins: [
    vue(),
    AutoImport({
      resolvers: [ElementPlusResolver()],
    }),
    Components({
      resolvers: [ElementPlusResolver()],
    }),
  ],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url))
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
      }
    }
  },
})

