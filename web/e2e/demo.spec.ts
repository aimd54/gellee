// Copyright The gellee Authors
// SPDX-License-Identifier: Apache-2.0

import { expect, test } from '@playwright/test';

test.use({ baseURL: 'http://127.0.0.1:8098' });

const views = [
  { path: '/', text: ['loom', 'pathfinder', 'seedbank'] },
  { path: '/projects', text: ['weft', 'almanac', 'trellis'] },
  { path: '/projects/studio/loom', text: ['Thread the heddle frame into the loom core'] },
  { path: '/usage', text: ['pathfinder'] },
  { path: '/ideas', text: ['Preview a pattern as woven cloth before it is threaded'] },
  { path: '/workers', text: ['laptop', 'desktop', 'worker-1'] },
];

for (const { path, text } of views) {
  test(`the demo shows invented data on ${path}`, async ({ page }) => {
    await page.goto(path);
    await expect(page.getByRole('heading', { level: 1 })).toBeVisible();
    await expect(page.getByRole('note', { name: 'Demo: invented data' })).toBeVisible();
    for (const t of text) await expect(page.getByRole('main')).toContainText(t);
  });
}

test('the demo bench holds four projects against a limit of three', async ({ page }) => {
  await page.goto('/');
  await expect(page.getByText('4 projects on the bench, the limit is 3.')).toBeVisible();
  await expect(page.getByText('1 over the limit')).toBeVisible();
});
