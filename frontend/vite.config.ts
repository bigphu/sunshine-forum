import { fileURLToPath, URL } from 'node:url'
import vue from '@vitejs/plugin-vue'
import { defineConfig } from 'vite'

// Dev chạy cổng 5180 và chuyển /api sang backend: cùng origin nên cookie phiên
// hoạt động, backend không cần CORS. 127.0.0.1 để Windows không thử IPv6 trước.
export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: { '@': fileURLToPath(new URL('./src', import.meta.url)) },
  },
  server: {
    port: 5180,
    strictPort: true,
    proxy: {
      '/api': { target: process.env.API_URL ?? 'http://127.0.0.1:8091', changeOrigin: false },
    },
    // Trong container dev (compose.yml): bind mount trên Windows không phát sự kiện file, và
    // HMR phải nối về cổng đã publish
    watch: process.env.VITE_USE_POLLING === 'true' ? { usePolling: true, interval: 300 } : undefined,
    hmr: process.env.VITE_HMR_CLIENT_PORT ? { clientPort: Number(process.env.VITE_HMR_CLIENT_PORT) } : undefined,
  },
})
