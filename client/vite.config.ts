import { defineConfig } from 'vite';
import vue from '@vitejs/plugin-vue';
import tailwindcss from '@tailwindcss/vite';

export default defineConfig({
  server: {
    port: 5173,
    proxy: {
      '/api': 'http://localhost:8080',
      // 图标集由后端托管并自动更新（端口见 server/config.go 的 Addr）
      '/icons/mdi.json': 'http://localhost:8080',
    },
  },
  plugins: [vue(), tailwindcss()],
});
