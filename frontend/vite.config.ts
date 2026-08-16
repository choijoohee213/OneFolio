import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// 개발 중에는 프록시로 백엔드에 붙여 CORS 를 피한다.
// 배포에서는 VITE_API_BASE 로 백엔드 주소를 준다.
export default defineConfig({
  plugins: [react()],
  server: {
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        timeout: 120000,
        proxyTimeout: 120000,
      },
      '/health': 'http://localhost:8080',
    },
  },
})
