import { defineConfig } from '@playwright/test'
export default defineConfig({
  testDir: './tests',
  workers: 1,
  fullyParallel: false,
  timeout: 30000,
  use: {
    channel: process.env.TRENDS_TEST_BROWSER,
    headless: true,
    viewport: { width: 1440, height: 960 },
  },
  reporter: 'list',
  outputDir: '../work/browser-results',
})
