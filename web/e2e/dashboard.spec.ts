// Copyright The gellee Authors
// SPDX-License-Identifier: Apache-2.0

import { expect, test, type Page } from '@playwright/test';

const rail = (page: Page) => page.getByRole('navigation', { name: 'Sections' });

test('the overview leads with the bench, in handoff order', async ({ page }) => {
  await page.goto('/');
  await expect(page.getByRole('heading', { level: 1 })).toHaveText('Pick up the thread.');
  await expect(page.getByText('Wednesday, 11 November · collected 10:00')).toBeVisible();
  await expect(page.getByText('3 projects on the bench, the limit is 2.')).toBeVisible();
  await expect(page.getByText('1 over the limit')).toBeVisible();
  await expect(page.getByRole('alert')).toContainText('alpha/broken.md: state "sleeping" is not active, parked or done');

  const cards = page.getByRole('region', { name: 'The bench' }).locator('a.gl-hero');
  await expect(cards).toHaveCount(3);
  expect(await cards.evaluateAll((els) => els.map((el) => el.getAttribute('data-project')))).toEqual([
    'beta/tool',
    'alpha/renamed',
    'alpha/fern',
  ]);
  await expect(cards.nth(0)).toContainText('Where you left off');
  await expect(cards.nth(0)).toContainText('The parser handles nested lists.');
  await expect(cards.nth(0)).toContainText('Add the table parser.');
  await expect(cards.nth(0)).toContainText('today');
  await expect(cards.nth(0)).toContainText('uncommitted work');
  await expect(cards.nth(0)).toContainText('555 in 7 days');
  await expect(cards.nth(1)).not.toContainText('Where you left off');
  await expect(cards.nth(1)).toContainText('yesterday');
  await expect(cards.nth(1)).toContainText('222 in 7 days');
  await expect(cards.nth(2)).toContainText('fieldwork group');
  await expect(cards.nth(2)).toContainText('9 November');
  await expect(cards.nth(2)).toContainText('Run the task 17 review, then merge the branch.');
  await expect(cards.nth(2)).toContainText('3.3k in 7 days');

  const lineOf = (project: string) =>
    page.locator('.gl-hero-wrap').filter({ has: page.locator(`a[data-project="${project}"]`) }).locator('.gl-hero-session');
  await expect(page.locator('.gl-hero-session')).toHaveCount(3);
  await expect(lineOf('beta/tool')).toContainText('Last session on m1, today, 09:55');
  await expect(lineOf('beta/tool').getByRole('link', { name: /^Open\b/ })).toHaveAttribute('href', 'https://claude.ai/code/session_01FixtureToolSession');
  await expect(lineOf('alpha/renamed')).toContainText('Last session on m2, 9 November, 15:00');
  await expect(lineOf('alpha/renamed').getByRole('link')).toHaveCount(0);
  await expect(lineOf('alpha/fern')).toContainText('Last session on m1, yesterday, 11:02');
  await expect(lineOf('alpha/fern').getByRole('link')).toHaveCount(0);

  const stats = page.locator('.gl-stat');
  await expect(stats).toHaveText(['5projects tracked', '1parked, 1 done', '3need attention', '4.9ktokens in 7 days']);
});

test('the workbench lists the attention sentences for the git facts', async ({ page }) => {
  await page.goto('/');
  const bench = page.getByRole('region', { name: 'On your workbench' });
  await expect(bench.getByRole('button', { name: 'Needs attention 4' })).toHaveAttribute('aria-pressed', 'true');
  const rows = bench.getByRole('listitem');
  await expect(rows).toHaveCount(4);
  await expect(rows.nth(0)).toContainText('tool');
  await expect(rows.nth(0)).toContainText('3 files are uncommitted');
  await expect(rows.nth(1)).toContainText('renamed');
  await expect(rows.nth(1)).toContainText('No remote: 5 commits exist on this machine only');
  await expect(rows.nth(2)).toContainText('2 commits are not pushed, on m2-task-17');
  await expect(rows.nth(2).locator('code')).toHaveText('m2-task-17');
  await expect(rows.nth(3)).toContainText('The status is older than the last commit, by 1 day');

  await bench.getByRole('button', { name: 'Parked 1' }).click();
  await expect(rows).toHaveCount(1);
  await expect(rows.nth(0)).toContainText('Parked: Waiting for milestone 2');
  await bench.getByRole('button', { name: 'All 5' }).click();
  await expect(rows).toHaveCount(5);
});

test('the side cards show the last 30 days and the first ideas', async ({ page }) => {
  await page.goto('/');
  const month = page.getByRole('region', { name: 'Last 30 days' });
  const chart = month.getByRole('img', { name: 'Tokens per day, by project' });
  await expect(chart.locator('g[data-day]')).toHaveCount(30);
  await expect(chart.locator('g[data-day="2026-11-11"] rect')).toHaveCount(3);
  await expect(chart.locator('g[data-day="2026-11-09"] rect')).toHaveCount(0);
  await expect(month).toContainText('4.9k tokens, 44k cache reads');

  const later = page.getByRole('region', { name: 'Set aside for later' });
  await expect(later.getByRole('listitem')).toHaveText([
    'Compare two seed mixes on the same field planfern',
    'Write up the soil sampling measurementsfern',
    'A note on renaming without losing historyrenamed',
  ]);
  await later.getByRole('link', { name: 'All ideas' }).click();
  await expect(page).toHaveURL('/ideas');
});

test('projects are grouped, a row opens the project page, and back returns to the list', async ({ page }) => {
  await page.goto('/');
  await rail(page).getByRole('link', { name: 'Projects' }).click();
  await expect(page).toHaveURL('/projects');
  await expect(page.getByText('5 tracked, across 2 roots.')).toBeVisible();
  await expect(page.getByRole('region').getByRole('heading', { level: 2 })).toHaveText(['fieldwork', 'No group']);
  await expect(page.getByText('2 projects · 3.3k tokens in 30 days')).toBeVisible();
  await expect(page.locator('a[data-project]')).toHaveCount(5);
  await expect(page.locator('a[data-project="beta/legacy"]')).toContainText('Replaced by fern');

  await page.getByRole('group', { name: 'State' }).getByRole('button', { name: 'Parked 1' }).click();
  await expect(page.locator('a[data-project]')).toHaveCount(1);
  await expect(page.locator('a[data-project="alpha/fern-docs"]')).toContainText('Parked: Waiting for milestone 2');
  await page.getByRole('group', { name: 'State' }).getByRole('button', { name: 'All 5' }).click();

  await page.locator('a[data-project="alpha/fern"]').click();
  await expect(page).toHaveURL('/projects/alpha/fern');
  await expect(page.getByRole('heading', { level: 1 })).toHaveText('fern');
  await expect(page.locator('a[data-project]')).toHaveCount(0);
  await expect(rail(page).getByRole('link', { name: 'Projects' })).toHaveAttribute('aria-current', 'page');

  await page.goBack();
  await expect(page).toHaveURL('/projects');
  await expect(page.locator('a[data-project]')).toHaveCount(5);
});

test('a project page loads from its address, with its sections, git facts and usage', async ({ page }) => {
  await page.goto('/projects/alpha/fern');
  await expect(page.getByRole('heading', { level: 1 })).toHaveText('fern');
  const head = page.locator('header.gl-head');
  await expect(head).toContainText('stale status');
  await expect(head).toContainText('unpushed work');
  await expect(head).toContainText('last handoff 9 November');
  await expect(page.getByRole('region', { name: 'Next action' })).toContainText('Run the task 17 review, then merge the branch.');
  await expect(page.getByRole('region', { name: 'Where I stopped' }).locator('code')).toHaveText('m2-task-17');
  await expect(page.getByRole('region', { name: 'Remaining' }).getByRole('listitem')).toHaveText('Milestone 2, tasks 18 to 24');

  const git = page.getByRole('region', { name: 'Git, m1' });
  await expect(git).toContainText('yesterday, 09:00 abc1234');
  await expect(git.getByText('2 commits')).toHaveClass('gl-late');

  const sessions = page.getByRole('region', { name: 'Sessions' }).getByRole('listitem');
  await expect(sessions).toHaveCount(2);
  await expect(sessions.nth(0)).toContainText('m1');
  await expect(sessions.nth(0)).toContainText('Review task 17');
  await expect(sessions.nth(0)).toContainText('yesterday, 11:02');
  await expect(sessions.nth(1)).toContainText('m2');
  await expect(sessions.nth(1).getByRole('link', { name: /^Open\b/ })).toHaveAttribute('href', 'https://claude.ai/code/session_01FixtureFernSession');

  const usage = page.getByRole('region', { name: 'Usage' });
  await expect(usage).toContainText('14.4k');
  await expect(usage).toContainText('mostly model-a');
  const fold = usage.getByRole('button', { name: 'Day by day, per model (3 rows)' });
  await expect(usage.getByRole('table')).toHaveCount(0);
  await fold.click();
  await expect(usage.getByRole('row')).toHaveCount(4);
  await expect(usage.getByRole('row').nth(3)).toContainText('1 October');
  await expect(usage.getByRole('row').nth(3)).toContainText('11.1k');

  await head.getByRole('link', { name: 'fern-docs' }).click();
  await expect(page).toHaveURL('/projects/alpha/fern-docs');
  await expect(page.getByRole('region', { name: 'Next action' })).toContainText('Parked: Waiting for milestone 2');
  // A short window puts the heading below the fold, so only a scroll
  // brings it into view.
  await page.setViewportSize({ width: 1440, height: 120 });
  await page.getByRole('navigation', { name: 'Breadcrumb' }).getByRole('link', { name: 'fieldwork' }).click();
  await expect(page).toHaveURL('/projects#group-fieldwork');
  await expect(page.getByRole('heading', { level: 2, name: 'fieldwork' })).toBeInViewport({ ratio: 1 });
});

test('a replaced project and an unknown address', async ({ page }) => {
  await page.goto('/projects/beta/legacy');
  await expect(page.getByRole('heading', { level: 1 })).toHaveText('legacy');
  await expect(page.locator('header.gl-head').getByRole('link', { name: 'fern' })).toHaveAttribute('href', '/projects/alpha/fern');
  await expect(page.getByText('No machine has reported this working tree.')).toBeVisible();

  await page.goto('/projects/alpha/nope');
  await expect(page.getByRole('heading', { level: 1 })).toHaveText('No project at this address');
  await page.getByRole('link', { name: 'Back to Projects' }).click();
  await expect(page).toHaveURL('/projects');
});

test('usage counts renamed, untracked and outside work, for the chosen period', async ({ page }) => {
  await page.goto('/usage');
  await expect(page.getByRole('heading', { level: 1 })).toHaveText('4.9k tokens');
  await expect(page.getByText('13 October to today, 44k cache reads')).toBeVisible();
  await expect(page.locator('li[data-key="alpha/renamed"]')).toContainText('222 usage');
  await expect(page.locator('li[data-key="alpha/old-name"]')).toHaveCount(0);
  await expect(page.locator('li[data-key="other"]')).toContainText('outside the roots');
  await expect(page.locator('li[data-key="alpha/gone"]')).toContainText('no status file');

  const period = page.getByRole('group', { name: 'Period' });
  await period.getByRole('button', { name: '7 days' }).click();
  await expect(page.getByText('5 November to today, 44k cache reads')).toBeVisible();
  await expect(page.locator('li[data-key="alpha/gone"]')).toHaveCount(0);

  await period.getByRole('button', { name: 'everything' }).click();
  await expect(page.getByText('1 October to today, 144k cache reads')).toBeVisible();
  await expect(page.locator('li[data-key="alpha/fern"]')).toContainText('14.4k usage');
});

test('ideas are notes, filtered by project', async ({ page }) => {
  await page.goto('/ideas');
  await expect(page.getByRole('heading', { level: 1 })).toHaveText('Set aside for later');
  await expect(page.getByText('3 ideas, from 2 projects.')).toBeVisible();
  await expect(page.locator('.gl-note')).toHaveCount(3);
  await page.getByRole('group', { name: 'Project' }).getByRole('button', { name: 'renamed 1' }).click();
  await expect(page.locator('.gl-note')).toHaveText(['A note on renaming without losing historyrenamed']);
  await page.locator('.gl-note').getByRole('link', { name: 'renamed' }).click();
  await expect(page).toHaveURL('/projects/alpha/renamed');
});

test('the theme switches and is remembered', async ({ page }) => {
  await page.goto('/');
  const html = page.locator('html');
  const first = await html.getAttribute('data-theme');
  await rail(page).getByRole('button', { name: /theme$/ }).click();
  await expect(html).not.toHaveAttribute('data-theme', first!);
  const second = await html.getAttribute('data-theme');
  await page.reload();
  await expect(html).toHaveAttribute('data-theme', second!);
});

test('the figures of the project rows share one right edge, whatever the warnings', async ({ page }) => {
  await page.goto('/projects');
  await expect(page.locator('a[data-project]')).toHaveCount(5);
  // The row with two warnings is the one that can push its figure aside.
  await expect(page.locator('a[data-project="alpha/fern"] .gl-pill.gl-tone-late')).toHaveCount(2);
  const edges = await page
    .locator('a[data-project] .gl-row-figure')
    .evaluateAll((els) => els.map((el) => Math.round(el.getBoundingClientRect().right)));
  expect(edges).toHaveLength(5);
  expect(new Set(edges).size).toBe(1);
  // Each warning stays inside the column that holds it.
  const outside = await page.locator('a[data-project] .gl-row-side').evaluateAll((sides) =>
    sides.flatMap((side) => {
      const box = side.getBoundingClientRect();
      return [...side.querySelectorAll('.gl-pill')]
        .map((pill) => pill.getBoundingClientRect())
        .filter((r) => r.left < box.left - 0.5 || r.right > box.right + 0.5)
        .map((r) => [r.left, r.right, box.left, box.right]);
    }),
  );
  expect(outside).toEqual([]);
});

for (const width of [1440, 720]) {
  for (const path of ['/', '/projects', '/projects/alpha/fern', '/usage', '/ideas']) {
    test(`nothing scrolls sideways at ${width}px on ${path}`, async ({ page }) => {
      await page.setViewportSize({ width, height: 900 });
      await page.goto(path);
      await expect(page.getByRole('heading', { level: 1 })).toBeVisible();
      if (path === '/projects/alpha/fern') await page.getByRole('button', { name: /^Day by day/ }).click();
      // The document never scrolls: the main element does. Both are
      // measured, and the main element is the one that can fail.
      const sizes = await page.evaluate(() => {
        const main = document.querySelector('main')!;
        const doc = document.documentElement;
        return { main: [main.scrollWidth, main.clientWidth], doc: [doc.scrollWidth, doc.clientWidth] };
      });
      expect(sizes.main[1]).toBeGreaterThan(300);
      expect(sizes.main[0]).toBeLessThanOrEqual(sizes.main[1]!);
      expect(sizes.doc[0]).toBeLessThanOrEqual(sizes.doc[1]!);
    });
  }
}

test('a route change moves focus to the heading and names the page', async ({ page }) => {
  await page.goto('/');
  await expect(page).toHaveTitle('Overview | gellee');
  await page.getByRole('navigation', { name: 'Sections' }).getByRole('link', { name: 'Ideas' }).click();
  const heading = page.getByRole('heading', { level: 1, name: 'Set aside for later' });
  await expect(heading).toBeFocused();
  await expect(page).toHaveTitle('Ideas | gellee');
  // A link that is only muted text is underlined at rest.
  const project = page.locator('.gl-note a').first();
  expect(await project.evaluate((a) => getComputedStyle(a).textDecorationLine)).toBe('underline');
});

test('the workers page shows each machine from its last collection', async ({ page }) => {
  await page.goto('/');
  await rail(page).getByRole('link', { name: 'Workers' }).click();
  await expect(page).toHaveURL('/workers');
  await expect(page.getByRole('heading', { level: 1 })).toHaveText('Workers');
  await expect(page.getByText("Every figure is from each worker's last collection.")).toBeVisible();

  const m1 = page.getByRole('region', { name: 'm1' });
  const m2 = page.getByRole('region', { name: 'm2' });
  await expect(m1).toContainText('today, 10:00');
  await expect(m1).toContainText('Claude Code 2.1.323');
  await expect(m1).toContainText('1 version set aside');
  await expect(m1.getByRole('link', { name: /^Open\b/ })).toHaveAttribute('href', 'https://claude.ai/code/session_01FixtureToolSession');
  await expect(m1.getByRole('link', { name: /^Open\b/ })).toHaveAttribute('target', '_blank');
  await expect(m2).toContainText('not seen since 9 November');
  await expect(m2.getByText('behind')).toBeVisible();
  await expect(m2.getByText('sync stale')).toBeVisible();

  // m1's session outside the roots has ended: folded apart from the two running.
  await expect(m1.locator('.gl-session')).toHaveCount(2);
  const ended = m1.getByRole('button', { name: 'Ended (1)' });
  await expect(ended).toHaveAttribute('aria-expanded', 'false');
  await ended.click();
  await expect(m1.locator('.gl-session')).toHaveCount(3);
  await expect(m1.locator('.gl-session').last()).toContainText('other');

  await m1.getByRole('button', { name: 'Checkouts (3)' }).click();
  await expect(m1.getByRole('listitem').filter({ hasText: 'alpha/fern' })).toContainText('unpushed work');
  await m1.getByRole('link', { name: 'tool' }).click();
  await expect(page).toHaveURL('/projects/beta/tool');
});
