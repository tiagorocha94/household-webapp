import { configDefaults, defineConfig } from 'vitest/config';
import { svelte } from '@sveltejs/vite-plugin-svelte';

// The UI normally runs from the Go server (make run). The dev server and its
// /api proxy exist only for `npm run dev:mock` (the UI against fake data).
export default defineConfig({
  plugins: [svelte()],
  server: {
    port: 5173,
    proxy: {
      '/api': 'http://127.0.0.1:8080',
    },
  },
  build: {
    outDir: 'dist',
  },
  test: {
    // The browser tests in e2e/ run with Playwright (npm run e2e), not here.
    exclude: [...configDefaults.exclude, 'e2e/**'],
  },
});
