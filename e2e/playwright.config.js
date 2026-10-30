// @ts-check
const crypto = require('node:crypto');
const os = require('node:os');
const path = require('node:path');
const { defineConfig, devices } = require('@playwright/test');

module.exports = defineConfig({
  testDir: './tests',
  // Fail CI if a test.only was left in by accident
  forbidOnly: !!process.env.CI,
  // Browser tests can fail on timing alone; one retry on CI keeps that from blocking a PR
  retries: process.env.CI ? 1 : 0,
  reporter: process.env.CI ? [['github'], ['html', { open: 'never' }]] : 'list',
  use: {
    baseURL: 'http://127.0.0.1:8080',
    // Keep a trace (screenshots, DOM, network) of failed tests to see what went wrong
    trace: 'retain-on-failure',
  },
  // One browser is enough to check our selectors; the simulator's browser users only need them to exist
  projects: [{ name: 'chromium', use: { ...devices['Desktop Chrome'] } }],

  // Start the Go app before the tests. It runs from backend/ because the templates are
  // loaded from ../frontend. The session key is random per run and the database is a
  // throwaway file, so no real secrets or data are involved.
  webServer: {
    command: 'go run .',
    cwd: '../backend',
    url: 'http://127.0.0.1:8080/',
    // Locally, reuse an app that is already running on 8080 (e.g. docker compose up)
    reuseExistingServer: !process.env.CI,
    // The first go run also compiles the app
    timeout: 120_000,
    env: {
      ...process.env,
      SESSION_HASH_KEY: crypto.randomBytes(32).toString('base64'),
      DB_PATH: path.join(os.tmpdir(), 'whoknows-e2e.db'),
    },
  },
});
