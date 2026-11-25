import { defineConfig } from '@playwright/test';

// Each project is one set of modules, with its own server on its own port, started on a copy of
// the demo household (scripts/demo.sh). The binary must be built first: `make web-build build`.
const server = process.env.E2E_SERVER ?? './household-app';

const sets = [
  { name: 'all', modules: '', port: 8171 },
  { name: 'cashflow', modules: 'cashflow', port: 8172 },
  { name: 'goals+savings', modules: 'goals,savings', port: 8173 },
];

export default defineConfig({
  testDir: './e2e',
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 1 : 0,
  reporter: process.env.CI ? 'github' : 'list',
  projects: sets.map((set) => ({
    name: set.name,
    use: { baseURL: `http://127.0.0.1:${set.port}` },
  })),
  webServer: sets.map((set) => ({
    command: `bash scripts/demo.sh ${server}`,
    cwd: '..',
    env: {
      DEMO_PORT: String(set.port),
      DEMO_DIR: `.demo-e2e-${set.port}`,
      DEMO_MODULES: set.modules,
    },
    url: `http://127.0.0.1:${set.port}/api/v1/info`,
    reuseExistingServer: false,
    timeout: 60_000,
  })),
});
