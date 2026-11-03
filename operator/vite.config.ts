import { fileURLToPath, URL } from 'node:url';
import react from '@vitejs/plugin-react';
import { defineConfig } from 'vitest/config';

/**
 * The operator console talks to the Go backend over the documented HTTP
 * surface. Only /healthz and /readyz exist today; /admin/v1 is proxied so the
 * future M05/M07 routes work without touching this file, but the client never
 * calls them until they are implemented.
 */
const backend = process.env.TTG_OPERATOR_BACKEND_URL ?? 'http://127.0.0.1:8080';

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
    },
  },
  server: {
    port: 5173,
    strictPort: true,
    proxy: {
      '/healthz': { target: backend, changeOrigin: false },
      '/readyz': { target: backend, changeOrigin: false },
      '/admin/v1': { target: backend, changeOrigin: false },
    },
  },
  preview: {
    port: 4173,
    strictPort: true,
  },
  build: {
    outDir: 'dist',
    sourcemap: false,
    target: 'es2022',
  },
  test: {
    environment: 'jsdom',
    globals: true,
    setupFiles: ['./src/test/setup.ts'],
    css: false,
    restoreMocks: true,
    clearMocks: true,
    include: ['src/**/*.test.{ts,tsx}'],
  },
});
