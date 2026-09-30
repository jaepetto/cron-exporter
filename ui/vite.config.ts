import tailwindcss from '@tailwindcss/vite';
import react from '@vitejs/plugin-react';
import { defineConfig } from 'vite';

export default defineConfig({
  base: './',
  plugins: [react(), tailwindcss()],
  build: {
    manifest: true,
    outDir: '../pkg/dashboard/web',
    emptyOutDir: false,
    sourcemap: false,
  },
});
