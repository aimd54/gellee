// Copyright The gellee Authors
// SPDX-License-Identifier: Apache-2.0

// Captures the demo overview in light and dark into docs/assets, for the
// README. Starts the built binary over the demo with a fixed date, so the
// figures on screen are the same at every run. Run it through
// `make readme-shots`.

import { spawn } from 'node:child_process';
import { chromium } from '@playwright/test';

const listen = '127.0.0.1:8097';
const base = `http://${listen}`;
const server = spawn('../bin/gellee', ['serve', '--demo', '--listen', listen, '--today', '2026-11-11'], {
  stdio: 'inherit',
});

async function ready() {
  for (let i = 0; i < 50; i++) {
    try {
      if ((await fetch(base)).ok) return;
    } catch {
      // not listening yet
    }
    await new Promise((r) => setTimeout(r, 100));
  }
  throw new Error(`the demo server did not answer on ${base}`);
}

const browser = await chromium.launch();
try {
  await ready();
  for (const scheme of ['light', 'dark']) {
    const page = await browser.newPage({ viewport: { width: 1280, height: 740 }, colorScheme: scheme });
    await page.goto(base);
    await page.getByRole('note', { name: 'Demo: invented data' }).waitFor();
    await page.getByText('4 projects on the bench').waitFor();
    const path = `../docs/assets/dashboard-${scheme}.png`;
    await page.screenshot({ path });
    console.log(`wrote ${path}`);
    await page.close();
  }
} finally {
  await browser.close();
  server.kill();
}
