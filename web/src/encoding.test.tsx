// Copyright The gellee Authors
// SPDX-License-Identifier: Apache-2.0

import { fireEvent, render, screen } from '@testing-library/react';
import { afterEach, expect, test, vi } from 'vitest';
import { stubApi } from './test/api';
import { project, usage } from './test/shapes';
import { Bench } from './views/Bench';
import { Ideas } from './views/Ideas';
import { ProjectPage } from './views/ProjectPage';
import { Projects } from './views/Projects';
import { UsageView } from './views/UsageView';
import { Workbench } from './views/Workbench';

const awkward = [
  project({ key: 'alpha/c#', warnings: ['unpushed'], git: { m1: { branch: 'main', head: 'abc1234', last_commit: '2026-11-09T09:00:00+01:00', remote: true, ahead: 2, dirty: 0 } } }),
  project({ key: 'alpha/100%', warnings: [] }),
];
const hrefs = { 'alpha/c#': '/projects/alpha/c%23', 'alpha/100%': '/projects/alpha/100%25' };
const go = vi.fn();

function expectLinks(root: HTMLElement | Document = document) {
  for (const [key, href] of Object.entries(hrefs)) {
    const link = root.querySelector(`a[data-project="${key}"]`);
    expect(link, key).not.toBeNull();
    expect(link).toHaveAttribute('href', href);
  }
}

afterEach(() => vi.unstubAllGlobals());

test('the bench links keys that need encoding', () => {
  render(<Bench projects={awkward} today="2026-11-11" shared={new Set()} go={go} />);
  expectLinks();
});

test('the workbench links keys that need encoding in every choice', () => {
  render(<Workbench projects={awkward} today="2026-11-11" shared={new Set()} go={go} />);
  expect(document.querySelector('a[data-project="alpha/c#"]')).toHaveAttribute('href', '/projects/alpha/c%23');
  fireEvent.click(screen.getByRole('button', { name: /^All/ }));
  expectLinks();
});

test('the project list links keys that need encoding', async () => {
  stubApi({ '/api/v1/projects': { projects: awkward }, '/api/v1/health': { today: '2026-11-11' } });
  render(<Projects go={go} />);
  await screen.findByRole('heading', { level: 1 });
  expectLinks();
});

test('an idea links its project when the key needs encoding', async () => {
  stubApi({ '/api/v1/ideas': { ideas: [{ project: 'alpha/c#', text: 'One.' }, { project: 'alpha/100%', text: 'Two.' }] } });
  render(<Ideas go={go} />);
  expect(await screen.findByRole('link', { name: 'c#' })).toHaveAttribute('href', '/projects/alpha/c%23');
  expect(screen.getByRole('link', { name: '100%' })).toHaveAttribute('href', '/projects/alpha/100%25');
});

test('usage links a tracked project when the key needs encoding', async () => {
  stubApi({
    '/api/v1/usage': {
      from: '2026-10-13', to: '2026-11-11', total: usage(10), days: [], models: [],
      projects: [{ key: 'alpha/c#', tracked: true, usage: usage(5) }, { key: 'alpha/100%', tracked: true, usage: usage(5) }],
    },
  });
  render(<UsageView go={go} />);
  expect(await screen.findByRole('link', { name: 'c#' })).toHaveAttribute('href', '/projects/alpha/c%23');
  expect(screen.getByRole('link', { name: '100%' })).toHaveAttribute('href', '/projects/alpha/100%25');
});

test('the project page requests an encoded path and links its group and its replacement', async () => {
  const p = project({ key: 'alpha/c#', state: 'done', group: 'a&b?c', superseded_by: 'alpha/100%' });
  stubApi({
    '/api/v1/projects/alpha/c%23': { project: p, group_members: ['alpha/100%'], days: [], sessions: [] },
    '/api/v1/health': { today: '2026-11-11' },
  });
  render(<ProjectPage projectKey="alpha/c#" go={go} />);
  const links = await screen.findAllByRole('link', { name: '100%' });
  expect(links).toHaveLength(2);
  for (const l of links) expect(l).toHaveAttribute('href', '/projects/alpha/100%25');
});

test('a replacement without a root is plain text', async () => {
  const p = project({ key: 'alpha/old', state: 'done', superseded_by: 'solo' });
  stubApi({
    '/api/v1/projects/alpha/old': { project: p, group_members: [], days: [], sessions: [] },
    '/api/v1/health': { today: '2026-11-11' },
  });
  render(<ProjectPage projectKey="alpha/old" go={go} />);
  await screen.findByRole('heading', { level: 1 });
  expect(screen.getByText(/replaced by solo/)).toBeVisible();
  expect(screen.queryByRole('link', { name: 'solo' })).toBeNull();
});
