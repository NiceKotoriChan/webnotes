import { defineConfig } from 'vite';
import vue from '@vitejs/plugin-vue';
import tailwindcss from '@tailwindcss/vite';

export default defineConfig({
  server: {
    port: 5173,
    proxy: {
      '/api': 'http://localhost:8080',
      // 图标集与两份图标规则文件都由后端托管（端口见 server/config.go 的 Addr）。
      // 这里转发整个前缀，别只转发 mdi.json —— 规则文件路径不同，漏了 dev 下会 404。
      '/icons': 'http://localhost:8080',
    },
  },
  plugins: [vue(), tailwindcss()],
});
