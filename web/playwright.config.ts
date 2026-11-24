// Copyright The gellee Authors
// SPDX-License-Identifier: Apache-2.0

import { defineConfig, devices } from '@playwright/test';

// The built binary serves the fixture state directory with a fixed date,
// so every figure on screen is known in advance.
export default defineConfig({
  testDir: 'e2e',
  fullyParallel: false,
  workers: 1,
  forbidOnly: !!process.env.CI,
  reporter: 'list',
  use: {
    baseURL: 'http://127.0.0.1:8099',
    viewport: { width: 1440, height: 900 },
    locale: 'en-GB',
    timezoneId: 'UTC',
    trace: 'retain-on-failure',
  },
  projects: [{ name: 'chromium', use: { ...devices['Desktop Chrome'], viewport: { width: 1440, height: 900 } } }],
  webServer: {
    command: '../bin/gellee serve --state ../tests/fixtures/state --listen 127.0.0.1:8099 --today 2026-11-11',
    url: 'http://127.0.0.1:8099/api/v1/health',
    reuseExistingServer: false,
    timeout: 20_000,
  },
});
