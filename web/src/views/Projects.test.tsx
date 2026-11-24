// Copyright The gellee Authors
// SPDX-License-Identifier: Apache-2.0

import { fireEvent, render, screen, waitFor, within } from '@testing-library/react';
import { afterEach, expect, test, vi } from 'vitest';
import { stubApi } from '../test/api';
import { project, usage } from '../test/shapes';
import { Projects } from './Projects';

const projects = [
  project({ key: 'alpha/fern', group: 'fieldwork', updated: '2026-11-09', next: 'Run the review.', usage_30: usage(3330), warnings: ['stale'] }),
  project({ key: 'alpha/fern-docs', state: 'parked', group: 'fieldwork', parked_reason: 'Waiting', updated: '2026-10-20', usage_30: usage(1000) }),
  project({ key: 'beta/kit', group: 'tools', updated: '2026-11-01', next: 'Cut a release.', usage_30: usage(9000) }),
  project({ key: 'beta/legacy', state: 'done', superseded_by: 'alpha/fern', updated: '2026-09-01' }),
  project({ key: 'beta/tool', updated: '2026-11-11', next: 'Add the table parser.', usage_30: usage(5550) }),
];

function stub() {
  stubApi({ '/api/v1/projects': { projects }, '/api/v1/health': { today: '2026-11-11' } });
}

const groups = () => screen.getAllByRole('region').map((r) => within(r).getByRole('heading', { level: 2 }).textContent);
const rows = (group: string) =>
  within(screen.getByRole('region', { name: group }))
    .getAllByRole('link')
    .map((a) => a.getAttribute('data-project'));
const row = (key: string) => document.querySelector(`a[data-project="${key}"]`)!;

afterEach(() => {
  window.history.replaceState(null, '', '/');
});

test('projects are rows under their group, groups ordered by their most recent handoff', async () => {
  stub();
  const go = vi.fn();
  render(<Projects go={go} />);
  expect(await screen.findByRole('heading', { level: 1 })).toHaveTextContent('Projects');
  expect(screen.getByText('5 tracked, across 2 roots.')).toBeVisible();
  expect(groups()).toEqual(['fieldwork', 'tools', 'No group']);
  expect(rows('fieldwork')).toEqual(['alpha/fern', 'alpha/fern-docs']);
  expect(rows('tools')).toEqual(['beta/kit']);
  expect(rows('No group')).toEqual(['beta/tool', 'beta/legacy']);
  expect(screen.getByText('2 projects · 4.3k tokens in 30 days')).toBeVisible();
  expect(screen.getByText('1 project · 9k tokens in 30 days')).toBeVisible();
  expect(within(screen.getByRole('region', { name: 'No group' })).getByText('2 projects')).toBeVisible();

  expect(row('alpha/fern')).toHaveTextContent('Run the review.');
  expect(row('alpha/fern')).toHaveTextContent('3.3k 30 days');
  // The figure sits over its label, as in the approved mockup.
  expect(within(row('alpha/fern') as HTMLElement).getByText('30 days')).toHaveClass('gl-under');
  expect(within(row('alpha/fern') as HTMLElement).getByText('stale status')).toBeVisible();
  expect(row('alpha/fern-docs')).toHaveTextContent('Parked: Waiting');
  expect(within(row('alpha/fern-docs') as HTMLElement).getByText('20 October')).toBeVisible();
  expect(row('beta/legacy')).toHaveTextContent('Replaced by fern');
  expect(within(row('beta/legacy') as HTMLElement).getByText('1 September')).toBeVisible();
  expect(within(row('beta/tool') as HTMLElement).getByText('today')).toBeVisible();

  fireEvent.click(row('beta/kit'));
  expect(go).toHaveBeenCalledWith('/projects/beta/kit');
});

test('the state filter counts and narrows, and the counts follow the root', async () => {
  stub();
  render(<Projects go={vi.fn()} />);
  const states = within(await screen.findByRole('group', { name: 'State' }));
  expect(states.getAllByRole('button').map((b) => b.textContent)).toEqual(['All 5', 'Active 3', 'Parked 1', 'Done 1']);
  expect(states.getByRole('button', { name: 'All 5' })).toHaveAttribute('aria-pressed', 'true');

  fireEvent.click(states.getByRole('button', { name: 'Parked 1' }));
  expect(groups()).toEqual(['fieldwork']);
  expect(rows('fieldwork')).toEqual(['alpha/fern-docs']);
  expect(screen.getByText('1 project · 1k tokens in 30 days')).toBeVisible();

  fireEvent.click(states.getByRole('button', { name: 'All 5' }));
  fireEvent.change(screen.getByLabelText('Root'), { target: { value: 'beta' } });
  expect(states.getAllByRole('button').map((b) => b.textContent)).toEqual(['All 3', 'Active 2', 'Parked 0', 'Done 1']);
  expect(groups()).toEqual(['tools', 'No group']);

  fireEvent.click(states.getByRole('button', { name: 'Parked 0' }));
  expect(screen.getByText('No project matches these filters.')).toBeVisible();
  expect(screen.queryAllByRole('region')).toHaveLength(0);
});

test('sorting by usage reorders the rows and the groups', async () => {
  stub();
  render(<Projects go={vi.fn()} />);
  await screen.findByRole('heading', { level: 1 });
  fireEvent.change(screen.getByLabelText('Sort'), { target: { value: 'usage' } });
  expect(groups()).toEqual(['tools', 'fieldwork', 'No group']);
  expect(rows('No group')).toEqual(['beta/tool', 'beta/legacy']);
});

test('an address that names a group scrolls to it', async () => {
  const seen: string[] = [];
  Element.prototype.scrollIntoView = function (this: Element) {
    seen.push(this.id);
  };
  window.history.replaceState(null, '', '/projects#group-tools');
  stub();
  render(<Projects go={vi.fn()} />);
  await screen.findByRole('heading', { level: 1 });
  await waitFor(() => expect(seen).toEqual(['group-tools']));
});

test('a failed request says so', async () => {
  stubApi({});
  render(<Projects go={vi.fn()} />);
  expect(await screen.findByRole('alert')).toHaveTextContent('The project list could not be loaded: 404 Not Found');
});

const odd = [
  project({ key: 'alpha/a', group: 'data fern', updated: '2026-11-09' }),
  project({ key: 'alpha/b', group: 'a&b?c', updated: '2026-11-08' }),
  project({ key: 'alpha/c', group: 'tools', updated: '2026-11-07' }),
];

test('a group whose name needs encoding is a region with its own name', async () => {
  stubApi({ '/api/v1/projects': { projects: odd }, '/api/v1/health': { today: '2026-11-11' } });
  render(<Projects go={vi.fn()} />);
  await screen.findByRole('heading', { level: 1 });
  expect(rows('data fern')).toEqual(['alpha/a']);
  expect(rows('a&b?c')).toEqual(['alpha/b']);
  expect(rows('tools')).toEqual(['alpha/c']);
});

test('an address that names a group needing encoding scrolls to it', async () => {
  const seen: string[] = [];
  Element.prototype.scrollIntoView = function (this: Element) {
    seen.push(this.textContent ?? '');
  };
  window.history.replaceState(null, '', '/projects#group-data%20fern');
  stubApi({ '/api/v1/projects': { projects: odd }, '/api/v1/health': { today: '2026-11-11' } });
  render(<Projects go={vi.fn()} />);
  await screen.findByRole('heading', { level: 1 });
  await waitFor(() => expect(seen).toEqual(['data fern']));
});

test('a broken escape in the address leaves the list showing', async () => {
  window.history.replaceState(null, '', '/projects#%');
  stubApi({ '/api/v1/projects': { projects: odd }, '/api/v1/health': { today: '2026-11-11' } });
  render(<Projects go={vi.fn()} />);
  expect(await screen.findByRole('heading', { level: 1 })).toHaveTextContent('Projects');
  expect(groups()).toEqual(['data fern', 'a&b?c', 'tools']);
});

test('sorting by usage reorders the rows inside a group, not only the groups', async () => {
  // In the group, the most recent handoff has the least usage.
  const mixed = [
    project({ key: 'alpha/new', group: 'pair', updated: '2026-11-10', usage_30: usage(100) }),
    project({ key: 'alpha/old', group: 'pair', updated: '2026-10-01', usage_30: usage(900) }),
    project({ key: 'beta/solo', group: 'single', updated: '2026-11-09', usage_30: usage(500) }),
  ];
  stubApi({ '/api/v1/projects': { projects: mixed }, '/api/v1/health': { today: '2026-11-11' } });
  render(<Projects go={vi.fn()} />);
  await screen.findByRole('heading', { level: 1 });
  expect(groups()).toEqual(['pair', 'single']);
  expect(rows('pair')).toEqual(['alpha/new', 'alpha/old']);
  fireEvent.change(screen.getByLabelText('Sort'), { target: { value: 'usage' } });
  expect(groups()).toEqual(['pair', 'single']);
  expect(rows('pair')).toEqual(['alpha/old', 'alpha/new']);
});

test('with no project the heading does not count roots', async () => {
  stubApi({ '/api/v1/projects': { projects: [] }, '/api/v1/health': { today: '2026-11-11' } });
  render(<Projects go={vi.fn()} />);
  await screen.findByRole('heading', { level: 1 });
  expect(screen.getByText('0 tracked.')).toBeVisible();
  expect(document.body).not.toHaveTextContent('roots');
});

test('the rows share one unit: tokens as soon as one project has an unpriced model', async () => {
  const mixed = projects.map((p) => (p.key === 'beta/kit' ? p : { ...p, usage_30: { ...p.usage_30, cost: 1.5 } }));
  stubApi({ '/api/v1/projects': { projects: mixed }, '/api/v1/health': { today: '2026-11-11' } });
  render(<Projects go={vi.fn()} />);
  await screen.findByRole('heading', { level: 1 });
  expect(row('alpha/fern')).toHaveTextContent('3.3k 30 days');
  expect(row('beta/kit')).toHaveTextContent('9k 30 days');
  expect(document.body).not.toHaveTextContent('$');
});

test('the rows show a cost when every project that has usage is priced', async () => {
  const nothing = { ...usage(0), messages: 0 };
  const priced = projects.map((p) =>
    p.key === 'beta/legacy' ? { ...p, usage_30: nothing } : { ...p, usage_30: { ...p.usage_30, cost: 1.5 } },
  );
  stubApi({ '/api/v1/projects': { projects: priced }, '/api/v1/health': { today: '2026-11-11' } });
  render(<Projects go={vi.fn()} />);
  await screen.findByRole('heading', { level: 1 });
  expect(row('alpha/fern')).toHaveTextContent('$1.50 30 days');
  expect(row('beta/legacy')).toHaveTextContent('$0.00 30 days');
});
