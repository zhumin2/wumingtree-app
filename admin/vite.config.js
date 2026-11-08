import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// dev 模式经代理访问本机 Go API（避免 CORS）；生产构建产物由 API 同源提供 /static/admin（架构设计 5.2）。
export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,
    proxy: {
      '/api': 'http://127.0.0.1:8080',
    },
  },
  base: '/static/admin/',
})
