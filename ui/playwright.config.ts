import { defineConfig, devices } from '@playwright/test';

export default defineConfig({
  testDir: 'e2e',
  fullyParallel: false,
  workers: 1,
  retries: 1,
  reporter: [['list'], ['html', { outputFolder: 'playwright-report', open: 'never' }]],
  outputDir: 'test-results',
  use: {
    baseURL: 'http://127.0.0.1:18080',
    httpCredentials: { username: 'admin', password: 'portal-e2e-admin-key' },
    trace: 'retain-on-failure',
    screenshot: 'only-on-failure',
  },
  projects: [
    { name: 'chromium', use: { ...devices['Desktop Chrome'] } },
    { name: 'firefox', use: { ...devices['Desktop Firefox'] } },
    { name: 'webkit', use: { ...devices['Desktop Safari'] } },
  ],
  webServer: {
    command: 'cd .. && mise run build && rm -f /tmp/cronmetrics_portal_e2e.db && ./bin/cronmetrics serve --config test/e2e-browser/config.yaml',
    url: 'http://127.0.0.1:18080/health',
    timeout: 120_000,
    reuseExistingServer: false,
  },
});