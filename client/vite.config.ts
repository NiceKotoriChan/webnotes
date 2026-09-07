import { defineConfig } from 'vite';
import { svelte } from '@sveltejs/vite-plugin-svelte';
import { VitePWA } from 'vite-plugin-pwa';

export default defineConfig({
  server: {
    port: 5173,
    proxy: {
      '/api': 'http://localhost:8080',
    },
  },
  plugins: [
    svelte(),
    VitePWA({
      registerType: 'autoUpdate',
      manifest: {
        name: 'WebNotes',
        short_name: 'WebNotes',
        display: 'standalone',
        background_color: '#ffffff',
        theme_color: '#1f2937',
        icons: [],
      },
    }),
  ],
});
